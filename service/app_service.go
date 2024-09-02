package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	env "github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal"
	appDeployments "github.com/nifetency/nife.io/internal/app_deployments"
	clusterDetails "github.com/nifetency/nife.io/internal/cluster_info"
	clusterInfo "github.com/nifetency/nife.io/internal/cluster_info"
	"github.com/nifetency/nife.io/internal/decode"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	"github.com/nifetency/nife.io/internal/users"
	"github.com/nifetency/nife.io/pkg/helper"
	_helper "github.com/nifetency/nife.io/pkg/helper"
	commonService "github.com/nifetency/nife.io/pkg/helper"
	"github.com/pelletier/go-toml"
	"k8s.io/client-go/kubernetes"
)

func CreateApp(app *model.NewApp, user_id, subOrgId, businessUnitId, workloadManagementId string) error {
	statement, err := database.Db.Prepare("INSERT INTO app(id, name, hostname, deployed, status, version, appUrl, organization_id,services, config, ipAddresses, createdAt, createdBy, updatedBy, sub_org_id, business_unit_id,workload_management_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	config, _ := json.Marshal(app.App.Config)
	ser, _ := json.Marshal(app.App.Services)
	ipAdd, _ := json.Marshal(app.App.IPAddresses)
	defer statement.Close()
	_, err = statement.Exec(app.App.ID, strings.ToLower(app.App.Name), app.App.Hostname, app.App.Deployed, app.App.Status, app.App.Version, app.App.AppURL, app.App.Organization.ID, ser, config, ipAdd, time.Now().UTC(), user_id, user_id, subOrgId, businessUnitId, workloadManagementId)
	if err != nil {
		return err
	}
	return nil
}

func GetApp(name, userId string) (*model.App, error) {
	var app model.App
	querystring := `
	SELECT 
		ap.id,
		ap.name,
		ap.deployed,
		ap.hostname,
		ap.status,
		ap.version,
		ap.appUrl,
		ap.services,
		ap.ipAddresses,
		ap.config,
		ap.regions,
		ap.backupRegions,
		ap.envArgs,
		ap.imageName,
		ap.port,
		ap.secrets_registry_id,		
		ap.instance_id,
		ap.docker_id,
		ap.host_id,
		ap.tenant_id,
		ap.deploy_type,
		ap.workload_management_id,
		ap.organization_id,
		ap.sub_org_id,
		ap.replicas,
		ap.deployment_time,
		ap.build_time,
		ap.build_logs_url,
		org.id,
		org.name,
		org.domains,
		org.slug,
		org.type
	FROM
		app ap
			JOIN
		organization org ON ap.organization_id = org.id
	WHERE
		ap.name = ?;
    `
	selDB, err := database.Db.Query(querystring, name)
	if err != nil {
		return &app, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var organization model.Organization

		var serv []uint8
		var service []*model.Service

		var ip []uint8
		var ipAddress model.IPAddresses

		var conf []uint8
		var config model.AppConfig

		var domain []uint8
		var domainModel *model.Domains

		var region []uint8
		var regions []*model.Region

		var backupRegion []uint8
		var backupRegions []*model.Region

		err = selDB.Scan(&app.ID, &app.Name, &app.Deployed, &app.Hostname, &app.Status, &app.Version, &app.AppURL, &serv, &ip, &conf, &region, &backupRegion, &app.EnvArgs, &app.ImageName, &app.Port, &app.SecretRegistryID, &app.InstanceID, &app.DockerID, &app.HostID, &app.TenantID, &app.DeployType, &app.WorkloadManagementID, &app.OrganizationID, &app.SubOrganizationID, &app.Replicas, &app.DeploymentTime, &app.BuildTime, &app.BuildLogsURL, &organization.ID, &organization.Name, &domain, &organization.Slug, &organization.Type)

		if err != nil {
			return &app, err
		}

		json.Unmarshal([]byte(string(domain)), &domainModel)
		organization.Domains = domainModel

		app.Organization = &organization

		json.Unmarshal([]byte(string(serv)), &service)
		app.Services = service

		json.Unmarshal([]byte(string(ip)), &ipAddress)
		app.IPAddresses = &ipAddress

		json.Unmarshal([]byte(string(conf)), &config)
		app.Config = &config

		app.ParseConfig = &config

		json.Unmarshal([]byte(string(region)), &regions)
		app.Regions = regions

		json.Unmarshal([]byte(string(backupRegion)), &backupRegions)
		app.BackupRegions = backupRegions
	}
	if app.Status == "Active" {
		var regionCode string
		var clusterDet model.ClusterDetail
		for _, reg := range app.Regions {
			regionCode = *reg.Code
		}
		userRole, err := GetRoleByUserId(userId)
		if userRole == "User" {
			companyName, err := users.GetCompanyNameById(userId)
			if err != nil {
				return nil, err
			}
			adminEmail, err := users.GetAdminByCompanyNameAndEmail(companyName)
			if err != nil {
				return nil, err
			}

			userid, err := users.GetUserIdByEmail(adminEmail)
			if err != nil {
				return nil, err
			}
			userId = strconv.Itoa(userid)
		}

		clsdet, err := clusterDetails.GetClusterDetailsByOrgId(*app.Organization.ID, regionCode, "code", userId)
		if err != nil {
			return &app, err
		}
		clusterDet = model.ClusterDetail{
			RegionCode:            &clsdet.Region_code,
			RegionName:            clsdet.RegionName,
			IsDefault:             &clsdet.IsDefault,
			ClusterConfigPath:     &clsdet.Cluster_config_path,
			EblEnabled:            &clsdet.EBL_enabled,
			Port:                  &clsdet.Port,
			CloudType:             &clsdet.CloudType,
			ClusterType:           &clsdet.ClusterType,
			ProviderType:          clsdet.ProviderType,
			ExternalBaseAddress:   clsdet.ExternalBaseAddress,
			ExternalAgentPlatform: clsdet.ExternalAgentPlatform,
			ExternalLBType:        clsdet.ExternalLBType,
			ExternalCloudType:     clsdet.ExternalCloudType,
			Interface:             clsdet.Interface,
			Route53CountryCode:    clsdet.Route53CountryCode,
			TenantID:              clsdet.TenantId,
			AllocationTag:         &clsdet.AllocationTag,
		}
		app.ClusterDetials = &clusterDet
	}
	app.Releases, err = getAppRelease(app.Name)
	if err != nil {
		return &app, err
	}

	return &app, nil
}

func GetAppStatus(userId, status string) (*model.Nodes, error) {

	var nodes model.Nodes

	var QueryString string

	if status == "Active" {
		QueryString = `
	SELECT 	
    ap.name,
	ap.config,
	ap.regions,		
	org.name
	
FROM
	app ap
		JOIN
	organization org ON ap.organization_id = org.id
WHERE
	ap.createdBy = ? and ap.status = "Active" ;	`
	} else {
		QueryString = `SELECT 	
		ap.name,
		ap.config,
		ap.regions,		
		org.name
		
	FROM
		app ap
			JOIN
		organization org ON ap.organization_id = org.id
	WHERE
		ap.createdBy = ?`
	}

	selDB, err := database.Db.Query(QueryString, userId)
	if err != nil {
		return &model.Nodes{}, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var app model.App

		var organization model.Organization

		var conf []uint8
		var config model.AppConfig

		var region []uint8
		var regions []*model.Region

		err = selDB.Scan(&app.Name, &conf, &region, &organization.Name)
		if err != nil {
			return &model.Nodes{}, err
		}

		json.Unmarshal([]byte(string(conf)), &config)
		app.Config = &config

		json.Unmarshal([]byte(string(region)), &regions)
		app.Regions = regions

		app.Releases, err = getAppReleaseStatus(userId, app.Name)
		if err != nil {
			return &model.Nodes{}, err
		}

		app.Organization = &organization

		nodes.Nodes = append(nodes.Nodes, &app)

	}

	return &nodes, nil
}

func AllApps(user_id string, userRegion string, orgSlug string) (*model.Nodes, error) {
	var nodes model.Nodes

	Querystring := `
	SELECT 
		ap.id,
		ap.name,
        IFNULL( ar.version, "") as version,       
		ap.deployed,
		ap.appUrl,
		ap.hostname,			
		ap.status,
		ap.regions,
		ap.config,
		ap.createdAt,
		ap.workload_management_id,
		ap.organization_id,
		ap.sub_org_id,
		ap.business_unit_id,
		ap.imageName,
        org.slug
	FROM
		app ap
	   JOIN organization org on ap.organization_id = org.id
       LEFT JOIN        
		 app_release ar on (ar.app_id = ap.name and ar.status = 'active')`
	if userRegion == "" && user_id != "" {
		Querystring = Querystring + `Where ap.status != 'Terminated' and ap.createdBy = ? `
	}
	if userRegion != "" && user_id != "" {
		Querystring = Querystring + `Where ap.status != 'Terminated' and ap.createdBy = ? and JSON_CONTAINS(regions, '{"code":"` + userRegion + `"}') = 1 `
	}

	if orgSlug != "" {
		Querystring = Querystring + `and org.slug = ` + `"` + orgSlug + `"`
	}

	Querystring = Querystring + ` order by createdAt desc`

	selDB, err := database.Db.Query(Querystring, user_id)
	if err != nil {
		return &nodes, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var app model.App

		var organization model.Organization
		var appVersion string
		// var createdAt time.Time

		var domain []uint8
		var domainModel *model.Domains

		//var createdAt string

		// var cr []uint8
		// var currentRelease model.Release

		var region []uint8
		var regions []*model.Region

		var config []uint8
		var configModel *model.AppConfig
		// fmt.Println(currentRelease)

		err = selDB.Scan(&app.ID, &app.Name, &appVersion, &app.Deployed, &app.AppURL, &app.Hostname, &app.Status, &region, &config, &app.CreatedAt, &app.WorkloadManagementID, &app.OrganizationID, &app.SubOrganizationID, &app.BusinessUnitID, &app.ImageName, &organization.Slug)
		if err != nil {
			fmt.Println(err)
			return &nodes, err
		}
		json.Unmarshal([]byte(string(domain)), &domainModel)
		organization.Domains = domainModel

		app.Organization = &organization

		// currentRelease.CreatedAt = &createdAt

		app.Releases, err = getAppRelease(app.Name)
		if err != nil {
			log.Println(err)
		}

		app.Version, _ = strconv.Atoi(appVersion)
		if err != nil {
			log.Println(err)
		}

		json.Unmarshal([]byte(string(region)), &regions)
		app.Regions = regions

		json.Unmarshal([]byte(string(config)), &configModel)
		app.Config = configModel

		if app.Status == "New" || configModel.Build == nil {
			defaultBuiltin1 := "-"
			app.BuiltinType = &defaultBuiltin1
		} else {
			app.BuiltinType = configModel.Build.Builder
		}

		orgName, err := GetOrgNameById(*app.OrganizationID)
		if err != nil {
			log.Println(err)
		}
		subOrgName, err := GetOrgNameById(*app.SubOrganizationID)
		if err != nil {
			log.Println(err)
		}
		businessUnitName, err := GetBusinessUnitById(*app.BusinessUnitID)
		if err != nil {
			log.Println(err)
		}
		var workloadName string
		var workloadEndPoint string

		if *app.WorkloadManagementID != "" {
			workloadName, workloadEndPoint, err = GetWorkloadNameById(*app.WorkloadManagementID)
			if err != nil {
				log.Println(err)
			}
		} else {
			workloadName = ""
			workloadEndPoint = ""
		}

		userDetAct, err := GetById(user_id)
		if err != nil {
			return nil, err
		}

		app.OrganizationName = &orgName
		app.SubOrganizationName = &subOrgName
		app.BusinessUnitName = &businessUnitName
		app.WorkloadManagementName = &workloadName
		app.WorkloadManagementEndPoint = &workloadEndPoint
		app.UserDetails = &userDetAct

		nodes.Nodes = append(nodes.Nodes, &app)
	}

	// nodes.Releases, err = getAppRelease(appId)

	return &nodes, nil
}

func AllSubApps(user_id string, userRegion string, subOrgSlug string) (*model.Nodes, error) {
	var nodes model.Nodes

	Querystring := `
	SELECT 
		ap.id,
		ap.name,
        IFNULL( ar.version, "") as version,       
		ap.deployed,
		ap.appUrl,
		ap.hostname,			
		ap.status,
		ap.regions,
		ap.config,
		ap.createdAt,
		ap.workload_management_id,
		ap.organization_id,
		ap.sub_org_id,
		ap.business_unit_id,
        org.slug
	FROM
		app ap
	   JOIN organization org on ap.sub_org_id = org.id
       LEFT JOIN        
		 app_release ar on (ar.app_id = ap.name and ar.status = 'active')`
	if userRegion == "" && user_id != "" {
		Querystring = Querystring + `Where ap.status != 'Terminated' and ap.createdBy = ? `
	}
	if userRegion != "" && user_id != "" {
		Querystring = Querystring + `Where ap.status != 'Terminated' and ap.createdBy = ? and JSON_CONTAINS(regions, '{"code":"` + userRegion + `"}') = 1 `
	}

	if subOrgSlug != "" {
		Querystring = Querystring + `and org.slug = ` + `"` + subOrgSlug + `"`
	}

	Querystring = Querystring + ` order by createdAt desc`

	selDB, err := database.Db.Query(Querystring, user_id)
	if err != nil {
		return &nodes, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var app model.App

		var organization model.Organization
		var appVersion string
		// var createdAt time.Time

		var domain []uint8
		var domainModel *model.Domains

		//var createdAt string

		// var cr []uint8
		// var currentRelease model.Release

		var region []uint8
		var regions []*model.Region

		var config []uint8
		var configModel *model.AppConfig
		// fmt.Println(currentRelease)

		err = selDB.Scan(&app.ID, &app.Name, &appVersion, &app.Deployed, &app.AppURL, &app.Hostname, &app.Status, &region, &config, &app.CreatedAt, &app.WorkloadManagementID, &app.OrganizationID, &app.SubOrganizationID, &app.BusinessUnitID, &organization.Slug)
		if err != nil {
			fmt.Println(err)
			return &nodes, err
		}
		json.Unmarshal([]byte(string(domain)), &domainModel)
		organization.Domains = domainModel

		app.Organization = &organization

		// currentRelease.CreatedAt = &createdAt

		app.Releases, err = getAppRelease(app.Name)
		if err != nil {
			log.Println(err)
		}

		app.Version, _ = strconv.Atoi(appVersion)
		if err != nil {
			log.Println(err)
		}

		json.Unmarshal([]byte(string(region)), &regions)
		app.Regions = regions

		json.Unmarshal([]byte(string(config)), &configModel)
		app.Config = configModel

		var configBuiltin string

		for _, i := range app.Releases.Nodes {
			if *i.Status == "active" {
				defaultBuiltin := "Image"
				app.BuiltinType = &defaultBuiltin

				if configModel.Build.Builtin == nil {
					configBuiltin = ""
				} else if *configModel.Build.Builtin == "" {
					configBuiltin = ""
				}

				if *i.ArchiveURL != "" && configBuiltin != "" {
					*app.BuiltinType = "Built-In"

				} else if *i.ArchiveURL != "" && configBuiltin == "" {
					*app.BuiltinType = "Docker File"

				} else {
					app.BuiltinType = &defaultBuiltin
				}

			}

		}
		if app.Status == "New" {
			defaultBuiltin1 := "-"
			app.BuiltinType = &defaultBuiltin1
		}

		orgName, err := GetOrgNameById(*app.OrganizationID)
		if err != nil {
			log.Println(err)
		}
		subOrgName, err := GetOrgNameById(*app.SubOrganizationID)
		if err != nil {
			log.Println(err)
		}
		businessUnitName, err := GetBusinessUnitById(*app.BusinessUnitID)
		if err != nil {
			log.Println(err)
		}
		var workloadName string
		var workloadEndPoint string

		if *app.WorkloadManagementID != "" {
			workloadName, workloadEndPoint, err = GetWorkloadNameById(*app.WorkloadManagementID)
			if err != nil {
				log.Println(err)
			}
		} else {
			workloadName = ""
			workloadEndPoint = ""
		}

		app.OrganizationName = &orgName
		app.SubOrganizationName = &subOrgName
		app.BusinessUnitName = &businessUnitName
		app.WorkloadManagementName = &workloadName
		app.WorkloadManagementEndPoint = &workloadEndPoint

		nodes.Nodes = append(nodes.Nodes, &app)
	}

	// nodes.Releases, err = getAppRelease(appId)

	return &nodes, nil
}

func AllBusinessUnitApps(user_id string, userRegion string, businessunit string) (*model.Nodes, error) {
	var nodes model.Nodes

	Querystring := `
	SELECT 
		ap.id,
		ap.name,
        IFNULL( ar.version, "") as version,       
		ap.deployed,
		ap.appUrl,
		ap.hostname,			
		ap.status,
		ap.regions,
		ap.config,
		ap.createdAt
	FROM
		app ap
	   JOIN business_unit bu on ap.business_unit_id = bu.id
       LEFT JOIN        
		 app_release ar on (ar.app_id = ap.name and ar.status = 'active')`
	if userRegion == "" && user_id != "" {
		Querystring = Querystring + `Where ap.status != 'Terminated' and ap.createdBy = ? `
	}
	if userRegion != "" && user_id != "" {
		Querystring = Querystring + `Where ap.status != 'Terminated' and ap.createdBy = ? and JSON_CONTAINS(regions, '{"code":"` + userRegion + `"}') = 1 `
	}

	if businessunit != "" {
		Querystring = Querystring + `and bu.name = ` + `"` + businessunit + `"`
	}

	Querystring = Querystring + ` order by createdAt desc`

	selDB, err := database.Db.Query(Querystring, user_id)
	if err != nil {
		return &nodes, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var app model.App

		var organization model.Organization
		var appVersion string
		// var createdAt time.Time

		var domain []uint8
		var domainModel *model.Domains

		//var createdAt string

		// var cr []uint8
		// var currentRelease model.Release

		var region []uint8
		var regions []*model.Region

		var config []uint8
		var configModel *model.AppConfig
		// fmt.Println(currentRelease)

		err = selDB.Scan(&app.ID, &app.Name, &appVersion, &app.Deployed, &app.AppURL, &app.Hostname, &app.Status, &region, &config, &app.CreatedAt)
		if err != nil {
			fmt.Println(err)
			return &nodes, err
		}
		json.Unmarshal([]byte(string(domain)), &domainModel)
		organization.Domains = domainModel

		app.Organization = &organization

		// currentRelease.CreatedAt = &createdAt

		app.Releases, err = getAppRelease(app.Name)
		if err != nil {
			log.Println(err)
		}

		app.Version, _ = strconv.Atoi(appVersion)
		if err != nil {
			log.Println(err)
		}

		json.Unmarshal([]byte(string(region)), &regions)
		app.Regions = regions

		json.Unmarshal([]byte(string(config)), &configModel)
		app.Config = configModel

		var configBuiltin string

		for _, i := range app.Releases.Nodes {
			if *i.Status == "active" {
				defaultBuiltin := "Image"
				app.BuiltinType = &defaultBuiltin

				if configModel.Build.Builtin == nil {
					configBuiltin = ""
				} else if *configModel.Build.Builtin == "" {
					configBuiltin = ""
				}

				if *i.ArchiveURL != "" && configBuiltin != "" {
					*app.BuiltinType = "Built-In"

				} else if *i.ArchiveURL != "" && configBuiltin == "" {
					*app.BuiltinType = "Docker File"

				} else {
					app.BuiltinType = &defaultBuiltin
				}

			}

		}
		if app.Status == "New" {
			defaultBuiltin1 := "-"
			app.BuiltinType = &defaultBuiltin1
		}

		buDetails, err := GetBusinessUnitByName(user_id, businessunit)
		if err != nil {
			return &model.Nodes{}, err
		}

		orgnName, err := GetOrgNameById(*buDetails.OrgID)
		if err != nil {
			return &model.Nodes{}, err
		}
		if *buDetails.SubOrgID != "" {
			subOrgName, err := GetOrgNameById(*buDetails.SubOrgID)
			if err != nil {
				return &model.Nodes{}, err
			}
			app.SubOrganizationID = buDetails.SubOrgID
			app.SubOrganizationName = &subOrgName

		}

		app.OrganizationID = buDetails.OrgID
		app.OrganizationName = &orgnName

		nodes.Nodes = append(nodes.Nodes, &app)
	}

	// nodes.Releases, err = getAppRelease(appId)

	return &nodes, nil
}

func ReturnFirstApp() (*model.App, error) {
	var app model.App
	querystring := `
	SELECT 
		ap.id,
		ap.name,
		ap.deployed,
		ap.hostname,
		ap.currentRelease,
		ap.status,
		org.id,
		org.name,
		org.domains,
		org.slug,
		org.type
	FROM
		app ap
			JOIN
		organization org ON ap.organization_id = org.id limit 1`
	selDB, err := database.Db.Query(querystring)
	if err != nil {
		return &app, err
	}
	defer selDB.Close()

	var cr []uint8
	var currentRelease model.Release

	var organization model.Organization

	var domain []uint8
	var domainModel *model.Domains

	err = selDB.Scan(&app.ID, &app.Name, &app.Deployed, &app.Hostname, &app.CurrentRelease, &app.Status, &organization.ID, &organization.Name, &domain, &organization.Slug, &organization.Type)
	if err == sql.ErrNoRows {
		return &app, errors.New("No app found")
	} else if err != nil {
		return &app, err
	}

	json.Unmarshal([]byte(string(domain)), &domainModel)
	organization.Domains = domainModel

	app.Organization = &organization

	json.Unmarshal([]byte(string(cr)), &currentRelease)
	app.CurrentRelease = &currentRelease
	return &app, nil
}

func ReturnFirstAppCompact() (*model.AppCompact, error) {
	var app model.AppCompact
	querystring := `
	SELECT 
		ap.id,
		ap.name,
		ap.deployed,
		ap.hostname,
		ap.status,
		org.id,
		org.name,
		org.domains,
		org.slug,
		org.type
	FROM
		app ap
			JOIN
		organization org ON ap.organization_id = org.id limit 1`
	row := database.Db.QueryRow(querystring)

	var organization model.Organization

	var domain []uint8
	var domainModel *model.Domains

	var rel []uint8
	var release model.Release

	var ip []uint8
	var ipAddress model.IPAddresses

	var serv []uint8
	var service []*model.Service

	err := row.Scan(&app.ID, &app.Name, &app.Deployed, &app.Hostname, &app.Status, &organization.ID, &organization.Name, &domain, &organization.Slug, &organization.Type)
	if err == sql.ErrNoRows {
		return &app, errors.New("No app found")
	} else if err != nil {
		return &app, err
	}

	json.Unmarshal([]byte(string(domain)), &domainModel)
	organization.Domains = domainModel

	app.Organization = &organization

	json.Unmarshal([]byte(string(rel)), &release)
	app.Release = &release

	json.Unmarshal([]byte(string(ip)), &ipAddress)
	app.IPAddresses = &ipAddress

	json.Unmarshal([]byte(string(serv)), &service)
	app.Services = service
	return &app, nil
}

func AddOrRemoveRegions(name string, addList []*string, delList []*string, backupList []*string, toDeploy bool, userId string) (*model.App, string, error) {
	app, err := GetApp(name, userId)
	if err != nil {
		return nil, "", err
	}
	var orgSlug string

	if *app.SubOrganizationID != "" {
		getSubOrg, err := GetOrganizationById(*app.SubOrganizationID)
		if err != nil {
			return nil, "", err
		}
		orgSlug = *getSubOrg.Slug
	} else {
		orgSlug = *app.Organization.Slug
	}
	var deploymentId string
	regions := make([]*model.Region, 0)
	backupRegionsList := make([]*model.Region, 0)
	regions = append(regions, app.Regions...)
	backupRegionsList = append(backupRegionsList, app.BackupRegions...)

	for _, addItem := range addList {
		found := false
		for _, existingItemAdd := range app.Regions {
			if *addItem == *existingItemAdd.Code {
				found = true
				break
			}
		}
		if !found {
			var roleUserId string
			userRole, err := GetRoleByUserId(userId)
			if userRole == "User" {
				companyName, err := users.GetCompanyNameById(userId)
				if err != nil {
					return nil, "", err
				}
				adminEmail, err := users.GetAdminByCompanyNameAndEmail(companyName)
				if err != nil {
					return nil, "", err
				}

				userid, err := users.GetUserIdByEmail(adminEmail)
				if err != nil {
					return nil, "", err
				}
				roleUserId = strconv.Itoa(userid)
			} else {
				roleUserId = userId
			}

			regionData, err := helper.GetRegionByCode(*addItem, "code", roleUserId)
			if err != nil {
				return nil, "", fmt.Errorf("%s Region does not exists", *addItem)
			}
			regions = append(regions, &model.Region{Code: regionData.Code, Name: regionData.Name, Latitude: regionData.Latitude, Longitude: regionData.Longitude})
			if toDeploy {

				currentRelease, err := _helper.GetAppRelease(app.Name, "active")
				if err != nil {
					log.Println(err)
				}
				clusterDetails, err := clusterDetails.GetClusterDetailsByOrgId(*app.Organization.ID, *addItem, "code", roleUserId)
				fmt.Println(clusterDetails)
				if err != nil {
					return &model.App{}, "", err
				}
				//Deployment code running
				port, _ := helper.GetInternalPort(app.Config.Definition)

				secretRegid := ""

				if app.SecretRegistryID != nil {
					secretRegid = *app.SecretRegistryID
				}

				memoryResource := _helper.GetResourceRequirement(app.Config.Definition)

				nullCheckStruct := model.Requirement{}
				empty := ""

				if memoryResource == nullCheckStruct {
					memoryResource = model.Requirement{
						RequestRequirement: &model.RequirementProperties{CPU: &empty, Memory: &empty},
						LimitRequirement:   &model.RequirementProperties{CPU: &empty, Memory: &empty},
					}
				}

				environmentArgument := ""
				if app.EnvArgs != nil && *app.EnvArgs != "" {

					type PortData struct {
						Port string `json:"PORT"`
					}
					unescapedValue := strings.ReplaceAll(*app.EnvArgs, `\"`, `"`)

					var data PortData
					err := json.Unmarshal([]byte(unescapedValue), &data)

					envArgsString := fmt.Sprintf("PORT=%s", data.Port)
					if err != nil {
						envArgsString = *app.EnvArgs
					}

					environmentArgument, err = env.EnvironmentArgument(envArgsString, *clusterDetails.Interface, "", model.Requirement{})
					if err != nil {
						return nil, "", err
					}
				}

				var image string
				if *app.Config.Build.Image == "" {
					image = currentRelease.ImageName
				} else {
					image = *app.Config.Build.Image
					currentRelease.Port, _ = strconv.Atoi(*app.Port)
				}

				deployOutput, err := Deploy(app.Name, image, secretRegid, orgSlug, int32(currentRelease.Port), *clusterDetails, environmentArgument, memoryResource, userId, true)

				if err != nil {
					return nil, "", err
				}
				// Create Deployment for Current Release
				release, err := commonService.GetAppRelease(app.Name, "active")
				if err != nil {
					return nil, "", err
				}

				deploymentId = uuid.NewString()
				deployment := appDeployments.AppDeployments{
					Id:            deploymentId,
					AppId:         app.Name,
					Region_code:   clusterDetails.Region_code,
					Status:        "running",
					Deployment_id: deployOutput.ID,
					Port:          fmt.Sprintf("%v", port),
					App_Url:       *deployOutput.LoadBalanceURL,
					Release_id:    release.Id,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
					ContainerID:   *deployOutput.ContainerID,
				}
				deploymentErr := commonService.CreateDeploymentsRecord(deployment)
				if deploymentErr != nil {
					return nil, "", deploymentErr
				}
				// _ = CreateOrDeleteDNSRecord(deploymentId, deployOutput.HostName, *deployOutput.LoadBalanceURL, clusterDetails.Region_code, clusterDetails.CloudType, false)  // DNS route53

				externalPort, _ := _helper.GetExternalPort(app.Config.Definition)
				_, err = CreateCLBRoute(app.Name, *deployOutput.LoadBalanceURL, strconv.Itoa(int(externalPort)), true)
				if err != nil {
					return nil, "", err
				}

			}
		}
	}
	// remove region
	for _, deleteItem := range delList {
		if app.Regions != nil {
			AppRegion := len(app.Regions)
			delRegion := len(delList)
			supportRegion := len(app.BackupRegions)

			if AppRegion == 1 && delRegion == 1 && supportRegion > 0 {
				for i, region := range app.BackupRegions {
					organizationRegionData, err := clusterDetails.GetClusterDetailsByOrgId(*app.Organization.ID, *region.Code, "code", userId)
					if err != nil {
						return &model.App{}, "", err
					}

					if toDeploy {
						UnDeploy(name, *organizationRegionData, app.Hostname, app.Organization, userId, true)
					}

					backupRegionsList = RemoveIndex(backupRegionsList, i)

				}
			}

			for i, existingItemAdd := range regions {
				if *deleteItem == *existingItemAdd.Code {
					regions = RemoveIndex(regions, i)
					var roleUserId string
					userRole, err := GetRoleByUserId(userId)
					if userRole == "User" {
						companyName, err := users.GetCompanyNameById(userId)
						if err != nil {
							return nil, "", err
						}
						adminEmail, err := users.GetAdminByCompanyNameAndEmail(companyName)
						if err != nil {
							return nil, "", err
						}

						userid, err := users.GetUserIdByEmail(adminEmail)
						if err != nil {
							return nil, "", err
						}
						roleUserId = strconv.Itoa(userid)
					} else {
						roleUserId = userId
					}

					organizationRegionData, err := clusterDetails.GetClusterDetailsByOrgId(*app.Organization.ID, *deleteItem, "code", roleUserId)
					if err != nil {
						return &model.App{}, "", err
					}
					if toDeploy {
						UnDeploy(name, *organizationRegionData, app.Hostname, app.Organization, userId, true)
						break
					}
				}
			}
		}
	}
	//backup region
	for _, backupItem := range backupList {
		backupFound := false
		for _, existingItemBackup := range app.BackupRegions {
			if *backupItem == *existingItemBackup.Code {
				backupFound = true
				break
			}
		}
		if !backupFound {
			support := []string{}
			support = append(support, *backupItem)
			for _, regCode := range app.Regions {
				checkCode := contains(support, *regCode.Code)
				if checkCode {
					return nil, "", fmt.Errorf("Cannot Add Region %s As Backup, Beacuse It is in Primary Region", *regCode.Code)
				}
			}

			backupRegionData, err := helper.GetRegionByCode(*backupItem, "code", userId)
			if err != nil {
				return nil, "", err
			}
			secretRegid := ""

			if app.SecretRegistryID != nil {
				secretRegid = *app.SecretRegistryID
			}

			clusterDetails, err := clusterDetails.GetClusterDetailsByOrgId(*app.Organization.ID, *backupItem, "code", userId)
			fmt.Println(clusterDetails)
			if err != nil {
				return &model.App{}, "", err
			}

			environmentArgument := ""
			if app.EnvArgs != nil && *app.EnvArgs != "" {
				environmentArgument, err = env.EnvironmentArgument(*app.EnvArgs, *clusterDetails.Interface, "", model.Requirement{})
				if err != nil {
					return nil, "", err
				}
			}

			release, err := commonService.GetAppRelease(app.Name, "active")
			if err != nil {
				return nil, "", err
			}

			memoryResource := _helper.GetResourceRequirement(app.Config.Definition)
			nullCheckStruct := model.Requirement{}
			empty := ""

			if memoryResource == nullCheckStruct {
				memoryResource = model.Requirement{
					RequestRequirement: &model.RequirementProperties{CPU: &empty, Memory: &empty},
					LimitRequirement:   &model.RequirementProperties{CPU: &empty, Memory: &empty},
				}
			}
			deployOutput, err := Deploy(app.Name, *app.Config.Build.Image, secretRegid, *app.Organization.Slug, int32(release.Port), *clusterDetails, environmentArgument, memoryResource, userId, true)
			if err != nil {
				return nil, "", err
			}
			fmt.Println(deployOutput)

			// deploymentId := uuid.NewString()
			// deployment := appDeployments.AppDeployments{
			// 	Id:            deploymentId,
			// 	AppId:         app.Name,
			// 	Region_code:   clusterDetails.Region_code,
			// 	Status:        "running",
			// 	Deployment_id: deployOutput.ID,
			// 	Port:          fmt.Sprintf("%v", release.Port),
			// 	App_Url:       *deployOutput.LoadBalanceURL,
			// 	Release_id:    release.Id,
			// 	CreatedAt:     time.Now(),
			// 	UpdatedAt:     time.Now(),
			// 	ContainerID:   *deployOutput.ContainerID,
			// }

			// deploymentErr := commonService.CreateDeploymentsRecord(deployment)
			// if deploymentErr != nil {
			// 	return nil, deploymentErr
			// }

			// _ = CreateOrDeleteDNSRecord(deploymentId, deployOutput.HostName, *deployOutput.LoadBalanceURL, clusterDetails.Region_code, clusterDetails.CloudType, false)

			backupRegionsList = append(backupRegionsList, &model.Region{Code: backupRegionData.Code, Name: backupRegionData.Name, Latitude: backupRegionData.Latitude, Longitude: backupRegionData.Longitude})
		}
	}

	statement, err := database.Db.Prepare("Update app set regions = ?, backupRegions = ? where name = ?")
	if err != nil {
		return nil, "", err
	}
	regionsJSON, _ := json.Marshal(regions)
	backupRegionsListJSON, _ := json.Marshal(backupRegionsList)
	_, err = statement.Exec(regionsJSON, backupRegionsListJSON, name)
	if err != nil {
		fmt.Println(err)
		return nil, "", err
	}

	// Update App Status
	UpdateAppStatus(name)
	updateapp, err := GetApp(name, userId)
	if err != nil {
		return nil, "", err
	}
	return updateapp, deploymentId, nil
}

func UpdateAppStatus(name string) error {

	release, err := commonService.GetAppRelease(name, "active")
	if err != nil {
		return err
	}

	appDeployments, err := commonService.GetDeploymentsByReleaseId(name, release.Id)
	if err != nil {
		return err
	}
	appStatus := ""

	if len(*appDeployments) == 0 {
		// update App Status to Terminated
		appStatus = string(internal.AppStatusTerminated)
	}
	for _, d := range *appDeployments {
		if d.Status == "running" {
			appStatus = string(internal.AppStatusActive)
		}
	}
	if appStatus == "" {
		appStatus = string(internal.AppStatusSuspended)
	}
	statement, err := database.Db.Prepare("Update app set status = ? where name = ?")
	if err != nil {
		return err
	}
	_, err = statement.Exec(appStatus, name)
	if err != nil {
		return err
	}
	return nil
}

func contains(s []string, searchterm string) bool {
	i := sort.SearchStrings(s, searchterm)
	return i < len(s) && s[i] == searchterm
}

func UpdateAppConfig(name string, appConfig *model.AppConfig) error {
	config, _ := json.Marshal(appConfig)
	fmt.Println(appConfig)
	statement, err := database.Db.Prepare("Update app set Config = ? , ParseConfig = ?, updatedAt =?  where name = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(config, config, time.Now().UTC(), name)
	if err != nil {
		return err
	}

	return nil
}

func UpdateVersion(appName string, vers int) error {
	statement, err := database.Db.Prepare("Update app set version = ? where name = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(vers, appName)
	if err != nil {
		return err
	}
	return nil
}

func GetRunningRegionApp(appId string, status string, userId string) (*model.AppDeploymentRegion, error) {
	appRegions, avalRegions, err := clusterInfo.GetActiveClusterByAppId(appId, status, userId)
	if err != nil {
		return nil, err
	}
	if avalRegions == nil {
		avalRegions = []*model.Region{}
	}
	return &model.AppDeploymentRegion{Regions: appRegions, AvailableRegions: avalRegions}, nil

}

func RemoveIndex(s []*model.Region, index int) []*model.Region {
	return append(s[:index], s[index+1:]...)
}

func GenarateAndValidateName(appId string) (string, bool, error) {
	if appId == "" {
		randomName := helper.GetRandomName(5)
		newAppId, exist, _ := validateName(randomName)
		if exist {
			GenarateAndValidateName("")
		}
		return newAppId, exist, nil
	}
	return validateName(appId)
}

func validateName(appId string) (string, bool, error) {
	appName := ""
	querystring := "SELECT name FROM app where name = ? LIMIT 1"
	selDB, err := database.Db.Query(querystring, appId)
	if err != nil {
		return appName, false, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&appName)
		if err != nil {
			return appName, false, err
		}
	}
	if appName != "" {
		return appId, true, nil
	}
	return appId, false, nil
}

func getAppRelease(appId string) (*model.Releases, error) {
	stable := true
	var releases model.Releases
	querystring := ` SELECT 
	id, status, reason, description, version, image_name, user_id, port, 
	COALESCE((SELECT email FROM user WHERE id = user_id), '') AS user_email,
	COALESCE((SELECT firstName FROM user WHERE id = user_id), '') AS first_name,
	COALESCE((SELECT lastName FROM user WHERE id = user_id), '') AS last_name,
	createdAt, archive_url, builder_type, routing_policy 
FROM app_release 
WHERE app_id = ? 
ORDER BY createdAt DESC`
	selDB, err := database.Db.Query(querystring, appId)
	if err != nil {
		return &model.Releases{}, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var release model.Release
		var user model.User
		err = selDB.Scan(&release.ID, &release.Status, &release.Reason, &release.Description, &release.Version, &release.Image, &user.ID, &release.Port, &user.Email, &user.FirstName, &user.LastName, &release.CreatedAt, &release.ArchiveURL, &release.BuilderType, &release.RoutingPolicy)
		if err != nil {
			return &model.Releases{}, err
		}
		appStatus, _ := GetAppStatusByAppName(appId)
		if appStatus == "Suspended" {
			*release.Status = "inactive"
		}
		namee := user.FirstName + "" + user.LastName
		user.Name = namee
		release.Stable = &stable
		release.User = &user
		releases.Nodes = append(releases.Nodes, &release)
	}
	return &releases, nil
}

func getAppReleaseStatus(userId string, appId string) (*model.Releases, error) {
	stable := true
	var releases model.Releases
	querystring := "SELECT id, status, reason, description, version,image_name,user_id , (SELECT email from user where id = user_id) as user_email, createdAt FROM app_release where user_id = ? and app_id = ?  order by createdAt desc"
	selDB, err := database.Db.Query(querystring, userId, appId)
	if err != nil {
		return &model.Releases{}, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var release model.Release
		var user model.User
		err = selDB.Scan(&release.ID, &release.Status, &release.Reason, &release.Description, &release.Version, &release.Image, &user.ID, &user.Email, &release.CreatedAt)

		if err != nil {
			return &model.Releases{}, err
		}

		release.Stable = &stable
		release.User = &user
		releases.Nodes = append(releases.Nodes, &release)
	}
	return &releases, nil
}

func GetAvailabilityCluster(isLatency, userId string) (*model.ClusterNodes, error) {
	var nodes model.ClusterNodes
	var queryString string

	if isLatency == "" {
		querystring := "SELECT ci.id, ci.region_code, ci.ip_address, ci.cluster_config_path, ci.load_balancer_url, ci.cluster_type, ci.name, ci.is_low_latency, regions.latitude, regions.longitude FROM cluster_info ci INNER JOIN regions ON ci.region_code=regions.code where ci.is_active = 1"
		queryString = querystring
	}

	if isLatency != "" {
		querystring := "SELECT ci.id, ci.region_code, ci.ip_address, ci.cluster_config_path, ci.load_balancer_url, ci.cluster_type, ci.name, ci.is_low_latency, regions.latitude, regions.longitude FROM cluster_info ci INNER JOIN regions ON ci.region_code=regions.code where ci.is_active = 1 and ci.is_low_latency = " + isLatency
		queryString = querystring
	}

	selDB, err := database.Db.Query(queryString)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var availability model.ClusterInfo
		err = selDB.Scan(&availability.ID, &availability.RegionCode, &availability.IPAddress, &availability.LoadBalancerURL, &availability.ClusterConfigPath, &availability.Clustertype,
			&availability.Name, &availability.IsLatency, &availability.Latitude, &availability.Longitude)
		if err != nil {
			return nil, err
		}
		res := model.ClusterInfo{
			ID:                availability.ID,
			RegionCode:        availability.RegionCode,
			IPAddress:         availability.IPAddress,
			ClusterConfigPath: availability.ClusterConfigPath,
			Clustertype:       availability.Clustertype,
			Name:              availability.Name,
			IsLatency:         availability.IsLatency,
			Latitude:          availability.Latitude,
			Longitude:         availability.Longitude,
			LoadBalancerURL:   availability.LoadBalancerURL,
		}

		nodes.Nodes = append(nodes.Nodes, &res)
	}

	availabilityCluster, err := GetAvailabilityAddedCluster(userId)
	if err != nil {
		return nil, err
	}

	nodes.Nodes = append(nodes.Nodes, availabilityCluster.Nodes...)

	return &nodes, nil

}

func GetAvailabilityAddedCluster(userId string) (*model.ClusterNodes, error) {
	var nodes model.ClusterNodes

	queryString := "SELECT ci.id, ci.region_code, ci.cluster_type, ci.location_name FROM cluster_info_user ci where ci.is_active = 1 and user_id = ?;"

	selDB, err := database.Db.Query(queryString, userId)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var availability model.ClusterInfo
		err = selDB.Scan(&availability.ID, &availability.RegionCode, &availability.Clustertype, &availability.Name)
		if err != nil {
			return nil, err
		}
		IsLatency := true
		res := model.ClusterInfo{
			ID:          availability.ID,
			RegionCode:  availability.RegionCode,
			Name:        availability.Name,
			IsLatency:   &IsLatency,
			Clustertype: availability.Clustertype,
		}
		nodes.Nodes = append(nodes.Nodes, &res)

	}

	return &nodes, nil

}

