package clusterInfo

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	"github.com/nifetency/nife.io/pkg/helper"
)

func GetClusterDetailsByOrgId(organizationId, input, requestType, userId string) (*ClusterDetail, error) {
	var clusterDetail ClusterDetail
	query := ""

	ccheckReg, err := helper.CheckRegionsByRegionCode(input)
	if err != nil {
		return nil, err
	}

	if ccheckReg != "" {

		if requestType == "code" {
			query = `select org.region_code, org.is_default,ci.name,ci.cluster_config_path , ci.ebl_enabled, ci.port ,ci.provided_type, ci.cluster_type,ci.external_base_address,
		ci.external_agent_platform, ci.external_cloud_type, ci.interface,ci.route53_country_code,ci.tenant_id,ci.allocation_tag
				FROM organization_regions org join cluster_info ci 
				on org.region_code = ci.region_code
		where org.organization_id = ? and org.region_code = ?`
		} else {
			query = `select org.region_code, org.is_default,ci.name,ci.cluster_config_path , ci.ebl_enabled, ci.port ,ci.provided_type, ci.cluster_type,ci.external_base_address,
		ci.external_agent_platform, ci.external_cloud_type, ci.interface,ci.route53_country_code,ci.tenant_id,ci.allocation_tag
				FROM organization_regions org join cluster_info ci 
				on org.region_code = ci.region_code
				where org.organization_id = ? and org.is_default = ?`
		}

		if requestType == "Code" {
			input = requestType
		}

		selDB, err := database.Db.Query(query, organizationId, input)
		if err != nil {
			return nil, err
		}
		defer selDB.Close()
		selDB.Next()
		err = selDB.Scan(&clusterDetail.Region_code, &clusterDetail.IsDefault, &clusterDetail.RegionName, &clusterDetail.Cluster_config_path, &clusterDetail.EBL_enabled, &clusterDetail.Port, &clusterDetail.ProviderType, &clusterDetail.ClusterType, &clusterDetail.ExternalBaseAddress,
			&clusterDetail.ExternalAgentPlatform, &clusterDetail.ExternalCloudType, &clusterDetail.Interface, &clusterDetail.Route53CountryCode, &clusterDetail.TenantId, &clusterDetail.AllocationTag)
		if err != nil {
			return nil, err
		}
	} else {
		if requestType == "code" {

			query = `select org.region_code, org.is_default, ci.location_name, ci.cluster_config_url , ci.ebl_enabled, ci.port ,ci.provided_type, ci.cluster_type, ci.interface
			FROM organization_regions org join cluster_info_user ci on org.region_code = ci.region_code
			where (org.organization_id = ? and org.region_code = ? ) and ( ci.is_active = 1 and ci.user_id = ?) `
		} else {
			query = `select org.region_code, org.is_default, ci.location_name, ci.cluster_config_url , ci.ebl_enabled, ci.port ,ci.provided_type, ci.cluster_type, ci.interface
			FROM organization_regions org join cluster_info_user ci on org.region_code = ci.region_code
			where (org.organization_id = ? and org.is_default = ?) and ci.user_id = ?;`
		}

		if requestType == "Code" {
			input = requestType
		}
		selDB, err := database.Db.Query(query, organizationId, input, userId)
		if err != nil {
			return nil, err
		}
		defer selDB.Close()
		selDB.Next()
		err = selDB.Scan(&clusterDetail.Region_code, &clusterDetail.IsDefault, &clusterDetail.RegionName, &clusterDetail.ClusterConfigURL, &clusterDetail.EBL_enabled, &clusterDetail.Port, &clusterDetail.ProviderType, &clusterDetail.ClusterType,
			&clusterDetail.Interface)
		if err != nil {
			return nil, err
		}

	}

	return &clusterDetail, nil
}

