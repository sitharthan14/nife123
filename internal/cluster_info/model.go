package clusterInfo

type ClusterDetail struct {
	Id 					  string  `json:"id"`
	Region_code           string  `json:"region_code"`
	RegionName            *string `json:"regionName"`
	IsDefault             int     `json:"is_default"`
	Cluster_config_path   string  `json:"cluster_config_path"`
	EBL_enabled           string  `json:"ebl_enabled"`
	Port                  string  `json:"port"`
	CloudType             string  `json:"cluster_type"`
	ClusterType           string  `json:"clusterType"`
	ProviderType          *string `json:"providerType"`
	ExternalBaseAddress   *string `json:"externalBaseAddress"`
	ExternalAgentPlatform *int    `json:"externalAgentPlatform"`
	ExternalLBType        *string `json:"externalLbType"`
	ExternalCloudType     *int `json:"externalCloudtype"`
	Interface             *string `json:"interface"`
	Route53CountryCode    *string `json:"route53Code"`
	TenantId              *string `json:"tenantId"`
	AllocationTag         string   `json:"allocationTag"`
	ClusterConfigURL      *string `json:"clusterConfigUrl"`
}
