package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"log"

	"github.com/alecthomas/log4go"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal/auth"
	clusterInfo "github.com/nifetency/nife.io/internal/cluster_info"
	deployment "github.com/nifetency/nife.io/pkg/helper"
	"github.com/nifetency/nife.io/service"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (r *mutationResolver) DeleteDuploApp(ctx context.Context, appName *string) (*model.OutputMessage, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	AppDetails, err := service.GetApp(*appName, user.ID)

	if err != nil {
		log4go.Error("Module: DeleteDuploApp, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: DeleteDuploApp, MethodName: GetApp, Message: Fetching app details by app name is successfully completed, user: %s", user.ID)

	var baseURL string
	var tenantId string

	for _, regCode := range AppDetails.Regions {
		clusterDetails, err := clusterInfo.GetClusterDetails(*regCode.Code, user.ID)
		if err != nil {
			return nil, fmt.Errorf(err.Error())
		}
		if *clusterDetails.InterfaceType == "REST" {
			baseURL = *clusterDetails.ExternalBaseAddress
			tenantId = *clusterDetails.TenantID
		}
	}

	if AppDetails.Status == "New" {
		goto StatusCheck
	}

	if AppDetails.Status != "Terminated" && AppDetails.Status != "" {
		err = service.DeleteDuploApp(tenantId, baseURL, *appName)
		if err != nil {
			log4go.Error("Module: DeleteDuploApp, MethodName: DeleteDuploApp, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: DeleteDuploApp, MethodName: DeleteDuploApp, Message: Duplo App is successfully deleted, user: %s", user.ID)

	} else {
		return nil, fmt.Errorf("App %s already terminated", *appName)
	}

	if AppDetails.Status == "Active" {
		err = service.DuploDeployStatus(*appName)
		if err != nil {
			log4go.Error("Module: DeleteDuploApp, MethodName: DuploDeployStatus, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: DeleteDuploApp, MethodName: DuploDeployStatus, Message: Deleting Duplo app is successfully completed, user: %s", user.ID)
	}

StatusCheck:

	err = service.DuploAppDeleteStatus(*appName, AppDetails.Status)
	if err != nil {
		log4go.Error("Module: DeleteDuploApp, MethodName: DuploAppDeleteStatus, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: DeleteDuploApp, MethodName: DuploAppDeleteStatus, Message: Deleting Duplo app is successfully completed, user: %s", user.ID)

	message := "Deleted Successfully"
	return &model.OutputMessage{
		Message: &message,
	}, nil
}

func (r *queryResolver) GetDuploStatus(ctx context.Context, appName *string) ([]*model.DuploDeployOutput, error) {
	getDuploStatus, err := service.GetDuploDeployStatus(*appName)
	if err != nil {
		log.Println(err)
		return []*model.DuploDeployOutput{}, err
	}
	return getDuploStatus, nil
}

func (r *queryResolver) GetclusterLog(ctx context.Context, appName *string, region *string) (*string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	var clientset *kubernetes.Clientset

	appDetails, err := service.GetApp(*appName, user.ID)
	if err != nil {
		log4go.Error("Module: GetclusterLog, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetclusterLog, MethodName: GetApp, Message: Get app details by app name is successfully completed, user: %s", user.ID)
	if appDetails.Status == "Suspended" {
		return nil, fmt.Errorf("App is suspended, please activate you app to see the logs")
	}

	if *appDetails.DeployType == 1 {

		if *region != "" {
			clusterDetails, err := clusterInfo.GetClusterDetailsByOrgId(*appDetails.Organization.ID, *region, "code", user.ID)
			if err != nil {
				log4go.Error("Module: GetclusterLog, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
				return nil, fmt.Errorf(err.Error())
			}
			log4go.Info("Module: GetclusterLog, MethodName: GetClusterDetailsByOrgId, Message: Get Cluster details based on organization is successfully completed, user: %s", user.ID)

			clientset, err = helper.LoadK8SConfig(clusterDetails.Cluster_config_path)

			if err != nil {
				log4go.Error("Module: GetclusterLog, MethodName: LoadK8SConfig, Message: %s user:%s", err.Error(), user.ID)
				return nil, fmt.Errorf(err.Error())
			}
			log4go.Info("Module: GetclusterLog, MethodName: LoadK8SConfig, Message:successfully reached, user: %s", user.ID)

		} else {
			clusterDetails, err := clusterInfo.GetClusterDetailsByOrgId(*appDetails.Organization.ID, "1", "default", user.ID)
			if err != nil {
				log4go.Error("Module: GetclusterLog, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
				return nil, fmt.Errorf(err.Error())
			}
			log4go.Info("Module: GetclusterLog, MethodName: GetClusterDetailsByOrgId, Message: Get Cluster details based on organization is successfully completed, user: %s", user.ID)

			clientset, err = helper.LoadK8SConfig(clusterDetails.Cluster_config_path)
			if err != nil {
				log4go.Error("Module: GetclusterLog, MethodName: LoadK8SConfig, Message: %s user:%s", err.Error(), user.ID)
				return nil, fmt.Errorf(err.Error())
			}
			log4go.Info("Module: GetclusterLog, MethodName: LoadK8SConfig, Message:successfully reached, user: %s", user.ID)

			region = &clusterDetails.Region_code
		}

		deployDetails, err := deployment.GetDeploymentsRecordSingle(*appName, *region, "running")

		if err != nil {
			log4go.Error("Module: GetclusterLog, MethodName: GetDeploymentsRecordSingle, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: GetclusterLog, MethodName: GetDeploymentsRecordSingle, Message: Fetching Deployment records is successfully completed, user: %s", user.ID)
		fmt.Println(deployDetails)

		podList, err := clientset.CoreV1().Pods(*appDetails.Organization.Slug).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			return nil, err
		}
		var containerId string

		for _, i := range podList.Items {
			if *appName == i.Labels["app"] {
				containerId = i.Name
			}
		}

		log, err := service.NifePodLog(*appDetails.Organization.Slug, *appName, containerId, clientset)
		if err != nil {
			return nil, fmt.Errorf(err.Error())
		}
		return &log, nil
	} else {

		logs, err := service.GetDuploLog(*appName, user.ID)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: GetclusterLog, MethodName: GetDuploLog, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: GetclusterLog, MethodName: GetDuploLog, Message: Fetching Duplo logs is successfully completed , user: %s", user.ID)

		return logs.Data, nil
	}
}
