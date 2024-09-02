package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/alecthomas/log4go"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal/auth"
	clusterDetails "github.com/nifetency/nife.io/internal/cluster_info"
	"github.com/nifetency/nife.io/internal/links"
	"github.com/nifetency/nife.io/internal/stripes"
	"github.com/nifetency/nife.io/internal/users"
	"github.com/nifetency/nife.io/service"
	yaml "gopkg.in/yaml.v3"
)

func (r *mutationResolver) AddRegionUsingKubeConfig(ctx context.Context, input *model.ClusterDetailsInput) (string, error) {
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
			log4go.Error("Module: AddRegionUsingKubeConfig, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.Email)
			return "", err
		}
		log4go.Info("Module: AddRegionUsingKubeConfig, MethodName: GetCustPlanName, Message: Get user plan with ProductId:"+user.StripeProductId+", user: %s", user.Email)
	}
	if checkFreePlan {
		planName = "Free Plan"
	}
	planAndPermission, err := users.GetCustomerPermissionByPlan(planName)
	if err != nil {
		log4go.Error("Module: AddRegionUsingKubeConfig, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return "", err
	}
	log4go.Info("Module: AddRegionUsingKubeConfig, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+user.StripeProductId+", user: %s", user.Email)

	k8sRegionCount, err := service.CheckUserAddedRegion(user.ID)

	if k8sRegionCount >= planAndPermission.K8sRegions {
		return "", fmt.Errorf("You've reached your maximum limit of current plan. Please upgrade your plan to add another Region")
	}
	if *input.RegionName == "" {
		return "", fmt.Errorf("region name cannot be empty")
	}
	if *input.ProviderType == "" {
		return "", fmt.Errorf("provider type cannot be empty")
	}

	fileNames, err := service.SplitUrl(*input.ClusterConfigURL)
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}

	_, err = links.GetFileFromPrivateS3kubeconfig(*input.ClusterConfigURL, fileNames)
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}

	var kubeConfig service.Config

	f, err := os.Open(fileNames)
	if err != nil {
		f.Close()
		helper.DeletedSourceFile(fileNames)
		return "", fmt.Errorf(err.Error())
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&kubeConfig)
	if err != nil {
		f.Close()
		helper.DeletedSourceFile(fileNames)
		return "", err
	}

	for _, clusterName := range kubeConfig.Clusters {
		clusterNames, err := service.SplitUrl(clusterName.Name)
		if err != nil {
			f.Close()
			helper.DeletedSourceFile(clusterNames)
			return "", fmt.Errorf(err.Error())
		}
		input.RegionCode = &clusterNames
	}

	err = service.AddUserK8sRegion(*input, user.ID)
	if err != nil {
		f.Close()
		helper.DeletedSourceFile(fileNames)
		return "", err
	}
	f.Close()
	err = helper.DeletedSourceFile(fileNames)

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: AddRegionUsingKubeConfig, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: AddRegionUsingKubeConfig, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	AddOperation := service.Activity{
		Type:       "BYOC REGION",
		UserId:     user.ID,
		Activities: "ADDED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Added a Byoc Region" + *input.RegionName,
		RefId:      user.ID,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	err = service.SendSlackNotification(user.ID, AddOperation.Message)
	if err != nil {
		log4go.Error("Module: AddRegionUsingKubeConfig, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return "Successfully Inserted ", nil
}

func (r *mutationResolver) DeleteKubeConfigRegion(ctx context.Context, id *string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	checkReg, err := service.CheckUserRegionById(user.ID, *id)

	if checkReg.RegionCode == nil {
		return "", fmt.Errorf("Can't find the selected region")
	}

	checkApps, err := service.AllApps(user.ID, *checkReg.RegionCode, "")
	if checkApps.Nodes != nil {
		return "", fmt.Errorf("Please delete the Apps mapped with the BYOC Region and try deleting the BYOC Region")
	}

	err = service.DeleteUserK8sRegion(*id, user.ID)
	if err != nil {
		return "", err
	}

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: DeleteKubeConfigRegion, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: DeleteKubeConfigRegion, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	AddOperation := service.Activity{
		Type:       "BYOC REGION",
		UserId:     user.ID,
		Activities: "REMOVED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Removed a Byoc Region" + *checkReg.RegionName,
		RefId:      user.ID,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	err = service.SendSlackNotification(user.ID, AddOperation.Message)
	if err != nil {
		log4go.Error("Module: DeleteKubeConfigRegion, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return "Successfully Deleted", nil
}

func (r *queryResolver) GetClusterDetails(ctx context.Context, regCode *string) (*model.ClusterDetails, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.ClusterDetails{}, fmt.Errorf("Access Denied")
	}
	result, err := clusterDetails.GetClusterDetails(*regCode, user.ID)

	if err != nil {
		return &model.ClusterDetails{}, err
	}

	return result, nil
}

func (r *queryResolver) GetClusterDetailsByOrgID(ctx context.Context, orgID *string) (*model.ClusterDetails, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.ClusterDetails{}, fmt.Errorf("Access Denied")
	}
	clusterDetails, err := clusterDetails.GetClusterDetailsByOrgId(*orgID, "IND", "default", user.ID)
	if err != nil {
		return &model.ClusterDetails{}, err
	}

	cluster := &model.ClusterDetails{
		RegionCode:            &clusterDetails.Region_code,
		ProviderType:          clusterDetails.ProviderType,
		ClusterType:           &clusterDetails.ClusterType,
		RegionName:            clusterDetails.RegionName,
		ExternalBaseAddress:   clusterDetails.ExternalBaseAddress,
		ExternalAgentPlatForm: clusterDetails.ExternalAgentPlatform,
		ExternalLBType:        clusterDetails.ExternalLBType,
		ExternalCloudType:     clusterDetails.ExternalCloudType,
		InterfaceType:         clusterDetails.Interface,
		Route53countryCode:    clusterDetails.Route53CountryCode,
		TenantID:              clusterDetails.TenantId,
		AllocationTag:         &clusterDetails.AllocationTag,
	}

	return cluster, nil
}

func (r *queryResolver) GetClusterDetailsByOrgIDMultipleReg(ctx context.Context, orgID *string) ([]*model.ClusterDetails, error) {
	clusterDetails, err := clusterDetails.GetAllClusterDetailsByOrgId(*orgID)
	if err != nil {
		return []*model.ClusterDetails{}, err
	}
	var result []*model.ClusterDetails
	for _, clus := range clusterDetails {

		cluster := &model.ClusterDetails{
			RegionCode:            &clus.Region_code,
			ProviderType:          clus.ProviderType,
			ClusterType:           &clus.ClusterType,
			RegionName:            clus.RegionName,
			ExternalBaseAddress:   clus.ExternalBaseAddress,
			ExternalAgentPlatForm: clus.ExternalAgentPlatform,
			ExternalLBType:        clus.ExternalLBType,
			ExternalCloudType:     clus.ExternalCloudType,
			InterfaceType:         clus.Interface,
			Route53countryCode:    clus.Route53CountryCode,
			TenantID:              clus.TenantId,
			AllocationTag:         &clus.AllocationTag,
			IsDefault:             &clus.IsDefault,
		}
		result = append(result, cluster)
	}

	return result, nil
}

func (r *queryResolver) GetUserAddedRegions(ctx context.Context) ([]*model.ClusterDetails, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.ClusterDetails{}, fmt.Errorf("Access Denied")
	}
	var adminClusterDetails []*model.ClusterDetails
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
		adminClusterDetails, err = clusterDetails.GetAllUserAddedClusterDetailsByUserId(userId)
		if err != nil {
			return []*model.ClusterDetails{}, err
		}
	}

	clusterDetails, err := clusterDetails.GetAllUserAddedClusterDetailsByUserId(user.ID)
	if err != nil {
		return []*model.ClusterDetails{}, err
	}

	adminClusterDetails = append(adminClusterDetails, clusterDetails...)
	return adminClusterDetails, nil
}

func (r *queryResolver) GetCloudRegions(ctx context.Context, typeArg *string) ([]*model.CloudRegions, error) {
	clusterRegions, err := clusterDetails.GetCloudRegion(*typeArg)
	if err != nil {
		return []*model.CloudRegions{}, err
	}

	return clusterRegions, nil
}
