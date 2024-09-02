package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/alecthomas/log4go"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/internal/auth"
	"github.com/nifetency/nife.io/internal/stripes"
	"github.com/nifetency/nife.io/internal/users"
	"github.com/nifetency/nife.io/service"
)

func (r *mutationResolver) CreateWorkloadManagement(ctx context.Context, input *model.WorkloadManagement) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	//---------plans and permissions-------
	var getPermission string
	idUser, _ := strconv.Atoi(user.ID)
	checkFreePlan, err := users.FreePlanDetails(idUser)
	if !checkFreePlan {

		getPermission, err = stripes.GetCustPlanName(user.CustomerStripeId)
		if err != nil {
			log4go.Error("Module: CreateWorkloadManagement, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.ID)
			return "", fmt.Errorf(err.Error())
		}
		log4go.Info("Module: CreateWorkloadManagement, MethodName: GetCustPlanName, Message: Checking for permission to create workload, user: %s", user.ID)
	}
	if checkFreePlan {
		getPermission = "free plan"
	}
	permissions, err := users.GetCustomerPermissionByPlan(getPermission)
	if err != nil {
		log4go.Error("Module: CreateWorkloadManagement, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: CreateWorkloadManagement, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:, user: %s", user.Email)

	if permissions.WorkloadManagement == false {
		return "", fmt.Errorf("Upgrade your plan to unlock Workload management feature")
	}
	//--------------
	orgName, err := service.GetOrgNameById(*input.OrganizationID)
	if err != nil {
		log4go.Error("Module: CreateWorkloadManagement, MethodName: GetOrgNameById, Message: %s user:%s", err.Error(), user.ID)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: CreateWorkloadManagement, MethodName: GetOrgNameById, Message: Check the provided organization is available for the user, Organization: "+*input.OrganizationID+", user: %s", user.ID)

	if orgName == "" {
		return "", fmt.Errorf(" Cannot find the Organization ")
	}

	checkwl, checkEndPoint, err := service.GetworkloadByName(*input.EnvironmentName, user.ID)
	if err != nil {
		log4go.Error("Module: CreateWorkloadManagement, MethodName: GetworkloadByName, Message: %s user:%s", err.Error(), user.ID)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: CreateWorkloadManagement, MethodName: GetworkloadByName, Message: Checking for the worload name with the user. Workload Name: "+*input.EnvironmentName+", user: %s", user.ID)

	if checkwl == *input.EnvironmentName {
		return "", fmt.Errorf(" The Workload management name " + *input.EnvironmentName + " is already created ")
	}
	if checkEndPoint == *input.EnvironmentEndpoint {
		return "", fmt.Errorf(" The Workload management endpoint " + *input.EnvironmentEndpoint + " is already created ")
	}

	err = service.CreateworkloadManagement(*input, user.ID)
	if err != nil {
		log4go.Error("Module: CreateWorkloadManagement, MethodName: CreateworkloadManagement, Message: %s user:%s", err.Error(), user.ID)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: CreateWorkloadManagement, MethodName: CreateworkloadManagement, Message: Creating workload management is successfully completed. envName: "+*input.EnvironmentName+" Organization Name: "+orgName+", user: %s", user.ID)

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: CreateWorkloadManagement, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: CreateWorkloadManagement, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	AddOperation := service.Activity{
		Type:       "WORKLOAD",
		UserId:     user.ID,
		Activities: "CREATED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Created a workload management " + *input.EnvironmentName,
		RefId:      *input.OrganizationID,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	err = service.SendSlackNotification(user.ID, AddOperation.Message)
	if err != nil {
		log4go.Error("Module: CreateWorkloadManagement, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return "Successfully Created", nil
}

func (r *mutationResolver) DeleteWorkloadManagement(ctx context.Context, id *string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	workloaddet, err := service.GetWorkLoadManagementById(*id, user.ID)
	if err != nil {
		log4go.Error("Module: DeleteWorkloadManagement, MethodName: GetWorkLoadManagementByUser, Message: %s user:%s", err.Error(), user.ID)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: DeleteWorkloadManagement, MethodName: GetWorkLoadManagementByUser, Message: Fetching workload management details based on workload Id: "+*id+", user: %s", user.ID)

	if workloaddet.EnvironmentName == nil {
		return "Can't find the workload", nil
	}

	_, err = service.DeleteWorkLoadManagement(*id, user.ID)
	if err != nil {
		log4go.Error("Module: DeleteWorkloadManagement, MethodName: DeleteWorkLoadManagement, Message: %s user:%s", err.Error(), user.ID)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: DeleteWorkloadManagement, MethodName: DeleteWorkLoadManagement, Message: Deleting workload management Id : "+*id+", user: %s", user.ID)

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: DeleteWorkloadManagement, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: DeleteWorkloadManagement, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	AddOperation := service.Activity{
		Type:       "WORKLOAD",
		UserId:     user.ID,
		Activities: "DELETED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Delete a workload management " + *workloaddet.EnvironmentName,
		RefId:      *workloaddet.OrganizationID,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	err = service.SendSlackNotification(user.ID, AddOperation.Message)
	if err != nil {
		log4go.Error("Module: DeleteWorkloadManagement, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return "Successfully Deleted", nil
}

func (r *mutationResolver) AddWorkloadRegions(ctx context.Context, workLoadID string, regionCode []*string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	checkWorkload, err := service.GetWorkLoadManagementById(workLoadID, user.ID)
	if err != nil {
		return "", err
	}
	if checkWorkload.EnvironmentName == nil {
		return "", fmt.Errorf("Cannot locate the given workload")
	}

	resultChan := make(chan error)

	var wg sync.WaitGroup

	for _, regCode := range regionCode {
		wg.Add(1)
		go func(regCode *string) {
			defer wg.Done()

			err := service.AddWorkloadRegion(*regCode, workLoadID, user.ID)
			if err != nil {
				resultChan <- err
			}
		}(regCode)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for err := range resultChan {
		if err != nil {
			return "", err
		}
	}

	return "Successfully Inserted", nil
}

func (r *mutationResolver) RemoveWorkloadRegions(ctx context.Context, wlid *string, wlRegion *string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	err = service.DeleteWorkloadRegion(*wlid, user.ID, *wlRegion)
	if err != nil {
		return "", err
	}

	return "Successfully Deleted", nil
}

func (r *queryResolver) GetWorkloadMangementByUser(ctx context.Context) ([]*model.WorkloadManagementList, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	var workloaddet []*model.WorkloadManagementList
	var workloadByOrganization []*model.WorkloadManagementList

	if user.RoleId != 1 {
		orgDet, err := service.AllOrganizations(user.ID)
		if err != nil {
			return nil, err
		}

		adminEmail, err := users.GetAdminByCompanyNameAndEmail(user.CompanyName)
		if err != nil {
			return nil, err
		}

		userid, err := users.GetUserIdByEmail(adminEmail)
		if err != nil {
			return nil, err
		}
		userId := strconv.Itoa(userid)

		// adminWorkloaddet, err = service.GetWorkLoadManagementByUser(userId)
		// if err != nil {
		// 	log4go.Error("Module: GetWorkloadMangementByUser, MethodName: GetWorkLoadManagementByUser, Message: %s user:%s", err.Error(), user.ID)
		// 	return []*model.WorkloadManagementList{}, fmt.Errorf(err.Error())
		// }
		// log4go.Info("Module: GetWorkloadMangementByUser, MethodName: GetWorkLoadManagementByUser, Message: Fetching workload management details based on user, user: %s", user.ID)

		for _, orgs := range orgDet.Nodes {
			workloadByOrganization, err = service.GetWorkLoadManagementByOrgIdSubOrgBusinessU(userId, *orgs.ID, "", "")
			if err != nil {
				return nil, err
			}
		}

	} else {
		var err error
		workloaddet, err = service.GetWorkLoadManagementByUser(user.ID)
		if err != nil {
			log4go.Error("Module: GetWorkloadMangementByUser, MethodName: GetWorkLoadManagementByUser, Message: %s user:%s", err.Error(), user.ID)
			return []*model.WorkloadManagementList{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: GetWorkloadMangementByUser, MethodName: GetWorkLoadManagementByUser, Message: Fetching workload management details based on user, user: %s", user.ID)
	}
	workloaddet = append(workloaddet, workloadByOrganization...)

	return workloaddet, nil
}

func (r *queryResolver) GetWorkloadMangementByorgnizationID(ctx context.Context, orgID *string, subOrgID *string, businessUnitID *string) ([]*model.WorkloadManagementList, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	var adminWorkloaddet []*model.WorkloadManagementList

	if user.RoleId != 1 {
		adminEmail, err := users.GetAdminByCompanyNameAndEmail(user.CompanyName)
		if err != nil {
			return nil, err
		}

		userid, err := users.GetUserIdByEmail(adminEmail)
		if err != nil {
			return nil, err
		}
		userId := strconv.Itoa(userid)

		adminWorkloaddet, err = service.GetWorkLoadManagementByOrgIdSubOrgBusinessU(userId, *orgID, *subOrgID, *businessUnitID)
		if err != nil {
			log4go.Error("Module: GetWorkloadMangementByUser, MethodName: GetWorkLoadManagementByUser, Message: %s user:%s", err.Error(), user.ID)
			return []*model.WorkloadManagementList{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: GetWorkloadMangementByUser, MethodName: GetWorkLoadManagementByUser, Message: Fetching workload management details based on user, user: %s", user.ID)

	}

	workloaddet, err := service.GetWorkLoadManagementByOrgIdSubOrgBusinessU(user.ID, *orgID, *subOrgID, *businessUnitID)
	if err != nil {
		log4go.Error("Module: GetWorkloadMangementByorgnizationID, MethodName: GetWorkLoadManagementByOrgId, Message: %s user:%s", err.Error(), user.ID)
		return []*model.WorkloadManagementList{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetWorkloadMangementByorgnizationID, MethodName: GetWorkLoadManagementByOrgId, Message: Fetching workload management details based on user and Organization: "+*orgID+", user: %s", user.ID)

	workloaddet = append(workloaddet, adminWorkloaddet...)

	return workloaddet, nil
}

func (r *queryResolver) GetWorkloadMangementByWlID(ctx context.Context, workloadID *string) (*model.WorkloadManagementList, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	workloaddet, err := service.GetWorkLoadManagementById(*workloadID, user.ID)
	if err != nil {
		log4go.Error("Module: GetWorkloadMangementByUser, MethodName: GetWorkLoadManagementByUser, Message: %s user:%s", err.Error(), user.ID)
		return &model.WorkloadManagementList{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetWorkloadMangementByUser, MethodName: GetWorkLoadManagementByUser, Message: Fetching workload management details based on workload Id: "+*workloadID+", user: %s", user.ID)

	return workloaddet, nil
}

func (r *queryResolver) GetWorkloadMangementByWlName(ctx context.Context, workloadName *string) (*model.WorkloadManagementList, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	workloaddett, err := service.GetWorkLoadManagementByName(*workloadName, user.ID)
	if err != nil {
		log4go.Error("Module: GetWorkloadMangementByWlName, MethodName: GetWorkLoadManagementByName, Message: %s user:%s", err.Error(), user.ID)
		return &model.WorkloadManagementList{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetWorkloadMangementByWlName, MethodName: GetWorkLoadManagementByName, Message: Fetching workload management details based on workload name : "+*workloadName+", user: %s", user.ID)

	return workloaddett, nil
}

func (r *queryResolver) GetWorkloadRegion(ctx context.Context, workloadID string) (*model.WorkLoadRegions, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	workLoadDet, err := service.GetWorkLoadManagementById(workloadID, user.ID)
	if err != nil {
		return nil, err
	}
	var userId string
	if user.RoleId != 1 {
		adminEmail, err := users.GetAdminByCompanyNameAndEmail(user.CompanyName)
		if err != nil {
			return nil, err
		}

		userid, err := users.GetUserIdByEmail(adminEmail)
		if err != nil {
			return nil, err
		}
		userId = strconv.Itoa(userid)
		workLoadDet, err = service.GetWorkLoadManagementById(workloadID, userId)
		if err != nil {
			return nil, err
		}

	} else {
		userId = user.ID
	}

	workloadRegions, err := service.GetWorkLoadRegionById(workloadID, userId)
	if err != nil {
		return nil, err
	}
	var wlRegions []*model.Region
	for _, reg := range workloadRegions {
		getRegionDet, err := service.GetRegionDetailsByCode(*reg, user.ID)
		if err != nil {
			return nil, err
		}
		wlRegions = append(wlRegions, &getRegionDet)
	}

	result := &model.WorkLoadRegions{
		ID:                   workLoadDet.ID,
		EnvironmentName:      workLoadDet.EnvironmentName,
		EnvironmentEndpoint:  workLoadDet.EnvironmentEndpoint,
		OrganizationID:       workLoadDet.OrganizationID,
		AddedWorkLoadRegions: wlRegions,
	}

	return result, nil
}
