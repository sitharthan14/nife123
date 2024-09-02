package organizationRegions

import (
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func GetOrganizationRegionByOrgId(organizationId, input, requestType string) (*OrganizationRegions, error) {
	var organizationRegion OrganizationRegions
	query := ""
	if requestType == "code" {
		query = "SELECT id, region_id, region_code, cluster_config, is_default from organization_regions where organization_id = ? and region_code = ?"
	} else {
		query = "SELECT id, region_id, region_code, cluster_config, is_default from organization_regions where organization_id = ? and is_default = ?"
	}

	selDB, err := database.Db.Query(query, organizationId, input)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&organizationRegion.Id, &organizationRegion.RegionId, &organizationRegion.RegionCode, &organizationRegion.ClusterConfig, &organizationRegion.IsDefault)
	if err != nil {
		return nil, err
	}

	return &organizationRegion, nil
}
