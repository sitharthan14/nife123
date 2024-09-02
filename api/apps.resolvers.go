package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/alecthomas/log4go"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/generated"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	apprelease "github.com/nifetency/nife.io/internal/app_release"
	"github.com/nifetency/nife.io/internal/auth"
	clusterDetails "github.com/nifetency/nife.io/internal/cluster_info"
	clusterInfo "github.com/nifetency/nife.io/internal/cluster_info"
	organizationInfo "github.com/nifetency/nife.io/internal/organization_info"
	secretregistry "github.com/nifetency/nife.io/internal/secret_registry"
	"github.com/nifetency/nife.io/internal/stripes"
	"github.com/nifetency/nife.io/internal/users"
	_helper "github.com/nifetency/nife.io/pkg/helper"
	"github.com/nifetency/nife.io/service"
	"google.golang.org/api/option"
)

func (r *mutationResolver) CreateApp(ctx context.Context, input model.CreateAppInput) (*model.NewApp, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.NewApp{}, fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	checkAppQuota, err := r.Query().AppQuotaExist(ctx)

	if checkAppQuota == false {
		return nil, fmt.Errorf("App Quota is exceeded, Upgrade your plan to create more applications")
	}

	if input.OrganizationID == "" && input.SubOrganizationID == "" && input.BusinessUnitID == "" {
		return nil, fmt.Errorf("Select an Organization or Sub-Organization")
	}
	uuid := uuid.New()

	err = helper.AppNameCheckWithBlankSpace(input.Name)
	if err != nil {
		log4go.Error("Module: CreateApp, MethodName: AppNameCheckWithBlankSpace, Message: %s user:%s", err.Error(), user.ID)
		return &model.NewApp{}, fmt.Errorf("App Name should not contain empty space %s", input.Name)
	}
	log4go.Info("Module: CreateApp, MethodName: AppNameCheckWithBlankSpace, Message: Check AppName With Blank Space completed successfully, user: %s", user.ID)

	// Validate App Name is Unique
	name, appNameExists, err := service.GenarateAndValidateName(input.Name)
	if err != nil {
		log4go.Error("Module: CreateApp, MethodName: GenarateAndValidateName, Message: %s user:%s", err.Error(), user.ID)
		return &model.NewApp{}, err
	}
	log4go.Info("Module: CreateApp, MethodName: GenarateAndValidateName, Message: Genarated App name or Validate App name is successfully completed, user: %s", user.ID)

	if appNameExists {
		return &model.NewApp{}, fmt.Errorf("Please enter Unique name , Name %s Already Exits", input.Name)
	}

	input.Name = name
	var appService []*model.Service
	serviceString := `[{"description":"app_service","protocol":"TCP","internalPort":3000,"externalPort":3000,"ports":[{"port":1,"handlers":["port_handlers"]}],"checks":[{"type":"check1","interval":1000,"timeout":3000,"httpMethod":"POST","httpPath":"http:\/\/localhost","httpProtocol":"TCP","httpSkipTLSVerify":true,"httpHeaders":[{"name":"Auhtorization","value":"Bearer xxxxxxxx"}]}]}]`
	err = json.Unmarshal([]byte(serviceString), &appService)
	if err != nil {
		return &model.NewApp{}, err
	}

	var ipAddresses model.IPAddresses
	ipAddressesString := `{"nodes":[{"id":"df63077c-28cb-48c3-a2b0-2ab486a39f2d","address":"node1","type":"worker","createdAt":"2015-09-15T11:00:12-00:00"}]}`
	err = json.Unmarshal([]byte(ipAddressesString), &ipAddresses)
	if err != nil {
		return &model.NewApp{}, err
	}

	memoryReq := os.Getenv("DEFAULT_MEMORY_REQUEST")
	memorylimit := os.Getenv("DEFAULT_MEMORY_LIMIT")
	cpuReq := os.Getenv("DEFAULT_CPU_REQUEST")
	cpuLimit := os.Getenv("DEFAULT_CPU_LIMIT")

	var appConfig model.AppConfig
	appConfigString := `{
		"definition":{
			"kill_signal":"SIGINT",
			"kill_timeout":5,
			"env":{
				
			},
			"experimental":{
				"allowed_public_ports":[
					
				],
				"auto_rollback":true
			},
			"services":[
				{
					"http_checks":[
						
					],
					"internal_port":4000,
					"external_port":80,
					"routing_policy":"Latency",
					"protocol":"tcp",
					"script_checks":[
						
					],

					"requests":{
						"memory":"` + memoryReq + `",
						"cpu":"` + cpuReq + `"
					},
					"limits":{
						"memory":"` + memorylimit + `",
						"cpu":"` + cpuLimit + `"
					},

					"ports":[
						{
							"handlers":[
								"http"
							],
							"port":80
						},
						{
							"handlers":[
								"tls",
								"http"
							],
							"port":443
						}
					],
					"tcp_checks":[
						{
							"grace_period":"1s",
							"interval":"15s",
							"restart_limit":6,
							"timeout":"2s"
						}
					],
					"concurrency":{
						"hard_limit":25,
						"soft_limit":20,
						"type":"connections"
					}
				}
			]
		},
		"valid":true,
		"errors":[
			"no error"
		]
	}`
	err = json.Unmarshal([]byte(appConfigString), &appConfig)
	if err != nil {
		return &model.NewApp{}, err
	}

	var currentRelease model.Release
	currentReleaseString := `{"id":"c5ec3b40-b26c-4e8b-b676-631449083cda","version":2,"stable":true,"inProgress":true,"reason":"new update waiting","description":"test description","status":"New","deployentStrategy":"No","deployment":{},"user":{},"createdAt":"2015-09-15T11:00:12-00:00"}`
	err = json.Unmarshal([]byte(currentReleaseString), &currentRelease)
	if err != nil {
		return &model.NewApp{}, err
	}

	appURL := "nife.io"

	deployed := false

	organizationDetails, err := service.GetOrganization(input.OrganizationID, "")
	if err != nil {
		log4go.Error("Module: CreateApp, MethodName: GetOrganization, Message: %s user:%s", err.Error(), user.ID)
		return &model.NewApp{}, err
	}

	log4go.Info("Module: CreateApp, MethodName: GetOrganization, Message: Get Organization is completed successfully, user: %s", user.ID)

	appId := uuid.String()
	app := &model.NewApp{
		App: &model.App{
			ID:             appId,
			Name:           input.Name,
			Hostname:       "nife.io",
			Deployed:       &deployed,
			Status:         "New",
			Version:        1,
			AppURL:         &appURL,
			Organization:   organizationDetails,
			Services:       appService,
			Config:         &appConfig,
			ParseConfig:    &appConfig,
			IPAddresses:    &ipAddresses,
			Release:        &currentRelease,
			CurrentRelease: &currentRelease,
		},
	}

	err = service.CreateApp(app, user.ID, input.SubOrganizationID, input.BusinessUnitID, input.WorkloadManagementID)
	if err != nil {
		log4go.Error("Module: CreateApp, MethodName: CreateApp, Message: %s user:%s", err.Error(), user.ID)
		return &model.NewApp{}, err
	}
	log4go.Info("Module: CreateApp, MethodName: CreateApp, Message: App created successfully, user: %s", user.ID)

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: CreateApp, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return &model.NewApp{}, err
	}
	log4go.Info("Module: CreateApp, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	AddOperation := service.Activity{
		Type:       "APP",
		UserId:     user.ID,
		Activities: "CREATED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Created a App " + input.Name,
		RefId:      appId,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	err = service.SendSlackNotification(user.ID, AddOperation.Message)
	if err != nil {
		log4go.Error("Module: CreateApp, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}
	return app, nil
}

func (r *mutationResolver) DeleteApp(ctx context.Context, appID string, regionCode string) (*model.App, error) {
	emptyRegion := make([]*string, 0)
	deleteRegionItem := make([]*string, 0)
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.App{}, fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	appdet, err := service.GetApp(appID, user.ID)
	if err != nil {
		log4go.Error("Module: DeleteApp, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
		return &model.App{}, err
	}

	if appdet.Name == "" {
		return nil, fmt.Errorf("Can't able to find the given App, Pass valid app name")
	}

	log4go.Info("Module: DeleteApp, MethodName: GetApp, Message: Get app details is successfully completed, user: %s", user.ID)

	if appdet.Status == "Suspended" {
		var OrgId string
		var orgDet model.Organization
		if *appdet.SubOrganizationID != "" {
			orgdett, err := service.GetOrganization(*appdet.SubOrganizationID, "")
			if err != nil {
				return &model.App{}, err
			}
			OrgId = *appdet.SubOrganizationID
			orgDet = *orgdett
		} else {
			OrgId = *appdet.Organization.ID
			orgDet = *appdet.Organization
		}

		clusterDetails, err := clusterDetails.GetClusterDetailsByOrgId(OrgId, regionCode, "code", user.ID)
		if err != nil {
			log4go.Error("Module: DeleteApp, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
			return &model.App{}, err
		}
		log4go.Info("Module: DeleteApp, MethodName: GetClusterDetailsByOrgId, Message: Fetching cluster details by organization is completed , user: %s", user.ID)

		_, err = service.SuspendResumePods(*clusterDetails, appID, "suspended", int32(1), &orgDet)
		if err != nil {
			log4go.Error("Module: DeleteApp, MethodName: SuspendResumePods, Message: %s user:%s", err.Error(), user.ID)
			return &model.App{}, err
		}
		log4go.Info("Module: DeleteApp, MethodName: SuspendResumePods, Message: Resume the suspended app is successfully completed, user: %s", user.ID)
	}
	app, err := service.GetApp(appID, user.ID)
	if err != nil {
		return &model.App{}, err
	}
	if *app.DeployType == 1 || *app.DeployType == 0 {

		if regionCode == "" {
			service.UpdateAppStatus(app.Name)

			userDetAct, err := service.GetById(user.ID)
			if err != nil {
				log4go.Error("Module: DeleteApp, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
				return &model.App{}, err
			}
			log4go.Info("Module: DeleteApp, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)
			if app.Status != "Active" {
				err = service.DeleteAppRecord(app.Name, "app")
			}
			DeleteOperation := service.Activity{
				Type:       "APP",
				UserId:     user.ID,
				Activities: "DELETED",
				Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Deleted the App " + appID,
				RefId:      app.ID,
			}

			_, err = service.InsertActivity(DeleteOperation)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}
			err = service.SendSlackNotification(user.ID, DeleteOperation.Message)
			if err != nil {
				log4go.Error("Module: DeleteApp, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
			}
			return &model.App{}, nil
		}

		var OrgId string
		var orgDet model.Organization
		if *appdet.SubOrganizationID != "" {
			orgdett, err := service.GetOrganization(*appdet.SubOrganizationID, "")
			if err != nil {
				return &model.App{}, err
			}
			OrgId = *appdet.SubOrganizationID
			orgDet = *orgdett
		} else {
			OrgId = *appdet.Organization.ID
			orgDet = *appdet.Organization
		}

		clusterDetails, err := clusterDetails.GetClusterDetailsByOrgId(OrgId, regionCode, "code", user.ID)
		if err != nil {
			log4go.Error("Module: DeleteApp, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
			return &model.App{}, fmt.Errorf("Can't find the app in given region")
		}
		log4go.Info("Module: DeleteApp, MethodName: GetClusterDetailsByOrgId, Message: Fetching cluster details by organization is completed , user: %s", user.ID)

		if app.SecretRegistryID != nil {

			privateRegistry, err := secretregistry.GetSecretDetails(*appdet.SecretRegistryID, "")
			if err != nil {
				log4go.Error("Module: DeleteApp, MethodName: GetSecretDetails, Message: %s user:%s", err.Error(), user.ID)
				return &model.App{}, err
			}

			var fileSavePath string
			if clusterDetails.Cluster_config_path == "" {
				k8sPath := "k8s_config/" + clusterDetails.Region_code
				err := os.Mkdir(k8sPath, 0755)
				if err != nil {
					return nil, err
				}

				fileSavePath = k8sPath + "/config"

				_, err = organizationInfo.GetFileFromPrivateS3kubeconfigs(*clusterDetails.ClusterConfigURL, fileSavePath)
				if err != nil {
					return nil, err
				}
				clusterDetails.Cluster_config_path = "./k8s_config/" + clusterDetails.Region_code + "/config"
			}

			clientset, err := helper.LoadK8SConfig(clusterDetails.Cluster_config_path)
			if err != nil {
				helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)
				return &model.App{}, err
			}
			helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)

			var dbType string

			if privateRegistry.RegistryType == nil {
				dbType = ""
			} else {
				dbType = *privateRegistry.RegistryType
			}

			if dbType == "mysql" || dbType == "postgres" {

				err = service.DeletePersistentVolume(clientset, appID, *appdet.Organization.Slug, privateRegistry)
				if err != nil {

				}
				_, err = service.DeleteSecretOrganization(*privateRegistry.Name, "")
				if err != nil {
					return &model.App{}, err
				}

			}
		}

		var fileSavePath string
		if clusterDetails.Cluster_config_path == "" {
			k8sPath := "k8s_config/" + clusterDetails.Region_code
			err := os.Mkdir(k8sPath, 0755)
			if err != nil {
				return nil, err
			}

			fileSavePath = k8sPath + "/config"

			_, err = organizationInfo.GetFileFromPrivateS3kubeconfigs(*clusterDetails.ClusterConfigURL, fileSavePath)
			if err != nil {
				return nil, err
			}
			clusterDetails.Cluster_config_path = "./k8s_config/" + clusterDetails.Region_code + "/config"
		}

		clientset, err := helper.LoadK8SConfig(clusterDetails.Cluster_config_path)
		if err != nil {
			helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)
			return &model.App{}, err
		}

		// if clusterDetails.ClusterType == "byoh" {

		// 	ingressName := appID + "-ingress"
		// 	err = service.DeleteIngress(clientset, ingressName, *appdet.Organization.Slug)
		// 	if err != nil {
		// 		return &model.App{}, err
		// 	}
		// }

		_, err = service.UnDeploy(appID, *clusterDetails, app.Hostname, &orgDet, user.ID, false)
		if err != nil {
			helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)
			return &model.App{}, err
		}
		helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)

		deleteRegionItem = append(deleteRegionItem, &clusterDetails.Region_code)

		service.AddOrRemoveRegions(appID, emptyRegion, deleteRegionItem, emptyRegion, false, user.ID)

		volumes, err := service.GetVolumeByAppName(appdet.Name)
		if err != nil {
			return &model.App{}, err
		}

		if volumes != nil {

			err = service.DeletePersistentVolumes(clientset, appID, *appdet.Organization.Slug)
			if err != nil {

			}
			err = service.DeleteVolume(appID)
			if err != nil {
				return &model.App{}, err
			}
		}

		//deleting the record from the DB
		appStatus, err := service.GetApp(appID, user.ID)
		if appStatus.Status != "Active" {
			err = service.DeleteAppRecord(appdet.Name, "app")
			err = service.DeleteAppRecord(appdet.Name, "deployment")
			err = service.DeleteAppRecord(appdet.Name, "release")
		}

		userDetAct, err := service.GetById(user.ID)
		if err != nil {
			log4go.Error("Module: DeleteApp, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
			return &model.App{}, err
		}
		log4go.Info("Module: DeleteApp, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

		DeleteOperation := service.Activity{
			Type:       "APP",
			UserId:     user.ID,
			Activities: "DELETED",
			Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Deleted the App " + appID + " from " + regionCode + " Region",
			RefId:      app.ID,
		}

		_, err = service.InsertActivity(DeleteOperation)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		err = service.SendSlackNotification(user.ID, DeleteOperation.Message)
		if err != nil {
			log4go.Error("Module: DeleteApp, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
		}

		return &model.App{}, nil
	} else {
		AppDetails, err := service.GetApp(appID, user.ID)

		if err != nil {
			return nil, fmt.Errorf(err.Error())
		}

		var baseURL string
		var tenantId string

		for _, regCode := range AppDetails.Regions {
			clusterDetails, err := clusterDetails.GetClusterDetails(*regCode.Code, user.ID)
			if err != nil {
				log4go.Error("Module: DeleteApp, MethodName: GetClusterDetails, Message: %s user:%s", err.Error(), user.ID)
				return nil, fmt.Errorf(err.Error())
			}
			log4go.Info("Module: DeleteApp, MethodName: GetClusterDetails, Message: Fetching cluster details is successfully completed, user: %s", user.ID)

			if *clusterDetails.InterfaceType == "REST" {
				baseURL = *clusterDetails.ExternalBaseAddress
				tenantId = *clusterDetails.TenantID
			}
		}

		if AppDetails.Status != "Terminated" && AppDetails.Status != "" {
			err = service.DeleteDuploApp(tenantId, baseURL, appID)
			if err != nil {
				log4go.Error("Module: DeleteApp, MethodName: DeleteDuploApp, Message: %s user:%s", err.Error(), user.ID)
				return nil, fmt.Errorf(err.Error())
			}
			log4go.Info("Module: DeleteApp, MethodName: DeleteDuploApp, Message: Deleting Duplo App is successfully completed, user: %s", user.ID)

			if AppDetails.Status == "New" {
				goto StatusCheck
			}
		} else {
			return nil, fmt.Errorf("App %s already terminated", appID)
		}

		if AppDetails.Status == "Active" {
			err = service.DuploDeployStatus(appID)
			if err != nil {
				return nil, fmt.Errorf(err.Error())
			}
		}

	StatusCheck:

		err = service.DuploAppDeleteStatus(appID, AppDetails.Status)
		if err != nil {
			log4go.Error("Module: DeleteApp, MethodName: DuploAppDeleteStatus, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}

		log4go.Info("Module: DeleteApp, MethodName: DuploAppDeleteStatus, Message: Deleting Duplo App status is successfully completed, user: %s", user.ID)

		if AppDetails.SecretRegistryID != nil {

			registry, err := secretregistry.GetSecretDetails(*AppDetails.SecretRegistryID, "")

			if err != nil {
				return nil, fmt.Errorf(err.Error())
			}

			DelSecret := service.SecretElement{
				RegistryName:        *registry.Name,
				TenantId:            tenantId,
				ExternalBaseAddress: baseURL,
			}

			err = DelSecret.CheckandDeleteSecret()

			if err != nil {
				log4go.Error("Module: DeleteApp, MethodName: CheckandDeleteSecret, Message: %s user:%s", err.Error(), user.ID)
				return nil, fmt.Errorf(err.Error())
			}
			log4go.Info("Module: DeleteApp, MethodName: CheckandDeleteSecret, Message: Fetching the secrets of the user and Deleting the secret, user: %s", user.ID)
		}
	}
	return &model.App{}, nil
}

func (r *mutationResolver) MoveApp(ctx context.Context, input model.MoveAppInput) (*model.NewApp, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.NewApp{}, fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	app, err := service.GetApp(input.AppID, user.ID)
	if err != nil {
		log4go.Error("Module: MoveApp, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: MoveApp, MethodName: GetApp, Message: Fetching App detail based on App Name reached successfully, user: %s", user.ID)

	checkReg, err := clusterDetails.GetClusterDetails(input.SourceRegCode, user.ID)
	if checkReg.RegionName == nil {
		return nil, fmt.Errorf("%s Region does not exists", input.SourceRegCode)
	}

	var result model.NewApp
	app, _, err = service.AddOrRemoveRegions(input.AppID, []*string{&input.DestRegCode}, []*string{&input.SourceRegCode}, []*string{}, true, user.ID)
	if err != nil {
		log4go.Error("Module: MoveApp, MethodName: AddOrRemoveRegions, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: MoveApp, MethodName: AddOrRemoveRegions, Message:Add Or Remove Regions reached successfully, user: %s", user.ID)

	result.App = app

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: MoveApp, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return &model.NewApp{}, err
	}
	log4go.Info("Module: MoveApp, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	UpdateOperation := service.Activity{
		Type:       "APP",
		UserId:     user.ID,
		Activities: "MOVED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Moved the App " + input.AppID + " from " + input.SourceRegCode + " to " + input.DestRegCode,
		RefId:      app.ID,
	}

	_, err = service.InsertActivity(UpdateOperation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	err = service.SendSlackNotification(user.ID, UpdateOperation.Message)
	if err != nil {
		log4go.Error("Module: MoveApp, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return &result, nil
}

func (r *mutationResolver) PauseApp(ctx context.Context, input model.PauseAppInput) (*model.SuspendApp, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.SuspendApp{}, fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	app, err := service.GetApp(input.AppID, user.ID)
	if err != nil {
		log4go.Error("Module: PauseApp, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
		return &model.SuspendApp{}, err
	}
	log4go.Info("Module: PauseApp, MethodName: GetApp, Message: Fetching App detail based on App Name reached successfully, user: %s", user.ID)

	clusterDetails, err := clusterDetails.GetClusterDetailsByOrgId(*app.Organization.ID, input.RegionCode, "code", user.ID)
	if err != nil {
		log4go.Error("Module: PauseApp, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
		return &model.SuspendApp{}, err
	}
	log4go.Info("Module: PauseApp, MethodName: GetClusterDetailsByOrgId, Message: Fetching Cluster details based on Organization successfully completed, user: %s", user.ID)

	_, err = service.SuspendResumePods(*clusterDetails, input.AppID, "running", int32(0), app.Organization)
	if err != nil {
		log4go.Error("Module: PauseApp, MethodName: SuspendResumePods, Message: %s user:%s", err.Error(), user.ID)
		return &model.SuspendApp{}, err
	}
	log4go.Info("Module: PauseApp, MethodName: SuspendResumePods, Message: Suspend or Resume Apps reached successfully, user: %s", user.ID)

	app, err = service.GetApp(input.AppID, user.ID)
	if err != nil {
		return &model.SuspendApp{}, err
	}

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: PauseApp, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return &model.SuspendApp{}, err
	}
	log4go.Info("Module: PauseApp, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	UpdateOperation := service.Activity{
		Type:       "APP",
		UserId:     user.ID,
		Activities: "SUSPENDED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Suspended the App " + input.AppID,
		RefId:      app.ID,
	}

	_, err = service.InsertActivity(UpdateOperation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	err = service.SendSlackNotification(user.ID, UpdateOperation.Message)
	if err != nil {
		log4go.Error("Module: PauseApp, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return &model.SuspendApp{App: app}, nil
}

func (r *mutationResolver) ResumeApp(ctx context.Context, input model.ResumeAppInput) (*model.ResumeApp, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.ResumeApp{}, fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	app, err := service.GetApp(input.AppID, user.ID)
	if err != nil {
		return &model.ResumeApp{}, err
	}

	clusterDetails, err := clusterDetails.GetClusterDetailsByOrgId(*app.Organization.ID, input.RegionCode, "code", user.ID)
	if err != nil {
		log4go.Error("Module: ResumeApp, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
		return &model.ResumeApp{}, err
	}
	log4go.Info("Module: ResumeApp, MethodName: GetClusterDetailsByOrgId, Message: Fetching cluster details based on organization, user: %s", user.ID)

	_, err = service.SuspendResumePods(*clusterDetails, input.AppID, "suspended", int32(1), app.Organization)
	if err != nil {
		log4go.Error("Module: ResumeApp, MethodName: SuspendResumePods, Message: %s user:%s", err.Error(), user.ID)
		return &model.ResumeApp{}, err
	}
	log4go.Info("Module: ResumeApp, MethodName: SuspendResumePods, Message: Suspend or Resume Apps reached successfully, user: %s", user.ID)

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: ResumeApp, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return &model.ResumeApp{}, err
	}
	log4go.Info("Module: ResumeApp, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	UpdateOperation := service.Activity{
		Type:       "APP",
		UserId:     user.ID,
		Activities: "RESUMED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Resumed the App " + input.AppID,
		RefId:      app.ID,
	}

	_, err = service.InsertActivity(UpdateOperation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	err = service.SendSlackNotification(user.ID, UpdateOperation.Message)
	if err != nil {
		log4go.Error("Module: ResumeApp, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	app, err = service.GetApp(input.AppID, user.ID)
	if err != nil {
		return &model.ResumeApp{}, err
	}

	return &model.ResumeApp{App: app}, nil
}

func (r *mutationResolver) RestartApp(ctx context.Context, input model.RestartAppInput) (*model.RestartApp, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.RestartApp{}, fmt.Errorf("Access Denied")
	}
	var node *model.Nodes
	var app *model.RestartApp
	dataBytes, err := ioutil.ReadFile("mock_data/apps_list.json")
	if err != nil {
		return app, err
	}
	err = json.Unmarshal(dataBytes, &node)
	if err != nil {
		return app, err
	}

	for _, app := range node.Nodes {
		if strings.ToLower(app.ID) == strings.ToLower(input.AppID) {
			restartApp := &model.RestartApp{
				App: app,
			}
			return restartApp, nil
		}
	}
	return app, nil
}

func (r *mutationResolver) ConfigureRegions(ctx context.Context, input *model.ConfigureRegionsInput) (*model.App, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.App{}, fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	if len(input.AllowRegions) == 0 && len(input.DenyRegions) == 0 && len(input.BackupRegions) == 0 {
		return nil, fmt.Errorf("Regions cannot be Empty")
	}

	app, err := service.GetApp(input.AppID, user.ID)
	if err != nil {
		return nil, err
	}

	res, _, err := service.AddOrRemoveRegions(input.AppID, input.AllowRegions, input.DenyRegions, input.BackupRegions, true, user.ID)
	if err != nil {
		err = service.ErrorActivity(user.ID, input.AppID, err.Error())
		log4go.Error("Module: ConfigureRegions, MethodName: AddOrRemoveRegions, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: ConfigureRegions, MethodName: AddOrRemoveRegions, Message: successfully reached, user: %s", user.ID)

	err = service.DeployType(1, input.AppID)
	if err != nil {
		return nil, err
	}
	for _, addregion := range input.AllowRegions {

		userDetAct, err := service.GetById(user.ID)
		if err != nil {
			log4go.Error("Module: ConfigureRegions, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
			return &model.App{}, err
		}
		log4go.Info("Module: ConfigureRegions, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

		UpdateOperation := service.Activity{
			Type:       "APP",
			UserId:     user.ID,
			Activities: "SCALED",
			Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Scaled the App " + input.AppID + " to " + *addregion,
			RefId:      app.ID,
		}

		_, err = service.InsertActivity(UpdateOperation)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		err = service.SendSlackNotification(user.ID, UpdateOperation.Message)
		if err != nil {
			log4go.Error("Module: ConfigureRegions, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
		}
	}

	return res, err
}

func (r *mutationResolver) UpdateApp(ctx context.Context, input model.UpdateAppInput) (*model.App, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.App{}, fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	if input.Replicas == 0 {
		return nil, fmt.Errorf("Replicas count cannot be 0")
	}

	//---------plans and permissions-------
	var planName string
	idUser, _ := strconv.Atoi(user.ID)
	checkFreePlan, err := users.FreePlanDetails(idUser)
	if !checkFreePlan {
		planName, err = stripes.GetCustPlanName(user.CustomerStripeId)
		if err != nil {
			log4go.Error("Module: UpdateApp, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.Email)
			return nil, err
		}
		log4go.Info("Module: UpdateApp, MethodName: GetCustPlanName, Message: Get user plan with ProductId:"+user.StripeProductId+", user: %s", user.Email)
	}
	if checkFreePlan {
		planName = "Free Plan"
	}
	permissions, err := users.GetCustomerPermissionByPlan(planName)
	if err != nil {
		log4go.Error("Module: UpdateApp, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return nil, err
	}
	log4go.Info("Module: UpdateApp, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+user.StripeProductId+", user: %s", user.Email)

	if permissions.PlanName == "Enterprise" {
		if permissions.Replicas < input.Replicas {
			return nil, fmt.Errorf("Maximum limit for Enterprise Plan is %d replicas. Please try again.", permissions.Replicas)
		}
	}
	if permissions.PlanName == "Premium" {
		if permissions.Replicas < input.Replicas {
			return nil, fmt.Errorf("Maximum limit for Premium Plan is %d replicas. Please try again.", permissions.Replicas)
		}
	}
	if permissions.PlanName == "free plan" || permissions.PlanName == "Starter" {
		if permissions.Replicas < input.Replicas {
			return nil, fmt.Errorf("Maximum limit for %s is %d replicas. Please try again.", permissions.PlanName, permissions.Replicas)
		}
	}

	app, err := service.GetApp(input.AppID, user.ID)
	if err != nil {
		return app, err
	}
	internalPort, _ := strconv.Atoi(input.InternalPort)
	if err != nil {
		return nil, err
	}

	externalPort, _ := strconv.Atoi(input.ExternalPort)
	if err != nil {
		return nil, err
	}

	// BUILD TYPE
	build := *&model.Builder{}
	json.Unmarshal([]byte(input.Build), &build)
	app.ParseConfig.Build = &build
	builderValue := "GitHub"
	if *app.ParseConfig.Build.Builtin == builderValue {
		builderValue = "Github"
		app.ParseConfig.Build.Builder = &builderValue
	}
	if *build.Builder == "" {
		*build.Builder = "Deploy Image"
	}

	memoryManage := model.Requirement{}

	json.Unmarshal([]byte(input.Resource), &memoryManage)

	// DEFINITIONS
	app.ParseConfig.Definition = _helper.SetInternalPort(app.ParseConfig.Definition, internalPort)
	app.ParseConfig.Definition = _helper.SetExternalPort(app.Config.Definition, externalPort)
	app.ParseConfig.Definition = _helper.SetRoutingPolicy(app.Config.Definition, input.RoutingPolicy)
	app.ParseConfig.Definition = _helper.SetResourceRequirement(app.Config.Definition, memoryManage)
	app.ParseConfig.Definition = _helper.SetRoutingPolicy(app.Config.Definition, input.RoutingPolicy)

	memoryRequestAndLimit := _helper.GetResourceRequirement(app.Config.Definition)
	memoLimit, err := strconv.Atoi(*memoryRequestAndLimit.LimitRequirement.Memory)
	memoReq, err := strconv.Atoi(*memoryRequestAndLimit.RequestRequirement.Memory)
	cpuLimit, err := strconv.Atoi(*memoryRequestAndLimit.LimitRequirement.CPU)
	cpuReq, err := strconv.Atoi(*memoryRequestAndLimit.RequestRequirement.CPU)

	if permissions.PlanName == "Enterprise" {
		if permissions.InfrastructureConfiguration < memoLimit || permissions.InfrastructureConfiguration < memoReq || permissions.InfrastructureConfiguration < cpuLimit || permissions.InfrastructureConfiguration < cpuReq {
			return nil, fmt.Errorf("Maximum limit for Enterprise Plan is %d CPU  and Memory resource. Please try again.", permissions.InfrastructureConfiguration)
		}
	}
	if permissions.PlanName == "Premium" {
		if permissions.InfrastructureConfiguration < memoLimit || permissions.InfrastructureConfiguration < memoReq || permissions.InfrastructureConfiguration < cpuLimit || permissions.InfrastructureConfiguration < cpuReq {
			return nil, fmt.Errorf("Maximum limit for Premium Plan is %d CPU  and Memory resource. Please try again.", permissions.InfrastructureConfiguration)
		}
	}
	if permissions.PlanName == "free plan" || permissions.PlanName == "Starter" {
		if permissions.InfrastructureConfiguration < memoLimit || permissions.InfrastructureConfiguration < memoReq || permissions.InfrastructureConfiguration < cpuLimit || permissions.InfrastructureConfiguration < cpuReq {
			return nil, fmt.Errorf("Please upgrade your plan to pass resource configuration")
		}
	}

	err = service.UpdateAppConfig(input.AppID, app.ParseConfig)
	if err != nil {
		log4go.Error("Module: UpdateApp, MethodName: UpdateAppConfig, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: UpdateApp, MethodName: UpdateAppConfig, Message: Update App config  based on App is successfully completed, user: %s", user.ID)

	err = _helper.UpdatePort(input.AppID, *build.Image, input.InternalPort, input.Replicas)
	if err != nil {
		return nil, err
	}

	updatedApp, err := service.GetApp(input.AppID, user.ID)
	if err != nil {
		return app, err
	}

	return updatedApp, nil
}

func (r *mutationResolver) UpdateImage(ctx context.Context, appName *string, imageName *string) (*model.UpdateImageOutput, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.UpdateImageOutput{}, fmt.Errorf("Access Denied")
	}

	appDetails, err := service.GetApp(*appName, user.ID)

	if err != nil {
		log.Println(err)
		log4go.Error("Module: UpdateImage, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UpdateImage, MethodName: GetApp, Message: Get App detail based on App name, user: %s", user.ID)

	internalPort, err := _helper.GetInternalPort(appDetails.Config.Definition)

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf(err.Error())
	}

	id := uuid.NewString()

	getRelease, err := _helper.GetAppRelease(*appName, "active")

	prevVerison, _ := strconv.Atoi(getRelease.Version)

	_helper.UpdateAppRelease("inactive", getRelease.Id)

	newVersion := strconv.Itoa(prevVerison + 1)

	release := apprelease.AppRelease{
		Id:        id,
		AppId:     *appName,
		Version:   newVersion,
		Status:    "active",
		UserId:    user.ID,
		ImageName: *imageName,
		Port:      int(internalPort),
		CreatedAt: time.Now(),
	}

	err = _helper.CreateAppRelease(release)

	if err != nil {
		log.Println(err)
		log4go.Error("Module: UpdateImage, MethodName: CreateAppRelease, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UpdateImage, MethodName: CreateAppRelease, Message: App release is successfully inserted, user: %s", user.ID)

	err = service.UpdateImage(*appName, *imageName, internalPort)

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf(err.Error())
	}

	message := "updated Successfully"
	return &model.UpdateImageOutput{
		Message: &message,
	}, nil
}

func (r *mutationResolver) EditApp(ctx context.Context, input *model.EditAppByOrganization) (*string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	err = service.EditAppByOrg(*input)
	if err != nil {
		log4go.Error("Module: EditApp, MethodName: EditAppByOrg, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: EditApp, MethodName: EditAppByOrg, Message: Edit App is successfully updated, user: %s", user.ID)

	message := "App Moved Successfully"
	return &message, nil
}

func (r *mutationResolver) UpdateConfigApps(ctx context.Context, input *model.UpdateConfig) (*model.UpdateAppConfig, error) {
	emptyRegion := make([]*string, 0)
	RegionItem := make([]*string, 0)
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	var Name string
	var status string
	var err error
	var archive_url string

	appIdcheck, err := service.GetApp(*input.AppName, user.ID)
	if err != nil {
		log4go.Error("Module: App, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
		return &model.UpdateAppConfig{}, err
	}

	_, err = service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	if *input.Version != 0 {
		appName, _, _, _, err := service.GetAppByAppId(appIdcheck.ID)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: GetAppByAppId, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: DeployImage, MethodName: GetAppByAppId, Message: Fetching the App Name with App Id = "+*input.AppID+", user: %s", user.ID)

		envArgs, _, err := _helper.GetAppsEnvArgs(appName, strconv.Itoa(*input.Version))
		if err != nil {
			return nil, err
		}
		var envvArgs []*string
		envvArgs = append(envvArgs, &envArgs)
		err = _helper.UpdateEnvArgs(appName, envvArgs)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: UpdateEnvArgs, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
	}

	if len(input.EnvMapArgs) != 0 {

		appName, _, _, _, err := service.GetAppByAppId(*input.AppID)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: GetAppByAppId, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: DeployImage, MethodName: GetAppByAppId, Message: Fetching the App Name with App Id = "+*input.AppID+", user: %s", user.ID)

		err = _helper.UpdateEnvArgs(appName, input.EnvMapArgs)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: UpdateEnvArgs, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: DeployImage, MethodName: UpdateEnvArgs, Message: EnvArgs updated successfully, user: %s", user.ID)
	}
	if len(input.EnvMapArgs) == 0 && *input.Version == 0 {
		appName, _, _, _, err := service.GetAppByAppId(*input.AppID)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: GetAppByAppId, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: DeployImage, MethodName: GetAppByAppId, Message: Fetching the App Name with App Id = "+*input.AppID+", user: %s", user.ID)

		err = _helper.UpdateEnvArgs(appName, input.EnvMapArgs)
		if err != nil {
			log4go.Error("Module: DeployImage, MethodName: UpdateEnvArgs, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: DeployImage, MethodName: UpdateEnvArgs, Message: EnvArgs updated successfully, user: %s", user.ID)
	}

	if *input.Version == 0 {

		if *input.AppID != "" {
			Name, status, _, _, err = service.GetAppByAppId(*input.AppID)
			if err != nil {
				log4go.Error("Module: UpdateConfigApps, MethodName: GetAppByAppId, Message: %s user:%s", err.Error(), user.ID)
				return nil, fmt.Errorf(err.Error())
			}
			log4go.Info("Module: UpdateConfigApps, MethodName: GetAppByAppId, Message: Get App details based on appname is successfully completed, user: %s", user.ID)

		} else {
			Name = *input.AppName
		}

		appDetails, err := service.GetApp(Name, user.ID)
		if err != nil {
			log4go.Error("Module: UpdateConfigApps, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: UpdateConfigApps, MethodName: GetApp, Message: Update app config based on app name is successfully completed, user: %s", user.ID)

		if status == "" {
			status = appDetails.Status
		}

		if input.ArchiveURL != nil {
			archive_url = *input.ArchiveURL
		}

		internalPort, _ := strconv.Atoi(*input.InternalPort)
		if err != nil {
			return nil, err
		}

		externalPort, _ := strconv.Atoi(*input.ExternalPort)
		if err != nil {
			return nil, err
		}

		if status == "New" {
			build := *&model.Builder{
				Image:   input.Image,
				Builder: appDetails.Config.Build.Builder,
				Builtin: appDetails.Config.Build.Builtin,
			}
			appDetails.ParseConfig.Build = &build
			// DEFINITIONS
			appDetails.ParseConfig.Definition = _helper.SetInternalPort(appDetails.ParseConfig.Definition, internalPort)
			appDetails.ParseConfig.Definition = _helper.SetExternalPort(appDetails.Config.Definition, externalPort)
			err = service.UpdateAppConfig(*input.AppName, appDetails.ParseConfig)

			if err != nil {
				return nil, err
			}

			appReplicas, err := _helper.GetAppReplicas(appDetails.Name)
			if err != nil {
				return nil, err
			}

			err = _helper.UpdatePort(*input.AppName, *input.Image, *input.InternalPort, appReplicas)
			if err != nil {
				return nil, err
			}

			return nil, nil
		} else if status == "Active" {
			for _, i := range appDetails.Regions {
				RegionItem = append(RegionItem, i.Code)
			}
			if *appDetails.DeployType == 1 {
				_, _, err := service.AddOrRemoveRegions(Name, nil, RegionItem, nil, true, user.ID)
				if err != nil {
					err = service.ErrorActivity(user.ID, *input.AppName, err.Error())
					log4go.Error("Module: UpdateConfigApps, MethodName: AddOrRemoveRegions, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}

				log4go.Info("Module: UpdateConfigApps, MethodName: AddOrRemoveRegions, Message: successfully reached, user: %s", user.ID)
			}

			err = service.ConfigAppChange(appDetails.Name, *input.AppName, "Active")
			if err != nil {
				log4go.Error("Module: UpdateConfigApps, MethodName: ConfigAppChange, Message: %s user:%s", err.Error(), user.ID)
				return nil, err
			}
			log4go.Info("Module: UpdateConfigApps, MethodName: ConfigAppChange, Message: Update app config based on app name is successfully completed, user: %s", user.ID)

			internalPort, _ := strconv.Atoi(*input.InternalPort)
			if err != nil {
				return nil, err
			}

			externalPort, _ := strconv.Atoi(*input.ExternalPort)
			if err != nil {
				return nil, err
			}

			build := *&model.Builder{
				Image:   input.Image,
				Builder: appDetails.Config.Build.Builder,
				Builtin: appDetails.Config.Build.Builtin,
			}

			appDetails.ParseConfig.Build = &build

			// DEFINITIONS
			appDetails.ParseConfig.Definition = _helper.SetInternalPort(appDetails.ParseConfig.Definition, internalPort)
			appDetails.ParseConfig.Definition = _helper.SetExternalPort(appDetails.Config.Definition, externalPort)
			err = service.UpdateAppConfig(*input.AppName, appDetails.ParseConfig)
			if err != nil {
				err = service.ErrorActivity(user.ID, *input.AppName, err.Error())
				log4go.Error("Module: UpdateConfigApps, MethodName: UpdateAppConfig, Message: %s user:%s", err.Error(), user.ID)
				return nil, err
			}
			log4go.Info("Module: UpdateConfigApps, MethodName: UpdateAppConfig, Message: Update app config based on app name is successfully completed, user: %s", user.ID)

			if *appDetails.DeployType == 1 {

				_, deploymentId, err := service.AddOrRemoveRegions(*input.AppName, RegionItem, nil, emptyRegion, true, user.ID)
				if err != nil {
					err = service.ErrorActivity(user.ID, *input.AppName, err.Error())
					log4go.Error("Module: UpdateConfigApps, MethodName: AddOrRemoveRegions, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: AddOrRemoveRegions, Message:successfully reached, user: %s", user.ID)

				currentRelease, _ := _helper.GetAppRelease(*input.AppName, "active")

				_helper.UpdateAppRelease("inactive", currentRelease.Id)

				vers, _ := _helper.GetLatestVersion(*input.AppName)

				internalPort, _ := strconv.Atoi(*input.InternalPort)
				routingPolicy, err := _helper.GetRoutingPolicy(appDetails.Config.Definition)

				newRelease := apprelease.AppRelease{
					Id:            uuid.NewString(),
					AppId:         *input.AppName,
					Status:        "active",
					Version:       fmt.Sprintf("%v", (vers + 1)),
					CreatedAt:     time.Now(),
					UserId:        user.ID,
					ImageName:     *input.Image,
					Port:          internalPort,
					ArchiveUrl:    archive_url,
					BuilderType:   *build.Builder,
					RoutingPolicy: routingPolicy,
				}
				err = _helper.CreateAppRelease(newRelease)
				if err != nil {
					log4go.Error("Module: UpdateConfigApps, MethodName: CreateAppRelease, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: CreateAppRelease, Message: App Release is successfully inserted, user: %s", user.ID)
				version, _ := strconv.Atoi(newRelease.Version)

				err = _helper.UpdateDeploymentsReleaseId(deploymentId, newRelease.Id)
				if err != nil {
					return nil, err
				}

				appReplicas, err := _helper.GetAppReplicas(appDetails.Name)
				if err != nil {
					return nil, err
				}

				if len(input.EnvMapArgs) != 0 {
					err := _helper.UpdateEnvArgsInRelease(*input.AppName, input.EnvMapArgs, newRelease.Version)
					if err != nil {
						err = service.ErrorActivity(user.ID, *input.AppName, err.Error())
						log4go.Error("Module: DeployImage, MethodName: UpdateEnvArgsInRelease, Message: %s user:%s", err.Error(), user.ID)
						return nil, fmt.Errorf(err.Error())
					}
					log4go.Info("Module: DeployImage, MethodName: UpdateEnvArgsInRelease, Message: Updated EnvArgs variable to the app release, user: %s", user.ID)
				}

				err = _helper.UpdatePort(*input.AppName, *input.Image, *input.InternalPort, appReplicas)
				if err != nil {
					return nil, err
				}
				err = _helper.UpdateAppVersion(*input.AppName, version)
				if err != nil {
					return nil, err
				}
				companyName, err := users.GetCompanyNameById(user.ID)
				if err != nil {
					return nil, err
				}

				adminMailid, adminName, err := users.GetAdminByCompanyName(companyName, 1)
				if err != nil {
					return nil, err
				}

				companyLogo, err := service.GetLogoByUserId(user.ID)
				// if companyLogo != "" {
				if companyLogo == "\"\"" {
					companyLogo = "https://user-profileimage.s3.ap-south-1.amazonaws.com/nifeLogo.png"
				}

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

				for _, i := range appDetails.Regions {
					RegionItem = append(RegionItem, i.Code)
					regName, err := clusterInfo.GetClusterDetails(*i.Code, adminUserId)
					if err != nil {
						return nil, err
					}
					err = helper.DeployMail(adminName, adminMailid, user.FirstName+" "+user.LastName, *input.AppName, *regName.RegionName, "Redeploy", companyLogo, "Re-Deployed")
					if err != nil {
						err = service.ErrorActivity(user.ID, *input.AppName, err.Error())
						return nil, err
					}
				}
				// }

				userDetAct, err := service.GetById(user.ID)
				if err != nil {
					log4go.Error("Module: UpdateConfigApps, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
					return &model.UpdateAppConfig{}, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

				AddOperation := service.Activity{
					Type:       "APP",
					UserId:     user.ID,
					Activities: "REDEPLOYED",
					Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Redeployed the App " + *input.AppName,
					RefId:      *input.AppID,
				}

				_, err = service.InsertActivity(AddOperation)
				if err != nil {
					fmt.Println(err)
					return nil, err
				}
				err = service.SendSlackNotification(user.ID, AddOperation.Message)
				if err != nil {
					log4go.Error("Module: UpdateConfigApps, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
				}

				return &model.UpdateAppConfig{
					AppName:      input.AppName,
					InternalPort: input.InternalPort,
					ExternalPort: input.ExternalPort,
					Image:        input.Image,
				}, nil

			} else {

				clusterDetails, err := clusterDetails.GetClusterDetailsByOrgId(*appDetails.Organization.ID, "1", "default", user.ID)
				if err != nil {
					log4go.Error("Module: UpdateConfigApps, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: GetClusterDetailsByOrgId, Message: Get cluster details based on organization is successfully reached, user: %s", user.ID)

				err = service.DuploDeployStatus(appDetails.Name)
				if err != nil {
					return nil, err
				}

				secretId := ""

				if appDetails.SecretRegistryID != nil {
					secretId = *appDetails.SecretRegistryID

				}

				reDeploy := service.UpdateDuplo{
					AppName:             *input.AppName,
					Image:               *input.Image,
					Status:              appDetails.Status,
					UserId:              user.ID,
					InternalPort:        int(internalPort),
					ExternalPort:        int(externalPort),
					AgentPlatForm:       *clusterDetails.ExternalAgentPlatform,
					ExternalBaseAddress: *clusterDetails.ExternalBaseAddress,
					TenantId:            *clusterDetails.TenantId,
					SecretRegistyId:     secretId,
				}

				err = reDeploy.UpdateDuploApp()
				if err != nil {
					return nil, err
				}

				err = service.ConfigAppChange(appDetails.Name, *input.AppName, "Active")
				if err != nil {
					log4go.Error("Module: UpdateConfigApps, MethodName: ConfigAppChange, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: ConfigAppChange, Message: Update config app is successfully completed, user: %s", user.ID)

				return &model.UpdateAppConfig{
					AppName:      input.AppName,
					InternalPort: input.InternalPort,
					ExternalPort: input.ExternalPort,
					Image:        input.Image,
				}, nil
			}

		} else {
			return nil, fmt.Errorf("This app is already terminated")
		}
	} else {

		//  Revert App

		if *input.AppID != "" {
			Name, status, _, _, err = service.GetAppByAppId(*input.AppID)
			if err != nil {
				log4go.Error("Module: UpdateConfigApps, MethodName: GetAppByAppId, Message: %s user:%s", err.Error(), user.ID)
				return nil, fmt.Errorf(err.Error())
			}
			log4go.Info("Module: UpdateConfigApps, MethodName: GetAppByAppId, Message: Get app details based on app is successfully completed, user: %s", user.ID)
		} else {
			Name = *input.AppName
		}

		appDetails, err := service.GetApp(Name, user.ID)
		if err != nil {
			log4go.Error("Module: UpdateConfigApps, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: UpdateConfigApps, MethodName: GetApp, Message: Fetching app details successfully completed, user: %s", user.ID)

		if status == "" {
			status = appDetails.Status
		}

		if input.ArchiveURL != nil {
			archive_url = *input.ArchiveURL
		}

		internalPort, _ := strconv.Atoi(*input.InternalPort)
		if err != nil {
			return nil, err
		}

		externalPort, _ := strconv.Atoi(*input.ExternalPort)
		if err != nil {
			return nil, err
		}

		if status == "New" {
			build := *&model.Builder{
				Image:   input.Image,
				Builder: appDetails.Config.Build.Builder,
				Builtin: appDetails.Config.Build.Builtin,
			}
			appDetails.ParseConfig.Build = &build
			// DEFINITIONS
			appDetails.ParseConfig.Definition = _helper.SetInternalPort(appDetails.ParseConfig.Definition, internalPort)
			appDetails.ParseConfig.Definition = _helper.SetExternalPort(appDetails.Config.Definition, externalPort)
			err = service.UpdateAppConfig(*input.AppName, appDetails.ParseConfig)
			if err != nil {
				log4go.Error("Module: UpdateConfigApps, MethodName: UpdateAppConfig, Message: %s user:%s", err.Error(), user.ID)
				return nil, err
			}
			log4go.Info("Module: UpdateConfigApps, MethodName: UpdateAppConfig, Message:  Update app config based on app is successfully completed , user: %s", user.ID)

			appReplicas, err := _helper.GetAppReplicas(appDetails.Name)
			if err != nil {
				return nil, err
			}

			err = _helper.UpdatePort(*input.AppName, *input.Image, *input.InternalPort, appReplicas)
			if err != nil {
				return nil, err
			}

			return nil, nil
		} else if status == "Active" {
			for _, i := range appDetails.Regions {
				RegionItem = append(RegionItem, i.Code)
			}
			if *appDetails.DeployType == 1 {
				_, _, err := service.AddOrRemoveRegions(Name, nil, RegionItem, nil, true, user.ID)
				if err != nil {
					err = service.ErrorActivity(user.ID, *input.AppName, err.Error())
					log4go.Error("Module: UpdateConfigApps, MethodName: AddOrRemoveRegions, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: AddOrRemoveRegions, Message:successfully reached, user: %s", user.ID)
			}

			err = service.ConfigAppChange(appDetails.Name, *input.AppName, "Active")
			if err != nil {
				return nil, err
			}

			_, buildType, err := _helper.GetAppsEnvArgs(appDetails.Name, strconv.Itoa(*input.Version))
			if err != nil {
				return nil, err
			}

			internalPort, _ := strconv.Atoi(*input.InternalPort)
			if err != nil {
				return nil, err
			}

			externalPort, _ := strconv.Atoi(*input.ExternalPort)
			if err != nil {
				return nil, err
			}

			build := *&model.Builder{
				Image:   input.Image,
				Builder: &buildType,
				Builtin: appDetails.Config.Build.Builtin,
			}
			appDetails.ParseConfig.Build = &build

			// DEFINITIONS
			appDetails.ParseConfig.Definition = _helper.SetInternalPort(appDetails.ParseConfig.Definition, internalPort)
			appDetails.ParseConfig.Definition = _helper.SetExternalPort(appDetails.Config.Definition, externalPort)

			routingPolicy, err := _helper.GetRoutingPolicyInAppRelease(*input.AppName, *input.Version)
			if err != nil {
				return nil, err
			}
			appDetails.ParseConfig.Definition = _helper.SetRoutingPolicy(appDetails.Config.Definition, routingPolicy)

			err = service.UpdateAppConfig(*input.AppName, appDetails.ParseConfig)
			if err != nil {
				log4go.Error("Module: UpdateConfigApps, MethodName: UpdateAppConfig, Message: %s user:%s", err.Error(), user.ID)
				return nil, err
			}
			log4go.Info("Module: UpdateConfigApps, MethodName: UpdateAppConfig, Message: Update app config based on app is successfully completed, user: %s", user.ID)

			if *appDetails.DeployType == 1 {

				_, _, err = service.AddOrRemoveRegions(*input.AppName, RegionItem, nil, emptyRegion, true, user.ID)
				if err != nil {
					err = service.ErrorActivity(user.ID, *input.AppName, err.Error())
					log4go.Error("Module: UpdateConfigApps, MethodName: AddOrRemoveRegions, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: AddOrRemoveRegions, Message: successfully reached, user: %s", user.ID)

				currentRelease, _ := _helper.GetAppRelease(*input.AppName, "active")

				_helper.UpdateAppRelease("inactive", currentRelease.Id)

				version := strconv.Itoa(*input.Version)

				internalPort, _ := strconv.Atoi(*input.InternalPort)

				newRelease := apprelease.AppRelease{
					AppId:      *input.AppName,
					Status:     "active",
					Version:    version,
					CreatedAt:  time.Now(),
					UserId:     user.ID,
					ImageName:  *input.Image,
					Port:       internalPort,
					ArchiveUrl: archive_url,
				}
				err = _helper.UpdateAppReleases(newRelease)
				if err != nil {
					log4go.Error("Module: UpdateConfigApps, MethodName: UpdateAppReleases, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: UpdateAppReleases, Message: App release is successfully updated , user: %s", user.ID)

				appReplicas, err := _helper.GetAppReplicas(appDetails.Name)
				if err != nil {
					return nil, err
				}

				err = _helper.UpdatePort(*input.AppName, *input.Image, *input.InternalPort, appReplicas)
				if err != nil {
					return nil, err
				}
				err = _helper.UpdateAppVersion(*input.AppName, *input.Version)
				if err != nil {
					return nil, err
				}

				userDetAct, err := service.GetById(user.ID)
				if err != nil {
					log4go.Error("Module: UpdateConfigApps, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
					return &model.UpdateAppConfig{}, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

				AddOperation := service.Activity{
					Type:       "APP",
					UserId:     user.ID,
					Activities: "REDEPLOYED",
					Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Redeployed the App " + *input.AppName,
					RefId:      newRelease.Id,
				}

				_, err = service.InsertActivity(AddOperation)
				if err != nil {
					fmt.Println(err)
					return nil, err
				}
				err = service.SendSlackNotification(user.ID, AddOperation.Message)
				if err != nil {
					log4go.Error("Module: UpdateConfigApps, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
				}

				return &model.UpdateAppConfig{
					AppName:      input.AppName,
					InternalPort: input.InternalPort,
					ExternalPort: input.ExternalPort,
					Image:        input.Image,
				}, nil

			} else {

				clusterDetails, err := clusterDetails.GetClusterDetailsByOrgId(*appDetails.Organization.ID, "1", "default", user.ID)
				if err != nil {
					log4go.Error("Module: UpdateConfigApps, MethodName: GetClusterDetailsByOrgId, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: GetClusterDetailsByOrgId, Message: Get cluster details based on organization is successfully completed, user: %s", user.ID)

				err = service.DuploDeployStatus(appDetails.Name)
				if err != nil {
					log4go.Error("Module: UpdateConfigApps, MethodName: DuploDeployStatus, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: DuploDeployStatus, Message: Deleting duplo deploy status is successfully completed, user: %s", user.ID)

				secretId := ""

				if appDetails.SecretRegistryID != nil {
					secretId = *appDetails.SecretRegistryID

				}

				reDeploy := service.UpdateDuplo{
					AppName:             *input.AppName,
					Image:               *input.Image,
					Status:              appDetails.Status,
					UserId:              user.ID,
					InternalPort:        int(internalPort),
					ExternalPort:        int(externalPort),
					AgentPlatForm:       *clusterDetails.ExternalAgentPlatform,
					ExternalBaseAddress: *clusterDetails.ExternalBaseAddress,
					TenantId:            *clusterDetails.TenantId,
					SecretRegistyId:     secretId,
				}

				err = reDeploy.UpdateDuploApp()
				if err != nil {
					log4go.Error("Module: UpdateConfigApps, MethodName: UpdateDuploApp, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: UpdateConfigApps, MethodName: UpdateDuploApp, Message: Duplo App is updated successfully , user: %s", user.ID)

				err = service.ConfigAppChange(appDetails.Name, *input.AppName, "Active")
				if err != nil {
					return nil, err
				}

				return &model.UpdateAppConfig{
					AppName:      input.AppName,
					InternalPort: input.InternalPort,
					ExternalPort: input.ExternalPort,
					Image:        input.Image,
				}, nil
			}

		} else {
			return nil, fmt.Errorf("This app is already terminated")
		}

	}
}

func (r *mutationResolver) AppTemplate(ctx context.Context, input model.ConfigTemplate) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	configDet, err := service.GetApp(*input.AppName, user.ID)
	if err != nil {
		log4go.Error("Module: AppTemplate, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: AppTemplate, MethodName: GetApp, Message: Fetching App details is successfully completed, user: %s", user.ID)
	var volSize int
	volumeDet, err := service.GetVolumeDetailsByAppName(*input.AppName)
	if err != nil {
		return "", err
	}

	for _, vol := range volumeDet {
		volSize, _ = strconv.Atoi(*vol.Size)
	}

	config, _ := json.Marshal(configDet.Config)

	err = service.CreateAppTemplate(input, string(config), user.ID, *configDet.EnvArgs, volSize)
	if err != nil {
		log4go.Error("Module: AppTemplate, MethodName: CreateAppTemplate, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: AppTemplate, MethodName: CreateAppTemplate, Message: App Template is successfully created, user: %s", user.ID)

	return "Created Successfully", nil
}

func (r *mutationResolver) UpdateAppTemplate(ctx context.Context, input model.ConfigTemplate) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	templateByName, err := service.GetConfigTemplatesByName(*input.ID)
	if err != nil {
		log4go.Error("Module: UpdateAppTemplate, MethodName: GetConfigTemplatesByName, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: UpdateAppTemplate, MethodName: GetConfigTemplatesByName, Message: Fetching config template by name is successfully completed, user: %s", user.ID)

	internalPort, _ := strconv.Atoi(*input.InternalPort)
	if err != nil {
		return "", err
	}
	externalPort, _ := strconv.Atoi(*input.ExternalPort)
	if err != nil {
		return "", err
	}
	templateByName.Config.Definition = _helper.SetInternalPort(templateByName.Config.Definition, internalPort)
	templateByName.Config.Definition = _helper.SetExternalPort(templateByName.Config.Definition, externalPort)
	templateByName.Config.Build.Image = input.Image

	templateByName.Config.Definition = _helper.SetRoutingPolicy(templateByName.Config.Definition, *input.RoutingPolicy)
	emptyStr := ""
	if input.EnvArgs == nil {
		input.EnvArgs = &emptyStr
	}

	err = service.UpdateConfigTemp(*input.ID, *input.Name, *input.EnvArgs, *input.CPULimit, *input.MemoryLimit, *input.CPURequests, *input.MemoryRequests, templateByName.Config, *input.VolumeSize)
	if err != nil {
		log4go.Error("Module: UpdateAppTemplate, MethodName: UpdateConfigTemp, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: UpdateAppTemplate, MethodName: UpdateConfigTemp, Message: Config template is successfully updated, user: %s", user.ID)

	return "Updated Successfully", nil
}

func (r *mutationResolver) DeleteAppTemplate(ctx context.Context, id *string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	err := service.DeleteConfigTemp(*id)
	if err != nil {
		log4go.Error("Module: DeleteAppTemplate, MethodName: DeleteConfigTemp, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: DeleteAppTemplate, MethodName: DeleteConfigTemp, Message: Config template is successfully deleted , user: %s", user.ID)

	return "Deleted Successfully", nil
}

func (r *mutationResolver) CheckGithubRepoPrivateOrPublic(ctx context.Context, githubURL *string) (*bool, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	resp, err := http.Get(*githubURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to get repository information: Please check the URL Or The given repository is not public")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to get repository information: Please check the URL Or The given repository is not public")
	}
	publicRepo := true
	return &publicRepo, nil
}

func (r *mutationResolver) CreateNifeTomlFile(ctx context.Context, input *model.CreateAppToml) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	var storageLink string
	service.CreateNifeToml(input)

	fileContent, err := ioutil.ReadFile(*input.AppName + ".toml")
	if err != nil {
		fmt.Println("Error reading nife.toml file:", err)
		return "", err
	}

	storageSelector := os.Getenv("STORAGE_SELECTOR")

	if storageSelector == "GCP" {

		cloudStorageBucketName := os.Getenv("GCP_BUCKET_NAME_INSTANCE")
		serviceAccountKey := os.Getenv("GCP_SERVICE_ACCOUNT_KEY")

		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAccountKey))
		if err != nil {
			return "", err
		}

		bucket := client.Bucket(cloudStorageBucketName)
		object := bucket.Object(*input.AppName + ".toml")

		file := ioutil.NopCloser(bytes.NewReader(fileContent))

		wc := object.NewWriter(context.Background())
		if _, err := io.Copy(wc, file); err != nil {
			return "", err
		}
		if err := wc.Close(); err != nil {
			return "", err
		}

		attrs, err := object.Attrs(context.Background())
		if err != nil {
			return "", err
		}

		storageLink = fmt.Sprintf("https://storage.cloud.google.com/%s/%s", attrs.Bucket, attrs.Name)
		fmt.Println(storageLink)
	} else if storageSelector == "AWS" {
		readFile := bytes.NewReader(fileContent)

		accessKey := os.Getenv("AWS_ACCESS_KEY")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY_ID")
		s3Region := os.Getenv("AWS_REGION")
		s3BucketName := os.Getenv("S3_BUCKET_NAME_INSTANCE")

		credentials, err := session.NewSession(&aws.Config{
			Region: aws.String(s3Region),
			Credentials: credentials.NewStaticCredentials(
				accessKey,
				secretKey,
				""),
		})

		if err != nil {
			return "", err
		}

		uploader := s3manager.NewUploader(credentials)

		s3link, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(s3BucketName),
			Key:    aws.String(*input.AppName + ".toml"),
			Body:   readFile,
			// ACL:    aws.String("public-read"),
			ACL: aws.String("public-read"),
		})
		if err != nil {
			return "", err
		}
		storageLink = s3link.Location
	}

	if err := os.Remove(*input.AppName + ".toml"); err != nil {
		fmt.Println("Error deleting nife.toml file:", err)
	}
	return storageLink, nil
}

func (r *queryResolver) App(ctx context.Context, name *string) (*model.App, error) {
	user := auth.ForContext(ctx)

	// for _, f := range graphql.CollectFieldsCtx(ctx, nil) {
	// 	fmt.Println(f.Name)
	// 	for _, a := range f.Arguments {
	// 		fmt.Println(a.Name, a.Value)
	// 	}
	// }
	if user == nil {
		return &model.App{}, fmt.Errorf("Access Denied")
	}
	var app *model.App
	var checkUser bool

	appOwner, err := service.GetUserIdByAppName(*name)
	if err != nil {
		return nil, err
	}
	if user.RoleId != 1 {
		userDet, err := service.GetById(user.ID)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: App, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: App, MethodName: GetById, Message:successfully reached, user: %s", user.ID)
		adminUsers, err := service.GetInviteUserByCompanyId(*userDet.CompanyID)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: App, MethodName: GetInviteUserByCompanyId, Message: %s user:%s", err.Error(), user.ID)
			return nil, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: App, MethodName: GetInviteUserByCompanyId, Message: Fetching invite user by company name is successfully completed, user: %s", user.ID)
		for _, admUser := range adminUsers {
			if appOwner == *admUser.ID {
				checkUser = true
				appOwner = user.ID
				break
			}
		}
		if !checkUser {
			return nil, fmt.Errorf("Incorrect Application name")
		}
	}

	if user.RoleId == 1 {
		userDet1, err := service.GetById(appOwner)
		if err != nil {
			return nil, fmt.Errorf(err.Error())
		}
		userDet2, err := service.GetById(user.ID)
		if err != nil {
			return nil, fmt.Errorf(err.Error())
		}
		if *userDet1.CompanyID == *userDet2.CompanyID {
			appOwner = user.ID
		}

	}

	if appOwner != user.ID {
		return nil, fmt.Errorf("Incorrect Application name")
	}

	app, err = service.GetApp(*name, user.ID)
	if err != nil {
		log4go.Error("Module: App, MethodName: GetApp, Message: %s user:%s", err.Error(), user.ID)
		return app, err
	}
	log4go.Info("Module: App, MethodName: GetApp, Message: Fetching app by appname is successfully completed, user: %s", user.ID)

	return app, nil
}

func (r *queryResolver) Apps(ctx context.Context, typeArg *string, first *int, region *string, orgSlug *string) (*model.Nodes, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Nodes{}, fmt.Errorf("Access Denied")
	}
	apps, _ := service.AllApps(user.ID, *region, *orgSlug)
	log4go.Info("Module: Apps, MethodName: AllApps, Message: Fetching all Apps based on user is successfully completed, user: %s", user.ID)

	if user.RoleId == 1 {
		userss, err := service.GetById(user.ID)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: Apps, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
			return &model.Nodes{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: Apps, MethodName: GetById, Message:successfully reached, user: %s", user.ID)

		adminUsers, err := service.GetInviteUserByCompanyId(*userss.CompanyID)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: Apps, MethodName: GetInviteUserByCompanyId, Message: %s user:%s", err.Error(), user.ID)
			return &model.Nodes{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: Apps, MethodName: GetInviteUserByCompanyId, Message: Fetching invite user by company name is successfully completed, user: %s", user.ID)

		var apps1 *model.Nodes

		for _, inviteUser := range adminUsers {
			if user.ID != *inviteUser.ID {
				apps1, _ = service.AllApps(*inviteUser.ID, *region, *orgSlug)
				apps.Nodes = append(apps.Nodes, apps1.Nodes...)
			}
		}
	} else {

		orgs, err := service.AllOrganizations(user.ID)
		if err != nil {
			log4go.Error("Module: Apps, MethodName: AllOrganizations, Message: %s user:%s", err.Error(), user.ID)
			return &model.Nodes{}, err
		}
		log4go.Info("Module: Apps, MethodName: AllOrganizations, Message: Fetching all organization is successfully completed, user: %s", user.ID)
		userDet, err := service.GetById(user.ID)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: Apps, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
			return &model.Nodes{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: Apps, MethodName: GetById, Message:successfully reached, user: %s", user.ID)

		adminUsers, err := service.GetInviteUserByCompanyId(*userDet.CompanyID)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: Apps, MethodName: GetInviteUserByCompanyId, Message: %s user:%s", err.Error(), user.ID)
			return &model.Nodes{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: Apps, MethodName: GetInviteUserByCompanyId, Message: Fetching invite user by company name is successfully completed, user: %s", user.ID)

		for _, orgSlug := range orgs.Nodes {
			for _, adminUser := range adminUsers {
				if user.ID != *adminUser.ID {

					appsOrg, err := service.AllApps(*adminUser.ID, "", *orgSlug.Slug)
					if err != nil {
						return nil, err
					}

					apps.Nodes = append(apps.Nodes, appsOrg.Nodes...)
				}
			}
		}

	}
	return apps, nil
}

func (r *queryResolver) AppsSubOrg(ctx context.Context, typeArg *string, first *int, region *string, subOrgSlug *string) (*model.Nodes, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Nodes{}, fmt.Errorf("Access Denied")
	}
	apps, _ := service.AllSubApps(user.ID, *region, *subOrgSlug)
	log4go.Info("Module: AppsSubOrg, MethodName: AllSubApps, Message: Fetching all Apps under suborganization based on user is successfully completed, user: %s", user.ID)

	return apps, nil
}

func (r *queryResolver) AppsBusinessUnit(ctx context.Context, typeArg *string, first *int, region *string, businessUnit *string) (*model.Nodes, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Nodes{}, fmt.Errorf("Access Denied")
	}
	apps, _ := service.AllBusinessUnitApps(user.ID, *region, *businessUnit)
	log4go.Info("Module: AppsBusinessUnit, MethodName: AllBusinessUnitApps, Message: Fetching all Apps under Business Unit based on user is successfully completed, user: %s", user.ID)

	return apps, nil
}

func (r *queryResolver) AppsWorkload(ctx context.Context, name *string, organiztionID *string) (*model.Nodes, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Nodes{}, fmt.Errorf("Access Denied")
	}
	appsWl, _ := service.AllAppsUnderWorkload(user.ID, *name, *organiztionID)
	log4go.Info("Module: Apps, MethodName: AllApps, Message: Fetching all Apps based on user is successfully completed, user: %s", user.ID)

	return appsWl, nil
}

func (r *queryResolver) AppsWorkloadIDOrUserRole(ctx context.Context, workloadID *string, userID *string) (*model.Nodes, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Nodes{}, fmt.Errorf("Access Denied")
	}
	var appsDet *model.Nodes
	var err error
	if *workloadID != "" {
		appsDet, err = service.AllAppsWithWorkload(user.ID, *workloadID)
		if err != nil {
			return nil, err
		}
	} else {
		appsDet, err = service.AllAppsWithWorkload(*userID, "")
		if err != nil {
			return nil, err
		}
	}

	return appsDet, nil
}

func (r *queryResolver) Appcompact(ctx context.Context, name *string) (*model.AppCompact, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.AppCompact{}, fmt.Errorf("Access Denied")
	}
	app, err := service.ReturnFirstAppCompact()
	if err != nil {
		log4go.Error("Module: Appcompact, MethodName: ReturnFirstAppCompact, Message: %s user:%s", err.Error(), user.ID)
		return &model.AppCompact{}, err
	}
	log4go.Info("Module: Appcompact, MethodName: ReturnFirstAppCompact, Message:successfully reached, user: %s", user.ID)

	return app, nil
}

func (r *queryResolver) GetAppRegion(ctx context.Context, name string, status string) (*model.AppDeploymentRegion, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.AppDeploymentRegion{}, fmt.Errorf("Access Denied")
	}
	regions, err := service.GetRunningRegionApp(name, status, user.ID)
	if err != nil {
		log4go.Error("Module: GetAppRegion, MethodName: GetRunningRegionApp, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: GetAppRegion, MethodName: GetRunningRegionApp, Message: Get the active region of the app is successfully completed , user: %s", user.ID)

	return regions, nil
}

func (r *queryResolver) GetAvailabilityCluster(ctx context.Context, isLatency *string, first *int) (*model.ClusterNodes, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.ClusterNodes{}, fmt.Errorf("Access Denied")
	}

	if user.RoleId != 1 {
		adminEmail, err := users.GetAdminByCompanyNameAndEmail(user.CompanyName)
		if err != nil {
			return nil, err
		}

		userId, err := users.GetUserIdByEmail(adminEmail)
		if err != nil {
			return nil, err
		}
		user.ID = strconv.Itoa(userId)

	}

	cluster, err := service.GetAvailabilityCluster(*isLatency, user.ID)
	if err != nil {
		log4go.Error("Module: GetAvailabilityCluster, MethodName: GetAvailabilityCluster, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: GetAvailabilityCluster, MethodName: GetAvailabilityCluster, Message: Fetching the available cluster based on latency is successfully completed, user: %s", user.ID)

	return cluster, nil
}

func (r *queryResolver) GetRegionStatus(ctx context.Context, appID string) (*model.RegionStatusNodes, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.RegionStatusNodes{}, fmt.Errorf("Access Denied")
	}

	regionStatus, err := service.GetRegionStatus(appID)
	if err != nil {
		log4go.Error("Module: GetRegionStatus, MethodName: GetRegionStatus, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: GetRegionStatus, MethodName: GetRegionStatus, Message: Fetching the region status of the app is successfully completed, user: %s", user.ID)

	return &regionStatus, err
}

func (r *queryResolver) Platform(ctx context.Context) (*model.Regions, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Regions{}, fmt.Errorf("Access Denied")
	}
	getRegionDet, err := service.GetAvailableRegion()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &getRegionDet, nil
}

func (r *queryResolver) AppStatusList(ctx context.Context, status *string) (*model.Nodes, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Nodes{}, fmt.Errorf("Access Denied")
	}

	getAppById, err := service.GetAppStatus(user.ID, *status)

	if err != nil {
		log4go.Error("Module: AppStatusList, MethodName: GetAppStatus, Message: %s user:%s", err.Error(), user.ID)
		return &model.Nodes{}, err
	}
	log4go.Info("Module: AppStatusList, MethodName: GetAppStatus, Message: successfully reached, user: %s", user.ID)

	return getAppById, nil
}

func (r *queryResolver) AppQuotaExist(ctx context.Context) (bool, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return false, fmt.Errorf("Access Denied")
	}

	userId, _ := strconv.Atoi(user.ID)

	res, err := users.FreePlanDetails(userId)

	if err != nil {
		log.Println(err)
		log4go.Error("Module: AppQuotaExist, MethodName: FreePlanDetails, Message: %s user:%s", err.Error(), user.ID)
		return false, err
	}
	log4go.Info("Module: AppQuotaExist, MethodName: FreePlanDetails, Message: Get free plan user or not is successfully completed, user: %s", user.ID)

	if res {
		appCount, err := service.UserAppCount(user.ID)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: AppQuotaExist, MethodName: UserAppCount, Message: %s user:%s", err.Error(), user.ID)
			return false, err
		}
		log4go.Info("Module: AppQuotaExist, MethodName: UserAppCount, Message: App count based on user is successfully completed, user: %s", user.ID)
		permissions1, err := users.GetCustomerPermissionByPlan("free plan")
		if err != nil {
			log4go.Error("Module: AppQuotaExist, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
			return false, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: AppQuotaExist, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:, user: %s", user.Email)
		if appCount >= permissions1.Apps {
			return false, nil
		}
		return true, nil
	}

	planName, err := stripes.GetCustPlanName(user.CustomerStripeId)
	if err != nil {
		log4go.Error("Module: AppQuotaExist, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.ID)
		return false, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: AppQuotaExist, MethodName: GetCustPlanName, Message: Checking for permission to check apps count, user: %s", user.ID)

	permissions, err := users.GetCustomerPermissionByPlan(planName)
	if err != nil {
		log4go.Error("Module: AppQuotaExist, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return false, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: AppQuotaExist, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:, user: %s", user.Email)

	if user.StripeProductId == "" {
		return false, fmt.Errorf("Something Wrong in Stripes Product Id")
	}

	metaData, err := stripes.GetCustPlan(user.StripeProductId)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: AppQuotaExist, MethodName: GetCustPlan, Message: %s user:%s", err.Error(), user.ID)
		return false, err
	}
	log4go.Info("Module: AppQuotaExist, MethodName: GetCustPlan, Message:successfully reached, user: %s", user.ID)
	fmt.Println(metaData)

	appCount, err := service.UserAppCount(user.ID)
	if err != nil {
		log.Println(err)
		return false, err
	}

	if appCount >= permissions.Apps {
		return false, err
	}

	return true, nil
}

func (r *queryResolver) CheckAppByID(ctx context.Context, name string) (bool, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return false, fmt.Errorf("Access Denied")
	}
	checkApp, err := service.AppExistUser(name, user.ID)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: CheckAppByID, MethodName: AppExistUser, Message: %s user:%s", err.Error(), user.ID)
		return false, err
	}
	log4go.Info("Module: CheckAppByID, MethodName: AppExistUser, Message: Checking app exist for the user is successfully completed, user: %s", user.ID)

	return checkApp, nil
}

func (r *queryResolver) GetAppByAppID(ctx context.Context, id *string) (*model.App, error) {
	appName, _, _, _, err := service.GetAppByAppId(*id)

	if err != nil {
		log.Println(err)
		log4go.Error("Module: GetAppByAppID, MethodName: GetAppByAppId, Message: %s user:%s", err.Error(), id)
		return &model.App{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetAppByAppID, MethodName: GetAppByAppId, Message: Fetching app details by app name is successfully completed, user: %s", id)

	result := model.App{
		Name: appName,
	}
	return &result, nil
}

func (r *queryResolver) AppsCount(ctx context.Context) (*model.AppCount, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.AppCount{}, fmt.Errorf("Access Denied")
	}

	var terminatedApps, totalApps, newApps, inActiveApps, activeApps, allInActiveApps, allTerminatedApps, allNewApps, allActiveApps int
	var countDetails []*model.RegionAppCount

	userss, err := service.GetById(user.ID)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: Apps, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return &model.AppCount{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: Apps, MethodName: GetById, Message:successfully reached, user: %s", user.ID)

	adminUsers, err := service.GetInviteUserByCompanyId(*userss.CompanyID)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: Apps, MethodName: GetInviteUserByCompanyId, Message: %s user:%s", err.Error(), user.ID)
		return &model.AppCount{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: Apps, MethodName: GetInviteUserByCompanyId, Message: Fetching invite user by company name is successfully completed, user: %s", user.ID)

	for _, inviteUser := range adminUsers {
		activeApps, err = service.GetActiveAppsById(*inviteUser.ID)
		if err != nil {
			log4go.Error("Module: AppsCount, MethodName: GetActiveAppsById, Message: %s user:%s", err.Error(), user.ID)
			return &model.AppCount{}, err
		}
		log4go.Info("Module: AppsCount, MethodName: GetActiveAppsById, Message: Fetching active app based on user is successfully completed, user: %s", user.ID)
		allActiveApps += activeApps
		newApps, err = service.GetNewAppsById(*inviteUser.ID)
		if err != nil {
			return &model.AppCount{}, err
		}
		allNewApps += newApps
		inActiveApps, err = service.GetInActiveAppsById(*inviteUser.ID)
		if err != nil {
			return &model.AppCount{}, err
		}
		allInActiveApps += inActiveApps
		terminatedApps, err = service.GetTerminatedAppsById(*inviteUser.ID)
		if err != nil {
			return &model.AppCount{}, err
		}
		allTerminatedApps += terminatedApps

		totalApps += newApps + activeApps + inActiveApps
		var countDetail []*model.RegionAppCount

		countDetail, err = service.GetAppsByRegion(*inviteUser.ID)
		if err != nil {
			return &model.AppCount{}, err
		}
		countDetails = append(countDetails, countDetail...)

	}

	return &model.AppCount{
		TotalApps:  &totalApps,
		New:        &allNewApps,
		Active:     &allActiveApps,
		InActive:   &allInActiveApps,
		Terminated: &allTerminatedApps,
		Region:     countDetails,
	}, nil
}

func (r *queryResolver) GetAppTemplates(ctx context.Context) ([]*model.ConfigAppTemplates, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.ConfigAppTemplates{}, fmt.Errorf("Access Denied")
	}
	getconfigTemplates, err := service.GetAppConfigTemplates(user.ID)
	if err != nil {
		log4go.Error("Module: GetAppTemplates, MethodName: GetAppConfigTemplates, Message: %s user:%s", err.Error(), user.ID)
		return []*model.ConfigAppTemplates{}, err
	}
	log4go.Info("Module: GetAppTemplates, MethodName: GetAppConfigTemplates, Message: Fetch app config template based on user is successfully completed, user: %s", user.ID)

	return getconfigTemplates, nil
}

func (r *queryResolver) GetAppsAndOrgsCountDetails(ctx context.Context) (*model.AppsAndOrgsCountDetails, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.AppsAndOrgsCountDetails{}, fmt.Errorf("Access Denied")
	}

	var roleId string

	getRole, err := service.GetRoleByUserId(user.ID)
	if err != nil {
		log4go.Error("Module: GetAppsAndOrgsCountDetails, MethodName: GetRoleByUserId, Message: %s user:%s", err.Error(), user.ID)
		return &model.AppsAndOrgsCountDetails{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetAppsAndOrgsCountDetails, MethodName: GetRoleByUserId, Message: Get Role based on user is successfully completed, user: %s", user.ID)

	if getRole == "Admin" {
		roleId = "1"
	} else {
		roleId = "2"
	}

	orgCount, err := service.GetOrganizationCountById(user.ID)
	if err != nil {
		log4go.Error("Module: GetAppsAndOrgsCountDetails, MethodName: GetOrganizationCountById, Message: %s user:%s", err.Error(), user.ID)
		return &model.AppsAndOrgsCountDetails{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetAppsAndOrgsCountDetails, MethodName: GetOrganizationCountById, Message: Fetch organization count based on user is successfully completed, user: %s", user.ID)

	totalApps, err := service.GetAppsCountById(user.ID)
	if err != nil {
		return &model.AppsAndOrgsCountDetails{}, fmt.Errorf(err.Error())
	}

	orgByAppCount, err := service.GetAppsCountByOrg(user.ID, roleId)
	if err != nil {
		return &model.AppsAndOrgsCountDetails{}, fmt.Errorf(err.Error())
	}

	orgAppsByRegion, err := service.GetAppByRegion(user.ID)
	if err != nil {
		log4go.Error("Module: GetAppsAndOrgsCountDetails, MethodName: GetAppByRegion, Message: %s user:%s", err.Error(), user.ID)
		return &model.AppsAndOrgsCountDetails{}, err
	}
	log4go.Info("Module: GetAppsAndOrgsCountDetails, MethodName: GetAppByRegion, Message: Fetch app by region is successfully completed, user: %s", user.ID)

	return &model.AppsAndOrgsCountDetails{
		TotalOrgCount: &orgCount,
		TotalAppCount: &totalApps,
		OrgByAppCount: orgByAppCount,
		Region:        orgAppsByRegion,
	}, nil
}

func (r *queryResolver) GetAppsAndOrgsandSubOrgCountDetails(ctx context.Context) (*model.AppsAndOrgsAndSubOrgCountDetails, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.AppsAndOrgsAndSubOrgCountDetails{}, fmt.Errorf("Access Denied")
	}

	var roleId string
	getRole, err := service.GetRoleByUserId(user.ID)
	if err != nil {
		log4go.Error("Module: GetAppsAndOrgsCountDetails, MethodName: GetRoleByUserId, Message: %s user:%s", err.Error(), user.ID)
		return &model.AppsAndOrgsAndSubOrgCountDetails{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetAppsAndOrgsCountDetails, MethodName: GetRoleByUserId, Message: Get Role based on user is successfully completed, user: %s", user.ID)

	if getRole == "Admin" {
		roleId = "1"
	} else {
		roleId = "2"
	}

	orgCount, err := service.GetOrganizationCountById(user.ID)
	if err != nil {
		log4go.Error("Module: GetAppsAndOrgsCountDetails, MethodName: GetOrganizationCountById, Message: %s user:%s", err.Error(), user.ID)
		return &model.AppsAndOrgsAndSubOrgCountDetails{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetAppsAndOrgsCountDetails, MethodName: GetOrganizationCountById, Message: Fetch organization count based on user is successfully completed, user: %s", user.ID)

	subOrgCount, err := service.GetSubOrganizationCountById(user.ID)
	if err != nil {
		log4go.Error("Module: GetAppsAndOrgsCountDetails, MethodName: GetSubOrganizationCountById, Message: %s user:%s", err.Error(), user.ID)
		return &model.AppsAndOrgsAndSubOrgCountDetails{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetAppsAndOrgsCountDetails, MethodName: GetSubOrganizationCountById, Message: Fetch sub organization count based on user is successfully completed, user: %s", user.ID)

	businessUnitCount, err := service.GetBusinessUnitCountById(user.ID)
	if err != nil {
		log4go.Error("Module: GetAppsAndOrgsCountDetails, MethodName: GetBusinessUnitCountById, Message: %s user:%s", err.Error(), user.ID)
		return &model.AppsAndOrgsAndSubOrgCountDetails{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetAppsAndOrgsCountDetails, MethodName: GetBusinessUnitCountById, Message: Fetch Business Unit count based on user is successfully completed, user: %s", user.ID)

	totalApps, err := service.GetAppsCountById(user.ID)
	if err != nil {
		return &model.AppsAndOrgsAndSubOrgCountDetails{}, fmt.Errorf(err.Error())
	}

	orgByAppCount, err := service.GetOrgByUserId(user.ID, roleId)
	if err != nil {
		return &model.AppsAndOrgsAndSubOrgCountDetails{}, fmt.Errorf(err.Error())
	}

	orgAppsByRegion, err := service.GetAppByRegion(user.ID)
	if err != nil {
		log4go.Error("Module: GetAppsAndOrgsCountDetails, MethodName: GetAppByRegion, Message: %s user:%s", err.Error(), user.ID)
		return &model.AppsAndOrgsAndSubOrgCountDetails{}, err
	}
	log4go.Info("Module: GetAppsAndOrgsCountDetails, MethodName: GetAppByRegion, Message: Fetch app by region is successfully completed, user: %s", user.ID)

	return &model.AppsAndOrgsAndSubOrgCountDetails{
		TotalOrgCount:          &orgCount,
		TotalSubOrgCount:       &subOrgCount,
		TotalBusinessUnitCount: &businessUnitCount,
		TotalAppCount:          &totalApps,
		OrgByAppCount:          orgByAppCount,
		Region:                 orgAppsByRegion,
	}, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