func GetRegionStatus(appId string) (model.RegionStatusNodes, error) {
	var nodes model.RegionStatusNodes

	selDB, err := database.Db.Query("select id,region_code, status, deployment_id, port,app_url, release_id, elb_record_name, elb_record_id from app_deployments where appId = '" + appId + "'and (status = 'suspended' or status = 'running' )")
	if err != nil {
		return model.RegionStatusNodes{}, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var regionDetails model.RegionStatus
		err = selDB.Scan(&regionDetails.ID, &regionDetails.RegionCode, &regionDetails.Status, &regionDetails.DeploymentID, &regionDetails.Port, &regionDetails.AppURL, &regionDetails.ReleaseID, &regionDetails.ElbRecordName, &regionDetails.ElbRecordID)
		if err != nil {
			return model.RegionStatusNodes{}, err
		}
		regionResponse := model.RegionStatus{
			ID:            regionDetails.ID,
			RegionCode:    regionDetails.RegionCode,
			Status:        regionDetails.Status,
			DeploymentID:  regionDetails.DeploymentID,
			Port:          regionDetails.Port,
			AppURL:        regionDetails.AppURL,
			ReleaseID:     regionDetails.ReleaseID,
			ElbRecordName: regionDetails.ElbRecordName,
			ElbRecordID:   regionDetails.ElbRecordID,
		}

		nodes.Nodes = append(nodes.Nodes, &regionResponse)

	}
	if nodes.Nodes == nil {
		return model.RegionStatusNodes{
			Nodes: []*model.RegionStatus{},
		}, err
	}
	return nodes, nil
}

