package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/log4go"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal/auth"
	ci "github.com/nifetency/nife.io/internal/cluster_info"
	clusterDetails "github.com/nifetency/nife.io/internal/cluster_info"
	k8ssecrets "github.com/nifetency/nife.io/internal/k8s_secrets"
	organizationInfo "github.com/nifetency/nife.io/internal/organization_info"
	registryauthenticate "github.com/nifetency/nife.io/internal/registry_authenticate"
	secretregistry "github.com/nifetency/nife.io/internal/secret_registry"
	"github.com/nifetency/nife.io/internal/stripes"
	"github.com/nifetency/nife.io/internal/users"
	"github.com/nifetency/nife.io/service"
)

func (r *mutationResolver) CreateOrganization(ctx context.Context, input model.CreateOrganizationInput) (*model.CreateOrganization, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.CreateOrganization{}, fmt.Errorf("Access Denied")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("Organization Name cannot be Empty")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	// ------- Plan and permission
	var planName string
	idUser, _ := strconv.Atoi(user.ID)
	checkFreePlan, err := users.FreePlanDetails(idUser)
	if !checkFreePlan {
		planName, err = stripes.GetCustPlanName(user.CustomerStripeId)
		if err != nil {
			log4go.Error("Module: CreateOrganization, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.Email)
			return nil, err
		}
		log4go.Info("Module: CreateOrganization, MethodName: GetCustPlanName, Message: Get user plan with ProductId:"+user.StripeProductId+", user: %s", user.Email)
	}
	if checkFreePlan {
		planName = "Free Plan"
	}
	planAndPermission, err := users.GetCustomerPermissionByPlan(planName)
	if err != nil {
		log4go.Error("Module: CreateOrganization, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return nil, err
	}
	log4go.Info("Module: CreateOrganization, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+user.StripeProductId+", user: %s", user.Email)

	parentOrgCount, err := service.GetOrganizationCountById(user.ID)
	if err != nil {
		log4go.Error("Module: CreateOrganization, MethodName: GetOrganizationCountById, Message: %s user:%s", err.Error(), user.ID)
		return &model.CreateOrganization{}, err
	}
	log4go.Info("Module: CreateOrganization, MethodName: GetOrganizationCountById, Message: Check Organization Count by Id, user: %s", user.ID)

	if parentOrgCount >= planAndPermission.OrganizationCount {
		return nil, fmt.Errorf("You've reached your maximum limit of current plan. Please upgrade your plan to create Organization")
	}
	//------------
	organization, err := service.GetOrganization("", input.Name)
	if err != nil {
		log4go.Error("Module: CreateOrganization, MethodName: GetOrganization, Message: %s user:%s", err.Error(), user.ID)
		return &model.CreateOrganization{}, err
	}
	log4go.Info("Module: CreateOrganization, MethodName: GetOrganization, Message: Check organization exist or not is successfully completed, user: %s", user.ID)

	if organization.ID != nil {
		checkOrg, err := organizationInfo.CheckOrgExistByUser(*organization.ID, user.ID)
		if err != nil {
			log4go.Error("Module: CreateOrganization, MethodName: CheckOrgExistByUser, Message: %s user:%s", err.Error(), user.ID)
			return &model.CreateOrganization{}, err
		}
		log4go.Info("Module: CreateOrganization, MethodName: CheckOrgExistByUser, Message: Check organization exist is successfully completed, user: %s", user.ID)

		if checkOrg {

			randomNumber := organizationInfo.RandomNumber4Digit()
			randno := strconv.Itoa(int(randomNumber))
			input.Name = input.Name + "-" + randno

		} else if !checkOrg {
			return nil, fmt.Errorf("This Organization Name already exist for another User")
		}
	}

	_, err = organizationInfo.CreateOrgainzation(input.Name, user.ID, "0")
	if err != nil {
		log4go.Error("Module: CreateOrganization, MethodName: CreateOrgainzation, Message: %s user:%s", err.Error(), user.ID)
		return &model.CreateOrganization{}, err
	}
	log4go.Info("Module: CreateOrganization, MethodName: CreateOrgainzation, Message: Organization is successfully created, user: %s", user.ID)

	re, err := regexp.Compile(`[^\w]`)
	if err != nil {
		fmt.Println(err)
	}

	slug := re.ReplaceAllString(input.Name, "")
	slug = strings.ToLower(slug)

	org, err := service.GetOrganization("", slug)
	if err != nil {
		return &model.CreateOrganization{}, err
	}

	clusterDetails, err := clusterDetails.GetClusterDetailsByOrgIdDefault(*org.ID, "1", user.ID)
	if err != nil {
		log4go.Error("Module: CreateOrganization, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: CreateOrganization, MethodName: GetClusterDetailsByOrgId, Message: Get Cluster details based on organization is successfully completed, user: %s", user.ID)

	var clusterRegion1 *model.ClusterDetails
	if clusterDetails.Cluster_config_path == "" {

		clusterRegion1, err = service.GetUserAddedClusterDetailsByRegionCode(clusterDetails.Region_code, user.ID)
		if err != nil {
			return nil, err
		}
		k8sPath := "k8s_config/" + *clusterRegion1.RegionCode
		var fileSavePath string
		err = os.Mkdir(k8sPath, 0755)
		if err != nil {
			return nil, err
		}

		fileSavePath = k8sPath + "/config"

		_, err = organizationInfo.GetFileFromPrivateS3kubeconfigs(*clusterRegion1.ClusterConfigURL, fileSavePath)
		if err != nil {
			return nil, err
		}
		clusterDetails.Cluster_config_path = "./k8s_config/" + *clusterRegion1.RegionCode + "/config"

	}

	clientset, err := helper.LoadK8SConfig(clusterDetails.Cluster_config_path)
	if err != nil {
		helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)
		log4go.Error("Module: CreateOrganization, MethodName: LoadK8SConfig, Message: %s user:%s", err.Error(), user.ID)
		return &model.CreateOrganization{}, err
	}
	log4go.Info("Module: CreateOrganization, MethodName: LoadK8SConfig, Message:successfully reached, user: %s", user.ID)
	helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)

	err = k8ssecrets.Secret(clientset, *org.Slug)

	if err != nil {
		return &model.CreateOrganization{}, err
	}

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: CreateOrganization, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: CreateOrganization, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	AddOperation := service.Activity{
		Type:       "ORGANIZATION",
		UserId:     user.ID,
		Activities: "CREATED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Created a Organization " + input.Name,
		RefId:      *org.ID,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	err = service.SendSlackNotification(user.ID, AddOperation.Message)
	if err != nil {
		log4go.Error("Module: CreateOrganization, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return &model.CreateOrganization{Organization: org}, nil
}

func (r *mutationResolver) CreateSubOrganization(ctx context.Context, input model.CreateSubOrganizationInput) (*model.CreateOrganization, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.CreateOrganization{}, fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	if input.Name == "" {
		return nil, fmt.Errorf("Sub Organization Name cannot be Empty")
	}
	// ------- Plan and permission
	var planName string
	idUser, _ := strconv.Atoi(user.ID)
	checkFreePlan, err := users.FreePlanDetails(idUser)
	if !checkFreePlan {
		planName, err = stripes.GetCustPlanName(user.CustomerStripeId)
		if err != nil {
			log4go.Error("Module: CreateSubOrganization, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.Email)
			return nil, err
		}
		log4go.Info("Module: CreateSubOrganization, MethodName: GetCustPlanName, Message: Get user plan with ProductId:"+user.StripeProductId+", user: %s", user.Email)
	}
	if checkFreePlan {
		planName = "free plan"
	}
	planAndPermission, err := users.GetCustomerPermissionByPlan(planName)
	if err != nil {
		log4go.Error("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return nil, err
	}
	log4go.Info("Module: CreateSubOrganization, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+user.StripeProductId+", user: %s", user.Email)

	subOrgCount, err := service.GetSubOrganizationCountById(user.ID)
	if err != nil {
		log4go.Error("Module: CreateSubOrganization, MethodName: GetOrganizationCountById, Message: %s user:%s", err.Error(), user.ID)
		return &model.CreateOrganization{}, err
	}
	log4go.Info("Module: CreateSubOrganization, MethodName: GetOrganizationCountById, Message: Check Organization Count by Id, user: %s", user.ID)

	if subOrgCount >= planAndPermission.SubOrganizationCount {
		if planAndPermission.SubOrganizationCount == 0 {
			return nil, fmt.Errorf("Upgrade your plan to unlock the Sub-Organization feature")

		}
		return nil, fmt.Errorf("You've reached your maximum limit of current plan. Please upgrade your plan to create Sub-Organization")
	}
	//-------------
	organization, err := service.GetOrganization("", input.Name)
	if err != nil {
		log4go.Error("Module: CreateSubOrganization, MethodName: GetOrganization, Message: %s user:%s", err.Error(), user.ID)
		return &model.CreateOrganization{}, err
	}
	log4go.Info("Module: CreateSubOrganization, MethodName: GetOrganization, Message: Fetching Organization based on OrgId to create sub organization, sub-Organization:"+input.Name+", Parent Organization : "+input.ParentOrgID+" , user: %s", user.ID)
	if organization.ID != nil {
		checkOrg, err := organizationInfo.CheckOrgExistByUser(*organization.ID, user.ID)
		if err != nil {
			log4go.Error("Module: CreateSubOrganization, MethodName: CheckOrgExistByUser, Message: %s user:%s", err.Error(), user.ID)
			return &model.CreateOrganization{}, err
		}
		log4go.Info("Module: CreateSubOrganization, MethodName: CheckOrgExistByUser, Message: Check Organization Active or Exist for the User, Organization:"+*organization.Name+", user: %s", user.ID)
		if checkOrg {

			randomNumber := organizationInfo.RandomNumber4Digit()
			randno := strconv.Itoa(int(randomNumber))
			input.Name = input.Name + "-" + randno

		} else if !checkOrg {
			return nil, fmt.Errorf("This Sub Organization Name already exist for another User")
		}
	}

	_, err = organizationInfo.CreateSubOrgainzation(input.Name, user.ID, "0", input.ParentOrgID)
	if err != nil {
		log4go.Error("Module: CreateSubOrganization, MethodName: CreateSubOrgainzation, Message: %s user:%s", err.Error(), user.ID)
		return &model.CreateOrganization{}, err
	}
	log4go.Info("Module: CreateSubOrganization, MethodName: CreateSubOrgainzation, Message: Creating Sub-Organization , Sub-Organization:"+input.Name+", ParentOrgId: "+input.ParentOrgID+", user: %s", user.ID)

	re, err := regexp.Compile(`[^\w]`)
	if err != nil {
		fmt.Println(err)
	}

	slug := re.ReplaceAllString(input.Name, "")
	slug = strings.ToLower(slug)

	org, err := service.GetOrganization("", slug)
	if err != nil {
		log4go.Error("Module: CreateSubOrganization, MethodName: GetOrganization, Message: %s user:%s", err.Error(), user.ID)
		return &model.CreateOrganization{}, err
	}
	log4go.Info("Module: CreateSubOrganization, MethodName: GetOrganization, Message: Fetch Organization using Slug name : "+slug+", user: %s", user.ID)

	clusterDetails, err := clusterDetails.GetClusterDetailsByOrgIdDefault(*org.ID, "1", user.ID)
	if err != nil {
		log4go.Error("Module: CreateSubOrganization, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: CreateSubOrganization, MethodName: GetClusterDetailsByOrgId, Message: Fetching the cluster details based on OrganizationId : "+*org.ID+", user: %s", user.ID)

	clientset, err := helper.LoadK8SConfig(clusterDetails.Cluster_config_path)

	if err != nil {
		return &model.CreateOrganization{}, err
	}

	err = k8ssecrets.Secret(clientset, *org.Slug)

	if err != nil {
		return &model.CreateOrganization{}, err
	}

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: CreateSubOrganization, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: CreateSubOrganization, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	AddOperation := service.Activity{
		Type:       "SUB ORGANIZATION",
		UserId:     user.ID,
		Activities: "CREATED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Created a Sub Organization " + input.Name,
		RefId:      *org.ID,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	err = service.SendSlackNotification(user.ID, AddOperation.Message)
	if err != nil {
		log4go.Error("Module: CreateSubOrganization, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return &model.CreateOrganization{Organization: org}, nil
}

func (r *mutationResolver) DeleteOrganization(ctx context.Context, input model.DeleteOrganizationInput) (*model.DeleteOrganization, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.DeleteOrganization{}, fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	org, err := service.GetOrganization(input.OrganizationID, "")
	if err != nil {
		log4go.Error("Module: DeleteOrganization, MethodName: GetOrganization, Message: %s user:%s", err.Error(), user.ID)
		return &model.DeleteOrganization{}, err
	}
	log4go.Info("Module: DeleteOrganization, MethodName: GetOrganization, Message: Fetch Organization details using Organization Id: "+input.OrganizationID+" , user: %s", user.ID)
	if org.ID == nil {
		return &model.DeleteOrganization{}, fmt.Errorf("Organization Doesn't Exists")
	}
	//Delete Workload inside the Organization
	workloaddet, err := service.GetWorkLoadManagementByOrgIdSubOrgBusinessU(user.ID, input.OrganizationID, "", "")
	if err != nil {
		log4go.Error("Module: DeleteOrganization, MethodName: GetWorkLoadManagementByOrgIdSubOrgBusinessU, Message: %s user:%s", err.Error(), user.ID)
		return &model.DeleteOrganization{}, err
	}
	log4go.Info("Module: DeleteOrganization, MethodName: GetWorkLoadManagementByOrgIdSubOrgBusinessU, Message: Fetching Workload based on OrganizationId : "+input.OrganizationID+", user: %s", user.ID)

	for _, wl := range workloaddet {
		r.DeleteWorkloadManagement(ctx, wl.ID)
	}

	//------business unit
	subOrg, err := service.GetSubOrganization(input.OrganizationID)
	if err != nil {
		log4go.Error("Module: DeleteOrganization, MethodName: GetSubOrganization, Message: %s user:%s", err.Error(), user.ID)
		return &model.DeleteOrganization{}, err
	}
	log4go.Info("Module: DeleteOrganization, MethodName: GetSubOrganization, Message: Fetching Sub-Organization based on OrganizationId : "+input.OrganizationID+", user: %s", user.ID)

	if subOrg != nil {
		for _, i := range subOrg {
			r.DeleteSubOrganization(ctx, model.DeleteSubOrganizationInput{SubOrganizationID: *i.ID})
		}
	}
	businessUnit, err := service.GetBusinessUnit(input.OrganizationID, "")
	if err != nil {
		return &model.DeleteOrganization{}, err
	}
	for _, k := range businessUnit {
		err = service.DeleteBusinessUnit(*k.ID)
		if err != nil {
			return &model.DeleteOrganization{}, err
		}
	}

	err = service.DestoryDNSRecord(org)
	if err != nil {
		log4go.Error("Module: DeleteOrganization, MethodName: DestoryDNSRecord, Message: %s user:%s", err.Error(), user.ID)
		return &model.DeleteOrganization{}, err
	}
	log4go.Info("Module: DeleteOrganization, MethodName: DestoryDNSRecord, Message: DNS record is successfully destroyed, user: %s", user.ID)

	err = service.DestoryOrganizationResources(org, user.ID)
	if err != nil {
		log4go.Error("Module: DeleteOrganization, MethodName: DestoryOrganizationResources, Message: %s user:%s", err.Error(), user.ID)
		return &model.DeleteOrganization{}, err
	}
	log4go.Info("Module: DeleteOrganization, MethodName: DestoryOrganizationResources, Message: Organization resource is successfully destroyed, user: %s", user.ID)

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: DeleteOrganization, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: DeleteOrganization, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	getAllS3Deployments, err := service.GetAllS3DeploymentsByOrgId(user.ID, input.OrganizationID)
	if err != nil {
		return nil, err
	}

	for _, v := range getAllS3Deployments {
		r.DeleteS3Deployment(ctx, v.S3AppName)
	}

	DeleteOperation := service.Activity{
		Type:       "ORGANIZATION",
		UserId:     user.ID,
		Activities: "DELETED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Deleted the Organization " + *org.Name,
		RefId:      *org.ID,
	}

	_, err = service.InsertActivity(DeleteOperation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	err = service.SendSlackNotification(user.ID, DeleteOperation.Message)
	if err != nil {
		log4go.Error("Module: DeleteOrganization, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return &model.DeleteOrganization{DeletedOrganizationID: &input.OrganizationID}, err
}

func (r *mutationResolver) DeleteSubOrganization(ctx context.Context, input model.DeleteSubOrganizationInput) (*model.DeleteSubOrganization, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.DeleteSubOrganization{}, fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	org, err := service.GetOrganization(input.SubOrganizationID, "")
	if err != nil {
		log4go.Error("Module: DeleteSubOrganization, MethodName: GetOrganization, Message: %s user:%s", err.Error(), user.ID)
		return &model.DeleteSubOrganization{}, err
	}
	log4go.Info("Module: DeleteSubOrganization, MethodName: GetOrganization, Message: Fetch Organization details based on Sub-OrganizationId : "+input.SubOrganizationID+", user: %s", user.ID)
	if org.ID == nil {
		return &model.DeleteSubOrganization{}, fmt.Errorf("Sub Organization Doesn't Exists")
	}
	err = service.DestoryDNSRecord(org)
	if err != nil {
		return &model.DeleteSubOrganization{}, err
	}
	err = service.DestoryOrganizationResources(org, user.ID)
	if err != nil {
		log4go.Error("Module: DeleteSubOrganization, MethodName: DestoryOrganizationResources, Message: %s user:%s", err.Error(), user.ID)
		return &model.DeleteSubOrganization{}, err
	}
	log4go.Info("Module: DeleteSubOrganization, MethodName: DestoryOrganizationResources, Message: Deleting the namespace in kubernetes clusters using Slug Name : "+*org.Slug+", user: %s", user.ID)
	err = service.DeleteBusinessUnitByOrg(input.SubOrganizationID)
	if err != nil {
		return &model.DeleteSubOrganization{}, err
	}

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: DeleteSubOrganization, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: DeleteSubOrganization, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	DeleteOperation := service.Activity{
		Type:       "SUB ORGANIZATION",
		UserId:     user.ID,
		Activities: "DELETED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Deleted the Sub Organization " + *org.Name,
		RefId:      *org.ID,
	}

	_, err = service.InsertActivity(DeleteOperation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	err = service.SendSlackNotification(user.ID, DeleteOperation.Message)
	if err != nil {
		log4go.Error("Module: DeleteSubOrganization, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return &model.DeleteSubOrganization{DeletedSubOrganizationID: &input.SubOrganizationID}, err
}

func (r *mutationResolver) CreateOrganizationSecret(ctx context.Context, input *model.CreateSecretInput) (*model.Response, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Response{}, fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	//---------plans and permissions
	var planName string
	idUser, _ := strconv.Atoi(user.ID)
	checkFreePlan, err := users.FreePlanDetails(idUser)
	if !checkFreePlan {
		planName, err = stripes.GetCustPlanName(user.CustomerStripeId)
		if err != nil {
			log4go.Error("Module: CreateSubOrganization, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.Email)
			return nil, err
		}
		log4go.Info("Module: CreateSubOrganization, MethodName: GetCustPlanName, Message: Get user plan with ProductId:"+user.StripeProductId+", user: %s", user.Email)
	}
	if checkFreePlan {
		planName = "free plan"
	}
	planAndPermission, err := users.GetCustomerPermissionByPlan(planName)
	if err != nil {
		log4go.Error("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return nil, err
	}
	log4go.Info("Module: CreateSubOrganization, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+user.StripeProductId+", user: %s", user.Email)

	if !planAndPermission.Secret {
		return nil, fmt.Errorf("Upgrade your plan to unlock the Secrets feature")
	}
	//-------------
	secretRegistry, err := secretregistry.SecretRegData(input.RegistryInfo, input.RegistryType, input.Name)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: CreateOrganizationSecret, MethodName: SecretRegData, Message: %s user:%s", err.Error(), user.ID)
		return &model.Response{}, err
	}
	log4go.Info("Module: CreateOrganizationSecret, MethodName: SecretRegData, Message:successfully reached, user: %s", user.ID)

	if input.RegistryType == "" {
		request := service.OrgSecret{
			UserId:         user.ID,
			Name:           input.Name,
			OrganizationId: input.OrganizationID,
			RegistryType:   "env",
			Response:       secretRegistry,
		}
		_, err := request.CreateSecret()
		if err != nil {
			log.Println(err)
			log4go.Error("Module: CreateOrganizationSecret, MethodName: CreateSecret, Message: %s user:%s", err.Error(), user.ID)
			return &model.Response{}, err
		}
		log4go.Info("Module: CreateOrganizationSecret, MethodName: CreateSecret, Message: Secrets is successfully created, user: %s", user.ID)

		return &model.Response{
			Message: "Validated and Created Successfully",
		}, nil

	} else if input.RegistryType == "PAT" {
		request := service.OrgSecret{
			UserId:         user.ID,
			Name:           input.Name,
			OrganizationId: input.OrganizationID,
			RegistryType:   "PAT",
			Response:       secretRegistry,
		}

		patCheck, err := service.GetUserSecretByRegistryType(input.Name, user.ID, input.RegistryType)
		if err != nil {
			return &model.Response{}, err
		}
		if patCheck != "" {
			return nil, fmt.Errorf("The given PAT name is already created")
		}

		_, err = request.CreateSecret()
		if err != nil {
			log.Println(err)
			return &model.Response{}, err
		}

		return &model.Response{
			Message: "Validated and Created Successfully",
		}, nil

	}
	_, createdBy, err := secretregistry.CheckRegistryType(input.Name)

	if createdBy != "" && createdBy == user.ID {
		return &model.Response{}, fmt.Errorf("Secret Exists for %s", input.Name)
	}

	if err != nil {
		log.Println(err)
		return &model.Response{}, err
	}

	secretRegistry, err = secretregistry.SecretRegData(input.RegistryInfo, input.RegistryType, "")
	if err != nil {
		log.Println(err)
		log4go.Error("Module: CreateOrganizationSecret, MethodName: SecretRegData, Message: %s user:%s", err.Error(), user.ID)
		return &model.Response{}, err
	}
	log4go.Info("Module: CreateOrganizationSecret, MethodName: SecretRegData, Message:successfully reached, user: %s", user.ID)
	if input.RegistryType != "mysql" && input.RegistryType != "postgres" {
		err = registryauthenticate.SecretAuthentication(input.RegistryType, secretRegistry.UserName, secretRegistry.Password)

		if err != nil {
			log.Println(err)
			log4go.Error("Module: CreateOrganizationSecret, MethodName: SecretAuthentication, Message: %s user:%s", err.Error(), user.ID)
			return &model.Response{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: CreateOrganizationSecret, MethodName: SecretAuthentication, Message: Authenticating secrets with secret registry is successfully completed, user: %s", user.ID)
	}

	request := service.OrgSecret{
		UserId:         user.ID,
		Name:           input.Name,
		OrganizationId: input.OrganizationID,
		RegistryType:   input.RegistryType,
		Response:       secretRegistry,
	}

	_, err = request.CreateSecret()
	if err != nil {
		log4go.Error("Module: CreateOrganizationSecret, MethodName: CreateSecret, Message: %s user:%s", err.Error(), user.ID)
		return &model.Response{}, err
	}
	log4go.Info("Module: CreateOrganizationSecret, MethodName: CreateSecret, Message: Inserting secrets to database, Secret Name: "+input.Name+", Registry Type: "+input.RegistryType+", OrganizationId: "+input.OrganizationID+" , user: %s", user.ID)

	secretDet, err := service.GetRegId(input.Name)
	if err != nil {
		log.Println(err)
		return &model.Response{}, err
	}

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: CreateOrganizationSecret, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: CreateOrganizationSecret, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	AddOperation := service.Activity{
		Type:       "SECRET",
		UserId:     user.ID,
		Activities: "CREATED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Created a Secret " + input.Name,
		RefId:      secretDet,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	err = service.SendSlackNotification(user.ID, AddOperation.Message)
	if err != nil {
		log4go.Error("Module: CreateOrganizationSecret, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return &model.Response{
		Message: "Validated and Created Successfully",
	}, nil
}

func (r *mutationResolver) UpdateOrganizationSecret(ctx context.Context, name *string, input *model.UpdateSecretInput) (*model.Response, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Response{}, fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	registrytype, createdBy, err := secretregistry.CheckRegistryType(*name)

	if err != nil {
		log.Println(err)
		log4go.Error("Module: UpdateOrganizationSecret, MethodName: CheckRegistryType, Message: %s user:%s", err.Error(), user.ID)
		return &model.Response{}, err
	}
	log4go.Info("Module: UpdateOrganizationSecret, MethodName: CheckRegistryType, Message: Checking registry type is successfully completed, user: %s", user.ID)

	if createdBy != user.ID {
		return &model.Response{}, fmt.Errorf("Incorrect Registry Please Check the Name %s", *name)
	}

	if registrytype != input.RegistryType {
		return &model.Response{}, fmt.Errorf("Incorrect Registry Type, Please Check the Type")
	}

	secretRegistry, err := secretregistry.SecretRegData(input.RegistryInfo, input.RegistryType, "")

	if err != nil {
		log.Println(err)
		log4go.Error("Module: UpdateOrganizationSecret, MethodName: SecretRegData, Message: %s user:%s", err.Error(), user.ID)
		return &model.Response{}, err
	}
	log4go.Info("Module: UpdateOrganizationSecret, MethodName: SecretRegData, Message:successfully reached, user: %s", user.ID)

	request := service.OrgSecret{
		RegistryType: input.RegistryType,
		Response:     secretRegistry,
	}
	if registrytype != "PAT" && registrytype != "env" {
		err = registryauthenticate.SecretAuthentication(input.RegistryType, secretRegistry.UserName, secretRegistry.Password)

		if err != nil {
			log.Println(err)
			log4go.Error("Module: UpdateOrganizationSecret, MethodName: SecretAuthentication, Message: %s user:%s", err.Error(), user.ID)
			return &model.Response{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: UpdateOrganizationSecret, MethodName: SecretAuthentication, Message: Authenticating secrets with secret registry is successfully completed, user: %s", user.ID)
	}
	_, err = request.UpdateSecret(*name)

	if err != nil {
		log.Println(err)
		return &model.Response{}, err
	}

	return &model.Response{
		Message: "Updated Successfully",
	}, nil
}

func (r *mutationResolver) DeleteOrganizationSecret(ctx context.Context, name *string, id *string) (*model.Response, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Response{}, fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	secretDet, err := service.GetRegId(*name)
	if err != nil {
		log.Println(err)
		return &model.Response{}, err
	}
	if secretDet == "" {
		return nil, fmt.Errorf("Invalid secret name")
	}

	_, err = service.DeleteSecretOrganization(*name, *id)

	if err != nil {
		log.Println(err)
		log4go.Error("Module: DeleteOrganizationSecret, MethodName: DeleteSecretOrganization, Message: %s user:%s", err.Error(), user.ID)
		return &model.Response{}, err
	}
	log4go.Info("Module: DeleteOrganizationSecret, MethodName: DeleteSecretOrganization, Message: Organization secrets is successfully deleted, user: %s", user.ID)

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: DeleteOrganizationSecret, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: DeleteOrganizationSecret, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	UpdateOperation := service.Activity{
		Type:       "SECRET",
		UserId:     user.ID,
		Activities: "DELETED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Deleted a Secret " + *name,
		RefId:      secretDet,
	}

	_, err = service.InsertActivity(UpdateOperation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	err = service.SendSlackNotification(user.ID, UpdateOperation.Message)
	if err != nil {
		log4go.Error("Module: DeleteOrganizationSecret, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return &model.Response{
		Message: "Deleted Successfully",
	}, nil
}

func (r *mutationResolver) UpdateRegistryIDToApp(ctx context.Context, appName string, name *string) (*model.Response, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Response{}, fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	resId, err := service.GetRegId(*name)

	if err != nil {
		return &model.Response{}, err
	}

	if resId == "" {
		return &model.Response{}, fmt.Errorf("Registry Doesn't Exist for %s name", *name)
	}

	err = service.UpdateRegId(appName, resId)

	if err != nil {
		return &model.Response{}, err
	}

	return &model.Response{
		Message: "Updated reg Id",
	}, nil
}

func (r *mutationResolver) UpdateOrganization(ctx context.Context, org *string, defaulttype *bool) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	roleId, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}
	if roleId == 2 {
		return "", fmt.Errorf("Members role is restricted to update the default organization")
	}

	allOrganizations, err := service.AllOrganizations(user.ID)
	if err != nil {
		return "", err
	}
	if *defaulttype == false {
		return "", fmt.Errorf("Cannot remove the organization from default. At least one organization must be set as the default")
	}
	for _, org1 := range allOrganizations.Nodes {
		if *org1.Slug == *org {
			err = service.UpdateOrganization(true, *org)
			if err != nil {
				log4go.Error("Module: UpdateOrganization, MethodName: UpdateOrganization, Message: %s user:%s", err.Error(), user.ID)
				return "", fmt.Errorf(err.Error())
			}
			log4go.Info("Module: UpdateOrganization, MethodName: UpdateOrganization, Message: Organization is successfully updated, user: %s", user.ID)
		} else {
			err = service.UpdateOrganization(false, *org1.Slug)
			if err != nil {
				log4go.Error("Module: UpdateOrganization, MethodName: UpdateOrganization, Message: %s user:%s", err.Error(), user.ID)
				return "", fmt.Errorf(err.Error())
			}
		}

	}
	return "Updated successfully", nil
}

func (r *mutationResolver) CreateNamespaceInCluster(ctx context.Context, input *model.CreateNamespace) (string, error) {
	clusterDetails, err := ci.GetActiveClusterDetails("")
	if err != nil {
		return "", err
	}

	for _, orgname := range input.Name {

		for _, c := range *clusterDetails {

			if c.Cluster_config_path != "" {

				err = organizationInfo.CreateNamespaceInCluster(*orgname, c.Cluster_config_path)
				if err != nil {
					log.Println(err)
				}
			}

		}
	}

	return "", nil
}

func (r *mutationResolver) AddUserAddedregionsToOrganizatiom(ctx context.Context, organizationID []*string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	allRegions, err := service.GetClusterDetails(user.ID)
	if err != nil {
		return "", err
	}

	missingRegions := []string{}

	for _, orgId := range organizationID {
		orgRegions, err := service.GetOrganizationRegionByOrgId(*orgId)
		if err != nil {
			return "", err
		}

		present := make(map[string]bool)
		for _, num := range orgRegions {
			present[*num.RegionCode] = true
		}

		for _, num := range allRegions {
			if !present[*num.RegionCode] {
				missingRegions = append(missingRegions, *num.RegionCode)
			}
		}
		for _, missingReg := range missingRegions {
			_, err = service.InsertOrganizationRegion(*orgId, missingReg, user.ID)

		}
		missingRegions = []string{}

	}

	return "Successfully Inserted", nil
}

func (r *queryResolver) Organizations(ctx context.Context) (*model.Organizations, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Organizations{}, fmt.Errorf("Access Denied")
	}

	orgs, err := service.AllOrganizations(user.ID)
	if err != nil {
		log4go.Error("Module: Organizations, MethodName: AllOrganizations, Message: %s user:%s", err.Error(), user.ID)
		return orgs, err
	}
	log4go.Info("Module: Organizations, MethodName: AllOrganizations, Message: Fetching all organization is successfully completed, user: %s", user.ID)

	return orgs, nil
}

func (r *queryResolver) GetAllParentOrganizations(ctx context.Context) (*model.Organizations, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Organizations{}, fmt.Errorf("Access Denied")
	}

	orgs, err := service.AllParentOrganizations(user.ID)
	if err != nil {
		log4go.Error("Module: Organizations, MethodName: AllOrganizations, Message: %s user:%s", err.Error(), user.ID)
		return orgs, err
	}
	log4go.Info("Module: Organizations, MethodName: AllOrganizations, Message: Fetching all organization is successfully completed, user: %s", user.ID)

	return orgs, nil
}

func (r *queryResolver) OrganizationsandBusinessUnit(ctx context.Context) (*model.OrganizationsandBusinessUnit, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.OrganizationsandBusinessUnit{}, fmt.Errorf("Access Denied")
	}

	orgs, err := service.AllOrganizationsandbus(user.ID)
	if err != nil {
		log4go.Error("Module: OrganizationsandBusinessUnit, MethodName: AllOrganizations, Message: %s user:%s", err.Error(), user.ID)
		return &model.OrganizationsandBusinessUnit{}, err
	}
	log4go.Info("Module: OrganizationsandBusinessUnit, MethodName: AllOrganizations, Message: Fetching all organization is successfully completed, user: %s", user.ID)

	businessU, err := service.GetAllBusinessUnit(user.ID)
	if err != nil {
		log4go.Error("Module: OrganizationsandBusinessUnit, MethodName: GetAllBusinessUnit, Message: %s user:%s", err.Error(), user.ID)
		return &model.OrganizationsandBusinessUnit{}, err
	}
	log4go.Info("Module: OrganizationsandBusinessUnit, MethodName: GetAllBusinessUnit, Message: Fetching all Business Uniy is successfully completed, user: %s", user.ID)

	result := model.OrganizationsandBusinessUnit{
		Nodes:        orgs.Nodes,
		BusinessUnit: businessU,
	}

	return &result, nil
}

func (r *queryResolver) SubOrganizations(ctx context.Context) (*model.Organizations, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Organizations{}, fmt.Errorf("Access Denied")
	}

	orgs, err := service.SubOrganizations(user.ID)
	if err != nil {
		log4go.Error("Module: SubOrganizations, MethodName: AllSubOrganizations, Message: %s user:%s", err.Error(), user.ID)
		return orgs, err
	}
	log4go.Info("Module: SubOrganizations, MethodName: AllSubOrganizations, Message: Fetching all organization is successfully completed, user: %s", user.ID)

	return orgs, nil
}

func (r *queryResolver) SubOrganizationsByParentID(ctx context.Context, parentOrgID *string) (*model.Organizations, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Organizations{}, fmt.Errorf("Access Denied")
	}
	orgs, err := service.SubOrganizationsById(user.ID, *parentOrgID)
	if err != nil {
		log4go.Error("Module: SubOrganizationsByParentID, MethodName: SubOrganizationsById, Message: %s user:%s", err.Error(), user.ID)
		return orgs, err
	}
	log4go.Info("Module: SubOrganizationsByParentID, MethodName: SubOrganizationsById, Message: Fetching all organization is successfully completed, user: %s", user.ID)

	return orgs, nil
}

func (r *queryResolver) GetParentIDBySubOrganization(ctx context.Context, subOrgID *string) (*model.Organizations, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Organizations{}, fmt.Errorf("Access Denied")
	}
	orgs, err := service.PrentIdBySubOrganization(user.ID, *subOrgID)
	if err != nil {
		log4go.Error("Module: GetParentIDBySubOrganization, MethodName: PrentIdBySubOrganization, Message: %s user:%s", err.Error(), user.ID)
		return orgs, err
	}
	log4go.Info("Module: GetParentIDBySubOrganization, MethodName: PrentIdBySubOrganization, Message: Fetching all organization is successfully completed, user: %s", user.ID)

	return orgs, nil
}

func (r *queryResolver) Organization(ctx context.Context, slug string) (*model.OrganizationDetails, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.OrganizationDetails{}, fmt.Errorf("Access Denied")
	}

	orgDetails, err := service.GetOrgDetails(slug)
	if err != nil {
		log4go.Error("Module: Organization, MethodName: GetOrgDetails, Message: %s user:%s", err.Error(), user.ID)
		return &model.OrganizationDetails{}, err
	}
	log4go.Info("Module: Organization, MethodName: GetOrgDetails, Message: Fetching organization details is successfully completed, user: %s", user.ID)

	Role, err := service.GetRoleByUserId(user.ID)
	if err != nil {
		log4go.Error("Module: Organization, MethodName: GetRoleByUserId, Message: %s user:%s", err.Error(), user.ID)
		return &model.OrganizationDetails{}, err
	}
	log4go.Info("Module: Organization, MethodName: GetRoleByUserId, Message: Get user role by user Id is successfully completed, user: %s", user.ID)

	orgDetails.ViewerRole = &Role

	return orgDetails, nil
}

func (r *queryResolver) GetOrganizationByOrgID(ctx context.Context, id string) (*model.Organization, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Organization{}, fmt.Errorf("Access Denied")
	}

	orgDetails, err := service.GetOrganization(id, "")
	if err != nil {
		log4go.Error("Module: GetOrganizationByOrgID, MethodName: GetOrganization, Message: %s user:%s", err.Error(), user.ID)
		return &model.Organization{}, err
	}
	log4go.Info("Module: GetOrganizationByOrgID, MethodName: GetOrganization, Message: Fetching organization details is by OrganizationId: "+id+", user: %s", user.ID)

	return orgDetails, nil
}

func (r *queryResolver) OrganizationRegistryType(ctx context.Context) ([]*model.OrganizationRegistryType, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.OrganizationRegistryType{}, fmt.Errorf("Access Denied")
	}
	orgType, err := service.GetRegistryType()
	if err != nil {
		log4go.Error("Module: OrganizationRegistryType, MethodName: GetRegistryType, Message: %s user:%s", err.Error(), user.ID)
		return []*model.OrganizationRegistryType{}, err
	}
	log4go.Info("Module: OrganizationRegistryType, MethodName: GetRegistryType, Message: Fetching registry type is successfully completed, user: %s", user.ID)

	return orgType, nil
}

func (r *queryResolver) GetSecret(ctx context.Context, name *string) ([]*model.GetUserSecret, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.GetUserSecret{}, fmt.Errorf("Access Denied")
	}
	var response []*model.GetUserSecret

	if user.RoleId != 1 {

		orgDetails, err := service.AllOrganizations(user.ID)
		if err != nil {
			return nil, err
		}
		for _, orgId := range orgDetails.Nodes {
			secretDet, err := service.GetUserSecretByOrgId(*orgId.ID)
			if err != nil {
				return nil, err
			}
			response = append(response, secretDet...)
		}

	} else {

		responses, err := service.GetUserSecret(*name, user.ID)

		if err != nil {
			log.Println(err)
			log4go.Error("Module: GetSecret, MethodName: GetUserSecret, Message: %s user:%s", err.Error(), user.ID)
			return []*model.GetUserSecret{}, err
		}
		log4go.Info("Module: GetSecret, MethodName: GetUserSecret, Message: Fetching secrets based on user is successfully completed, user: %s", user.ID)
		response = append(response, responses...)
	}
	return response, nil
}

func (r *queryResolver) GetRegistryByUser(ctx context.Context, orgID string, regType string) ([]*model.GetSecRegistry, error) {
	user := auth.ForContext(ctx)

	if user == nil {
		return []*model.GetSecRegistry{}, fmt.Errorf("Access Denied")
	}

	regList, err := service.GetRegistryNameList(user.ID, orgID, regType)

	if err != nil {
		log.Println(err)
		log4go.Error("Module: GetRegistryByUser, MethodName: GetRegistryNameList, Message: %s user:%s", err.Error(), user.ID)
		return []*model.GetSecRegistry{}, err
	}
	log4go.Info("Module: GetRegistryByUser, MethodName: GetRegistryNameList, Message: Fetching registry name list is successfully completed, user: %s", user.ID)

	if regList == nil {
		return []*model.GetSecRegistry{}, fmt.Errorf("There is no registry for the user, create a new one")
	}
	return regList, nil
}

func (r *queryResolver) GetAppByRegionCount(ctx context.Context) (*model.OrgCountDetails, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.OrgCountDetails{}, fmt.Errorf("Access Denied")
	}
	var roleId string

	getRole, err := service.GetRoleByUserId(user.ID)
	if err != nil {
		log4go.Error("Module: GetAppByRegionCount, MethodName: GetRoleByUserId, Message: %s user:%s", err.Error(), user.ID)
		return &model.OrgCountDetails{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetAppByRegionCount, MethodName: GetRoleByUserId, Message: Get user role by user Id is successfully completed, user: %s", user.ID)

	if getRole == "Admin" {
		roleId = "1"
	} else {
		roleId = "2"
	}

	countDetails, err := service.GetAppCountByOrg(user.ID, roleId)

	if err != nil {
		log4go.Error("Module: GetAppByRegionCount, MethodName: GetAppCountByOrg, Message: %s user:%s", err.Error(), user.ID)
		return &model.OrgCountDetails{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetAppByRegionCount, MethodName: GetAppCountByOrg, Message: Fetching apps count based on organization is successfully completed, user: %s", user.ID)

	orgCount, err := service.GetOrganizationCountById(user.ID)

	if err != nil {
		log4go.Error("Module: GetAppByRegionCount, MethodName: GetOrganizationCountById, Message: %s user:%s", err.Error(), user.ID)
		return &model.OrgCountDetails{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetAppByRegionCount, MethodName: GetOrganizationCountById, Message: Fetching organization count by userId is successfully completed, user: %s", user.ID)

	return &model.OrgCountDetails{
		TotalOrgCount: &orgCount,
		OrgByAppCount: countDetails,
	}, nil
}

func (r *queryResolver) GetSecretByRegistryID(ctx context.Context, secretID *string) (*model.GetUserSecret, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.GetUserSecret{}, fmt.Errorf("Access Denied")
	}

	secDet, err := service.GetSecretBySecId(*secretID, user.ID)
	if err != nil {
		log4go.Error("Module: GetSecretByRegistryID, MethodName: GetSecretBySecId, Message: %s user:%s", err.Error(), user.ID)
		return &model.GetUserSecret{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetSecretByRegistryID, MethodName: GetSecretBySecId, Message: Fetching secrets details using secretId: "+*secretID+", user: %s", user.ID)

	return secDet, nil
}
