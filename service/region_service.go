package service

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

type RegionDetails struct {
	ID        string
	Name      string
	Latitude  string
	Longitude string
}

type Config struct {
	APIVersion  string   `yaml:"apiVersion"`
	Kind        string   `yaml:"kind"`
	Preferences struct{} `yaml:"preferences"`
	Clusters    []struct {
		Name    string `yaml:"name"`
		Cluster struct {
			CertificateAuthorityData string `yaml:"certificate-authority-data"`
			Server                   string `yaml:"server"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
	Contexts []struct {
		Name    string `yaml:"name"`
		Context struct {
			Cluster string `yaml:"cluster"`
			User    string `yaml:"user"`
		} `yaml:"context"`
	} `yaml:"contexts"`
	CurrentContext string `yaml:"current-context"`
	Users          []struct {
		Name string `yaml:"name"`
		User struct {
			Exec struct {
				APIVersion string   `yaml:"apiVersion"`
				Command    string   `yaml:"command"`
				Args       []string `yaml:"args"`
			} `yaml:"exec"`
		} `yaml:"user"`
	} `yaml:"users"`
}

func UpdateDefaultRegion(region model.DefaultRegionInput) error {
	statement, err := database.Db.Prepare(`UPDATE organization_regions
	SET is_default = ?
	WHERE organization_id = ? and region_code = ? `)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(true, region.OrganizationID, region.Region)
	if err != nil {
		return err
	}

	return nil
}

func RemoveDefaultRegion(orgId string) error {
	statement, err := database.Db.Prepare(`UPDATE organization_regions
	SET is_default = ?
	WHERE organization_id = ? and is_default = ? `)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(false, orgId, true)
	if err != nil {
		return err
	}

	return nil
}

func GetRegionDetails(code string) (RegionDetails, error) {

	query := "select id, name, latitude,longitude from regions where code = ?"

	selDB, err := database.Db.Query(query, code)
	if err != nil {
		return RegionDetails{}, err
	}
	var regionDetails RegionDetails

	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&regionDetails.ID, &regionDetails.Name, &regionDetails.Latitude, &regionDetails.Longitude)
	if err != nil {
		return RegionDetails{}, err
	}
	return regionDetails, nil

}

func UpdateMultipleRegion(isDefault bool, organizationID, region string) error {
	statement, err := database.Db.Prepare(`UPDATE organization_regions
	SET is_default = ?
	WHERE organization_id = ? and region_code = ? `)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(isDefault, organizationID, region)
	if err != nil {
		return err
	}

	return nil
}
func DeleteRequestRegion(id string) error {
	statement, err := database.Db.Prepare(`DELETE FROM requested_region WHERE id = ?`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(id)
	if err != nil {
		return err
	}

	return nil
}

func InsertNewRegionRequest(userName, userId, userRegion string) error {

	statement, err := database.Db.Prepare("INSERT INTO requested_region(id, user_name, status, created_by, created_at, requested_region) VALUES(?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, userName, "Pending", userId, time.Now(), userRegion)
	if err != nil {
		return err
	}
	return nil
}

func GetRegionRequestByUserId(userid string) ([]*model.RequestedRegions, error) {
	query := `SELECT id,user_name, status, created_by, created_at, requested_region FROM requested_region WHERE created_by = ? AND requested_region IS NOT NULL`

	selDB, err := database.Db.Query(query, userid)

	if err != nil {
		return []*model.RequestedRegions{}, err
	}

	defer selDB.Close()

	var result []*model.RequestedRegions
	for selDB.Next() {

		var region model.RequestedRegions

		err := selDB.Scan(&region.ID, &region.UserName, &region.Status, &region.CreatedBy, &region.CreatedAt, &region.RequestedRegion)

		if err != nil {
			return []*model.RequestedRegions{}, err
		}

		result = append(result, &region)
	}
	return result, nil
}

func GetRequestedRegion(userId, region string) (string, error) {

	query := "SELECT requested_region FROM requested_region WHERE created_by = ? and requested_region = ?; "

	selDB, err := database.Db.Query(query, userId, region)
	if err != nil {
		return "", err
	}
	var requestedRegion string

	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&requestedRegion)

	return requestedRegion, nil

}

func AddUserK8sRegion(k8sRegions model.ClusterDetailsInput, userId string) error {
	statement, err := database.Db.Prepare("INSERT INTO cluster_info_user(id, region_code, provided_type, cluster_type, location_name, interface, cluster_config_url, user_id, ebl_enabled, port, is_active) VALUES(?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, k8sRegions.RegionCode, k8sRegions.ProviderType, "byoc", k8sRegions.RegionName, "kube_config", k8sRegions.ClusterConfigURL, userId, false, "3000", true)
	if err != nil {
		return err
	}
	return nil
}

func CheckUserAddedRegion(userId string) (int, error) {

	query := "SELECT region_code,  provided_type, cluster_type, location_name, interface, cluster_config_url, ebl_enabled, port FROM cluster_info_user where user_id = ? and is_active = 1"

	selDB, err := database.Db.Query(query, userId)
	if err != nil {
		return 0, err
	}
	var result []model.ClusterDetails

	defer selDB.Close()
	for selDB.Next() {
		var userAddedRegion model.ClusterDetails

		err = selDB.Scan(&userAddedRegion.RegionCode, &userAddedRegion.ProviderType, &userAddedRegion.ClusterType, &userAddedRegion.RegionName, &userAddedRegion.InterfaceType, &userAddedRegion.ClusterConfigURL, &userAddedRegion.EblEnabled, &userAddedRegion.Port)

		result = append(result, userAddedRegion)
	}

	k8sRegionCount := len(result)

	return k8sRegionCount, nil

}

func DeleteUserK8sRegion(id, user_id string) error {
	statement, err := database.Db.Prepare("UPDATE cluster_info_user set is_active = ? where id = ? and user_id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(false, id, user_id)
	if err != nil {
		return err
	}
	return nil
}

func CheckUserRegionById(userId, regId string) (model.ClusterDetails, error) {

	query := "SELECT region_code,  provided_type, cluster_type, location_name, interface, cluster_config_url, ebl_enabled, port FROM cluster_info_user where id = ? and user_id = ? and is_active = 1"

	selDB, err := database.Db.Query(query, regId, userId)
	if err != nil {
		return model.ClusterDetails{}, err
	}

	defer selDB.Close()
	selDB.Next()
	var userAddedRegion model.ClusterDetails
	err = selDB.Scan(&userAddedRegion.RegionCode, &userAddedRegion.ProviderType, &userAddedRegion.ClusterType, &userAddedRegion.RegionName, &userAddedRegion.InterfaceType, &userAddedRegion.ClusterConfigURL, &userAddedRegion.EblEnabled, &userAddedRegion.Port)

	return userAddedRegion, nil

}

func SplitUrl(Url string) (string, error) {

	parts := strings.Split(Url, "/")

	fileNames := parts[len(parts)-1]

	return fileNames, nil
}

func GetClusterDetails(userId string) ([]*model.ClusterDetails, error) {

	query := `select ci.region_code, ci.name,ci.provided_type, ci.cluster_type,ci.external_base_address,
		ci.external_agent_platform,ci.external_cloud_type,ci.interface,ci.route53_country_code,ci.tenant_id,ci.allocation_tag,ci.load_balancer_url
				FROM cluster_info ci where ci.is_active = 1`

	selDB, err := database.Db.Query(query)
	if err != nil {
		return []*model.ClusterDetails{}, err
	}

	var result []*model.ClusterDetails
	defer selDB.Close()
	for selDB.Next() {
		var ciDetails model.ClusterDetails
		err := selDB.Scan(&ciDetails.RegionCode, &ciDetails.RegionName, &ciDetails.ProviderType, &ciDetails.ClusterType, &ciDetails.ExternalBaseAddress,
			&ciDetails.ExternalAgentPlatForm, &ciDetails.ExternalCloudType, &ciDetails.InterfaceType, &ciDetails.Route53countryCode, &ciDetails.TenantID, &ciDetails.AllocationTag, &ciDetails.LoadBalancerURL)
		if err != nil {
			return []*model.ClusterDetails{}, err
		}

		result = append(result, &ciDetails)
	}

	userAddedRegions, err := GetUserAddedClusterDetails(userId)
	if err != nil {
		return nil, err
	}

	result = append(result, userAddedRegions...)

	return result, nil
}

func GetUserAddedClusterDetails(userId string) ([]*model.ClusterDetails, error) {
	var clusterDetails []*model.ClusterDetails
	query := `select ci.region_code, ci.location_name, ci.cluster_config_url , ci.ebl_enabled, ci.port ,ci.provided_type, ci.cluster_type, ci.interface
	FROM cluster_info_user ci where ci.is_active = 1 and ci.user_id = ?;`

	selDB, err := database.Db.Query(query, userId)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var clusterDetail model.ClusterDetails
		err := selDB.Scan(&clusterDetail.RegionCode, &clusterDetail.RegionName, &clusterDetail.ClusterConfigURL, &clusterDetail.EblEnabled, &clusterDetail.Port, &clusterDetail.ProviderType, &clusterDetail.ClusterType,
			&clusterDetail.InterfaceType)
		if err != nil {
			return nil, err
		}

		clusterDetails = append(clusterDetails, &clusterDetail)
	}

	return clusterDetails, nil
}

func GetOrganizationRegionByOrgId(organizationId string) ([]*model.OrganizationRegionTable, error) {
	var result []*model.OrganizationRegionTable
	query := "SELECT id, organization_id, region_code, is_default from organization_regions where organization_id = ? "

	selDB, err := database.Db.Query(query, organizationId)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var organizationRegion model.OrganizationRegionTable

		err = selDB.Scan(&organizationRegion.ID, &organizationRegion.OrganizationID, &organizationRegion.RegionCode, &organizationRegion.IsDefault)
		if err != nil {
			return nil, err
		}
		result = append(result, &organizationRegion)
	}

	return result, nil
}

func GetRegionDetailsByCode(code, userId string) (model.Region, error) {

	query := "select code, name, latitude,longitude from regions where code = ?"

	selDB, err := database.Db.Query(query, code)
	if err != nil {
		return model.Region{}, err
	}
	var regionDetails model.Region

	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&regionDetails.Code, &regionDetails.Name, &regionDetails.Latitude, &regionDetails.Longitude)

	if regionDetails.Name == nil {
		regionDet, err := GetUserAddedRegionsDet(code, userId)
		if err != nil {
			return model.Region{}, err
		}
		regionDetails = regionDet
	}

	return regionDetails, nil

}

func GetUserAddedRegionsDet(code, userId string) (model.Region, error) {

	query := "SELECT region_code, location_name FROM cluster_info_user where (region_code = ? and user_id = ?) and is_active = ?;"

	selDB, err := database.Db.Query(query, code, userId, true)
	if err != nil {
		return model.Region{}, err
	}
	var regionDetails model.Region

	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&regionDetails.Code, &regionDetails.Name)
	if err != nil {
		return model.Region{}, err
	}

	return regionDetails, nil
}