func GetAvailableRegion() (model.Regions, error) {
	getActivePlatformCluster, err := clusterInfo.GetActivePlatormClusterDetails()
	if err != nil {
		return model.Regions{}, err
	}
	var nodes model.Regions

	selDB, err := database.Db.Query("select code, name, latitude, longitude from regions")
	if err != nil {
		return model.Regions{}, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var regionDetails model.PlatFormOutput
		err = selDB.Scan(&regionDetails.Code, &regionDetails.Name, &regionDetails.Latitude, &regionDetails.Longitude)
		if err != nil {
			return model.Regions{}, err
		}
		result := model.PlatFormOutput{
			Code:      regionDetails.Code,
			Name:      regionDetails.Name,
			Latitude:  regionDetails.Latitude,
			Longitude: regionDetails.Longitude,
		}
		for _, activeCluster := range *getActivePlatformCluster {
			if activeCluster.Region_code == *regionDetails.Code {
				nodes.Regions = append(nodes.Regions, &result)
			}
		}
	}
	return nodes, err
}

func UserAppCount(userId string) (int, error) {

	selDB, err := database.Db.Query("SELECT COUNT(name) FROM app  where (app.status = 'Active' or app.status = 'New' or app.status = 'Suspended') and app.createdBy = ?", userId)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	var count int

	for selDB.Next() {
		err = selDB.Scan(&count)
		if err != nil {
			return 0, err
		}
	}

	s3DeploymentCount, err := GetS3DeploymentsCountByUsreId(userId)
	if err != nil {
		return 0, nil
	}

	totalCount := count + s3DeploymentCount
	return totalCount, nil

}

