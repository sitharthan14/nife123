package datadog

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/nifetency/nife.io/helper"
	clusterInfo "github.com/nifetency/nife.io/internal/cluster_info"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	"github.com/nifetency/nife.io/service"
)

type DataDog struct {
	AppName string `json:"appName"`
	UserId  string `json:"userId"`
}

var DataDogRequest = []string{"kubernetes.cpu.usage.total", "kubernetes.memory.rss", "container.uptime", "container.net.sent", "container.net.rcvd", "container.cpu.usage", "container.cpu.limit", "container.memory.limit", "container.memory.usage", "kubernetes.containers.restarts", "kubernetes.containers.running", "container.io.read", "container.io.write", "container.pid.open_files", "container.pid.thread_count", "containerd.cpu.user", "containerd.cpu.limit", "containerd.cpu.system", "containerd.cpu.total"}

func GetDataDogGraphs(w http.ResponseWriter, r *http.Request) {

	var dataBody DataDog
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	appDet, err := service.GetApp(dataBody.AppName, dataBody.UserId)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	var getclust *clusterInfo.ClusterDetail

	for _, regdep := range appDet.Regions {
		getclust, err = clusterInfo.GetClusterDetailsStruct(*regdep.Code, dataBody.UserId)
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
	}
	var apiKey, appKey, endpoint string
	if getclust.ClusterType == "byoc" {
		apiKey, appKey, endpoint, err =	GetDataDogKeysByUser(dataBody.UserId, getclust.Id)
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "There is no datadog is mapped with this cluster"})
			return
		}
	} else {
		apiKey, appKey, endpoint, err = GetDataDogKeys()
	}
	response := make(map[string]interface{})

	for _, i := range DataDogRequest {
		datagr := DataDogGraphRequest(dataBody.AppName, endpoint, apiKey, appKey, i, i)
		response[i] = datagr
	}

	helper.RespondwithJSON(w, http.StatusOK, response)

}

func GetDataDogKeysByUser(userId, clusterId string) (string, string, string, error) {

	query := "SELECT api_key, app_key, api_endpoint FROM data_dog where user_id = ? and cluster_id = ?;"

	selDB, err := database.Db.Query(query, userId, clusterId)
	if err != nil {
		return "", "", "", err
	}
	var apiKey string
	var appKey string
	var endPoint string

	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&apiKey, &appKey, &endPoint)
	if err != nil {
		return "", "", "", err
	}
	return apiKey, appKey, endPoint, nil
}

func GetDataDogKeys() (string, string, string, error) {
	id := "630a612c-2d5e-4ea7-8753-89be61396529"

	query := "SELECT api_key, app_key, api_endpoint FROM data_dog where id = ?;"

	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return "", "", "", err
	}
	var apiKey string
	var appKey string
	var endPoint string

	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&apiKey, &appKey, &endPoint)
	if err != nil {
		return "", "", "", err
	}
	return apiKey, appKey, endPoint, nil
}

func DataDogGraphRequest(appName, endpoint, apiKey, appKey, request, title string) map[string]interface{} {

	postBody, _ := json.Marshal(map[string]interface{}{
		"graph_json": map[string]interface{}{
			"requests": []map[string]interface{}{
				map[string]interface{}{
					"q": "sum:" + request + "{kube_container_name:" + appName + "} by {pod_name}",
				},
			},
			"viz":    "timeseries",
			"events": []map[string]interface{}{},
		},
		"title":     title,
		"timeframe": "1_hour",
		"size":      "medium",
	})

	resp, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(postBody))
	if err != nil {
		return nil
	}

	resp.Header.Add("Accept", "application/json")
	resp.Header.Add("DD-API-KEY", apiKey)
	resp.Header.Add("DD-APPLICATION-KEY", appKey)

	clt := http.Client{}

	res, err := clt.Do(resp)
	if err != nil {
		return nil
	}

	var graphDet map[string]interface{}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}

	json.Unmarshal(body, &graphDet)

	return graphDet
}