func GetClusterDetailsByOrgIdDefault(organizationId, input, userId string) (*ClusterDetail, error) {
	var clusterDetails []*ClusterDetail
	query := `select org.region_code, org.is_default,ci.name,ci.cluster_config_path , ci.ebl_enabled, ci.port ,ci.provided_type, ci.cluster_type,ci.external_base_address,
		ci.external_agent_platform, ci.external_cloud_type, ci.interface,ci.route53_country_code,ci.tenant_id,ci.allocation_tag
				FROM organization_regions org join cluster_info ci 
				on org.region_code = ci.region_code
				where org.organization_id = ? and org.is_default = ?`

	selDB, err := database.Db.Query(query, organizationId, input)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var clusterDetail ClusterDetail
		err := selDB.Scan(&clusterDetail.Region_code, &clusterDetail.IsDefault, &clusterDetail.RegionName, &clusterDetail.Cluster_config_path, &clusterDetail.EBL_enabled, &clusterDetail.Port, &clusterDetail.ProviderType, &clusterDetail.ClusterType, &clusterDetail.ExternalBaseAddress,
			&clusterDetail.ExternalAgentPlatform, &clusterDetail.ExternalCloudType, &clusterDetail.Interface, &clusterDetail.Route53CountryCode, &clusterDetail.TenantId, &clusterDetail.AllocationTag)
		if err != nil {
			return nil, err
		}

		clusterDetails = append(clusterDetails, &clusterDetail)
	}

	userAddedReg, err := GetUserAddedClusterDetailsByOrgIdArr(organizationId, input, userId)
	if err != nil {
		return nil, err
	}

	clusterDetails = append(clusterDetails, userAddedReg...)

	return clusterDetails[0], nil
}

func GetClusterDetailsByOrgIdArr(organizationId, input, userId string) ([]*ClusterDetail, error) {
	var clusterDetails []*ClusterDetail
	query := `select org.region_code, org.is_default,ci.name,ci.cluster_config_path , ci.ebl_enabled, ci.port ,ci.provided_type, ci.cluster_type,ci.external_base_address,
		ci.external_agent_platform, ci.external_cloud_type, ci.interface,ci.route53_country_code,ci.tenant_id,ci.allocation_tag
				FROM organization_regions org join cluster_info ci 
				on org.region_code = ci.region_code
				where org.organization_id = ? and org.is_default = ?`

	selDB, err := database.Db.Query(query, organizationId, input)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var clusterDetail ClusterDetail
		err := selDB.Scan(&clusterDetail.Region_code, &clusterDetail.IsDefault, &clusterDetail.RegionName, &clusterDetail.Cluster_config_path, &clusterDetail.EBL_enabled, &clusterDetail.Port, &clusterDetail.ProviderType, &clusterDetail.ClusterType, &clusterDetail.ExternalBaseAddress,
			&clusterDetail.ExternalAgentPlatform, &clusterDetail.ExternalCloudType, &clusterDetail.Interface, &clusterDetail.Route53CountryCode, &clusterDetail.TenantId, &clusterDetail.AllocationTag)
		if err != nil {
			return nil, err
		}

		clusterDetails = append(clusterDetails, &clusterDetail)
	}

	userAddedReg, err := GetUserAddedClusterDetailsByOrgIdArr(organizationId, input, userId)
	if err != nil {
		return nil, err
	}

	clusterDetails = append(clusterDetails, userAddedReg...)

	return clusterDetails, nil
}

func GetUserAddedClusterDetailsByOrgIdArr(organizationId, input, userId string) ([]*ClusterDetail, error) {
	var clusterDetails []*ClusterDetail
	query := `select org.region_code, org.is_default, ci.location_name, ci.cluster_config_url , ci.ebl_enabled, ci.port ,ci.provided_type, ci.cluster_type, ci.interface
	FROM organization_regions org join cluster_info_user ci 
			   on org.region_code = ci.region_code
			   where (org.organization_id = ? and org.is_default = ?) and (ci.is_active = 1 and ci.user_id = ?);`

	selDB, err := database.Db.Query(query, organizationId, input, userId)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var clusterDetail ClusterDetail
		err := selDB.Scan(&clusterDetail.Region_code, &clusterDetail.IsDefault, &clusterDetail.RegionName, &clusterDetail.ClusterConfigURL, &clusterDetail.EBL_enabled, &clusterDetail.Port, &clusterDetail.ProviderType, &clusterDetail.ClusterType,
			&clusterDetail.Interface)
		if err != nil {
			return nil, err
		}

		clusterDetails = append(clusterDetails, &clusterDetail)
	}

	return clusterDetails, nil
}