func AppExistUser(appName, userId string) (bool, error) {

	selDB, err := database.Db.Query("select id from app where name = ? and createdBy = ?", appName, userId)
	if err != nil {
		return false, err
	}
	defer selDB.Close()

	var id string
	for selDB.Next() {
		err = selDB.Scan(&id)
		if err != nil {
			return false, err
		}
	}

	if id == "" {
		return false, err
	}
	return true, nil

}

func DuploAppDeleteStatus(appName, status string) error {

	queryString := `UPDATE app `
	if status == "New" {
		queryString = queryString + ` SET app.status = 'Terminated' where app.name = ` + `"` + appName + `"`
	} else {
		queryString = queryString + ` INNER JOIN  
	app_release  
	ON app_release.app_id = app.name  
	SET app_release.status = 'inactive', app.status = 'Terminated' where app.name =` + `"` + appName + `"`
	}

	statement, err := database.Db.Prepare(queryString)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec()
	if err != nil {
		return err
	}

	return nil
}

func DuploDeployStatus(appName string) error {

	statement, err := database.Db.Prepare("Delete from duplo_deploy_status where app_name = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(appName)
	if err != nil {
		return err
	}
	return nil
}

func UpdateImage(appName, imageName string, internalPort int64) error {

	queryString := `UPDATE app  
	INNER JOIN  
	app_release  
	ON app_release.app_id = app.name  
	SET app.imageName = ?, app_release.image_name = ?,app_release.port = ? where app_release.app_id = ? and app_release.status='active';`

	statement, err := database.Db.Prepare(queryString)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(imageName, imageName, internalPort, appName)
	if err != nil {
		return err
	}
	return nil
}

func ConfigAppChange(appName, NewApp, status string) error {

	queryString := `UPDATE app `
	if status != "New" {
		queryString = queryString + `INNER JOIN  
 	app_release
    ON app_release.app_id = app.name 
    INNER JOIN
    app_deployments 
 	ON app_deployments.appId = app.name`
	}
	queryString = queryString + ` SET app.name = ` + `"` + NewApp + `"`
	if status != "New" {
		queryString = queryString + ` ,app_release.app_id =` + `"` + NewApp + `"` + `,app_deployments.appId =` + `"` + NewApp + `"`
	}
	queryString = queryString + ` where app.name = ?`

	fmt.Println(queryString)

	statement, err := database.Db.Prepare(queryString)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(appName)
	if err != nil {
		return err
	}
	return nil
}

func DeployType(depType int, appName string) error {

	statement, err := database.Db.Prepare("UPDATE app SET deploy_type = ? where name = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(depType, appName)
	if err != nil {
		return err
	}
	return nil

}

func GetAppByAppId(id string) (string, string, string, string, error) {

	selDB, err := database.Db.Query("select name, status, organization_id, sub_org_id from app where id = ?", id)
	if err != nil {
		return "", "", "", "", err
	}
	defer selDB.Close()

	var name, status, orgId, subOrgId string
	for selDB.Next() {
		err = selDB.Scan(&name, &status, &orgId, &subOrgId)
		if err != nil {
			return "", "", "", "", err
		}
	}

	if name == "" {
		return "", "", "", "", fmt.Errorf("App Doesn't Exists")
	}
	return name, status, orgId, subOrgId, nil

}

func GetActiveAppsById(id string) (int, error) {
	TotalActiveApps := 0
	selDB, err := database.Db.Query("SELECT COUNT(*) FROM app where createdBy = ? and status = 'Active'", id)

	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&TotalActiveApps)
		if err != nil {
			return 0, err
		}
	}
	return TotalActiveApps, err
}
func GetTerminatedAppsById(id string) (int, error) {
	TotalTerminatedApps := 0
	selDB, err := database.Db.Query("SELECT COUNT(*) FROM app where createdBy = ? and status = 'Terminated'", id)

	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&TotalTerminatedApps)
		if err != nil {
			return 0, err
		}
	}
	return TotalTerminatedApps, err
}

