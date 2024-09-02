package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"strings"

	"github.com/alecthomas/log4go"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/internal/auth"
	"github.com/nifetency/nife.io/internal/regions"
	"github.com/nifetency/nife.io/service"
)

func (r *mutationResolver) UpdateDefaultRegion(ctx context.Context, input *model.DefaultRegionInput) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	err := service.RemoveDefaultRegion(*input.OrganizationID)
	if err != nil {
		log4go.Error("Module: UpdateDefaultRegion, MethodName: RemoveDefaultRegion, Message: %s user:%s", err.Error(), user.ID)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UpdateDefaultRegion, MethodName: RemoveDefaultRegion, Message: Default region removed successfully, user: %s", user.ID)

	err = service.UpdateDefaultRegion(*input)
	if err != nil {
		log4go.Error("Module: UpdateDefaultRegion, MethodName: UpdateDefaultRegion, Message: %s user:%s", err.Error(), user.ID)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UpdateDefaultRegion, MethodName: UpdateDefaultRegion, Message: Default region is updated successfully, user: %s", user.ID)

	return "Updated Successfully", err
}

func (r *mutationResolver) NewRegionRequest(ctx context.Context, input *model.RegionRequest) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	if len(input.Region) == 0 {
		return "Regions Must Not Be Null", nil
	}

	stringArray := []string{}
	for _, i := range input.Region {
		stringArray = append(stringArray, *i)
	}
	reg := strings.Join(stringArray, ", ")

	username := *input.FirstName + " " + *input.LastName

	err := regions.SentRegRequest(username, *input.Email, reg)

	if err != nil {
		log4go.Error("Module: NewRegionRequest, MethodName: SentRegRequest, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: NewRegionRequest, MethodName: SentRegRequest, Message: Region request mail is successfully sent , user: %s", user.ID)

	return "Email Sent Successfully", nil
}

func (r *mutationResolver) NewRegionsRequest(ctx context.Context, input *model.RegionRequest) (*model.RequestedRegionsResponse, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.RequestedRegionsResponse{}, fmt.Errorf("access Denied")
	}

	if len(input.Region) == 0 {
		mess := "Regions Must Not Be Null"
		return &model.RequestedRegionsResponse{Message: &mess}, nil
	}

	userName := user.FirstName + " " + user.LastName
	var reqestedRegion, alreadyRequestedRegion []*string
	for _, reg := range input.Region {

		reqRegion, err := service.GetRequestedRegion(user.ID, *reg)
		if err != nil {
			return &model.RequestedRegionsResponse{}, err
		}

		if reqRegion == "" {
			err = service.InsertNewRegionRequest(userName, user.ID, *reg)

			if err != nil {
				return &model.RequestedRegionsResponse{}, err
			}
			reqestedRegion = append(reqestedRegion, reg)
		} else {
			alreadyRequestedRegion = append(alreadyRequestedRegion, reg)
		}
	}
	return &model.RequestedRegionsResponse{
		RequestedRegions:        reqestedRegion,
		AlreadyRequestedRegions: alreadyRequestedRegion,
	}, nil
}

func (r *mutationResolver) MutipleRegion(ctx context.Context, input *model.MultipleRegionInput) (*model.MultipleRegionResponse, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.MultipleRegionResponse{}, fmt.Errorf("Access Denied")
	}

	roleId, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}
	if roleId == 2 {
		return nil, fmt.Errorf("Members role is restricted to update the organization default region")
	}

	region, err := service.GetRegionByOrgId(*input.OrganizationID, user.ID)

	if err != nil {
		return &model.MultipleRegionResponse{}, err
	}
	if *input.IsDefault == false {
		count := 0
		for _, regionCount := range region {
			if *regionCount.IsDefault == true {
				count++
			}
		}
		if count == len(input.Region) {
			return &model.MultipleRegionResponse{}, fmt.Errorf("There should be atleast one default region")
		}
	}

	for _, reg := range input.Region {
		err := service.UpdateMultipleRegion(*input.IsDefault, *input.OrganizationID, *reg)
		if err != nil {
			return &model.MultipleRegionResponse{}, err
		}
	}

	return &model.MultipleRegionResponse{Region: input.Region, IsDefault: input.IsDefault}, nil
}

func (r *mutationResolver) DeleteRequestedRegion(ctx context.Context, id *string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("access Denied")
	}

	err := service.DeleteRequestRegion(*id)
	if err != nil {
		return "", err
	}

	return "Successfully Deleted", nil
}

func (r *queryResolver) GetRequestedRegions(ctx context.Context) ([]*model.RequestedRegions, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("access Denied")
	}

	result, err := service.GetRegionRequestByUserId(user.ID)

	if err != nil {
		return []*model.RequestedRegions{}, err
	}

	return result, nil
}
