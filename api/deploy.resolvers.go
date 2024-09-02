package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/log4go"
	"github.com/codeclysm/extract"
	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	appDeployments "github.com/nifetency/nife.io/internal/app_deployments"
	apprelease "github.com/nifetency/nife.io/internal/app_release"
	"github.com/nifetency/nife.io/internal/auth"
	"github.com/nifetency/nife.io/internal/builtin"
	clusterDetails "github.com/nifetency/nife.io/internal/cluster_info"
	clusterInfo "github.com/nifetency/nife.io/internal/cluster_info"
	customerdisplayimage "github.com/nifetency/nife.io/internal/customer_display_image"
	"github.com/nifetency/nife.io/internal/decode"
	organizationInfo "github.com/nifetency/nife.io/internal/organization_info"
	secretregistry "github.com/nifetency/nife.io/internal/secret_registry"
	"github.com/nifetency/nife.io/internal/stripes"
	"github.com/nifetency/nife.io/internal/users"
	_helper "github.com/nifetency/nife.io/pkg/helper"
	"github.com/nifetency/nife.io/service"
)

func (r *mutationResolver) DeployImage(ctx context.Context, input model.DeployImageInput) (*model.DeployImage, error) {
	emptyRegion := make([]*string, 0)
	addRegionItem := make([]*string, 0)
	stable, inprogress := true, true
	description := ""
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	startTime := time.Now()

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	// companyName, err := users.GetCompanyNameById(user.ID)
	// if err != nil {
	// 	return nil, err
	// }

	// adminMailid, adminName, err := users.GetAdminByCompanyName(companyName, 1)
	// if err != nil {
	// 	return nil, err
	// }

	// regName, err := clusterInfo.GetClusterDetails("IND")
	// if err != nil {
	// 	return nil, err
	// }

	// err = helper.DeployMail(adminName, adminMailid, user.FirstName+" "+user.LastName, input.AppID, *regName.RegionName, "Deploy")
	// if err != nil {
	// 	return nil, err
	// }

	if len(input.EnvMapArgs) != 0 {
		err := _helper.UpdateEnvArgs(input.AppID, input.EnvMapArgs)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: UpdateEnvArgs, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: DeployImage, MethodName: UpdateEnvArgs, Message: EnvArgs updated successfully, user: %s", user.ID)
	}

	currentApp, err := service.GetApp(input.AppID, user.ID)
	if err != nil {
		log4go.Error("Module: DeployImage, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: DeployImage, MethodName: GetApp, Message: Get app details by app name, user: %s", user.ID)
	var OrgId string
	var orgSlug string

	if *currentApp.SubOrganizationID != "" {
		getSubOrg, err := service.GetOrganizationById(*currentApp.SubOrganizationID)
		if err != nil {
			return nil, err
		}

		OrgId = *currentApp.SubOrganizationID
		orgSlug = *getSubOrg.Slug
	} else {
		OrgId = *currentApp.Organization.ID
		orgSlug = *currentApp.Organization.Slug
	}
	var clusterDet []*clusterInfo.ClusterDetail
	var adminUserId string
	if user.RoleId != 1 {
		adminEmail, err := users.GetAdminByCompanyNameAndEmail(user.CompanyName)
		if err != nil {
			return nil, err
		}

		userid, err := users.GetUserIdByEmail(adminEmail)
		if err != nil {
			return nil, err
		}
		adminUserId = strconv.Itoa(userid)
	} else {
		adminUserId = user.ID
	}
	if *currentApp.WorkloadManagementID == "" {
		clusterDet, err = clusterDetails.GetClusterDetailsByOrgIdArr(OrgId, "1", adminUserId)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: DeployImage, MethodName: GetClusterDetailsByOrgId, Message: Get Cluster details based on organization is successfully completed, user: %s", user.ID)
	} else {
		workloadRegions, err := service.GetWorkLoadRegionById(*currentApp.WorkloadManagementID, adminUserId)
		if err != nil {
			return nil, err
		}

		if workloadRegions != nil {

			for _, reg := range workloadRegions {

				clusterDets, err := clusterDetails.GetClusterDetailsStruct(*reg, user.ID)
				if err != nil {
					return nil, err
				}
				clusterDets.IsDefault = 1
				clusterDet = append(clusterDet, clusterDets)
			}
		} else {
			clusterDet, err = clusterDetails.GetClusterDetailsByOrgIdArr(OrgId, "1", user.ID)
			if err != nil {
				log4go.Error("Module: DeployImage, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
				return nil, err
			}
			log4go.Info("Module: DeployImage, MethodName: GetClusterDetailsByOrgId, Message: Get Cluster details based on organization is successfully completed, user: %s", user.ID)
		}
	}
	var clusterDetails *clusterInfo.ClusterDetail
	clusterDetails = clusterDet[0]

	internalPort, _ := _helper.GetInternalPort(input.Definition)

	fmt.Println(internalPort)

	secretRegid := ""
	registryName := ""

	if currentApp.SecretRegistryID != nil {
		secretRegid = *currentApp.SecretRegistryID
		registry, err := secretregistry.GetSecretDetails(secretRegid, "")
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: GetSecretDetails, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: DeployImage, MethodName: GetSecretDetails, Message: Fetching secret details is succesfully completed, user: %s", user.ID)

		registryName = *registry.Name

	}

	memoryResource := _helper.GetResourceRequirement(currentApp.Config.Definition)

	nullCheckStruct := model.Requirement{}
	empty := ""

	if memoryResource == nullCheckStruct {
		memoryResource = model.Requirement{
			RequestRequirement: &model.RequirementProperties{CPU: &empty, Memory: &empty},
			LimitRequirement:   &model.RequirementProperties{CPU: &empty, Memory: &empty},
		}
	}

	var environmentArgument string

	if currentApp.EnvArgs != nil && *currentApp.EnvArgs != "" || registryName != "" || *memoryResource.LimitRequirement.Memory != "" {

		env := ""
		if currentApp.EnvArgs != nil {
			env = *currentApp.EnvArgs

		}
		environmentArgument, err = helper.EnvironmentArgument(env, *clusterDetails.Interface, registryName, memoryResource)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: EnvironmentArgument, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: DeployImage, MethodName: EnvironmentArgument, Message:successfully reached, user: %s", user.ID)
	}

	externalPort, _ := _helper.GetExternalPort(currentApp.Config.Definition)

	if *clusterDetails.Interface == "kube_config" {

		err := service.DeployType(1, input.AppID)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: DeployType, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: DeployImage, MethodName: DeployType, Message: Get the deploy type of an App is successfully completed, user: %s", user.ID)

		deployOutput, err := service.Deploy(input.AppID, input.Image, secretRegid, orgSlug, int32(internalPort), *clusterDetails, environmentArgument, memoryResource, user.ID, false)

		if err != nil {
			_ = service.ErrorActivity(user.ID, input.AppID, err.Error())
			log4go.Error("Module: DeployImage, MethodName: Deploy, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: DeployImage, MethodName: Deploy, Message: Deploying app in successfully completed, user: %s", user.ID)

		if input.Image != "mysql:5.6" && input.Image != "postgres:10.1" && clusterDetails.ClusterType != "byoh" {
			// deployOutput.URL = fmt.Sprintf(deployOutput.URL+":%v", externalPort)
		}

		if deployOutput.ReleaseID != nil {

			// CREATE NEW RELEASE BY TERMINATING DEPLOYMENTS
			currentReleae, err := service.CreateNewAppRelease(input.AppID, input.Image, *deployOutput.ReleaseID, user.ID, currentApp.Organization)
			v, _ := strconv.Atoi(currentReleae.Version)
			if err != nil {
				log4go.Error("Module: DeployImage, MethodName: CreateNewAppRelease, Message: %s user:%s", err.Error(), user.ID)
				return nil, err
			}
			log4go.Info("Module: DeployImage, MethodName: CreateNewAppRelease, Message: New app release is successfully inserted, user: %s", user.ID)

			err = service.UpdateImage(input.AppID, input.Image, internalPort)
			if err != nil {
				return nil, err
			}

			currentApp.Config.Build = &model.Builder{
				Image: &input.Image,
			}

			err = service.UpdateAppConfig(input.AppID, currentApp.Config)

			if err != nil {
				return nil, err
			}

			err = service.UpdateVersion(input.AppID, v)

			if err != nil {
				return nil, err
			}

			release := &model.Release{ID: &currentReleae.Id, Version: &v, Stable: &stable, InProgress: &inprogress, Reason: &currentApp.Hostname, Description: &description, Status: &currentReleae.Status, CreatedAt: &currentReleae.CreatedAt, User: &model.User{ID: user.ID, Email: user.Email}}
			return &model.DeployImage{
				Release: release,
			}, nil
		}

		// Create Release and Deployment Record
		releaseId := uuid.NewString()
		version := "1"
		status := "active"
		routingPolicy, err := _helper.GetRoutingPolicy(currentApp.Config.Definition)
		currentRelease := apprelease.AppRelease{
			Id:            releaseId,
			AppId:         input.AppID,
			Status:        status,
			Version:       version,
			CreatedAt:     time.Now(),
			UserId:        user.ID,
			ImageName:     input.Image,
			Port:          int(internalPort),
			ArchiveUrl:    *input.ArchiveURL,
			BuilderType:   *currentApp.Config.Build.Builder,
			RoutingPolicy: routingPolicy,
		}

		err = _helper.CreateAppRelease(currentRelease)
		if err != nil {
			return nil, err
		}
		var elb string
		if input.Image == "mysql:5.6" || input.Image == "postgres:10.1" {
			elb = deployOutput.URL
		} else {
			elb = *deployOutput.LoadBalanceURL
		}

		deploymentId := uuid.NewString()
		deployment := appDeployments.AppDeployments{
			Id:            deploymentId,
			AppId:         input.AppID,
			Region_code:   clusterDetails.Region_code,
			Status:        "running",
			Deployment_id: deployOutput.ID,
			Port:          fmt.Sprintf("%v", internalPort),
			App_Url:       elb,
			Release_id:    releaseId,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			ContainerID:   *deployOutput.ContainerID,
		}
		err = _helper.CreateDeploymentsRecord(deployment)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: CreateDeploymentsRecord, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: DeployImage, MethodName: CreateDeploymentsRecord, Message: App deployment record is successfully inserted, user: %s", user.ID)

		addRegionItem = append(addRegionItem, &clusterDetails.Region_code)
		service.AddOrRemoveRegions(input.AppID, addRegionItem, emptyRegion, emptyRegion, false, adminUserId)

		v, _ := strconv.Atoi(currentRelease.Version)
		// return Release Object
		release := &model.Release{ID: &releaseId, Version: &v, Stable: &stable, InProgress: &inprogress, Reason: &deployOutput.URL, Description: &description, Status: &status, CreatedAt: &currentRelease.CreatedAt, User: &model.User{ID: user.ID, Email: user.Email}}
		if input.Image != "mysql:5.6" && input.Image != "postgres:10.1" && clusterDetails.ClusterType != "byoh" {
			// err = service.CreateOrDeleteDNSRecord(deploymentId, deployOutput.HostName, *deployOutput.LoadBalanceURL, clusterDetails.Region_code, *clusterDetails.ProviderType, false)
			// if err != nil {
			// 	log4go.Error("Module: DeployImage, MethodName: CreateOrDeleteDNSRecord, Message: %s user:%s", err.Error(), user.ID)
			// 	return nil, err
			// }
			// log4go.Info("Module: DeployImage, MethodName: CreateOrDeleteDNSRecord, Message: successfully reached, user: %s", user.ID)

			_, err := service.CreateCLBRoute(input.AppID, *deployOutput.LoadBalanceURL, strconv.Itoa(int(externalPort)), true)
			if err != nil {
				return nil, err
			}
		}
		compName, err := users.GetCompanyNameById(user.ID)
		if err != nil {
			return nil, err
		}

		adminMailid, adminName, err := users.GetAdminByCompanyName(compName, 1)
		if err != nil {
			return nil, err
		}
		// err = service.UpdateAppbySubOrgAndBusinessUnit(input.AppID, *input.SubOrgID, *input.BusinessUnitID)
		// if err != nil {
		// 	log4go.Error("Module: DeployImage, MethodName: UpdateAppbySubOrgAndBusinessUnit, Message: %s user:%s", err.Error(), user.ID)
		// 	return nil, err
		// }
		// log4go.Info("Module: DeployImage, MethodName: UpdateAppbySubOrgAndBusinessUnit, Message: Update the Update based on SubOrganization and Business Unit, user: %s", user.ID)

		compLogo, err := service.GetLogoByUserId(user.ID)
		if compLogo == "\"\"" {
			compLogo = "https://user-profileimage.s3.ap-south-1.amazonaws.com/nifeLogo.png"
		}

		for _, reg := range addRegionItem {

			regName, err := clusterInfo.GetClusterDetails(*reg, adminUserId)
			if err != nil {
				log4go.Error("Module: DeployImage, MethodName: GetClusterDetails, Message: %s user:%s", err.Error(), user.ID)
				return nil, err
			}
			log4go.Info("Module: DeployImage, MethodName: GetClusterDetails, Message: Fetching cluster details is successfully completed, user: %s", user.ID)

			err = helper.DeployMail(adminName, adminMailid, user.FirstName+" "+user.LastName, input.AppID, *regName.RegionName, "Deploy", compLogo, "Deployed")
			if err != nil {
				endTime := time.Now()
				duration := endTime.Sub(startTime)
				err = service.UpdateDeploymentsTime(currentRelease.AppId, duration.String())
				log4go.Error("Module: DeployImage, MethodName: DeployMail, Message: %s user:%s", err.Error(), user.ID)
				return nil, err
			}
			log4go.Info("Module: DeployImage, MethodName: DeployMail, Message: Deploy mail is successfully send, user: %s", user.ID)
		}
		// }
		if len(input.EnvMapArgs) != 0 {
			err := _helper.UpdateEnvArgsInRelease(input.AppID, input.EnvMapArgs, currentRelease.Version)
			if err != nil {
				log4go.Error("Module: DeployImage, MethodName: UpdateEnvArgsInRelease, Message: %s user:%s", err.Error(), user.ID)
				return nil, fmt.Errorf(err.Error())
			}
			log4go.Info("Module: DeployImage, MethodName: UpdateEnvArgsInRelease, Message: Updated EnvArgs variable to the app release, user: %s", user.ID)
		}

		userDetAct, err := service.GetById(user.ID)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
			return &model.DeployImage{}, err
		}
		log4go.Info("Module: DeployImage, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

		AddOperation := service.Activity{
			Type:       "APP",
			UserId:     user.ID,
			Activities: "DEPLOYED",
			Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Deployed the App " + input.AppID + " to " + clusterDetails.Region_code + " Region",
			RefId:      currentApp.ID,
		}

		_, err = service.InsertActivity(AddOperation)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		err = service.SendSlackNotification(user.ID, AddOperation.Message)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
		}

		//Deploy in multiple region
		var clustRegions []*string
		for _, clust := range clusterDet {
			if clust.Region_code != clusterDetails.Region_code {
				clustRegions = append(clustRegions, &clust.Region_code)
			}
		}
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		err = service.UpdateDeploymentsTime(currentRelease.AppId, duration.String())

		if len(clusterDet) != 1 {
			r.ConfigureRegions(ctx, &model.ConfigureRegionsInput{AppID: input.AppID, AllowRegions: clustRegions})
		}

		endTime = time.Now()
		duration = endTime.Sub(startTime)

		err = service.UpdateDeploymentsTime(currentRelease.AppId, duration.String())

		return &model.DeployImage{
			Release: release,
		}, nil
	} else {

		err := service.DeployType(2, input.AppID)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: DeployType, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: DeployImage, MethodName: DeployType, Message: Get deploy type of the App is successfully completed, user: %s", user.ID)

		if currentApp.Status == "Active" {

			err = service.DuploDeployStatus(currentApp.Name)
			if err != nil {
				log4go.Error("Module: DeployImage, MethodName: DuploDeployStatus, Message: %s user:%s", err.Error(), user.ID)
				return nil, err
			}
			log4go.Info("Module: DeployImage, MethodName: DuploDeployStatus, Message: Deleting Duplo app deploy status is successfully completed, user: %s", user.ID)

			reDeploy := service.UpdateDuplo{
				AppName:             input.AppID,
				Image:               input.Image,
				Status:              currentApp.Status,
				UserId:              user.ID,
				InternalPort:        int(internalPort),
				ExternalPort:        int(externalPort),
				AgentPlatForm:       *clusterDetails.ExternalAgentPlatform,
				ExternalBaseAddress: *clusterDetails.ExternalBaseAddress,
				TenantId:            *clusterDetails.TenantId,
				SecretRegistyId:     secretRegid,
			}

			err = reDeploy.UpdateDuploApp()

			if err != nil {
				log4go.Error("Module: DeployImage, MethodName: UpdateDuploApp, Message: %s user:%s", err.Error(), user.ID)
				return nil, err
			}
			log4go.Info("Module: DeployImage, MethodName: UpdateDuploApp, Message: Duplo app is successfully updated, user: %s", user.ID)

			return &model.DeployImage{}, nil
		}

		duplo := service.DuploDetails{
			AppName:             input.AppID,
			Image:               input.Image,
			UserId:              user.ID,
			AllocationTag:       clusterDetails.AllocationTag,
			EnvArgs:             environmentArgument,
			InternalPort:        int(internalPort),
			ExternalPort:        int(externalPort),
			AgentPlatForm:       *clusterDetails.ExternalAgentPlatform,
			ExternalBaseAddress: *clusterDetails.ExternalBaseAddress,
			TenantId:            *clusterDetails.TenantId,
			Cloud:               *clusterDetails.ExternalCloudType,
			Region:              clusterDetails.Region_code,
			RegionName:          *clusterDetails.RegionName,
			SecretRegistryID:    secretRegid,
		}

		err = duplo.CreateDuploService()
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: CreateDuploService, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: DeployImage, MethodName: CreateDuploService, Message: Duplo service is successfully created, user: %s", user.ID)

		// err = service.UpdateAppbySubOrgAndBusinessUnit(input.AppID, *input.SubOrgID, *input.BusinessUnitID)
		// if err != nil {
		// 	log4go.Error("Module: DeployImage, MethodName: UpdateAppbySubOrgAndBusinessUnit, Message: %s user:%s", err.Error(), user.ID)
		// 	return nil, err
		// }
		// log4go.Info("Module: DeployImage, MethodName: UpdateAppbySubOrgAndBusinessUnit, Message: Update the Update based on SubOrganization and Business Unit, user: %s", user.ID)

		return &model.DeployImage{}, nil
	}
}

func (r *mutationResolver) OptimizeImage(ctx context.Context, input model.OptimizeImageInput) (*model.OptimizeImage, error) {
	status := "running"
	optimizeImage := &model.OptimizeImage{
		Status: &status,
	}
	return optimizeImage, nil
}

func (r *mutationResolver) DeployK8s(ctx context.Context, input model.DeployInput) (*model.DeployOutput, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) StartBuild(ctx context.Context, input model.StartBuildInput) (*model.StartBuild, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.StartBuild{}, fmt.Errorf("Access Denied")
	}
	startTime := time.Now()
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	var dockerFilePath string
	if input.DockerFilePath != nil {
		dockerFilePath = *input.DockerFilePath
	} else {
		dockerFilePath = ""
	}

	appDetails, err := service.GetApp(*input.AppID, user.ID)
	if err != nil {
		return nil, err
	}
	var secDetails model.GetUserSecret
	var patPassword string

	if appDetails.SecretRegistryID != nil {
		secDetails, err = secretregistry.GetSecretDetails(*appDetails.SecretRegistryID, "")
		if err != nil {
			return nil, err
		}
		patPassword = *secDetails.PassWord

	}

	var dockerFile string
	if input.DockerFile == nil {
		dockerFile = ""
	}
	if input.DockerFile != nil {
		dockerFile = *input.DockerFile
	}

	appInternalPort, err := builtin.GetAppDetails(*input.AppID, user.ID)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: StartBuild, MethodName: GetAppDetails, Message: %s user:%s", err.Error(), user.ID)
		return &model.StartBuild{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: StartBuild, MethodName: GetAppDetails, Message: Fetch app detail by app name is successfully completed, user: %s", user.ID)
	patPassword = decode.DePwdCode(patPassword)
	postBody, _ := json.Marshal(map[string]string{
		"appId":          *input.AppID,
		"sourceUrl":      *input.SourceURL,
		"sourceType":     *input.SourceType,
		"buildType":      *input.BuildType,
		"imageTag":       *input.ImageTag,
		"fileExtension":  input.FileExtension,
		"internalPort":   fmt.Sprintf("%v", appInternalPort),
		"dockerFile":     dockerFile,
		"dockerFilePath": dockerFilePath,
		"secretPAT":      patPassword,
	})

	responseBody := bytes.NewBuffer(postBody)

	remoteBuildURL := os.Getenv("START_BUILD_URL")

	resp, err := http.Post(remoteBuildURL+"/start_build", "application/json", responseBody)
	if err != nil {
		err = service.ErrorActivity(user.ID, *input.AppID, err.Error())
		log.Println(err)
		return &model.StartBuild{}, fmt.Errorf(err.Error())
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	var responseData service.ResponseData
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return &model.StartBuild{}, fmt.Errorf(err.Error())
	}
	imageName := responseData.ImageName
	buildLogs := responseData.BuildLogs

	if resp.StatusCode == 200 {
		if err != nil {
			log.Println(err)
			return &model.StartBuild{}, fmt.Errorf(err.Error())
		}

		defer resp.Body.Close()

		Imagetag := string(imageName)
		var buildlogsPointer []*string
		for _, tag := range buildLogs {
			tagPointer := tag
			buildlogsPointer = append(buildlogsPointer, &tagPointer)
		}

		remoteBuild := &model.StartBuild{
			Build:     &model.Build{Image: &Imagetag},
			BuildLogs: buildlogsPointer,
		}
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		err := service.UpdateBuildTime(*input.AppID, duration.String())
		if err != nil {
			return nil, err
		}

		dockerBuildFileUrl := customerdisplayimage.UploadDockerBuildLogs(*input.AppID+".log", body)

		err = service.UpdateBuiLogsURL(*input.AppID, dockerBuildFileUrl)
		if err != nil {
			return nil, err
		}

		return remoteBuild, nil
	} else {
		var jsonMap map[string]interface{}
		json.Unmarshal([]byte(string(body)), &jsonMap)
		message := jsonMap["message"]
		s := fmt.Sprintf("%v", message)
		return nil, fmt.Errorf(s)
	}
}

func (r *mutationResolver) S3Deployment(ctx context.Context, input *model.S3DeployInput) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	checkAppQuota, err := r.Query().AppQuotaExist(ctx)

	if checkAppQuota == false {
		return "", fmt.Errorf("App Quota is exceeded, Upgrade your plan to create more applications")
	}
	if input.OrganizationID == nil {
		return "", fmt.Errorf("Organization should be selected")
	}
	err = helper.AppNameCheckWithBlankSpace(*input.S3AppName)
	if err != nil {
		log4go.Error("Module: S3Deployment, MethodName: AppNameCheckWithBlankSpace, Message: %s user:%s", err.Error(), user.ID)
		return "", fmt.Errorf("App Name should not contain empty space %s", *input.S3AppName)
	}
	log4go.Info("Module: S3Deployment, MethodName: AppNameCheckWithBlankSpace, Message: Check AppName With Blank Space completed successfully, user: %s", user.ID)

	checkAppName, err := service.CheckS3AppName(*input.S3AppName)
	if err != nil {
		log4go.Error("Module: S3Deployment, MethodName: CheckS3AppName, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: S3Deployment, MethodName: CheckS3AppName, Message: Checking App Name, user: %s", user.ID)
	if checkAppName != "" {
		randomNumber := organizationInfo.RandomNumber4Digit()
		randno := strconv.Itoa(int(randomNumber))
		checkAppName = *input.S3AppName + "-" + randno
		input.S3AppName = &checkAppName
	}

	oldDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	fmt.Println(oldDir)
	fmt.Println(oldDir)
	fmt.Println(oldDir)

	accessKey := os.Getenv("AWS_ACCESS_KEY")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY_ID")
	s3Region := os.Getenv("AWS_REGION")

	var buildDuration time.Duration
	var buildCommandsS3String, envVariablesS3String []byte
	var extractedDir string

	if !*input.DeployBuildFile {
		startTime := time.Now()

		downloadPath := "extracted_app/" + *input.S3AppName + ".zip"
		err = service.DownloadFile(*input.S3Url, downloadPath)
		if err != nil {
			return "", err
		}

		archiveFile, err := os.Open(downloadPath)
		if err != nil {
			return "", err
		}
		defer archiveFile.Close()

		tempDir := "extracted_app/" + *input.S3AppName

		defer os.RemoveAll(tempDir)

		err = extract.Zip(context.Background(), archiveFile, tempDir, nil)
		if err != nil {
			return "", err
		}

		firstLevelDir1, err := service.FindPackageJSON(tempDir)
		if err != nil {
			return "", err
		}

		firstLevelDir1 = strings.ReplaceAll(firstLevelDir1, "\\", "/")
		extractedDir = firstLevelDir1
		commands := make([][]string, len(input.BuildCommandsS3))
		for i, cmds := range input.BuildCommandsS3 {
			if cmds.S3Cmd != nil {
				commands[i] = strings.Split(*cmds.S3Cmd, " ")
			}
		}
		buildCommandsS3String, err = json.Marshal(input.BuildCommandsS3)
		if err != nil {
			return "", err
		}

		envVariablesS3String, err = json.Marshal(input.EnvVariablesS3)
		if err != nil {
			return "", err
		}

		err = os.Chdir(extractedDir)
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(extractedDir)
		currentDir, err := os.Getwd()
		if err != nil {
			return "", err
		}

		fmt.Println(currentDir)
		fmt.Println(currentDir)
		fmt.Println(currentDir)

		service.SetEnvironmentVariables(input.EnvVariablesS3)

		for _, cmd := range commands {
			err := service.ExecuteCommand(cmd[0], cmd[1:]...)
			if err != nil {
				err = os.Chdir(oldDir)
				if err != nil {
					return "", err
				}
				return "", fmt.Errorf("error while running this command - ", cmd, "  Error - ", err, " Try building the file locally and upload the build")
			}
		}

		fmt.Println("Application built successfully!")
		//--------------------------------------------------------------
		endTime := time.Now()
		buildDuration = endTime.Sub(startTime)
	} else if *input.DeployBuildFile {
		startTime := time.Now()
		downloadPath := "extracted_app/" + *input.S3AppName + ".zip"
		err = service.DownloadFile(*input.S3Url, downloadPath)
		if err != nil {
			return "", err
		}

		archiveFile, err := os.Open(downloadPath)
		if err != nil {
			return "", err
		}
		defer archiveFile.Close()

		tempDir := "extracted_app/" + *input.S3AppName

		defer os.RemoveAll(tempDir)

		err = extract.Zip(context.Background(), archiveFile, tempDir, nil)
		if err != nil {
			err = os.Chdir(oldDir)
			if err != nil {
				return "", err
			}
			return "", err
		}

		indexPath := service.FindIndexHTMLFilePath(tempDir)
		if indexPath == ""{
			return "", fmt.Errorf("Index.html file not found.")
		}

		extractedDir = filepath.Join(indexPath, "")
		err = os.Chdir(extractedDir)
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(extractedDir)
		endTime := time.Now()
		buildDuration = endTime.Sub(startTime)

	}

	startTime1 := time.Now()
	webStaticURL, err := service.DeployOnS3(*input.S3AppName, *input.BuildFileName, accessKey, secretKey, s3Region, oldDir)
	if err != nil {
		err = os.Chdir(oldDir)
		if err != nil {
			return "", err
		}
		return "", err
	}

	err = os.Chdir(oldDir)
	if err != nil {
		return "", err
	}

	endTime1 := time.Now()
	deploymentDuration := endTime1.Sub(startTime1)

	appId, err := service.CreateS3Deployment(*input, user.ID, webStaticURL, string(envVariablesS3String), string(buildCommandsS3String), deploymentDuration.String(), buildDuration.String())
	if err != nil {
		return "", err
	}

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		return "", err
	}

	AddOperation := service.Activity{
		Type:       "SITE",
		UserId:     user.ID,
		Activities: "DEPLOYED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Deployed the SITE " + *input.S3AppName + " to object storage services",
		RefId:      appId,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return webStaticURL, nil
}

func (r *mutationResolver) RemoveFiles(ctx context.Context, s3appName *string) (*string, error) {
	originalDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting working directory:", err)
		return nil, err
	}

	folder := filepath.Join(originalDir, "extracted_app", *s3appName+".zip")

	err = os.Remove(folder)
	if err != nil {
		fmt.Println("Error removing file:", err)
		return nil, err
	}
	return nil, nil
}

func (r *mutationResolver) DeleteS3Deployment(ctx context.Context, s3appName *string) (*string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	accessKey := os.Getenv("AWS_ACCESS_KEY")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY_ID")
	s3Region := os.Getenv("AWS_REGION")

	checkUser, err := service.CheckS3DeploymentUser(*s3appName, user.ID)
	if err != nil {
		return nil, err
	}
	if checkUser != user.ID {
		return nil, fmt.Errorf("access denied")
	}

	err = service.DeleteS3BucketDeployment(*s3appName, accessKey, secretKey, s3Region)
	if err != nil {
		return nil, err
	}

	err = service.DeleteS3Deployment(*s3appName, user.ID)
	if err != nil {
		return nil, err
	}

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		return nil, err
	}

	DeleteOperation := service.Activity{
		Type:       "SITE",
		UserId:     user.ID,
		Activities: "DELETED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Deleted the SITE " + *s3appName,
		RefId:      *s3appName,
	}

	_, err = service.InsertActivity(DeleteOperation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	deleteMsg := "Deleted Successfully"

	return &deleteMsg, nil
}

func (r *queryResolver) GetAvailableBuiltIn(ctx context.Context, first *int) ([]string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return make([]string, 0), fmt.Errorf("Access Denied")
	}

	builtInProperties, _ := builtin.GetBuiltIn()

	var getBuiltIn []string

	for builtinName := range builtInProperties {
		getBuiltIn = append(getBuiltIn, builtinName)
	}
	return getBuiltIn, nil
}