func GetNewAppsById(id string) (int, error) {
	TotalNewApps := 0
	selDB, err := database.Db.Query("SELECT COUNT(*) FROM app where createdBy = ? and status = 'New'", id)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&TotalNewApps)
		if err != nil {
			return 0, err
		}
	}
	return TotalNewApps, err
}

func GetInActiveAppsById(id string) (int, error) {
	TotalInActiveApps := 0
	selDB, err := database.Db.Query("SELECT COUNT(*) FROM app where createdBy = ? and status = 'Suspended';", id)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&TotalInActiveApps)
		if err != nil {
			return 0, err
		}
	}
	return TotalInActiveApps, err
}

func GetAppsCountById(id string) (int, error) {
	TotalApps := 0
	AllAppsCount := 0
	selDB, err := database.Db.Query(`select COUNT(app.name) from app where (app.status = "Active" or app.status = "Suspended" or app.status = "New") and createdBy = ?`, id)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&TotalApps)
		if err != nil {
			return 0, err
		}
	}
	s3DeploymentCount, err := GetS3DeploymentsCountByUsreId(id)
	if err != nil {
		return 0, nil
	}
	AllAppsCount = TotalApps + s3DeploymentCount
	return AllAppsCount, err
}

func GetAppsByRegion(userId string) ([]*model.RegionAppCount, error) {

	query := `SELECT count(*) , region_code  FROM app_deployments 
	inner join app on app.name = app_deployments.appId where app.createdBy = ? and app.status = "Active" and app_deployments.status = "running"
	group by region_code`

	selDB, err := database.Db.Query(query, userId)
	if err != nil {
		return []*model.RegionAppCount{}, fmt.Errorf(err.Error())
	}
	defer selDB.Close()
	var apps *int
	var regions *string

	var regionByApp []*model.RegionAppCount

	for selDB.Next() {
		err = selDB.Scan(&apps, &regions)
		if err != nil {
			return []*model.RegionAppCount{}, fmt.Errorf(err.Error())
		}

		result := model.RegionAppCount{
			Region: regions,
			Apps:   apps,
		}

		regionByApp = append(regionByApp, &result)
	}

	return regionByApp, nil

}

