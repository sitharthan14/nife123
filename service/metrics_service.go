package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	"github.com/sendgrid/sendgrid-go"

	clusterInfo "github.com/nifetency/nife.io/internal/cluster_info"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

type Metrics struct {
	DataBody map[string]interface{}
}

type kubernetesMetrics struct {
	EventMeta EventmetaData
	Text      string
	Time      string
}

type EventmetaData struct {
	Kind      string
	Name      string
	Namespace string
	Reason    string
}

type DeployDetails struct {
	Email           string
	FirstName       string
	LastName        string
	RegionCode      string
	AppName         string
	OrganiztionName string
}

func GetUserMetrics(hostName string) ([]*model.GetUserMetrics, error) {

	query := `select resolver_ip, timestamp, query_type from domain_logs where query_name = ?`

	selDB, err := database.Db.Query(query, hostName)

	if err != nil {
		return []*model.GetUserMetrics{}, err
	}
	defer selDB.Close()

	var metricsGrp []*model.GetUserMetrics

	for selDB.Next() {
		var metrics model.GetUserMetrics

		err = selDB.Scan(&metrics.ResolverIP, &metrics.TimeStamp, &metrics.QueryType)
		if err != nil {
			return []*model.GetUserMetrics{}, err
		}

		metricsGrp = append(metricsGrp, &model.GetUserMetrics{
			ResolverIP: metrics.ResolverIP,
			TimeStamp:  metrics.TimeStamp,
			QueryType:  metrics.QueryType,
		})

	}

	return metricsGrp, nil
}

func PrintMetrics(w http.ResponseWriter, r *http.Request) {

	readBody, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(readBody))
	var k8sMetrics kubernetesMetrics
	json.Unmarshal(readBody, &k8sMetrics)

	parts := strings.Split(k8sMetrics.EventMeta.Name, "/")

	smsDetails, err := GetUserDataByK8s(parts[1])
	if err != nil {
		return
	}

	if k8sMetrics.EventMeta.Kind == "service" || k8sMetrics.EventMeta.Kind == "deployment" {
		return
	}

	switch {
	case k8sMetrics.EventMeta.Reason == "deleted":
		err = SentNotification(smsDetails, k8sMetrics, "High")
		if err != nil {
			fmt.Println(err)
			return
		}
	default:
		fmt.Println("Invalid")
	}

}

func GetUserDataByK8s(containerId string) (DeployDetails, error) {

	query := `SELECT app_deployments.region_code,organization.name,app_deployments.appId,user.email, user.firstName, user.lastName FROM app_deployments
	INNER JOIN app ON app.name = app_deployments.appId
    INNER JOIN user ON user.id = app.createdBy
    INNER JOIN organization ON organization.id = app.organization_id 
	WHERE app_deployments.container_id = ?`

	selDB, err := database.Db.Query(query, containerId)

	if err != nil {
		return DeployDetails{}, err
	}
	defer selDB.Close()

	var deployDetails DeployDetails

	for selDB.Next() {

		err = selDB.Scan(&deployDetails.RegionCode, &deployDetails.OrganiztionName, &deployDetails.AppName, &deployDetails.Email, &deployDetails.FirstName, &deployDetails.LastName)
		if err != nil {
			return DeployDetails{}, err
		}
	}

	return deployDetails, nil

}

func SentNotification(smsDetails DeployDetails, K8smetrics kubernetesMetrics, status string) error {

	apiKey := os.Getenv("SENDGRID_API_KEY")

	userName := os.Getenv("SENDGRID_USER_NAME")

	templateId := os.Getenv("KUBEWATCH_TEMPLATE_ID")

	request := sendgrid.GetRequest(apiKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"

	description := `A ` + K8smetrics.EventMeta.Kind + ` has been ` + K8smetrics.EventMeta.Reason + ` for the app ` + smsDetails.AppName
	subject := "Incident from Nife Monitoring-" + smsDetails.AppName
	username := smsDetails.FirstName + smsDetails.LastName

	request.Body = []byte(` {
		"from": {
			"email": "` + userName + `"
		},    
		"personalizations": [
		  {
			
			"to": [
				{
					"email": "` + smsDetails.Email + `"
				}
			],
			
			"dynamic_template_data":{ 
				"subject": "` + subject + `"
				"name": "` + username + `"
				"alert_name": "` + username + `"
				"description": "` + description + `"
				"severity": "` + status + `"	
				"resource": "` + smsDetails.OrganiztionName + `"
				"start_time": "` + K8smetrics.Time + `"		
			}
			
		  }
		],
		
		"template_id":"` + templateId + `"
	  }`)

	response, err := sendgrid.API(request)
	if err != nil {
		log.Println(err)
		return err
	}

	fmt.Println(response)
	return nil

}

func CreateDataDog(input model.DataDogInput, userId string) error {
	statement, err := database.Db.Prepare("INSERT INTO data_dog (id, api_key, app_key, api_endpoint, cluster_id, user_id, is_active) VALUES (?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, input.APIKey, input.AppKey, input.APIEndpoint, input.ClusterID, userId, true)
	if err != nil {
		return err
	}
	return nil
}

func UpdateDataDog(dataDogUpdate model.DataDogInput) error {

	statement, err := database.Db.Prepare("UPDATE data_dog SET api_key=?, app_key=?,api_endpoint=? where id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(dataDogUpdate.APIKey, dataDogUpdate.AppKey, dataDogUpdate.APIEndpoint, dataDogUpdate.ID)
	if err != nil {
		return err
	}
	return nil
}

func DeleteDataDog(id string) error {

	statement, err := database.Db.Prepare("UPDATE data_dog SET is_active=? where id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(false, id)
	if err != nil {
		return err
	}
	return nil
}

func GetUserAddedDataDog(userId string) ([]*model.AddedDataDog, error) {

	query := `SELECT id, api_key, app_key, api_endpoint, cluster_id FROM data_dog where user_id = ? and is_active = ?;`

	selDB, err := database.Db.Query(query, userId, true)

	if err != nil {
		return []*model.AddedDataDog{}, err
	}
	defer selDB.Close()

	var dataDogDetails []*model.AddedDataDog

	for selDB.Next() {
		var dataDogDet model.AddedDataDog

		err = selDB.Scan(&dataDogDet.ID, &dataDogDet.APIKey, &dataDogDet.AppKey, &dataDogDet.APIEndpoint, &dataDogDet.ClusterID)
		if err != nil {
			return []*model.AddedDataDog{}, err
		}

		clusterDet, err := clusterInfo.GetUserAddedClusterDetailsByclusterId(*dataDogDet.ClusterID)
		if err != nil {
			return nil, err
		}

		dataDogDet.ClusterDetails = clusterDet

		dataDogDetails = append(dataDogDetails, &dataDogDet)
	}
	return dataDogDetails, nil

}
