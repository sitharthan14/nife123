package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/internal/duplo"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	secretregistry "github.com/nifetency/nife.io/internal/secret_registry"
)

type DuploDetails struct {
	AppName             string
	Image               string
	UserId              string
	EnvArgs             string
	AgentPlatForm       int
	InternalPort        int
	ExternalPort        int
	ExternalBaseAddress string
	TenantId            string
	AllocationTag       string
	Cloud               int
	Region              string
	RegionName          string
	SecretRegistryID    string
	VolumeId            string
}

type UpdateDuplo struct {
	AppName             string
	Image               string
	Status              string
	UserId              string
	InternalPort        int
	ExternalPort        int
	AgentPlatForm       int
	ExternalBaseAddress string
	TenantId            string
	SecretRegistyId     string
	SecretType          string
}

func (data *DuploDetails) CreateDuploService() error {

	volume, err := GetVolumes(data.AppName)
    if err != nil {
		return fmt.Errorf(err.Error())
	}

	if data.SecretRegistryID != "" {

		privateRegistry, err := secretregistry.GetSecretDetails(data.SecretRegistryID, "")
		if err != nil {
			return err
		}
		secDetails := duplo.CreateSecret{
			SecretName: *privateRegistry.Name,
			SecretType: *privateRegistry.SecretType,
		}

		err = secDetails.CreateSecret(*privateRegistry.UserName, *privateRegistry.PassWord, data.ExternalBaseAddress, data.TenantId, *privateRegistry.URL)
		if err != nil {
			return fmt.Errorf(err.Error())
		}
	}


	payLoad := duplo.CreateService{
		Replicas:          1,
		Cloud:             data.Cloud,
		AgentPlatform:     data.AgentPlatForm,
		Name:              data.AppName,
		Volumes:           volume,
		AllocationTags:    data.AllocationTag,
		DockerImage:       data.Image,
		TenantId:          data.TenantId,
		NetworkId:         "default",
		OtherDockerConfig: data.EnvArgs,
	}

	postBody, _ := json.Marshal(payLoad)

	response, err := duplo.DuploHttpPostRequest(postBody, data.ExternalBaseAddress+"/subscriptions/", "POST", "ReplicationControllerUpdate", data.TenantId)

	if err != nil {
		log.Println(err)
		return err
	}

	body, _ := ioutil.ReadAll(response.Body)
	bodyResponse := string([]byte(body))
	if bodyResponse != "null" && response.StatusCode != 200 {
		return fmt.Errorf("Something went wrong to create EC2 service %s", bodyResponse)
	}

	if response.StatusCode == 200 {
		err = duplo.CreateDuploStatus(data.AppName, "container_creating", data.UserId, "", 1, 0)
		if err != nil {
			return fmt.Errorf(err.Error())
		}

		poll := duplo.PollingDuplo{
			AppName:      data.AppName,
			Status:       "New",
			UserId:       data.UserId,
			Image:        data.Image,
			InternalPort: data.InternalPort,
			ExternalPort: data.ExternalPort,
			BaseAddress:  data.ExternalBaseAddress,
			TenantId:     data.TenantId,
			Region:       data.Region,
			RegionName:   data.RegionName,
		}

		go poll.PollingDuplo()
		return nil
	}

	return nil

}

func (d *UpdateDuplo) UpdateDuploApp() error {

	if d.SecretRegistyId != "" {

		registry, err := secretregistry.GetSecretDetails(d.SecretRegistyId, "")
		if err != nil {
			return err
		}

		delSecret := SecretElement{
			RegistryName:        *registry.Name,
			TenantId:            d.TenantId,
			ExternalBaseAddress: d.ExternalBaseAddress,
		}
		err = delSecret.CheckandDeleteSecret()
		if err != nil {
			return err
		}

		secDetails := duplo.CreateSecret{
			SecretName: *registry.RegistryName,
			SecretType: *registry.SecretType,
		}

		err = secDetails.CreateSecret(*registry.UserName, *registry.PassWord, d.ExternalBaseAddress, d.TenantId, *registry.URL)

		if err != nil {
			return err
		}

	}

	postBody, _ := json.Marshal(map[string]interface{}{
		"Name":          d.AppName,
		"Image":         d.Image,
		"AgentPlatform": strconv.Itoa(d.AgentPlatForm),
	})

	response, err := duplo.DuploHttpPostRequest(postBody, d.ExternalBaseAddress+"/subscriptions/", "POST", "ReplicationControllerChange", d.TenantId)

	if err != nil {
		log.Println(err)
		return err
	}

	body, _ := ioutil.ReadAll(response.Body)
	bodyResponse := string([]byte(body))
	if bodyResponse != "null" && response.StatusCode != 200 {
		return fmt.Errorf("Something went wrong to update service %s", bodyResponse)
	}

	if response.StatusCode == 200 {
		err = duplo.CreateDuploStatus(d.AppName, "container_creating", d.UserId, "", 1, 0)
		if err != nil {
			return fmt.Errorf(err.Error())
		}

		poll := duplo.PollingDuplo{
			AppName:      d.AppName,
			Status:       "Active",
			UserId:       d.UserId,
			Image:        d.Image,
			InternalPort: d.InternalPort,
			ExternalPort: d.ExternalPort,
			BaseAddress:  d.ExternalBaseAddress,
			TenantId:     d.TenantId,
		}
		go poll.PollingDuplo()
		return nil
	}

	return nil
}