func EditAppByOrg(app model.EditAppByOrganization) error {
	statement, err := database.Db.Prepare(`UPDATE app
	SET organization_id = ? , sub_org_id = ? , business_unit_id = ? , workload_management_id = ?
	WHERE name = ? `)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(app.OrganizationID, app.SubOrganizationID, app.BusinessUnitID, app.WorkloadManagementID, app.AppName)
	if err != nil {
		return err
	}

	return nil
}

func NifePodLog(slug, appname, containerId string, clientset *kubernetes.Clientset) (string, error) {
	clusterlogs := clientset.CoreV1().Pods(slug).GetLogs(containerId, &apiv1.PodLogOptions{})

	podLogs, err := clusterlogs.Stream(context.TODO())
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}
	str := buf.String()
	return str, nil
}

func CreateAppTemplate(apptem model.ConfigTemplate, config, user_id, envArgs string, volumeSize int) error {
	statement, err := database.Db.Prepare("INSERT INTO app_template (id, name, config, env_args, volume_size ,is_active, created_by, created_at) VALUES (?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, apptem.Name, config, envArgs, volumeSize, true, user_id, time.Now().UTC())
	if err != nil {
		return err
	}
	return nil
}

func UpdateAppVersion(appName string, version int) error {
	statement, err := database.Db.Prepare("UPDATE app SET version = ? WHERE name = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(version, appName)
	if err != nil {
		return err
	}
	return nil
}

func GetAppConfigTemplates(user_id string) ([]*model.ConfigAppTemplates, error) {

	selDB, err := database.Db.Query("SELECT id, name, config, is_active, created_by, created_at, env_args, volume_size, cpu_limit, memory_limit, cpu_requests, memory_requests FROM app_template where created_by = ? and is_active = 1", user_id)
	if err != nil {
		return []*model.ConfigAppTemplates{}, err
	}
	defer selDB.Close()
	var allTemplates []*model.ConfigAppTemplates
	for selDB.Next() {
		var temp model.ConfigAppTemplates

		var conf []uint8
		var config model.AppConfig

		err = selDB.Scan(&temp.ID, &temp.Name, &conf, &temp.IsActive, &temp.CreatedBy, &temp.CreatedAt, &temp.EnvArgs, &temp.VolumeSize, &temp.CPULimit, &temp.MemoryLimit, &temp.CPURequests, &temp.MemoryRequests)
		if err != nil {
			return []*model.ConfigAppTemplates{}, err
		}

		json.Unmarshal([]byte(string(conf)), &config)
		temp.Config = &config

		rountingPolicy, err := helper.GetRoutingPolicy(config.Definition)
		if err != nil {
		}
		temp.RoutingPolicy = &rountingPolicy
		allTemplates = append(allTemplates, &temp)
	}
	return allTemplates, nil
}

func GetConfigTemplatesByName(id string) (*model.ConfigAppTemplates, error) {

	selDB, err := database.Db.Query("SELECT name, config, is_active, created_by, created_at FROM app_template where id = ? ", id)
	if err != nil {
		return &model.ConfigAppTemplates{}, err
	}
	defer selDB.Close()
	var temp model.ConfigAppTemplates
	for selDB.Next() {

		var conf []uint8
		var config model.AppConfig

		err = selDB.Scan(&temp.Name, &conf, &temp.IsActive, &temp.CreatedBy, &temp.CreatedAt)
		if err != nil {
			return &model.ConfigAppTemplates{}, err
		}

		json.Unmarshal([]byte(string(conf)), &config)
		temp.Config = &config
	}

	return &temp, nil
}

func UpdateConfigTemp(id, name, envArgs, cpu_limit, memory_limit, cpu_requests, memory_requests string, appConfig *model.AppConfig, volumeSize int) error {
	config, _ := json.Marshal(appConfig)
	fmt.Println(appConfig)
	statement, err := database.Db.Prepare("Update app_template set name = ?, config = ?, volume_size = ?, env_args = ? , cpu_limit = ?, memory_limit = ? , cpu_requests = ? , memory_requests = ? where id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(name, config, volumeSize, envArgs, cpu_limit, memory_limit, cpu_requests, memory_requests, id)
	if err != nil {
		return err
	}

	return nil
}

func DeleteConfigTemp(id string) error {
	statement, err := database.Db.Prepare("Update app_template set is_active = ?  where id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(false, id)
	if err != nil {
		return err
	}

	return nil
}

func GetAppsCountByOrg(userId, roleId string) ([]*model.AppsOrgsCount, error) {

	query := `select COUNT(app.name), org.name from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	INNER JOIN app ON app.organization_id = org.id
	where organization_users.user_id =? and (app.status = "Active" or app.status = "Suspended" or app.status = "New") and organization_users.role_id = ? group by org.name`

	selDB, err := database.Db.Query(query, userId, roleId)
	if err != nil {
		return []*model.AppsOrgsCount{}, err
	}
	defer selDB.Close()

	var RegionCount []*model.AppsOrgsCount

	for selDB.Next() {

		var organization string
		var appsCountByOrg int
		err = selDB.Scan(&appsCountByOrg, &organization)
		if err != nil {
			return []*model.AppsOrgsCount{}, err
		}

		newApps, err := GetAppsCountByStatus(userId, "New", roleId, organization)
		if err != nil {
			return []*model.AppsOrgsCount{}, err
		}
		activeApps, err := GetAppsCountByStatus(userId, "Active", roleId, organization)
		if err != nil {
			return []*model.AppsOrgsCount{}, err
		}
		inActiveApps, err := GetAppsCountByStatus(userId, "Suspended", roleId, organization)
		if err != nil {
			return []*model.AppsOrgsCount{}, err
		}

		result := model.AppsOrgsCount{
			Organization: &organization,
			AppsCount:    &appsCountByOrg,
			NewApp:       &newApps,
			ActiveApp:    &activeApps,
			InActiveApp:  &inActiveApps,
		}

		RegionCount = append(RegionCount, &result)
	}

	return RegionCount, nil
}

func GetAppsCountByStatus(userId, status, roleId, orgName string) (int, error) {
	TotalApps := 0

	query := `select COUNT(app.name) from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	INNER JOIN app ON app.organization_id = org.id
	where organization_users.user_id = ? and app.status = ? and organization_users.role_id = ? and org.name = ?  group by org.name;`

	selDB, err := database.Db.Query(query, userId, status, roleId, orgName)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&TotalApps)
		if err != nil {
			return 0, err
		}
	}

	return TotalApps, nil
}

func GetOrgByUserId(userId, roleId string) ([]*model.AppsOrgsSubCount, error) {

	query := ` select org.id, org.name from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	where (organization_users.user_id = ?  and organization_users.role_id = ?) and org.is_deleted = 0 and org.parent_orgid = "" ;`

	selDB, err := database.Db.Query(query, userId, roleId)
	if err != nil {
		return []*model.AppsOrgsSubCount{}, err
	}
	defer selDB.Close()

	var Count []*model.AppsOrgsSubCount

	for selDB.Next() {
		var orgId string
		var name string
		var newApp int
		var activeApp int
		var inActiveApp int
		var totalApp int
		err = selDB.Scan(&orgId, &name)
		if err != nil {
			return []*model.AppsOrgsSubCount{}, err
		}

		nameSubOrg, err := GetSubOrgCountByOrg(orgId, userId)
		if err != nil {
			return []*model.AppsOrgsSubCount{}, err
		}

		if nameSubOrg == nil {
			newApp, err = GetAppsCountByOrgname(userId, "New", roleId, name)
			if err != nil {
				return []*model.AppsOrgsSubCount{}, err
			}
			activeApp, err = GetAppsCountByOrgname(userId, "Active", roleId, name)
			if err != nil {
				return []*model.AppsOrgsSubCount{}, err
			}
			inActiveApp, err = GetAppsCountByOrgname(userId, "Suspended", roleId, name)
			if err != nil {
				return []*model.AppsOrgsSubCount{}, err
			}
			totalApp = newApp + activeApp + inActiveApp
		}

		result := model.AppsOrgsSubCount{
			Organization:    &name,
			AppsCount:       &totalApp,
			NewApp:          &newApp,
			ActiveApp:       &activeApp,
			InActiveApp:     &inActiveApp,
			SubOrganization: nameSubOrg,
		}

		Count = append(Count, &result)
	}

	return Count, nil
}

func GetSubOrgCountByOrg(orgId, userId string) ([]*model.SubOrgCount, error) {

	query := `SELECT id, parent_orgid, name FROM organization where parent_orgid = ? and is_deleted = 0`

	selDB, err := database.Db.Query(query, orgId)
	if err != nil {
		return []*model.SubOrgCount{}, err
	}
	defer selDB.Close()

	var RegionCount []*model.SubOrgCount

	for selDB.Next() {
		var subOrgId string
		var parentOrgId string
		var name string
		err = selDB.Scan(&subOrgId, &parentOrgId, &name)
		if err != nil {
			return []*model.SubOrgCount{}, err
		}

		nameBusinessUnit, err := GetBusinessUnitCountByOrg(parentOrgId, subOrgId, userId)
		if err != nil {
			return []*model.SubOrgCount{}, err
		}

		result := model.SubOrgCount{
			SubOrganizationCount: &name,
			BusinessUnit:         nameBusinessUnit,
		}

		RegionCount = append(RegionCount, &result)
	}

	return RegionCount, nil
}

func GetBusinessUnitCountByOrg(orgId, parentOrgId, userId string) ([]*model.BusinessUnitCount, error) {

	query := `SELECT id, name FROM business_unit where (org_id = ? and sub_org_id = ?) and  is_active = 1`

	selDB, err := database.Db.Query(query, orgId, parentOrgId)
	if err != nil {
		return []*model.BusinessUnitCount{}, err
	}
	defer selDB.Close()

	var RegionCount []*model.BusinessUnitCount

	for selDB.Next() {
		var id string
		var name string
		err = selDB.Scan(&id, &name)
		if err != nil {
			return []*model.BusinessUnitCount{}, err
		}

		allAppsCount, err := GetAppsCountByOrgNew(id, userId)
		if err != nil {
			return []*model.BusinessUnitCount{}, err
		}

		result := model.BusinessUnitCount{
			BusinessUnitCount: &name,
			AppsCount:         &allAppsCount,
		}

		RegionCount = append(RegionCount, &result)
	}

	return RegionCount, nil
}

func GetAppsCountByOrgNew(businessUnitId, userId string) (model.AppsCountbyBusinessUnit, error) {

	query := `select COUNT(app.name), business_unit.name from business_unit INNER JOIN app ON app.business_unit_id = business_unit.id
	where (app.createdBy = ? and app.business_unit_id = ?) and (app.status = "Active" or app.status = "Suspended" or app.status = "New") group by business_unit.name;`

	selDB, err := database.Db.Query(query, userId, businessUnitId)
	if err != nil {
		return model.AppsCountbyBusinessUnit{}, err
	}
	defer selDB.Close()

	var result model.AppsCountbyBusinessUnit

	for selDB.Next() {

		var businessUnitName string
		var appsCountByBusinessUnit int
		err = selDB.Scan(&appsCountByBusinessUnit, &businessUnitName)
		if err != nil {
			return model.AppsCountbyBusinessUnit{}, err
		}

		newApps, err := GetAppCountByStatus(userId, "New", businessUnitName)
		if err != nil {
			return model.AppsCountbyBusinessUnit{}, err
		}
		activeApps, err := GetAppCountByStatus(userId, "Active", businessUnitName)
		if err != nil {
			return model.AppsCountbyBusinessUnit{}, err
		}
		inActiveApps, err := GetAppCountByStatus(userId, "Suspended", businessUnitName)
		if err != nil {
			return model.AppsCountbyBusinessUnit{}, err
		}

		result = model.AppsCountbyBusinessUnit{
			AppsCount:   &appsCountByBusinessUnit,
			NewApp:      &newApps,
			ActiveApp:   &activeApps,
			InActiveApp: &inActiveApps,
		}

	}

	return result, nil
}

func GetAppCountByStatus(userId, status, businessUnitName string) (int, error) {
	TotalApps := 0

	query := `select COUNT(app.name) from business_unit INNER JOIN app ON app.business_unit_id = business_unit.id
	where (app.createdBy = ? and app.status = ?) and business_unit.name = ? group by business_unit.name`

	selDB, err := database.Db.Query(query, userId, status, businessUnitName)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&TotalApps)
		if err != nil {
			return 0, err
		}
	}

	return TotalApps, nil
}

