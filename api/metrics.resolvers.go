package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/alecthomas/log4go"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/internal/auth"
	clusterDetails "github.com/nifetency/nife.io/internal/cluster_info"
	"github.com/nifetency/nife.io/service"
)

func (r *mutationResolver) AddDataDogByoc(ctx context.Context, input model.DataDogInput) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	byocReg, err := clusterDetails.GetAllUserAddedClusterDetailsByUserId(user.ID)
	if err != nil {
		return "", err
	}
	if byocReg == nil {
		return "", fmt.Errorf("There is no BYOC regions for this account")
	}

	checkByocRegion, err := clusterDetails.GetUserAddedClusterDetailsByclusterId(*input.ClusterID)
	if err != nil {
		return "", err
	}

	if checkByocRegion == nil {
		return "", fmt.Errorf("Can't find the selected BYOC region")
	}

	err = service.CreateDataDog(input, user.ID)
	if err != nil {
		return "", err
	}

	return "Created Successfully", nil
}

func (r *mutationResolver) UpdateDataDogByoc(ctx context.Context, input *model.DataDogInput) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	err := service.UpdateDataDog(*input)
	if err != nil {
		return "", err
	}
	return "Successfully updated", err
}

func (r *mutationResolver) DeleteDataDogByoc(ctx context.Context, dataDogID string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	err := service.DeleteDataDog(dataDogID)
	if err != nil {
		return "", err
	}
	return "Successfully deleted", err
}

func (r *queryResolver) UserMetrics(ctx context.Context, appName *string) ([]*model.GetUserMetrics, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	appDetails, err := service.GetApp(*appName, user.ID)

	if err != nil {
		log.Println(err)
		log4go.Error("Module: UserMetrics, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: UserMetrics, MethodName: GetApp, Message: Get app details by app name is successfully completed, user: %s", user.ID)

	if appDetails.Hostname == "" {
		return nil, fmt.Errorf("Something went wrong with your App Name")
	}

	hostNameSplit := strings.Split(appDetails.Hostname, "://")

	metricsGroup, err := service.GetUserMetrics(hostNameSplit[1])

	if err != nil {
		log.Println(err)
		log4go.Error("Module: UserMetrics, MethodName: GetUserMetrics, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: UserMetrics, MethodName: GetUserMetrics, Message: Fetching user metrics is successfully completed, user: %s", user.ID)

	return metricsGroup, nil
}

func (r *queryResolver) GetDataDogByUserID(ctx context.Context) ([]*model.AddedDataDog, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	datadog, err := service.GetUserAddedDataDog(user.ID)
	if err != nil {
		return nil, err
	}
	return datadog, nil
}
