package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"strconv"

	"github.com/alecthomas/log4go"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/internal/auth"
	"github.com/nifetency/nife.io/internal/stripes"
	"github.com/nifetency/nife.io/internal/users"
	"github.com/nifetency/nife.io/service"
)

func (r *mutationResolver) CreateDuploVolume(ctx context.Context, input []*model.DuploVolumeInput) (*model.OutputMessage, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	//---------plans and permissions-------
	var planName string
	idUser, _ := strconv.Atoi(user.ID)
	checkFreePlan, err := users.FreePlanDetails(idUser)
	if !checkFreePlan {
		planName, err = stripes.GetCustPlanName(user.CustomerStripeId)
		if err != nil {
			log4go.Error("Module: CreateDuploVolume, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.Email)
			return nil, err
		}
		log4go.Info("Module: CreateDuploVolume, MethodName: GetCustPlanName, Message: Get user plan with ProductId:"+user.StripeProductId+", user: %s", user.Email)
	}
	if checkFreePlan {
		planName = "Free Plan"
	}
	permissions, err := users.GetCustomerPermissionByPlan(planName)
	if err != nil {
		log4go.Error("Module: CreateDuploVolume, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return nil, err
	}
	log4go.Info("Module: CreateDuploVolume, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+user.StripeProductId+", user: %s", user.Email)

	//--------------

	if permissions.PlanName == "Premium" {
		for _, vol := range input {
			size, err := strconv.Atoi(*vol.Size)
			if err != nil {
				return nil, err
			}
			if size > 10 {
				return nil, fmt.Errorf("Maximum limit for Premium Plan is 10Gi. Please try again.")
			}
		}
	} else if permissions.PlanName == "Enterprise" {
		for _, vol := range input {
			size, err := strconv.Atoi(*vol.Size)
			if err != nil {
				return nil, err
			}

			if size > 15 {
				return nil, fmt.Errorf("Maximum limit for Enterprise Plan is 15Gi. Please try again.")
			}
		}
	} else if permissions.PlanName == "Starter" || permissions.PlanName == "free plan" {
		return nil, fmt.Errorf("Upgrade your plan to unlock creating volume feature")
	}

	for _, inputdata := range input {
		// appid, err := service.GetAppIdByName(*inputdata.AppName)
		// if err != nil {
		// 	return nil, fmt.Errorf(err.Error())
		// }

		err := service.CreateVolumes(inputdata)
		if err != nil {
			log4go.Error("Module: CreateDuploVolume, MethodName: CreateVolumes, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: CreateDuploVolume, MethodName: CreateVolumes, Message: Volumes successfully created, user: %s", user.ID)
	}

	res := "Inserted Successfully"
	return &model.OutputMessage{Message: &res}, nil
}

func (r *mutationResolver) UpdateVolume(ctx context.Context, input *model.UpdateVolumeInput) (*string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	if *input.AppName == "" || *input.VolumeSize == "" {
		return nil, fmt.Errorf("App Name and Volume size cannot be empty")
	}

	volumeDet, err := service.GetVolumeDetailsByAppName(*input.AppName)
	if err != nil {
		log4go.Error("Module: UpdateVolume, MethodName: GetVolumeDetailsByAppName, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UpdateVolume, MethodName: GetVolumeDetailsByAppName, Message: Checking the App- "+*input.AppName+" is available for this user: %s", user.ID)

	if volumeDet == nil {
		return nil, fmt.Errorf("Can't find the given App")
	}

	err = service.UpdateVolume(*input.AppName, *input.VolumeSize)
	if err != nil {
		log4go.Error("Module: UpdateVolume, MethodName: UpdateVolume, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UpdateVolume, MethodName: UpdateVolume, Message: Volumes updated successfully to the App- "+*input.AppName+" , size- "+*input.VolumeSize+", user: %s", user.ID)
	res := "Updated Successfully"

	return &res, nil
}

func (r *queryResolver) GetVolumeType(ctx context.Context) ([]*model.VolumeType, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.VolumeType{}, fmt.Errorf("Access Denied")
	}

	VolType, err := service.GetVolumeType()
	if err != nil {
		log4go.Error("Module: GetVolumeType, MethodName: GetVolumeType, Message: %s user:%s", err.Error(), user.ID)
		return []*model.VolumeType{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetVolumeType, MethodName: GetVolumeType, Message: Fetch volume type is successfully completed, user: %s", user.ID)

	return VolType, err
}

func (r *queryResolver) GetVolumeByAppID(ctx context.Context, appID *string) ([]*model.VolumeByApp, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.VolumeByApp{}, fmt.Errorf("Access Denied")
	}

	volumebyId, err := service.GetVolumeDetailsByAppName(*appID)
	if err != nil {
		log4go.Error("Module: GetVolumeByAppID, MethodName: GetVolumeDetailsByAppName, Message: %s user:%s", err.Error(), user.ID)
		return []*model.VolumeByApp{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetVolumeByAppID, MethodName: GetVolumeDetailsByAppName, Message: Fetching volume details by App name is successfully completed, user: %s", user.ID)

	var volume model.VolumeByApp
	for _, vol := range volumebyId {
		volume = model.VolumeByApp{
			AppID:      vol.AppID,
			AccessMode: vol.AccessMode,
			Name:       vol.Name,
			Path:       vol.Path,
			Size:       vol.Size,
		}
	}

	return []*model.VolumeByApp{&volume}, nil
}
