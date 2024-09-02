package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func MoveWorkload(newOrg, oldOrg, userId string) error {

	statement, err := database.Db.Prepare("update workload_management set organization_id = ? where organization_id = ? and user_id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(newOrg, oldOrg, userId)
	if err != nil {
		return err
	}
	return nil

}

func MoveBusinessUnit(newOrg, buId, userId string) error {

	statement, err := database.Db.Prepare("update business_unit set org_id = ? where id = ? and created_by = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(newOrg, buId, userId)
	if err != nil {
		return err
	}
	return nil

}

func MoveSubOrg(newOrg, oldOrg string) error {

	statement, err := database.Db.Prepare("update organization set parent_orgid = ? where id = ? ")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(newOrg, oldOrg)
	if err != nil {
		return err
	}
	return nil

}

func MigrateApps(newOrg, appName string) error {

	statement, err := database.Db.Prepare("update app set organization_id = ? where name = ? ")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(newOrg, appName)
	if err != nil {
		return err
	}
	return nil

}

func GetAppOnlyInOrganization(user_id string, userRegion string, orgSlug string) (*model.Nodes, error) {
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
		Querystring = Querystring + `Where (ap.status != 'Terminated' and ap.createdBy = ?) and ap.sub_org_id = "" `
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

		var domain []uint8
		var domainModel *model.Domains

		var region []uint8
		var regions []*model.Region

		var config []uint8
		var configModel *model.AppConfig

		err = selDB.Scan(&app.ID, &app.Name, &appVersion, &app.Deployed, &app.AppURL, &app.Hostname, &app.Status, &region, &config, &app.CreatedAt, &app.WorkloadManagementID, &app.OrganizationID, &app.SubOrganizationID, &app.BusinessUnitID, &app.ImageName, &organization.Slug)
		if err != nil {
			fmt.Println(err)
			return &nodes, err
		}
		json.Unmarshal([]byte(string(domain)), &domainModel)
		organization.Domains = domainModel

		app.Organization = &organization

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

		app.OrganizationName = &orgName
		app.SubOrganizationName = &subOrgName
		app.BusinessUnitName = &businessUnitName
		app.WorkloadManagementName = &workloadName
		app.WorkloadManagementEndPoint = &workloadEndPoint

		nodes.Nodes = append(nodes.Nodes, &app)
	}

	return &nodes, nil
}