func GetActiveClusterByAppId(appId, status, userId string) (appRegions, avalRegions []*model.Region, err error) {
	appRegions, err = GetRegionListByStatusIn(appId, status)
	if err != nil {
		return nil, nil, err
	}
	avalRegions1, err := GetActiveClusterDetails(userId)
	if err != nil {
		return nil, nil, err
	}

	for _, region := range *avalRegions1 {
		avalRegions2 := []*model.Region{
			&model.Region{
				Code:      &region.Region_code,
				Name:      region.RegionName,
				Latitude:  nil,
				Longitude: nil,
			},
		}
		avalRegions = append(avalRegions, avalRegions2...)
	}

	return appRegions, avalRegions, nil

}

func GetRegionListByStatusIn(appId, status string) ([]*model.Region, error) {
	query := `select r.code, r.Name,r.latitude,r.longitude
	from app_deployments ad join regions r on ad.region_code = r.code 
	where ad.appId = ? `

	query, _ = GetStatusINString(query, status)

	selDB, err := database.Db.Query(query, appId)
	if err != nil {
		return nil, err
	}
	return ExecuteGetRegionQuery(query, selDB)
}

func GetClusterDetails(regionCode, userId string) (*model.ClusterDetails, error) {

	ccheckReg, err := helper.CheckRegionsByRegionCode(regionCode)
	if err != nil {
		return nil, err
	}
	var ciDetails model.ClusterDetails

	if ccheckReg != "" {

		query := `select ci.name,ci.provided_type, ci.cluster_type,ci.external_base_address,
	ci.external_agent_platform,ci.external_cloud_type,ci.interface,ci.route53_country_code,ci.tenant_id,ci.allocation_tag,ci.load_balancer_url
			FROM cluster_info ci where region_code = ? `

		selDB, err := database.Db.Query(query, regionCode)
		if err != nil {
			return &model.ClusterDetails{}, err
		}

		for selDB.Next() {
			err := selDB.Scan(&ciDetails.RegionName, &ciDetails.ProviderType, &ciDetails.ClusterType, &ciDetails.ExternalBaseAddress,
				&ciDetails.ExternalAgentPlatForm, &ciDetails.ExternalCloudType, &ciDetails.InterfaceType, &ciDetails.Route53countryCode, &ciDetails.TenantID, &ciDetails.AllocationTag, &ciDetails.LoadBalancerURL)
			if err != nil {
				return &model.ClusterDetails{}, err
			}

		}
	} else {
		query := `select ci.location_name,ci.provided_type, ci.cluster_type, ci.interface
		FROM cluster_info_user ci where (ci.region_code = ? and ci.is_active = 1)  and ci.user_id = ?`

		selDB, err := database.Db.Query(query, regionCode, userId)
		if err != nil {
			return &model.ClusterDetails{}, err
		}

		for selDB.Next() {
			err := selDB.Scan(&ciDetails.RegionName, &ciDetails.ProviderType, &ciDetails.ClusterType, &ciDetails.InterfaceType)
			if err != nil {
				return &model.ClusterDetails{}, err
			}

		}
	}
	return &ciDetails, nil
}

func GetAppAvalilableRegionList(appId string, appRegions []*model.Region) ([]*model.Region, error) {
	var regions []*model.Region
	query := `select code, Name, latitude , longitude from nife.regions`

	selDB, err := database.Db.Query(query)
	if err != nil {
		return nil, err
	}

	defer selDB.Close()
	for selDB.Next() {
		found := false
		row := model.Region{}
		err := selDB.Scan(&row.Code, &row.Name, &row.Latitude, &row.Longitude)
		if err != nil {
			return nil, err
		}
		for _, app := range appRegions {
			if *row.Code == *app.Code {
				found = true
				break
			}
		}
		if !found {
			regions = append(regions, &row)
		}

	}
	return regions, nil
}

func ExecuteGetRegionQuery(query string, selDB *sql.Rows) ([]*model.Region, error) {
	regions := []*model.Region{}
	defer selDB.Close()
	for selDB.Next() {
		row := model.Region{}
		err := selDB.Scan(&row.Code, &row.Name, &row.Latitude, &row.Longitude)
		if err != nil {
			return nil, err
		}
		regions = append(regions, &row)
	}

	return regions, nil
}

