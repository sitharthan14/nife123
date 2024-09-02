package duplo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	appDeployments "github.com/nifetency/nife.io/internal/app_deployments"
	apprelease "github.com/nifetency/nife.io/internal/app_release"
	_helper "github.com/nifetency/nife.io/pkg/helper"
)

type PollingDuplo struct {
	AppName      string
	Status       string
	UserId       string
	Image        string
	InternalPort int
	ExternalPort int
	BaseAddress  string
	TenantId     string
	Region       string
	RegionName   string
}

func (poll *PollingDuplo) PollingDuplo() error {

	CurrentStatus := -2.0

	count := 0


	regions := make([]*model.Region, 0)

	lat := 1.0
	long := 1.0
	regions = append(regions, &model.Region{Code: &poll.Region, Name: &poll.RegionName, Latitude: &lat, Longitude: &long})

	err := UpdateRegionStatus(poll.AppName, regions)

	for i := 0; ; i++ {

		response, err := DuploHTTPGetRequest(poll.BaseAddress+"/subscriptions/", "GetPods", poll.TenantId)

		if err != nil {
			log.Println(err)
			return err
		}

		defer response.Body.Close()

		body, _ := ioutil.ReadAll(response.Body)

		var result []map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err != nil {
			log.Println(err)
			return err
		}

		if result == nil {
			continue
		}

		for _, definition := range result {
			CurrentStatus, err = helper.GetPodStatus(definition, poll.AppName)
			if err != nil {
				log.Println(err)
				return err
			}
			if CurrentStatus == 1 {
				helper.UpdateDuploId(definition, poll.AppName, poll.TenantId)
			}
			if CurrentStatus > 7 || CurrentStatus == 1 || CurrentStatus > 1 && CurrentStatus <= 6 {
				break
			}

		}
		if CurrentStatus > 7 || CurrentStatus == 1 {
			break
		}
		time.Sleep(time.Second * 15)
		count++
		if count > 40 {
			return err
		}
	}

	if CurrentStatus > 7 {

		response, err := DuploHTTPGetRequest(poll.BaseAddress+"/subscriptions/", "GetFaultsByTenant", poll.TenantId)

		if err != nil {
			return err
		}

		defer response.Body.Close()

		body, _ := ioutil.ReadAll(response.Body)

		var result []FaultByTenant
		err = json.Unmarshal(body, &result)

		if err != nil {
			return err
		}

		length := len(poll.AppName)
		splitAppName := ""
		description := ""
		for _, j := range result {
			splitAppName = j.ResourceName[0:length]
			description = j.Description
		}

		if splitAppName == poll.AppName && splitAppName != "" {
			err = CreateDuploStatus(poll.AppName, "container_failed", poll.UserId, description, 0, count)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("Something wrong in fault by tenant")
		}
		return nil
	}

	err = CreateDuploStatus(poll.AppName, "container_created", poll.UserId, "", 2, count)
	if err != nil {
		return err
	}

	LB := LoadBalancerDetails{
		AppName:      poll.AppName,
		InternalPort: poll.InternalPort,
		ExternalPort: 443,
		UserId:       poll.UserId,
		BaseAddress:  poll.BaseAddress,
		TenantId:     poll.TenantId,
	}

	if poll.Status != "Active" {

		time.Sleep(time.Second*10)

		err = LB.CreateLoadBalancer()
		if err != nil {
			log.Println(err)
			return fmt.Errorf(err.Error())
		}
	}

	version := "1"

	if poll.Status == "Active" {

		LB := LoadBalancerDetails{
			AppName:      poll.AppName,
			InternalPort: poll.InternalPort,
			ExternalPort: poll.ExternalPort,
			UserId:       poll.UserId,
			BaseAddress:  poll.BaseAddress,
			TenantId:     poll.TenantId,
		}

		err = LB.UpdateLoadBalancer()

		if err != nil {
			log.Println(err)
			return fmt.Errorf(err.Error())
		}

		releases, err := _helper.GetAppRelease(poll.AppName, "active")

		if err != nil {
			log.Println(err)
			return fmt.Errorf(err.Error())
		}

		relversion, _ := strconv.Atoi(releases.Version)

		version = fmt.Sprintf("%v", (relversion + 1))

		err = _helper.UpdateAppRelease("inactive", releases.Id)

		if err != nil {
			log.Println(err)
			return fmt.Errorf(err.Error())
		}
		vers,_ := strconv.Atoi(version)
		 err = UpdateVersions(poll.AppName, vers)
		 if err != nil {
			return fmt.Errorf(err.Error())
		}

		appDeployment,err := _helper.GetDeploymentsByReleaseId(poll.AppName,releases.Id)
		
		if err != nil {
			log.Println(err)
			return fmt.Errorf(err.Error())
		}
        for _ ,appDep := range *appDeployment{

            err = UpdateDeploymentsRecord("destroyed",appDep.AppId , "", time.Now())
			if err != nil {
				log.Println(err)
				return fmt.Errorf(err.Error())
			}
		} 
        
		

		time.Sleep(30 * time.Second)

	}

	time.Sleep(time.Second * 10)

	LBStatus := LoadBalancerDetails{
		AppName:      poll.AppName,
		UserId:       poll.UserId,
		BaseAddress:  poll.BaseAddress,
		InternalPort: poll.InternalPort,
		ExternalPort: poll.ExternalPort,
		TenantId:     poll.TenantId,
	}

	DNS, err := LBStatus.GetLoadBalancerStatus()

	if err != nil {
		log.Println(err)
		return fmt.Errorf(err.Error())
	}

	

	id := uuid.New().String()

	release := apprelease.AppRelease{
		Id:        id,
		AppId:     poll.AppName,
		Version:   version,
		Status:    "active",
		ImageName: poll.Image,
		UserId:    poll.UserId,
		CreatedAt: time.Now(),
		Port:      poll.InternalPort,
	}

	err = _helper.CreateAppRelease(release)

	if err != nil {
		log.Println(err)
		return fmt.Errorf(err.Error())
	}
	deploy := appDeployments.AppDeployments{
		Id:            uuid.NewString(),
		AppId:         poll.AppName,
		Region_code:   *regions[0].Code,
		Status:        "running",
		Deployment_id: "",
		Release_id:    id,
		Port:          strconv.Itoa(poll.InternalPort),
		App_Url:       DNS,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		ELBRecordName: "",
		ELBRecordId:   "",
	}
	err = _helper.CreateDeploymentsRecord(deploy)
	if err != nil {
		log.Println(err)
		return fmt.Errorf(err.Error())
	}

	 err = UpdateImageandPort(poll.AppName,poll.Image, poll.InternalPort)
	 if err != nil {
		return fmt.Errorf(err.Error())
	}



	// regions := make([]*model.Region, 0)

	// lat := 1.0
	// long := 1.0
	// regions = append(regions, &model.Region{Code: &poll.Region, Name: &poll.RegionName, Latitude: &lat, Longitude: &long})

	// err = UpdateRegionStatus(poll.AppName, regions)

	if err != nil {
		log.Println(err)
		return fmt.Errorf(err.Error())
	}

	return nil
}