func GetAppsCountByOrgname(userId, status, roleId, orgName string) (int, error) {
	TotalApps := 0

	query := `select COUNT(app.name) from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	INNER JOIN app ON app.organization_id = org.id
	where organization_users.user_id = ? and app.status = ? and organization_users.role_id = ? and org.name = ?  group by org.name;`

	selDB, err := database.Db.Query(query, userId, status, roleId, orgName)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&TotalApps)
		if err != nil {
			return 0, err
		}
	}

	return TotalApps, nil
}

func GetAppByRegion(userId string) ([]*model.RegionAppCount, error) {

	query := `SELECT count(*) , region_code  FROM app_deployments 
	inner join app on app.name = app_deployments.appId where app.createdBy = ? and app.status = "Active" and app_deployments.status = "running"
	group by region_code`

	selDB, err := database.Db.Query(query, userId)
	if err != nil {
		return []*model.RegionAppCount{}, fmt.Errorf(err.Error())
	}
	defer selDB.Close()
	var apps *int
	var regions *string

	var regionByApp []*model.RegionAppCount

	for selDB.Next() {
		err = selDB.Scan(&apps, &regions)
		if err != nil {
			return []*model.RegionAppCount{}, fmt.Errorf(err.Error())
		}

		result := model.RegionAppCount{
			Region: regions,
			Apps:   apps,
		}

		regionByApp = append(regionByApp, &result)
	}

	return regionByApp, nil

}

func GetLogoByUserId(userId string) (string, error) {
	statement, err := database.Db.Prepare("SELECT company.logo FROM company INNER JOIN user on user.company_name = company.company_name where user.id = ?;")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(userId)

	var logoUrl string
	defer statement.Close()
	err = row.Scan(&logoUrl)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return "", err
	}

	return logoUrl, nil
}

func AllAppsUnderWorkload(user_id string, appName string, orgId string) (*model.Nodes, error) {
	var nodes model.Nodes

	Querystring := `
	SELECT 
		ap.id,
		ap.name,
        IFNULL( ar.version, "") as version,       
		ap.deployed,
		ap.appUrl,
		ap.hostname,			
		ap.status,
		ap.regions,
		ap.config,
		ap.createdAt,
		ap.workload_management_id,
		ap.organization_id,
		ap.sub_org_id,
		ap.business_unit_id,
		ap.imageName,
        org.slug
	FROM
		app ap
	   JOIN organization org on ap.organization_id = org.id
       LEFT JOIN        
		 app_release ar on (ar.app_id = ap.name and ar.status = 'active')
		 Where (ap.status != 'Terminated' and ap.createdBy = ?) and (ap.organization_id = ? and ap.name LIKE ?) and ap.name != ?
		`

	Querystring = Querystring + ` order by createdAt desc`

	selDB, err := database.Db.Query(Querystring, user_id, orgId, "%"+appName+"%", appName)
	if err != nil {
		return &nodes, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var app model.App

		var organization model.Organization
		var appVersion string
		// var createdAt time.Time

		var domain []uint8
		var domainModel *model.Domains

		//var createdAt string

		// var cr []uint8
		// var currentRelease model.Release

		var region []uint8
		var regions []*model.Region

		var config []uint8
		var configModel *model.AppConfig
		// fmt.Println(currentRelease)

		err = selDB.Scan(&app.ID, &app.Name, &appVersion, &app.Deployed, &app.AppURL, &app.Hostname, &app.Status, &region, &config, &app.CreatedAt, &app.WorkloadManagementID, &app.OrganizationID, &app.SubOrganizationID, &app.BusinessUnitID, &app.ImageName, &organization.Slug)
		if err != nil {
			fmt.Println(err)
			return &nodes, err
		}
		json.Unmarshal([]byte(string(domain)), &domainModel)
		organization.Domains = domainModel

		app.Organization = &organization

		// currentRelease.CreatedAt = &createdAt

		app.Releases, err = getAppRelease(app.Name)
		if err != nil {
			log.Println(err)
		}

		app.Version, _ = strconv.Atoi(appVersion)
		if err != nil {
			log.Println(err)
		}

		json.Unmarshal([]byte(string(region)), &regions)
		app.Regions = regions

		json.Unmarshal([]byte(string(config)), &configModel)
		app.Config = configModel

		var configBuiltin string

		for _, i := range app.Releases.Nodes {
			if *i.Status == "active" {
				defaultBuiltin := "Image"
				app.BuiltinType = &defaultBuiltin

				if configModel.Build.Builtin == nil {
					configBuiltin = ""
				} else if *configModel.Build.Builtin == "" {
					configBuiltin = ""
				}

				if *i.ArchiveURL != "" && configBuiltin != "" {
					*app.BuiltinType = "Built-In"

				} else if *i.ArchiveURL != "" && configBuiltin == "" {
					*app.BuiltinType = "Docker File"

				} else {
					app.BuiltinType = &defaultBuiltin
				}

			}

		}
		if app.Status == "New" {
			defaultBuiltin1 := "-"
			app.BuiltinType = &defaultBuiltin1
		}

		orgName, err := GetOrgNameById(*app.OrganizationID)
		if err != nil {
			log.Println(err)
		}
		subOrgName, err := GetOrgNameById(*app.SubOrganizationID)
		if err != nil {
			log.Println(err)
		}
		businessUnitName, err := GetBusinessUnitById(*app.BusinessUnitID)
		if err != nil {
			log.Println(err)
		}

		app.OrganizationName = &orgName
		app.SubOrganizationName = &subOrgName
		app.BusinessUnitName = &businessUnitName

		nodes.Nodes = append(nodes.Nodes, &app)
	}

	// nodes.Releases, err = getAppRelease(appId)

	return &nodes, nil
}

