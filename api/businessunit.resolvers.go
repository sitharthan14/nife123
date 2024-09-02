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

func (r *mutationResolver) CreateBusinessUnit(ctx context.Context, input model.BusinessUnitInput) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	// ------- Plan and permission
	var planName string
	idUser, _ := strconv.Atoi(user.ID)
	checkFreePlan, err := users.FreePlanDetails(idUser)
	if !checkFreePlan {
		planName, err = stripes.GetCustPlanName(user.CustomerStripeId)
		if err != nil {
			log4go.Error("Module: CreateSubOrganization, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.Email)
			return "", err
		}
		log4go.Info("Module: CreateSubOrganization, MethodName: GetCustPlanName, Message: Get user plan with ProductId:"+user.StripeProductId+", user: %s", user.Email)
	}
	if checkFreePlan {
		planName = "free plan"
	}
	planAndPermission, err := users.GetCustomerPermissionByPlan(planName)
	if err != nil {
		log4go.Error("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return "", err
	}
	log4go.Info("Module: CreateSubOrganization, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+user.StripeProductId+", user: %s", user.Email)

	businessUnitCount, err := service.GetBusinessUnitCountById(user.ID)
	if err != nil {
		log4go.Error("Module: CreateSubOrganization, MethodName: GetOrganizationCountById, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: CreateSubOrganization, MethodName: GetOrganizationCountById, Message: Check Organization Count by Id, user: %s", user.ID)

	if businessUnitCount >= planAndPermission.BusinessunitCount {
		if planAndPermission.BusinessunitCount == 0 {
			return "", fmt.Errorf("Upgrade your plan to unlock Business Unit feature")

		}
		return "", fmt.Errorf("You've reached your maximum limit of current plan. Please upgrade your plan to create Business Unit")
	}
	//-------------

	// if input.OrgID == nil || input.SubOrg == nil || *input.OrgID == "" || *input.SubOrg == "" {
	// 	return "Select Orgnization or Sub-Organization to Create Business Unit", nil
	// }
	checkWL, err := service.GetBusinessUnitByName(user.ID, *input.Name)
	if err != nil {
		log4go.Error("Module: CreateBusinessUnit, MethodName: GetBusinessUnitByName, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: CreateBusinessUnit, MethodName: GetBusinessUnitByName, Message: Creating business unit, Name:"+*input.Name+", Organization: "+*input.OrgID+", Sub-Organization: "+*input.SubOrg+" , user: %s", user.ID)

	if checkWL.Name != nil {
		return "", fmt.Errorf("Given business unit name has already taken. Try with another name")
	}

	err = service.CreateBusinessUnit(input, user.ID)
	if err != nil {
		log4go.Error("Module: CreateBusinessUnit, MethodName: CreateBusinessUnit, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: CreateBusinessUnit, MethodName: CreateBusinessUnit, Message: Creating business unit, Name:"+*input.Name+", Organization: "+*input.OrgID+", Sub-Organization: "+*input.SubOrg+" , user: %s", user.ID)

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: CreateBusinessUnit, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: CreateBusinessUnit, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	AddOperation := service.Activity{
		Type:       "BUSINESSUNIT",
		UserId:     user.ID,
		Activities: "CREATED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Created a business unit " + *input.Name,
		RefId:      *input.OrgID,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	err = service.SendSlackNotification(user.ID, AddOperation.Message)
	if err != nil {
		log4go.Error("Module: CreateBusinessUnit, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return "Successfully Created", nil
}

func (r *mutationResolver) UpdateBusinessUnit(ctx context.Context, input model.BusinessUnitInput) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	err = service.UpdateBusinessUnit(input, user.ID)
	if err != nil {
		log4go.Error("Module: UpdateBusinessUnit, MethodName: UpdateBusinessUnit, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: UpdateBusinessUnit, MethodName: UpdateBusinessUnit, Message: Updating business unit, Name:"+*input.Name+", Organization: "+*input.OrgID+", Sub-Organization: "+*input.SubOrg+" , user: %s", user.ID)
	return "Updated Created", nil
}

func (r *mutationResolver) DeleteBusinessUnit(ctx context.Context, id string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	businessUnitName, err := service.GetBusinessUnitById(id)
	if err != nil {
		log4go.Error("Module: DeleteBusinessUnit, MethodName: GetBusinessUnitById, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: DeleteBusinessUnit, MethodName: GetBusinessUnitById, Message: Fetching business unit details using BusinessUnitId:"+id+" , user: %s", user.ID)

	businessUnitOrg, err := service.GetBusinessUnitByName(user.ID, businessUnitName)
	if err != nil {
		log4go.Error("Module: DeleteBusinessUnit, MethodName: GetBusinessUnitByName, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: DeleteBusinessUnit, MethodName: GetBusinessUnitByName, Message: Fetching business unit details using BusinessUnitName:"+businessUnitName+" , user: %s", user.ID)

	err = service.DeleteBusinessUnit(id)
	if err != nil {
		log4go.Error("Module: DeleteBusinessUnit, MethodName: DeleteBusinessUnit, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: DeleteBusinessUnit, MethodName: DeleteBusinessUnit, Message: Deleting business unit, BusinessUnitId:"+id+" , user: %s", user.ID)

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: DeleteBusinessUnit, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: DeleteBusinessUnit, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	DeleteOperation := service.Activity{
		Type:       "BUSINESSUNIT",
		UserId:     user.ID,
		Activities: "DELETED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Deleted the Business unit " + businessUnitName,
		RefId:      *businessUnitOrg.OrgID,
	}

	_, err = service.InsertActivity(DeleteOperation)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	err = service.SendSlackNotification(user.ID, DeleteOperation.Message)
	if err != nil {
		log4go.Error("Module: DeleteBusinessUnit, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return "Deleted Successfully", nil
}

func (r *queryResolver) BusinessUnitList(ctx context.Context) ([]*model.ListBusinessUnit, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.ListBusinessUnit{}, fmt.Errorf("Access Denied")
	}

	businessUnit, err := service.GetAllBusinessUnit(user.ID)
	if err != nil {
		log4go.Error("Module: BusinessUnitList, MethodName: GetAllBusinessUnit, Message: %s user:%s", err.Error(), user.ID)
		return []*model.ListBusinessUnit{}, err
	}
	log4go.Info("Module: BusinessUnitList, MethodName: GetAllBusinessUnit, Message: Fetching Business unit based on user , user: %s", user.ID)
	return businessUnit, nil
}

func (r *queryResolver) GetBusinessUnitByID(ctx context.Context, name *string) (*model.ListBusinessUnit, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.ListBusinessUnit{}, fmt.Errorf("Access Denied")
	}
	buDet, err := service.GetBusinessUnitByName(user.ID, *name)
	if err != nil {
		log4go.Error("Module: GetBusinessUnitByID, MethodName: GetAllBusinessUnit, Message: %s user:%s", err.Error(), user.ID)
		return &model.ListBusinessUnit{}, err
	}
	log4go.Info("Module: GetBusinessUnitByID, MethodName: GetAllBusinessUnit, Message: Fetching Business unit details based on name , user: %s", user.ID)
	return buDet, nil
}

func (r *queryResolver) GetBusinessUnitByOrgID(ctx context.Context, orgID *string, subOrgID *string) ([]*model.ListBusinessUnit, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.ListBusinessUnit{}, fmt.Errorf("Access Denied")
	}

	businessUnitlist, err := service.GetBusinessUnitByOrgIdOrSubOrgId(*orgID, *subOrgID)
	if err != nil {
		log4go.Error("Module: GetBusinessUnitByOrgID, MethodName: GetBusinessUnitByOrgIdOrSubOrgId, Message: %s user:%s", err.Error(), user.ID)
		return []*model.ListBusinessUnit{}, err
	}
	log4go.Info("Module: GetBusinessUnitByOrgID, MethodName: GetBusinessUnitByOrgIdOrSubOrgId, Message: Fetching Business unit details based on OrganizationID:"+*orgID+" and Sub-OrganizationID: "+*subOrgID+" , user: %s", user.ID)

	return businessUnitlist, nil
}

func (r *queryResolver) GetBusinessUnit(ctx context.Context, orgID string, subOrgID string) ([]*model.GetBusinessUnit, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.GetBusinessUnit{}, fmt.Errorf("Access Denied")
	}

	businessUnit, err := service.GetBusinessUnit(orgID, subOrgID)
	if err != nil {
		log4go.Error("Module: GetBusinessUnit, MethodName: GetBusinessUnit, Message: %s user:%s", err.Error(), user.ID)
		return []*model.GetBusinessUnit{}, err
	}
	log4go.Info("Module: GetBusinessUnit, MethodName: GetBusinessUnit, Message: Fetching Business unit based on Organization: "+orgID+" sub-organization: "+subOrgID+" , user: %s", user.ID)
	return businessUnit, nil
}
