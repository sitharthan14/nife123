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

func (r *mutationResolver) UpdateRole(ctx context.Context, userID *string, roleID *int) (*string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	users, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: UpdateRole, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UpdateRole, MethodName: GetById, Message:successfully reached, user: %s", user.ID)

	if users.RoleID == 2 {
		return nil, fmt.Errorf("Permission Denied")
	}

	err = service.UpdateRole(*userID, *roleID)
	if err != nil {
		log4go.Error("Module: UpdateRole, MethodName: UpdateRole, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UpdateRole, MethodName: UpdateRole, Message: User role is successfully updated, user: %s", user.ID)

	message := "Update Successfully"
	return &message, nil
}

func (r *queryResolver) GetUserPermissions(ctx context.Context) ([]*model.Permission, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.Permission{}, fmt.Errorf("Access Denied")
	}

	userPermissions, err := service.GetUserPermission(user.ID)
	if err != nil {
		log4go.Error("Module: GetUserPermissions, MethodName: GetUserPermission, Message: %s user:%s", err.Error(), user.ID)
		return []*model.Permission{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetUserPermissions, MethodName: GetUserPermission, Message: Fetching user permissions is successfully completed, user: %s", user.ID)

	return userPermissions, nil
}

func (r *queryResolver) GetUserPermissionsByPlan(ctx context.Context) (*model.PlanAndPermission, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	//---------plans and permissions
	var planName string
	idUser, _ := strconv.Atoi(user.ID)
	checkFreePlan, err := users.FreePlanDetails(idUser)

	if !checkFreePlan {
		var custStripeId string

		if user.CustomerStripeId == "" {
			custstripeId, err := users.GetCustomerStripeId(idUser)
			if err != nil {
				return nil, err
			}
			_, _ = stripes.ListSubscription(custstripeId)
			if err != nil {
				return nil, err
			}
		} else {
			custStripeId = user.CustomerStripeId
		}

		planName, err = stripes.GetCustPlanName(custStripeId)
		if err != nil {
			log4go.Error("Module: CreateOrganization, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.Email)
			return nil, err
		}
		log4go.Info("Module: CreateOrganization, MethodName: GetCustPlanName, Message: Get user plan with ProductId:"+user.StripeProductId+", user: %s", user.Email)
	}
	if checkFreePlan {
		planName = "free plan"
	}
	planAndPermission, err := service.GetCustomerPermissionByPlans(planName)
	if err != nil {
		log4go.Error("Module: CreateOrganization, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return nil, err
	}
	log4go.Info("Module: CreateOrganization, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+user.StripeProductId+", user: %s", user.Email)

	return planAndPermission, nil
	//-----------
}