func GetStatusINString(query, status string) (string, error) {
	result, inString := "", ""
	s := strings.Split(status, ",")
	if len(s) == 1 {
		result = query + fmt.Sprintf("and ad.status IN ('%s')", status)
		fmt.Println(result)
		return result, nil

	} else {
		for i, status := range s {
			if i == (len(s) - 1) {
				inString = fmt.Sprintf(inString+"'%s'", status)
			} else {
				inString = fmt.Sprintf(inString+"'%s', ", status)
			}

		}
		result = query + fmt.Sprintf("and ad.status IN (%s)", inString)
		fmt.Println(result)
		return result, nil
	}
}

func GetActiveClusterDetails(userId string) (*[]ClusterDetail, error) {
	var clusterDetails []ClusterDetail
	query := `select region_code,name,cluster_config_path, ebl_enabled, port
		FROM cluster_info WHERE is_active = 1`

	selDB, err := database.Db.Query(query)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var clusterDetail ClusterDetail
		err = selDB.Scan(&clusterDetail.Region_code, &clusterDetail.RegionName, &clusterDetail.Cluster_config_path, &clusterDetail.EBL_enabled, &clusterDetail.Port)
		if err != nil {
			return nil, err
		}

		clusterDetails = append(clusterDetails, clusterDetail)
	}
	clusDet, err := GetActiveClusterDetailsAndUserAddedClusters(userId)
	if err != nil {
		return nil, err
	}

	clusterDetails = append(clusterDetails, *clusDet...)

	return &clusterDetails, nil
}

func GetActivePlatormClusterDetails() (*[]ClusterDetail, error) {
	var clusterDetails []ClusterDetail
	query := `select region_code,name,cluster_config_path, ebl_enabled, port
		FROM cluster_info WHERE is_active = 1`

	selDB, err := database.Db.Query(query)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var clusterDetail ClusterDetail
		err = selDB.Scan(&clusterDetail.Region_code, &clusterDetail.RegionName, &clusterDetail.Cluster_config_path, &clusterDetail.EBL_enabled, &clusterDetail.Port)
		if err != nil {
			return nil, err
		}

		clusterDetails = append(clusterDetails, clusterDetail)
	}

	return &clusterDetails, nil
}

func GetActiveClusterDetailsAndUserAddedClusters(userId string) (*[]ClusterDetail, error) {
	var clusterDetails []ClusterDetail
	query := `select region_code,location_name,cluster_config_url, ebl_enabled, port
		FROM cluster_info_user WHERE is_active = 1 and user_id = ?`

	selDB, err := database.Db.Query(query, userId)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var clusterDetail ClusterDetail
		err = selDB.Scan(&clusterDetail.Region_code, &clusterDetail.RegionName, &clusterDetail.ClusterConfigURL, &clusterDetail.EBL_enabled, &clusterDetail.Port)
		if err != nil {
			return nil, err
		}

		clusterDetails = append(clusterDetails, clusterDetail)
	}

	return &clusterDetails, nil
}

func GetAllClusterDetailsByOrgId(organizationId string) ([]*ClusterDetail, error) {
	var clusterDetails []*ClusterDetail
	query := `select org.region_code, org.is_default,ci.name,ci.cluster_config_path , ci.ebl_enabled, ci.port ,ci.provided_type, ci.cluster_type,ci.external_base_address,
		ci.external_agent_platform, ci.external_cloud_type, ci.interface,ci.route53_country_code,ci.tenant_id,ci.allocation_tag
				FROM organization_regions org join cluster_info ci 
				on org.region_code = ci.region_code
				where org.organization_id = ? `

	selDB, err := database.Db.Query(query, organizationId)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var clusterDetail ClusterDetail
		err := selDB.Scan(&clusterDetail.Region_code, &clusterDetail.IsDefault, &clusterDetail.RegionName, &clusterDetail.Cluster_config_path, &clusterDetail.EBL_enabled, &clusterDetail.Port, &clusterDetail.ProviderType, &clusterDetail.ClusterType, &clusterDetail.ExternalBaseAddress,
			&clusterDetail.ExternalAgentPlatform, &clusterDetail.ExternalCloudType, &clusterDetail.Interface, &clusterDetail.Route53CountryCode, &clusterDetail.TenantId, &clusterDetail.AllocationTag)
		if err != nil {
			return nil, err
		}

		clusterDetails = append(clusterDetails, &clusterDetail)
	}

	return clusterDetails, nil
}

