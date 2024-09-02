package helper

import (
	"fmt"
	"log"

	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func GetPodStatus(Definition map[string]interface{}, userAppName string) (float64, error) {
	podAppName, _ := Definition["Name"].(interface{}).(string)
	if podAppName == userAppName {
		status, _ := Definition["CurrentStatus"].(interface{}).(float64)
		return status, nil
	}
	return -1, nil
}

func GetPort(Definition map[string]interface{}, userAppName string) (string, error) {
	podAppName, _ := Definition["ReplicationControllerName"].(interface{}).(string)
	if podAppName == userAppName {
		port, _ := Definition["Port"].(interface{}).(string)
		return port, nil
	}
	return "", nil
}

type containerDetails struct {
	InstanceId string
	HostId     string
	DockerId   string
	TenantId   string
}

func UpdateDuploId(Definition map[string]interface{}, appName, tenantId string) {

	podAppName, _ := Definition["Name"].(interface{}).(string)
	if podAppName == appName {
		instanceId, _ := Definition["InstanceId"].(interface{}).(string)
		hostId, _ := Definition["Host"].(interface{}).(string)
		fmt.Println(instanceId, hostId)
		container := Definition["Containers"].([]interface{})
		container0 := container[0].(map[string]interface{})
		dockerid := container0["DockerId"].(string)

		result := containerDetails{
			InstanceId: instanceId,
			HostId:     hostId,
			DockerId:   dockerid,
			TenantId:   tenantId,
		}

		err := UpdateDuploDetails(appName, result)
		if err != nil {
			return
		}

		return
	}

}

func GetFaultTenant(Definition map[string]interface{}, userAppName string) (string, error) {
	resourceName, _ := Definition["ResourceName"].(interface{}).(string)

	length := len(userAppName)
	splitAppName := resourceName[0:length]

	if splitAppName == userAppName {
		status, _ := Definition["Description"].(interface{}).(string)
		return status, nil
	}
	return "", nil
}

func GetLBStatus(Definition map[string]interface{}) string {
	LBStatus, _ := Definition["State"].(map[string]interface{})
	serializeDef := LBStatus["Code"].(map[string]interface{})
	status := serializeDef["Value"].(string)
	return status
}

func GetDNS(PayLoad map[string]interface{}, appName string) string {
	serviceName := PayLoad["Name"].(interface{}).(string)
	if serviceName == appName {
		if _, ok := PayLoad["FqdnEx"]; ok {
			DNS := PayLoad["FqdnEx"].(interface{}).(string)
			return DNS
		}
		if _, ok := PayLoad["Fqdn"]; ok {
			DNS := PayLoad["Fqdn"].(interface{}).(string)
			return DNS
		}

	}
	return ""
}

func UpdateDuploDetails(appName string, container containerDetails) error {

	statement, err := database.Db.Prepare("UPDATE app SET instance_id = ?, docker_id = ?, host_id =?, tenant_id = ?	WHERE name = ?")
	if err != nil {
		log.Println(err)
		return err
	}
	_, err = statement.Exec(container.InstanceId, container.DockerId, container.HostId, container.TenantId, appName)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
