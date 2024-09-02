package service

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func CreateworkloadManagement(wl model.WorkloadManagement, userId string) error {
	statement, err := database.Db.Prepare("INSERT INTO workload_management (id, environment_name, environment_endpoint, organization_id, user_id, created_at, is_active) VALUES (?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, wl.EnvironmentName, wl.EnvironmentEndpoint, wl.OrganizationID, userId, time.Now(), true)
	if err != nil {
		return err
	}
	return nil
}

func DeleteWorkLoadManagement(workloadId, userId string) (string, error) {
	statement, err := database.Db.Prepare(`UPDATE workload_management SET is_active = 0  WHERE id = ? and user_id = ?`)
	if err != nil {
		return "", err
	}
	defer statement.Close()
	_, err = statement.Exec(workloadId, userId)
	if err != nil {
		return "", err
	}
	return "", err
}

func GetWorkLoadManagementByUser(userId string) ([]*model.WorkloadManagementList, error) {
	

	query := `SELECT id, environment_name, environment_endpoint, organization_id, sub_organization_id, business_unit_id, user_id, created_at FROM workload_management where user_id = ? and is_active = 1;`

	selDB, err := database.Db.Query(query, userId)
	if err != nil {
		return []*model.WorkloadManagementList{}, err
	}
	defer selDB.Close()

	var RegionCount []*model.WorkloadManagementList

	for selDB.Next() {

		var wl model.WorkloadManagementList
		err = selDB.Scan(&wl.ID, &wl.EnvironmentName, &wl.EnvironmentEndpoint, &wl.OrganizationID, &wl.SubOrganizationID, &wl.BusinessUnitID, &wl.UserID, &wl.CreatedAt)
		if err != nil {
			return []*model.WorkloadManagementList{}, err
		}

		orgName, err := GetOrgNameById(*wl.OrganizationID)
		if err != nil {
			return nil, err
		}
		wl.OrganizationName = &orgName

		wlApps, err := AllAppsByWlId(*wl.ID)
		if err != nil {
			return nil, err
		}
		wl.Apps = wlApps

		result := model.WorkloadManagementList{
			ID:                  wl.ID,
			EnvironmentName:     wl.EnvironmentName,
			EnvironmentEndpoint: wl.EnvironmentEndpoint,
			OrganizationID:      wl.OrganizationID,
			OrganizationName:    wl.OrganizationName,
			SubOrganizationID:   wl.SubOrganizationID,
			BusinessUnitID:      wl.BusinessUnitID,
			UserID:              wl.UserID,
			CreatedAt:           wl.CreatedAt,
			Apps:                wl.Apps,
		}

		RegionCount = append(RegionCount, &result)
	}

	return RegionCount, nil
}

func GetWorkLoadManagementById(workloadId, userId string) (*model.WorkloadManagementList, error) {

	query := `SELECT id, environment_name, environment_endpoint, organization_id, sub_organization_id, business_unit_id, user_id, created_at FROM workload_management where (user_id = ? and id = ? ) and is_active = 1;`

	selDB, err := database.Db.Query(query, userId, workloadId)
	if err != nil {
		return &model.WorkloadManagementList{}, err
	}
	defer selDB.Close()
	var wl model.WorkloadManagementList

	for selDB.Next() {
		err = selDB.Scan(&wl.ID, &wl.EnvironmentName, &wl.EnvironmentEndpoint, &wl.OrganizationID, &wl.SubOrganizationID, &wl.BusinessUnitID, &wl.UserID, &wl.CreatedAt)
		if err != nil {
			return &model.WorkloadManagementList{}, err
		}
	}
	appDet, err := AllAppsByWlId(workloadId)
	wl.Apps = appDet

	return &wl, nil
}

func GetWorkLoadRegionById(workloadId, userId string) ([]*string, error) {

	query := `SELECT region_code FROM workload_management_regions where workload_id = ? and user_id = ?;`

	selDB, err := database.Db.Query(query, workloadId, userId)
	if err != nil {
		return []*string{}, err
	}
	defer selDB.Close()
	var regionCodeArr []*string

	for selDB.Next() {
		var regionCode string
		err = selDB.Scan(&regionCode)
		if err != nil {
			return []*string{}, err
		}

		regionCodeArr = append(regionCodeArr, &regionCode)
	}

	return regionCodeArr, nil
}

func GetWorkLoadManagementByName(workloadName, userId string) (*model.WorkloadManagementList, error) {

	query := `SELECT id, environment_name, environment_endpoint, organization_id, sub_organization_id, business_unit_id, user_id, created_at FROM workload_management where (user_id = ? and environment_name = ? ) and is_active = 1;`

	selDB, err := database.Db.Query(query, userId, workloadName)
	if err != nil {
		return &model.WorkloadManagementList{}, err
	}
	defer selDB.Close()
	var wl model.WorkloadManagementList

	for selDB.Next() {
		err = selDB.Scan(&wl.ID, &wl.EnvironmentName, &wl.EnvironmentEndpoint, &wl.OrganizationID, &wl.SubOrganizationID, &wl.BusinessUnitID, &wl.UserID, &wl.CreatedAt)
		if err != nil {
			return &model.WorkloadManagementList{}, err
		}
	}

	return &wl, nil
}

func GetWorkLoadManagementByOrgIdSubOrgBusinessU(userId, orgId, subOrgId, businessUnitId string) ([]*model.WorkloadManagementList, error) {
	query := `SELECT id, environment_name, environment_endpoint, organization_id, user_id, created_at FROM workload_management where (user_id = ? and organization_id = ?) and (sub_organization_id = ? and business_unit_id = ?) and is_active = 1;`

	selDB, err := database.Db.Query(query, userId, orgId, subOrgId, businessUnitId)
	if err != nil {
		return []*model.WorkloadManagementList{}, err
	}
	defer selDB.Close()

	var RegionCount []*model.WorkloadManagementList

	for selDB.Next() {

		var wl model.WorkloadManagementList
		err = selDB.Scan(&wl.ID, &wl.EnvironmentName, &wl.EnvironmentEndpoint, &wl.OrganizationID, &wl.UserID, &wl.CreatedAt)
		if err != nil {
			return []*model.WorkloadManagementList{}, err
		}

		appDet, err := AllAppsByWlId(*wl.ID)
		if err != nil {
			return []*model.WorkloadManagementList{}, err
		}
		orgName, err := GetOrgNameById(orgId)
		if err != nil {
			return nil, err
		}

		result := model.WorkloadManagementList{
			ID:                  wl.ID,
			EnvironmentName:     wl.EnvironmentName,
			EnvironmentEndpoint: wl.EnvironmentEndpoint,
			OrganizationID:      wl.OrganizationID,
			UserID:              wl.UserID,
			CreatedAt:           wl.CreatedAt,
			OrganizationName: 	 &orgName,
			Apps:                appDet,
		}

		RegionCount = append(RegionCount, &result)
	}

	return RegionCount, nil
}

func AllAppsByWlId(wlId string) (*model.Nodes, error) {
	var nodes model.Nodes

	Querystring := `
	SELECT	ap.id,
		ap.name,
		ap.deployed,
		ap.hostname,
		ap.status,
		ap.version,
		ap.appUrl,
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
		ap.business_unit_id,
		org.id,
		org.name,
		org.domains,
		org.slug,
		org.type
	FROM
		app ap JOIN organization org ON ap.organization_id = org.id
	WHERE
		ap.workload_management_id = ? and ap.status != "Terminated";`

	selDB, err := database.Db.Query(Querystring, wlId)
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
		var ip []uint8
		var ipAddress model.IPAddresses

		var region []uint8
		var regions []*model.Region

		var config []uint8
		var configModel *model.AppConfig
		// fmt.Println(currentRelease)

		var backupRegion []uint8
		var backupRegions []*model.Region

		err = selDB.Scan(&app.ID, &app.Name, &app.Deployed, &app.Hostname, &app.Status, &app.Version, &app.AppURL, &ip, &config, &region, &backupRegion, &app.EnvArgs, &app.ImageName, &app.Port, &app.SecretRegistryID, &app.InstanceID, &app.DockerID, &app.HostID, &app.TenantID, &app.DeployType, &app.WorkloadManagementID, &app.OrganizationID, &app.SubOrganizationID, &app.BusinessUnitID, &organization.ID, &organization.Name, &domain, &organization.Slug, &organization.Type)
		if err != nil {
			return &nodes, err
		}
		json.Unmarshal([]byte(string(domain)), &domainModel)
		organization.Domains = domainModel

		app.Organization = &organization

		// currentRelease.CreatedAt = &createdAt

		app.Releases, err = getAppRelease(app.Name)
		if err != nil {
			return nil, err
		}

		app.Version, _ = strconv.Atoi(appVersion)
		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(string(region)), &regions)
		app.Regions = regions

		json.Unmarshal([]byte(string(config)), &configModel)
		app.Config = configModel

		json.Unmarshal([]byte(string(ip)), &ipAddress)
		app.IPAddresses = &ipAddress

		json.Unmarshal([]byte(string(backupRegion)), &backupRegions)
		app.BackupRegions = backupRegions

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
			return nil, err
		}
		subOrgName, err := GetOrgNameById(*app.SubOrganizationID)
		if err != nil {
			return nil, err
		}
		businessUnitName, err := GetBusinessUnitById(*app.BusinessUnitID)
		if err != nil {
			return nil, err
		}

		app.OrganizationName = &orgName
		app.SubOrganizationName = &subOrgName
		app.BusinessUnitName = &businessUnitName

		nodes.Nodes = append(nodes.Nodes, &app)
	}

	// nodes.Releases, err = getAppRelease(appId)

	return &nodes, nil
}

func AddWorkloadRegion(regionCode, workLoadId, userId string) error {
	statement, err := database.Db.Prepare("INSERT INTO workload_management_regions (id, workload_id, region_code, user_id) VALUES (?,?,?,?)")
	if err != nil {
		return err
	}
	defer statement.Close()
	id := uuid.NewString()
	_, err = statement.Exec(id, workLoadId, regionCode, userId)
	if err != nil {
		return err
	}
	return nil
}

func DeleteWorkloadRegion(workLoadId, userId, regionCode string) error {
	statement, err := database.Db.Prepare("DELETE FROM workload_management_regions WHERE (workload_id = ? and user_id = ?) and region_code = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(workLoadId, userId, regionCode)
	if err != nil {
		return err
	}
	return nil
}
