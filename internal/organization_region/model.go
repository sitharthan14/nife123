package organizationRegions

type OrganizationRegions struct {
	Id             string `json:"id"`
	OrganizationId string `json:"organization_id"`
	RegionId       string `json:"region_id"`
	RegionCode     string `json:"region_code"`
	ClusterConfig  string `json:"cluster_config"`
	IsDefault      int    `json:"is_default"`
}