func (r *queryResolver) GetElbURL(ctx context.Context, input *model.ElbURLInput) (*model.ElbURL, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
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

	if !planAndPermission.CustomDomain {
		return nil, fmt.Errorf("Upgrade your plan to get the Custom domain feature")
	}
	//---------
	elbUrl, err := service.GetElbUrlByAppName(*input.AppName)
	if err != nil {
		log4go.Error("Module: GetElbURL, MethodName: GetElbUrlByAppName, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: GetElbURL, MethodName: GetElbUrlByAppName, Message: User "+user.ID+" is requested for load balancer url - "+*elbUrl.ElbURL+" , user: %s", user.ID)

	return &elbUrl, nil
}

func (r *queryResolver) GetAllS3deployments(ctx context.Context) ([]*model.S3Deployments, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	result, err := service.GetAllS3Deployments(user.ID)
	if err != nil {
		return nil, err
	}

	var pointerResult []*model.S3Deployments
	for i := range result {
		pointerResult = append(pointerResult, &result[i])
	}

	return pointerResult, nil
}

func (r *queryResolver) GetS3deployments(ctx context.Context, s3appName *string) (*model.S3Deployments, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	result, err := service.GetS3DeploymentsByAppName(user.ID, *s3appName)
	if err != nil {
		return nil, err
	}
	if result.S3AppName == nil {
		return nil, err
	}

	return &result, nil
}