func GetDuploDeployStatus(appName string) ([]*model.DuploDeployOutput, error) {

	var duploDetails []*model.DuploDeployOutput

	QueryString := "select id, status, user_id, info, created_at, updated_at,progress,poll_count from duplo_deploy_status where app_name = ? and is_active = 1 ORDER BY created_at DESC"
	selDB, err := database.Db.Query(QueryString, appName)
	if err != nil {
		return []*model.DuploDeployOutput{}, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var duplo model.DuploDeployOutput
		err = selDB.Scan(&duplo.ID, &duplo.Status, &duplo.UserID, &duplo.Info, &duplo.CreatedAt, &duplo.UpdatedAt, &duplo.Progress, &duplo.PollCount)
		if err != nil {
			return []*model.DuploDeployOutput{}, err
		}

		duploDetails = append(duploDetails, &duplo)
	}

	return duploDetails, nil
}

func GetDuploLog(appName, userId string) (model.Duplolog, error) {

	appDetails, err := GetApp(appName, userId)
	if err != nil {
		log.Println(err)
		return model.Duplolog{}, err
	}

	tenantId := os.Getenv("TENANT_ID")

	duploUrl := os.Getenv("Duplo_Deploy_URL")

	postBody, _ := json.Marshal(map[string]interface{}{
		"HostName": appDetails.HostID,
		"DockerId": appDetails.DockerID,
		"Tail":     100,
	})

	response, err := duplo.DuploHttpPostRequest(postBody, duploUrl, "POST", "findContainerLogs", tenantId)

	if err != nil {
		return model.Duplolog{}, err
	}

	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)

	var result model.Duplolog
	json.Unmarshal(body, &result)

	return result, nil

}

func DeleteDuploApp(tenantId, duploUrl, appName string) error {

	res, err := duplo.DuploHttpPostRequest(nil, duploUrl+"/v2/subscriptions/", "DELETE", "ReplicationControllerApiV2/"+appName, tenantId)

	if err != nil {
		return fmt.Errorf(err.Error())
	}

	fmt.Println(res.Status)
	return nil

}

func GetVolumes(appName string) (string ,error) {
	var volume []byte
	getAppidByName, err := GetAppIdByName(appName)
	if err != nil {
		return "",fmt.Errorf(err.Error())
	}
	duploVol, err := GetVolumeDetailsByAppName(getAppidByName)
	if err != nil {
		return "",fmt.Errorf(err.Error())
	}
    
	for _, vol := range duploVol {
		if *vol.IsHostVolume == false {
			var volgrp []map[string]interface{}
			for _, duplovolume := range duploVol {
				volumes := map[string]interface{}{
					"AccessMode": "ReadWriteOnce",
					"Name": duplovolume.Name,
					"Path": duplovolume.Path,
					"Size": *duplovolume.Size+"Gi",
				}
				volgrp = append(volgrp, volumes)
				volume, _ = json.Marshal(volgrp)
			}
				
		} else if *vol.IsHostVolume == true && *vol.IsRead == false {

			var volgrp []map[string]interface{}
			for _, duplovolume := range duploVol {
				volumes := map[string]interface{}{
					"Name": duplovolume.Name,
					"Path": duplovolume.ContainerPath,
					"Spec": map[string]interface{}{
						"HostPath": map[string]interface{}{
							"Path": duplovolume.HostPath,
						},
					},
				}
				volgrp = append(volgrp, volumes)
				volume, _ = json.Marshal(volgrp)
				
			}

		} else {

			var volgrp []map[string]interface{}
			for _, duplovolume := range duploVol {
				volumes := map[string]interface{}{
					"Name":     duplovolume.Name,
					"Path":     duplovolume.ContainerPath,
					"ReadOnly": true,
					"Spec": map[string]interface{}{
						"HostPath": map[string]interface{}{
							"Path": duplovolume.HostPath,
						},
					},
				}
				volgrp = append(volgrp, volumes)
				volume, _ = json.Marshal(volgrp)
				
			}

		}
	}
	return string(volume), nil
}