func GetAllUserAddedClusterDetailsByUserId(userId string) ([]*model.ClusterDetails, error) {
	var clusterDetails []*model.ClusterDetails
	query := `SELECT id, region_code, provided_type, cluster_type, location_name, interface, cluster_config_url, ebl_enabled, port FROM cluster_info_user
	where user_id = ? and is_active = ?`

	selDB, err := database.Db.Query(query, userId, true)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var clusterDetail model.ClusterDetails
		err := selDB.Scan(&clusterDetail.ID, &clusterDetail.RegionCode, &clusterDetail.ProviderType, &clusterDetail.ClusterType, &clusterDetail.RegionName, &clusterDetail.InterfaceType, &clusterDetail.ClusterConfigURL, &clusterDetail.EblEnabled, &clusterDetail.Port)
		if err != nil {
			return nil, err
		}

		clusterDetails = append(clusterDetails, &clusterDetail)
	}

	return clusterDetails, nil
}
func GetUserAddedClusterDetailsByclusterId(id string) (*model.ClusterDetails, error) {
	var clusterDetail model.ClusterDetails
	query := `SELECT id, region_code, provided_type, cluster_type, location_name, interface, cluster_config_url, ebl_enabled, port FROM cluster_info_user
	where id = ? and is_active = ?`

	selDB, err := database.Db.Query(query, id, true)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		err := selDB.Scan(&clusterDetail.ID, &clusterDetail.RegionCode, &clusterDetail.ProviderType, &clusterDetail.ClusterType, &clusterDetail.RegionName, &clusterDetail.InterfaceType, &clusterDetail.ClusterConfigURL, &clusterDetail.EblEnabled, &clusterDetail.Port)
		if err != nil {
			return nil, err
		}
	}
	return &clusterDetail, nil
}

func GetCloudRegion(cloudType string) ([]*model.CloudRegions, error) {
	var clusterRegions []*model.CloudRegions
	query := `SELECT region_code, region_name, cloud_type FROM cloud_regions where cloud_type = ?`

	selDB, err := database.Db.Query(query, cloudType)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var clusterRegion model.CloudRegions
		err := selDB.Scan(&clusterRegion.Code, &clusterRegion.Name, &clusterRegion.Type)
		if err != nil {
			return nil, err
		}

		clusterRegions = append(clusterRegions, &clusterRegion)
	}

	return clusterRegions, nil
}

func GetClusterDetailsStruct(regionCode, userId string) (*ClusterDetail, error) {

	ccheckReg, err := helper.CheckRegionsByRegionCode(regionCode)
	if err != nil {
		return nil, err
	}
	var ciDetails ClusterDetail

	if ccheckReg != "" {

		query := `select ci.region_code, ci.name,ci.provided_type, ci.cluster_type,ci.external_base_address,
	ci.external_agent_platform,ci.external_cloud_type,ci.interface,ci.route53_country_code,ci.tenant_id,ci.allocation_tag, ci.cluster_config_path
			FROM cluster_info ci where region_code = ? `

		selDB, err := database.Db.Query(query, regionCode)
		if err != nil {
			return &ClusterDetail{}, err
		}

		for selDB.Next() {
			err := selDB.Scan(&ciDetails.Region_code, &ciDetails.RegionName, &ciDetails.ProviderType, &ciDetails.ClusterType, &ciDetails.ExternalBaseAddress,
				&ciDetails.ExternalAgentPlatform, &ciDetails.ExternalCloudType, &ciDetails.Interface, &ciDetails.Route53CountryCode, &ciDetails.TenantId, &ciDetails.AllocationTag, &ciDetails.Cluster_config_path)
			if err != nil {
				return &ClusterDetail{}, err
			}

		}
	} else {
		query := `select ci.id, ci.region_code, ci.location_name,ci.provided_type, ci.cluster_type, ci.interface, ci.cluster_config_url
		FROM cluster_info_user ci where (ci.region_code = ? and ci.is_active = 1)  and ci.user_id = ?`

		selDB, err := database.Db.Query(query, regionCode, userId)
		if err != nil {
			return &ClusterDetail{}, err
		}

		for selDB.Next() {
			err := selDB.Scan(&ciDetails.Id, &ciDetails.Region_code, &ciDetails.RegionName, &ciDetails.ProviderType, &ciDetails.ClusterType, &ciDetails.Interface, &ciDetails.ClusterConfigURL)
			if err != nil {
				return &ClusterDetail{}, err
			}

		}
	}
	return &ciDetails, nil
}
