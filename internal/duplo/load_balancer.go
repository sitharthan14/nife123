package duplo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/nifetency/nife.io/helper"
	// "github.com/nifetency/nife.io/pkg/aws"
)

type LoadBalancerDetails struct {
	InternalPort int
	ExternalPort int
	AppName      string
	UserId       string
	BaseAddress  string
	TenantId     string
}

func (LB *LoadBalancerDetails) CreateLoadBalancer() error {

	// certificateArn := os.Getenv("DUPLO_BYOH_CERTIFICATE")
	certificateArn := os.Getenv("DUPLO_K8S_CERTIFICATE")

	payLoad := LoadBalancer{
		HealthCheckConfig:         "{}",
		LbType:                    1,
		Port:                      LB.InternalPort,
		ExternalPort:              443,
		IsInternal:                false,
		IsNative:                  false,
		HealthCheckUrl:            "/",
		Protocol:                  "http",
		CertificateArn:            certificateArn,
		ReplicationControllerName: LB.AppName,
	}

	postBody, _ := json.Marshal(payLoad)

	response, err := DuploHttpPostRequest(postBody, LB.BaseAddress+"/subscriptions/", "POST", "LBConfigurationUpdate", LB.TenantId)

	if err != nil {
		return err
	}

	body, _ := ioutil.ReadAll(response.Body)
	bodyResponse := string([]byte(body))

	if bodyResponse != "null" && response.StatusCode != 200 {
		return fmt.Errorf("Something wrong in creating load balancer")
	}
	err = CreateDuploStatus(LB.AppName, "service_creating", LB.UserId, "", 3, 0)
	if err != nil {
		return err
	}
	time.Sleep(time.Second*10)
	return nil
}

func (LB *LoadBalancerDetails) UpdateLoadBalancer() error {

	prevPort, err := GetPort(LB.TenantId, LB.BaseAddress, LB.AppName)

	if err != nil {
		return err
	}

	state := "delete"

	newInternalPort := strconv.Itoa(LB.InternalPort)

	for i := 0; i < 2; i++ {

		payload := UpdateLBConfig{
			ReplicationControllerName: LB.AppName,
			LBType:                    1,
			Protocol:                  "http",
			Port:                      prevPort,
			ExternalPort:              LB.ExternalPort,
			State:                     state,
			IsInternal:                true,
			HealthCheckUrl:            "/",
			CertificateArn:            "arn:aws:acm:us-west-2:430786739711:certificate/6cd2b0cc-0703-4806-848a-236f0beab49c",
		}

		postBody, _ := json.Marshal(payload)

		response, err := DuploHttpPostRequest(postBody, LB.BaseAddress+"/subscriptions/", http.MethodPost, "LBConfigurationUpdate", LB.TenantId)

		if err != nil {
			return err
		}
		body, _ := ioutil.ReadAll(response.Body)
		bodyResponse := string([]byte(body))

		if bodyResponse != "null" && response.StatusCode != 200 {
			return fmt.Errorf("Something wrong in updating load balancer")
		}

		state = ""

		prevPort = newInternalPort
	}

	err = CreateDuploStatus(LB.AppName, "service_creating", LB.UserId, "", 3, 0)

	if err != nil {
		return err
	}

	return nil

}

func (Lb *LoadBalancerDetails) GetLoadBalancerStatus() (string, error) {

	LBStatus := ""

	count := 0

	for i := 0; ; i++ {

		response, err := DuploHTTPGetRequest(Lb.BaseAddress+"/subscriptions/", "GetLbDetailsInService/"+Lb.AppName, Lb.TenantId)

		if err != nil {
			return "", err
		}

		defer response.Body.Close()
		body, _ := ioutil.ReadAll(response.Body)

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err != nil {
			return "", err
		}

		if result == nil {
			continue
		}

		LBStatus = helper.GetLBStatus(result)
		if err != nil {
			log.Println(err)
			return "", err
		}

		if LBStatus == "active" {
			break
		}
		time.Sleep(time.Second * 15)
		count++
		if count > 40 {
			break
		}
	}

	err := CreateDuploStatus(Lb.AppName, "service_created", Lb.UserId, "", 4, count)

	if err != nil {
		return "", err
	}

	resp, err := DuploHTTPGetRequest(Lb.BaseAddress+"/subscriptions/", "GetReplicationControllers", Lb.TenantId)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var result []map[string]interface{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	if result == nil {
		return "", fmt.Errorf("SomeThing went wrong in DNS")
	}

	DNS := ""

	for _, definiton := range result {
		DNS = helper.GetDNS(definiton, Lb.AppName)
	}

	if DNS == "" {
		return "", fmt.Errorf("something wrong in Get load balancer status")
	}

	// route53DNS,_,_,err:=  aws.CreateOrDeleteRecordSetRoute53("mpl5.apps.nifetency.com", DNS, "AS","",false,"latency")

	// if err != nil {
	// 	return "",err
	// }

	err = UpdateHostName(Lb.AppName, DNS)
	if err != nil {
		return "", err
	}
	return DNS, nil

}