func GetAppStatusByAppName(appName string) (string, error) {
	selDB, err := database.Db.Query("SELECT status FROM app where name = ?", appName)
	if err != nil {
		return "", err
	}
	defer selDB.Close()
	var status string
	for selDB.Next() {
		err = selDB.Scan(&status)
		if err != nil {
			return "", err
		}
	}
	return status, nil
}

func DeleteAppRecord(appName, table string) error {
	var statement *sql.Stmt

	if table == "app" {
		statement, _ = database.Db.Prepare(`DELETE FROM app WHERE name = ?`)
	} else if table == "deployment" {
		statement, _ = database.Db.Prepare(`DELETE FROM app_deployments WHERE appId = ?`)

	} else if table == "release" {
		statement, _ = database.Db.Prepare(`DELETE FROM app_release WHERE app_id = ?`)
	}

	_, err := statement.Exec(appName)
	if err != nil {
		return err
	}

	return nil
}

func AllAppsWithWorkload(user_id, workloadId string) (*model.Nodes, error) {
	var nodes model.Nodes
	var res string

	Querystring := `
	SELECT 
		ap.id,
		ap.name,
        IFNULL( ar.version, "") as version,       
		ap.deployed,
		ap.appUrl,
		ap.hostname,			
		ap.status,
		ap.regions,
		ap.config,
		ap.createdAt,
		ap.workload_management_id,
		ap.organization_id,
		ap.sub_org_id,
		ap.business_unit_id,
		ap.imageName,
        org.slug
	FROM
		app ap
	   JOIN organization org on ap.organization_id = org.id
       LEFT JOIN        
		 app_release ar on (ar.app_id = ap.name and ar.status = 'active') `

	if workloadId != "" {
		Querystring = Querystring + `Where ap.status != 'Terminated' and ap.workload_management_id = ? `
		res = workloadId
	} else {
		Querystring = Querystring + `Where ap.status != 'Terminated' and ap.createdBy = ? `
		res = user_id
	}

	Querystring = Querystring + ` order by createdAt desc`

	selDB, err := database.Db.Query(Querystring, res)
	if err != nil {
		return &nodes, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var app model.App

		var organization model.Organization
		var appVersion string
		// var createdAt time.Time

		var domain []uint8
		var domainModel *model.Domains

		//var createdAt string

		// var cr []uint8
		// var currentRelease model.Release

		var region []uint8
		var regions []*model.Region

		var config []uint8
		var configModel *model.AppConfig
		// fmt.Println(currentRelease)

		err = selDB.Scan(&app.ID, &app.Name, &appVersion, &app.Deployed, &app.AppURL, &app.Hostname, &app.Status, &region, &config, &app.CreatedAt, &app.WorkloadManagementID, &app.OrganizationID, &app.SubOrganizationID, &app.BusinessUnitID, &app.ImageName, &organization.Slug)
		if err != nil {
			fmt.Println(err)
			return &nodes, err
		}
		json.Unmarshal([]byte(string(domain)), &domainModel)
		organization.Domains = domainModel

		app.Organization = &organization

		// currentRelease.CreatedAt = &createdAt

		app.Releases, err = getAppRelease(app.Name)
		if err != nil {
			log.Println(err)
		}

		app.Version, _ = strconv.Atoi(appVersion)
		if err != nil {
			log.Println(err)
		}

		json.Unmarshal([]byte(string(region)), &regions)
		app.Regions = regions

		json.Unmarshal([]byte(string(config)), &configModel)
		app.Config = configModel

		if app.Status == "New" || configModel.Build == nil {
			defaultBuiltin1 := "-"
			app.BuiltinType = &defaultBuiltin1
		} else {
			app.BuiltinType = configModel.Build.Builder
		}

		orgName, err := GetOrgNameById(*app.OrganizationID)
		if err != nil {
			log.Println(err)
		}
		subOrgName, err := GetOrgNameById(*app.SubOrganizationID)
		if err != nil {
			log.Println(err)
		}
		businessUnitName, err := GetBusinessUnitById(*app.BusinessUnitID)
		if err != nil {
			log.Println(err)
		}
		var workloadName string
		var workloadEndPoint string

		if *app.WorkloadManagementID != "" {
			workloadName, workloadEndPoint, err = GetWorkloadNameById(*app.WorkloadManagementID)
			if err != nil {
				log.Println(err)
			}
		} else {
			workloadName = ""
			workloadEndPoint = ""
		}

		userDetAct, err := GetById(user_id)
		if err != nil {
			return nil, err
		}

		app.OrganizationName = &orgName
		app.SubOrganizationName = &subOrgName
		app.BusinessUnitName = &businessUnitName
		app.WorkloadManagementName = &workloadName
		app.WorkloadManagementEndPoint = &workloadEndPoint
		app.UserDetails = &userDetAct

		nodes.Nodes = append(nodes.Nodes, &app)
	}

	// nodes.Releases, err = getAppRelease(appId)

	return &nodes, nil
}

func CreateNifeToml(input *model.CreateAppToml) {

	appId := decode.EnPwdCode(*input.OrganizationID)

	timestamp := time.Now().Format("2006-01-02T15:04:05-07:00")
	fmt.Println(timestamp)

	if input.CPULimit == nil {
		CPULimit := "1"
		MemoryLimit := "256"
		CPURequests := "0.5"
		MemoryRequests := "128"
		input.CPULimit = &CPULimit
		input.MemoryLimit = &MemoryLimit
		input.CPURequests = &CPURequests
		input.MemoryRequests = &MemoryRequests
	}
	var builtins map[string]interface{}
	if input.Builtin != nil && *input.Builtin != "" {
		builtins = map[string]interface{}{
			"builtin": *input.Builtin,
		}
	} else {
		builtins = map[string]interface{}{
			"image": *input.Image,
		}
	}
	// builtin
	// Create a map to represent the TOML structure
	configMap := map[string]interface{}{
		"app":          *input.AppName,
		"github":       "",
		"id":           appId,
		"build":        builtins,
		"kill_signal":  "SIGINT",
		"kill_timeout": 5,
		"env":          map[string]interface{}{},
		"experimental": map[string]interface{}{
			"allowed_public_ports": []string{},
			"auto_rollback":        true,
		},
		"services": []interface{}{
			map[string]interface{}{
				"external_port":  *input.ExternalPort,
				"http_checks":    []string{},
				"internal_port":  *input.InternalPort,
				"protocol":       "tcp",
				"routing_policy": *input.RoutingPolicy,
				"script_checks":  []string{},
				"concurrency": map[string]interface{}{
					"hard_limit": 25,
					"soft_limit": 20,
					"type":       "connections",
				},
				"limits": map[string]interface{}{
					"cpu":    *input.CPULimit,
					"memory": *input.MemoryLimit,
				},
				"ports": []interface{}{
					map[string]interface{}{
						"handlers": []string{"http"},
						"port":     80,
					},
					map[string]interface{}{
						"handlers": []string{"tls", "http"},
						"port":     443,
					},
				},
				"requests": map[string]interface{}{
					"cpu":    *input.CPURequests,
					"memory": *input.MemoryRequests,
				},
				"tcp_checks": []interface{}{
					map[string]interface{}{
						"grace_period":  "1s",
						"interval":      "15s",
						"restart_limit": 6,
						"timeout":       "2s",
					},
				},
			},
		},
	}

	configTree, err := toml.TreeFromMap(configMap)
	if err != nil {
		fmt.Println("Error creating TOML tree:", err)
	}

	configString, err := configTree.ToTomlString()
	if err != nil {
		fmt.Println("Error encoding TOML:", err)
	}

	file, err := os.Create(*input.AppName + ".toml")
	if err != nil {
		fmt.Println("Error creating nife.toml file:", err)
	}
	defer file.Close()

	_, err = file.WriteString(configString)
	if err != nil {
		fmt.Println("Error writing to nife.toml file:", err)
	}
	fmt.Println(configString)
}

func GetUserIdByAppName(name string) (string, error) {
	var userId string
	selDB, err := database.Db.Query("SELECT createdBy FROM app where name = ?;", name)
	if err != nil {
		return "", err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&userId)
		if err != nil {
			return "", err
		}
	}
	return userId, err
}
