package duplo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal/decode"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func CreateDuploStatus(appName, status, userId, info string, progress, lastPoll int) error {
	statement, err := database.Db.Prepare("INSERT INTO duplo_deploy_status(id, status, user_id, info, created_at, app_name, updated_at,progress,poll_count)VALUES (?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}

	id := uuid.NewString()

	_, err = statement.Exec(id, status, userId, info, time.Now(), appName, time.Now(), progress, lastPoll)
	if err != nil {
		return err
	}
	return nil
}

func UpdateHostName(appName, hostName string) error {

	queryString := `UPDATE app SET hostname = ?, status =?, deployed = ? WHERE name = ?;`

	statement, err := database.Db.Prepare(queryString)
	if err != nil {
		return err
	}

	_, err = statement.Exec(hostName, "Active", true ,appName)
	if err != nil {
		return err
	}
	return nil
}

func DuploHttpPostRequest(postBody []byte, duploUrl, Method, endPoint, tenantId string) (http.Response, error) {

	accessToken := os.Getenv("ACCESS_TOKEN")
	url := fmt.Sprintf(duploUrl+"%s/"+endPoint, tenantId)
	req, err := http.NewRequest(Method, url, bytes.NewBuffer(postBody))

	if err != nil {
		log.Println(err)
		return http.Response{}, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{}
	response, err := client.Do(req)

	if err != nil {
		log.Println(err)
		return http.Response{}, err
	}

	return *response, nil

}

func DuploHTTPGetRequest(duploUrl, endPoint, tenantId string) (http.Response, error) {

	accessToken := os.Getenv("ACCESS_TOKEN")

	url := fmt.Sprintf(duploUrl+"%s/"+endPoint, tenantId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return http.Response{}, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	response, err := client.Do(req)

	if err != nil {
		return http.Response{}, err
	}

	return *response, nil

}

func (sec *CreateSecret) CreateSecret(Username, Password, externalBaseAddress, tenantId, link string) error {

	Password = decode.DePwdCode(Password)

	postBody, _ := json.Marshal(map[string]interface{}{
		"SecretName": sec.SecretName,
		"SecretType": sec.SecretType,
		"SecretData": map[string]interface{}{
			".dockerconfigjson": map[string]interface{}{
				"auths": map[string]interface{}{
					"https://index.docker.io/v1/": map[string]interface{}{
						"username": Username,
						"password": Password,
						"email":    "",
						"auth":     "",
					},
				},
			},
		},
	})

	fmt.Println(string(postBody))

	// responseBody := bytes.NewBuffer(postBody)

	response, err := DuploHttpPostRequest(postBody, externalBaseAddress+"/subscriptions/", "POST", "CreateOrUpdateK8Secret", tenantId)

	if err != nil {
		log.Println(err)
		return err
	}

	body, _ := ioutil.ReadAll(response.Body)
	bodyResponse := string([]byte(body))
	if bodyResponse != "null" && response.StatusCode != 200 {
		return fmt.Errorf("Something went wrong to create secret %s", bodyResponse)
	}
	return nil
}

func (data *DeleteSecret) DeleteDuploSecret() error {

	payLoad := DeleteSecret{
		SecretName: data.SecretName,
	}

	postBody, _ := json.Marshal(payLoad)

	res, err := DuploHttpPostRequest(postBody, data.ExternalBaseAddress+"/subscriptions", "DELETE", "DeleteK8Secret", data.TenantId)

	if err != nil {
		return fmt.Errorf(err.Error())
	}

	fmt.Println(res.Status)
	return nil
}

func (data *Getsecret) GetDuploSecret() (bool, error) {
	res, err := DuploHTTPGetRequest(data.ExternalBaseAddress+"/subscriptions/", "GetAllK8Secrets", data.TenantId)

	if err != nil {
		return false, fmt.Errorf(err.Error())
	}

	if res.Body == nil {
		return false, fmt.Errorf("Response Body Should Not Be Null")
	}

	body, _ := ioutil.ReadAll(res.Body)
	var result []map[string]interface{}
	json.Unmarshal(body, &result)

	for _, i := range result {
		value, err := helper.CheckRequiredSecret(data.Name, i)
		if err != nil {
			return false, fmt.Errorf(err.Error())
		}
		if value {
			return true, nil
		}
	}

	return false, nil

}
