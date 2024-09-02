package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	ad "github.com/nifetency/nife.io/internal/app_deployments"
	appRelease "github.com/nifetency/nife.io/internal/app_release"
	clusterDetails "github.com/nifetency/nife.io/internal/cluster_info"
	_helper "github.com/nifetency/nife.io/pkg/helper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateNewAppRelease(appId, imageToDeploy, prevReleaseId, userId string, org *model.Organization) (appRelease.AppRelease, error) {

	releaseId, prevRelease, err := updateDeloyments(imageToDeploy, appId, prevReleaseId, org, userId)
	if err != nil {
		return appRelease.AppRelease{}, err
	}
	version, _ := strconv.Atoi(prevRelease.Version)
	if err != nil {
		return appRelease.AppRelease{}, err
	}
	// CALL GO-ROUTINE
	currentRelease := appRelease.AppRelease{
		Id:        releaseId,
		AppId:     appId,
		Status:    "active",
		Version:   fmt.Sprintf("%v", (version + 1)),
		CreatedAt: time.Now(),
		UserId:    userId,
		ImageName: imageToDeploy,
		Port:      prevRelease.Port,
	}
	_helper.CreateAppRelease(currentRelease)
	return currentRelease, nil
}

func updateDeloyments(imageToDeploy, appId, prevReleaseId string, org *model.Organization, userId string) (string, appRelease.AppRelease, error) {

	currentRelease, err := _helper.GetAppRelease(appId, "active")
	if err != nil {
		log.Println(err)
	}

	appDetails, err := GetApp(appId, userId)

	deployments, err := _helper.GetDeploymentsByReleaseId(appId, prevReleaseId)
	if err != nil {
		log.Println(err)
	}

	appDeployments := *deployments
	noOfDeployments := len(appDeployments)
	releaseId := uuid.NewString()

	// INITIATE WAITGROUP
	var wg sync.WaitGroup
	wg.Add(noOfDeployments)

	fmt.Println(" Updating Release and Deployments ")
	for i := 0; i < noOfDeployments; i++ {

		go func(i int) {
			defer wg.Done()
			oldDeployment := appDeployments[i]
			//GET REGION CLUSTER DETAILS
			clusterDetails, err := clusterDetails.GetClusterDetailsByOrgId(*org.ID, oldDeployment.Region_code, "code", userId)
			if err != nil {
				log.Println(err)
			}
			clientset, err := helper.LoadK8SConfig(clusterDetails.Cluster_config_path)
			if err != nil {
				log.Println(err)
			}
			deploymentsClient := clientset.AppsV1().Deployments(*org.Slug)

			deletePolicy := metav1.DeletePropagationForeground

			if err := deploymentsClient.Delete(context.TODO(), oldDeployment.Deployment_id, metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			}); err != nil {
				log.Println(err)
			}

			UpdateDeploymentsRecord("destroyed", appId, oldDeployment.Deployment_id, time.Now())

			port, _ := strconv.Atoi(oldDeployment.Port)
			if err != nil {
				log.Println(err)
			}

			storage := _helper.GetResourceRequirement(appDetails.ParseConfig.Definition)

			appReplicas, err := _helper.GetAppReplicas(appId)
		if err != nil {
			log.Println(err)
		}

			// CREATE KUBERNETES DEPLOYMENT
			newDeploymentId, containerId,err := Deployment(*org.Slug, clientset, int32(port), imageToDeploy, appId, "", nil, storage,"", appReplicas)
			if err != nil {
				log.Println(err)
			}

			deployment := ad.AppDeployments{
				Id:            uuid.NewString(),
				AppId:         appId,
				Region_code:   clusterDetails.Region_code,
				Status:        "running",
				Deployment_id: newDeploymentId,
				Port:          oldDeployment.Port,
				App_Url:       oldDeployment.App_Url,
				Release_id:    releaseId,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
				ContainerID:   containerId,
			}
			err = _helper.CreateDeploymentsRecord(deployment)
			if err != nil {
				log.Println(err)
			}

		}(i)
	}

	wg.Wait()

	_helper.UpdateAppRelease("inactive", prevReleaseId)
	return releaseId, *currentRelease, nil
}
