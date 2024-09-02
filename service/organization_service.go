package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"encoding/json"

	"time"

	//	"github.com/google/uuid"
	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	ci "github.com/nifetency/nife.io/internal/cluster_info"
	orgUser "github.com/nifetency/nife.io/internal/organizaiton_users"
	organizationInfo "github.com/nifetency/nife.io/internal/organization_info"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	"github.com/nifetency/nife.io/internal/users"
	awsService "github.com/nifetency/nife.io/pkg/aws"
	ad "github.com/nifetency/nife.io/pkg/helper"

	//	apiv1 "k8s.io/api/core/v1"
	secretregistry "github.com/nifetency/nife.io/internal/secret_registry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OrgSecret struct {
	UserId         string
	Name           string
	Response       secretregistry.SecretResponse
	OrganizationId string
	RegistryType   string
}

func AllOrganizations(userId string) (*model.Organizations, error) {
	var orgs model.Organizations

	selDB, err := database.Db.Query(`select org.id, org.parent_orgid, org.name, org.slug, org.type, org.domains, organization_users.is_active from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	where org.is_deleted = 0 and organization_users.is_active = 1 and organization_users.user_id = ? `, userId)
	if err != nil {
		return &orgs, err
	}

	defer selDB.Close()

	for selDB.Next() {
		var org model.Organization
		var dom []uint8
		var domains model.Domains
		err = selDB.Scan(&org.ID, &org.ParentID, &org.Name, &org.Slug, &org.Type, &dom, &org.IsActive)
		if err != nil {
			return &orgs, err
		}

		json.Unmarshal([]byte(string(dom)), &domains)
		org.Domains = &domains
		_, err := json.Marshal(org)
		if err != nil {
			return &orgs, err
		}

		if *org.Type == "" || *org.Type == "0" {
			*org.Type = "false"
		} else {
			*org.Type = "true"
		}

		region, err := GetRegionByOrgId(*org.ID, userId)

		if err != nil {
			return &orgs, err
		}
		org.Region = region
		subOrg, err := AllSubOrganizations(userId, *org.ID)
		org.SubOrg = subOrg

		orgs.Nodes = append(orgs.Nodes, &org)
	}
	return &orgs, nil
}

func AllParentOrganizations(userId string) (*model.Organizations, error) {
	var orgs model.Organizations

	selDB, err := database.Db.Query(`select org.id, org.parent_orgid, org.name, org.slug, org.type, org.domains, organization_users.is_active from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	where (org.parent_orgid = "" and org.is_deleted = 0) and (organization_users.is_active = 1 and organization_users.user_id = ?) ; `, userId)
	if err != nil {
		return &orgs, err
	}

	defer selDB.Close()

	for selDB.Next() {
		var org model.Organization
		var dom []uint8
		var domains model.Domains
		err = selDB.Scan(&org.ID, &org.ParentID, &org.Name, &org.Slug, &org.Type, &dom, &org.IsActive)
		if err != nil {
			return &orgs, err
		}

		json.Unmarshal([]byte(string(dom)), &domains)
		org.Domains = &domains
		_, err := json.Marshal(org)
		if err != nil {
			return &orgs, err
		}

		if *org.Type == "" || *org.Type == "0" {
			*org.Type = "false"
		} else {
			*org.Type = "true"
		}

		region, err := GetRegionByOrgId(*org.ID, userId)

		if err != nil {
			return &orgs, err
		}
		org.Region = region
		subOrg, err := AllSubOrganizations(userId, *org.ID)
		org.SubOrg = subOrg

		orgs.Nodes = append(orgs.Nodes, &org)
	}
	return &orgs, nil
}

func AllOrganizationsandbus(userId string) (*model.OrganizationsandBusinessUnit, error) {
	var orgs model.OrganizationsandBusinessUnit

	selDB, err := database.Db.Query(`select org.id, org.parent_orgid, org.name, org.slug, org.type, org.domains, organization_users.is_active from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	where org.is_deleted = 0 and organization_users.is_active = 1 and organization_users.user_id = ? `, userId)
	if err != nil {
		return &model.OrganizationsandBusinessUnit{}, err
	}

	defer selDB.Close()

	for selDB.Next() {
		var org model.Organization
		var dom []uint8
		var domains model.Domains
		err = selDB.Scan(&org.ID, &org.ParentID, &org.Name, &org.Slug, &org.Type, &dom, &org.IsActive)
		if err != nil {
			return &model.OrganizationsandBusinessUnit{}, err
		}

		json.Unmarshal([]byte(string(dom)), &domains)
		org.Domains = &domains
		_, err := json.Marshal(org)
		if err != nil {
			return &model.OrganizationsandBusinessUnit{}, err
		}

		if *org.Type == "" || *org.Type == "0" {
			*org.Type = "false"
		} else {
			*org.Type = "true"
		}

		region, err := GetRegionByOrgId(*org.ID, userId)

		if err != nil {
			return &model.OrganizationsandBusinessUnit{}, err
		}
		org.Region = region
		subOrg, err := AllSubOrganizations(userId, *org.ID)
		org.SubOrg = subOrg
		parentOrgName, err := GetOrgNameById(*org.ParentID)
		org.ParentOrgName = &parentOrgName

		orgs.Nodes = append(orgs.Nodes, &org)
	}

	return &orgs, nil
}

func SubOrganizations(userId string) (*model.Organizations, error) {
	var orgs model.Organizations

	selDB, err := database.Db.Query(`select org.id, org.parent_orgid,org.name, org.slug, org.type, org.domains, organization_users.is_active from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	where (org.is_deleted = 0 and org.parent_orgid != "" ) and organization_users.is_active = 1 and organization_users.user_id = ? `, userId)
	if err != nil {
		return &orgs, err
	}

	defer selDB.Close()

	for selDB.Next() {
		var org model.Organization
		var dom []uint8
		var domains model.Domains
		err = selDB.Scan(&org.ID, &org.ParentID, &org.Name, &org.Slug, &org.Type, &dom, &org.IsActive)
		if err != nil {
			return &orgs, err
		}

		json.Unmarshal([]byte(string(dom)), &domains)
		org.Domains = &domains
		_, err := json.Marshal(org)
		if err != nil {
			return &orgs, err
		}

		if *org.Type == "" || *org.Type == "0" {
			*org.Type = "false"
		} else {
			*org.Type = "true"
		}

		region, err := GetRegionByOrgId(*org.ID, userId)

		if err != nil {
			return &orgs, err
		}
		org.Region = region
		subOrg, err := AllSubOrganizations(userId, *org.ID)
		org.SubOrg = subOrg

		parentOrgName, err := GetOrgNameById(*org.ParentID)
		org.ParentOrgName = &parentOrgName

		orgs.Nodes = append(orgs.Nodes, &org)
	}
	return &orgs, nil
}

func SubOrganizationsById(userId, parentOrgId string) (*model.Organizations, error) {
	var orgs model.Organizations

	selDB, err := database.Db.Query(`select org.id, org.parent_orgid,org.name, org.slug, org.type, org.domains, organization_users.is_active from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	where (org.is_deleted = 0 and org.parent_orgid = ? ) and organization_users.is_active = 1 and organization_users.user_id = ?`, parentOrgId, userId)
	if err != nil {
		return &orgs, err
	}

	defer selDB.Close()

	for selDB.Next() {
		var org model.Organization
		var dom []uint8
		var domains model.Domains
		err = selDB.Scan(&org.ID, &org.ParentID, &org.Name, &org.Slug, &org.Type, &dom, &org.IsActive)
		if err != nil {
			return &orgs, err
		}

		json.Unmarshal([]byte(string(dom)), &domains)
		org.Domains = &domains
		_, err := json.Marshal(org)
		if err != nil {
			return &orgs, err
		}

		if *org.Type == "" || *org.Type == "0" {
			*org.Type = "false"
		} else {
			*org.Type = "true"
		}

		region, err := GetRegionByOrgId(*org.ID, userId)

		if err != nil {
			return &orgs, err
		}
		org.Region = region
		subOrg, err := AllSubOrganizations(userId, *org.ID)
		org.SubOrg = subOrg

		orgs.Nodes = append(orgs.Nodes, &org)
	}
	return &orgs, nil
}

func AllSubOrganizations(userId, orgId string) ([]*model.SubOrganization, error) {
	var subOrgs []*model.SubOrganization

	query := `select org.id, org.name, org.slug, org.type, org.domains, organization_users.is_active from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	where  org.parent_orgid = ? and (organization_users.is_active = 1 and organization_users.user_id = ?) and org.is_deleted = 0`

	selDB, err := database.Db.Query(query, orgId, userId)
	if err != nil {
		return []*model.SubOrganization{}, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var subOrg model.SubOrganization
		var dom []uint8
		var domains model.Domains
		err = selDB.Scan(&subOrg.ID, &subOrg.Name, &subOrg.Slug, &subOrg.Type, &dom, &subOrg.IsActive)
		if err != nil {
			return []*model.SubOrganization{}, err
		}

		json.Unmarshal([]byte(string(dom)), &domains)
		subOrg.Domains = &domains
		_, err := json.Marshal(subOrg)
		if err != nil {
			return []*model.SubOrganization{}, err
		}

		if *subOrg.Type == "" || *subOrg.Type == "0" {
			*subOrg.Type = "false"
		} else {
			*subOrg.Type = "true"
		}

		region, err := GetRegionByOrgId(*subOrg.ID, userId)

		if err != nil {
			return []*model.SubOrganization{}, err
		}

		subOrg.Region = region

		subOrgs = append(subOrgs, &subOrg)

	}
	return subOrgs, nil
}

func GetRegionByOrgId(orgId, userId string) ([]*model.RegionDetails, error) {

	userDet, err := users.GetEmailById(userId)
	if err != nil {
		return nil, err
	}
	if userDet.RoleId != 1 {
		adminEmail, err := users.GetAdminByCompanyNameAndEmail(userDet.CompanyName)
		if err != nil {
			return nil, err
		}

		userid, err := users.GetUserIdByEmail(adminEmail)
		if err != nil {
			return nil, err
		}
		userId = strconv.Itoa(userid)
	}

	query := `select organization_regions.region_code, organization_regions.is_default, regions.name, cluster_info.cluster_type from organization_regions 
	inner join regions on regions.code = organization_regions.region_code
	inner join cluster_info on cluster_info.region_code = regions.code
	where organization_id = ?`

	selDB, err := database.Db.Query(query, orgId)
	if err != nil {
		return []*model.RegionDetails{}, err
	}
	defer selDB.Close()

	var RegionDetails []*model.RegionDetails

	for selDB.Next() {
		var reg model.RegionDetails
		err = selDB.Scan(&reg.RegCode, &reg.IsDefault, &reg.RegionName, &reg.ClusterType)
		if err != nil {
			return []*model.RegionDetails{}, err
		}

		RegionDetails = append(RegionDetails, &reg)

	}
	userAddedReg, err := GetUserAddedRegionByOrgId(orgId, userId)
	if err != nil {
		return []*model.RegionDetails{}, err
	}

	RegionDetails = append(RegionDetails, userAddedReg...)

	return RegionDetails, nil

}

func GetUserAddedRegionByOrgId(orgId, userId string) ([]*model.RegionDetails, error) {
	query := `select organization_regions.region_code, organization_regions.is_default, cluster_info_user.location_name, cluster_info_user.cluster_type from organization_regions 
	inner join cluster_info_user on cluster_info_user.region_code = organization_regions.region_code 
   where (organization_id = ? and cluster_info_user.is_active = 1) and cluster_info_user.user_id = ?`

	selDB, err := database.Db.Query(query, orgId, userId)
	if err != nil {
		return []*model.RegionDetails{}, err
	}
	defer selDB.Close()

	var RegionDetails []*model.RegionDetails

	for selDB.Next() {
		var reg model.RegionDetails
		err = selDB.Scan(&reg.RegCode, &reg.IsDefault, &reg.RegionName, &reg.ClusterType)
		if err != nil {
			return []*model.RegionDetails{}, err
		}

		RegionDetails = append(RegionDetails, &reg)

	}
	return RegionDetails, nil

}

func GetOrganization(id, slug string) (*model.Organization, error) {
	var org model.Organization
	constraintString := slug
	query := "SELECT id, parent_orgid, name, slug, type, domains FROM organization WHERE slug=? and is_deleted = 0"
	if id != "" {
		query = "SELECT id, parent_orgid, name, slug, type, domains FROM organization WHERE id=? and is_deleted = 0"
		constraintString = id
	}
	selDB, err := database.Db.Query(query, constraintString)
	if err != nil {
		return &org, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var dom []uint8
		var domains model.Domains
		err = selDB.Scan(&org.ID, &org.ParentID, &org.Name, &org.Slug, &org.Type, &dom)
		if err != nil {
			return &org, err
		}
		json.Unmarshal([]byte(string(dom)), &domains)
		org.Domains = &domains
	}
	return &org, nil
}

func GetSubOrganization(id string) ([]*model.Organization, error) {

	query := "SELECT id, name, slug, type, domains FROM organization WHERE parent_orgid=? and is_deleted = 0"

	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return []*model.Organization{}, err
	}
	defer selDB.Close()

	result := []*model.Organization{}

	for selDB.Next() {
		var org model.Organization
		var dom []uint8
		var domains model.Domains
		err = selDB.Scan(&org.ID, &org.Name, &org.Slug, &org.Type, &dom)
		if err != nil {
			return []*model.Organization{}, err
		}
		json.Unmarshal([]byte(string(dom)), &domains)
		org.Domains = &domains

		result = append(result, &org)
	}
	return result, nil
}

func GetRoleByRoleId(roleId int) (string, error) {

	query := `SELECT  role.name FROM user 
	INNER JOIN role ON role.id = user.role_id 
	where role.id = ?`
	selDB, err := database.Db.Query(query, roleId)
	if err != nil {
		return "", err
	}
	var role string

	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&role)
		if err != nil {
			log.Println(err)
			return "", err
		}
	}

	return role, nil

}

func GetRoleByUserId(userId string) (string, error) {

	query := `SELECT role.name FROM user 
	INNER JOIN role ON role.id = user.role_id 
	where  user.id = ?`
	selDB, err := database.Db.Query(query, userId)
	if err != nil {
		return "", err
	}
	var role string
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&role)
		if err != nil {
			log.Println(err)
			return "", err
		}
	}

	return role, nil

}

func GetOrgDetails(slug string) (*model.OrganizationDetails, error) {

	role := "Admin"
	org, err := GetOrganization("", slug)
	if err != nil {
		return &model.OrganizationDetails{}, err
	}

	if org.ID == nil {
		return nil, fmt.Errorf("Invalid orgnaization %s ", slug)
	}

	orgUsers, err := orgUser.GetOrgUserDetails(*org.ID)
	if err != nil {
		return &model.OrganizationDetails{}, err
	}

	edges := []*model.OrganizationMembershipEdge{}
	for _, orgUser := range orgUsers {

		role, err := GetRoleByRoleId(orgUser.RoleId)
		if err != nil {
			return &model.OrganizationDetails{}, err
		}

		ou := orgUser

		Name := orgUser.FirstName + " " + orgUser.LastName

		user := model.User{ID: orgUser.UserId, Name: Name, Email: orgUser.UserEmail}

		edges = append(edges, &model.OrganizationMembershipEdge{Node: &user, Cursor: &ou.Id, ID: &ou.Id, Role: &role, JoinedAt: &ou.JoinedAt})
	}

	orgDetails := &model.OrganizationDetails{
		ID:         org.ID,
		Name:       org.Name,
		Slug:       org.Slug,
		Type:       org.Type,
		ViewerRole: &role,
		Members: &model.Members{
			Edges: edges,
		},
	}
	return orgDetails, nil
}

func DestoryOrganizationResources(org *model.Organization, userId string) error {
	var result int
	clusterDetails, err := ci.GetActiveClusterDetails(userId)
	if err != nil {
		return err
	}

	for _, clusPath := range *clusterDetails {
		var fileSavePath string
		if clusPath.Cluster_config_path == "" {
			k8sPath := "k8s_config/" + clusPath.Region_code
			err := os.Mkdir(k8sPath, 0755)
			if err != nil {
				return err
			}

			fileSavePath = k8sPath + "/config"

			_, err = organizationInfo.GetFileFromPrivateS3kubeconfigs(*clusPath.ClusterConfigURL, fileSavePath)
			if err != nil {
				return err
			}

		}
	}

	for _, c := range *clusterDetails {

		if c.Cluster_config_path == "" {
			c.Cluster_config_path = "./k8s_config/" + c.Region_code + "/config"
		}

		if c.Cluster_config_path != "" {
			fmt.Println(c.Cluster_config_path)
			err = DeleteNamespaceInCluster(strings.ToLower(strings.ReplaceAll(*org.Slug, " ", "-")), c.Cluster_config_path)
			if err != nil {
				helper.DeletedSourceFile("k8s_config/" + c.Region_code)
				log.Println(err)
				continue
			}
			helper.DeletedSourceFile("k8s_config/" + c.Region_code)
		}
	}
	appsDet, err := AllApps(userId, "", *org.Slug)
	if err != nil {
		return err
	}
	rows, err := database.Db.Query("call update_app_release_deployment_status(?)", org.ID)
	if err != nil {
		return err
	}

	rows.Next()
	err = rows.Scan(&result)
	if err != nil {
		return err
	}

	if result != 1 {
		return err
	}

	for _, appIds := range appsDet.Nodes {
		userDetails, err := GetById(userId)
		DeleteOperation := Activity{
			Type:       "APP",
			UserId:     userId,
			Activities: "DELETED",
			Message:    *userDetails.FirstName + " " + *userDetails.LastName + " has Deleted the App " + appIds.Name,
			RefId:      appIds.ID,
		}

		_, err = InsertActivity(DeleteOperation)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		err = SendSlackNotification(userId, DeleteOperation.Message)

	}
	return nil
}

func DeleteNamespaceInCluster(orgName, configPath string) error {
	clientset, err := helper.LoadK8SConfig(configPath)
	if err != nil {
		return err
	}

	deletePolicy := metav1.DeletePropagationBackground
	err = clientset.CoreV1().Namespaces().Delete(context.Background(), orgName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil {
		return err
	}
	return nil
}

func DestoryDNSRecord(org *model.Organization) error {

	appDeployments, err := ad.GetAppDeploymentsByOrg(*org.ID)

	if err != nil {
		return err
	}

	if len(*appDeployments) == 0 {
		return nil
	}

	awsService.DeleteDNSRecordBatch(appDeployments)
	// DELETE DNS RECORD FOR RUNNING INSTANCE
	return nil
}

func (s *OrgSecret) CreateSecret() (string, error) {

	statement, err := database.Db.Prepare("INSERT INTO organization_secrets (id,name,organization_id,registry_type,username, password,url,key_file_content, registry_name,createdBy, createdAt,secret_type) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, s.Name, s.OrganizationId, s.RegistryType, s.Response.UserName, s.Response.Password, s.Response.Url, s.Response.KeyFileContent, s.Response.RegistryName, s.UserId, time.Now(), s.Response.SecretType)
	if err != nil {
		return "", err
	}
	return "", nil
}

func (s *OrgSecret) UpdateSecret(name string) (string, error) {

	statement, err := database.Db.Prepare("UPDATE organization_secrets SET username=?, password=?,url=?,key_file_content=?, registry_name=?,updatedAt=? where name = ?")
	if err != nil {
		return "", err
	}
	defer statement.Close()
	_, err = statement.Exec(s.Response.UserName, s.Response.Password, s.Response.Url, s.Response.KeyFileContent, s.Response.RegistryName, time.Now().UTC(), name)
	if err != nil {
		return "", err
	}
	return "", nil
}

func UpdateOrganization(defaultType bool, slug string) error {
	statement, err := database.Db.Prepare("UPDATE organization SET type=? where slug = ?")
	if err != nil {
		return err
	}

	var DefaultType string
	if defaultType {
		DefaultType = "1"
	} else {
		DefaultType = "0"
	}
	defer statement.Close()
	_, err = statement.Exec(DefaultType, slug)
	if err != nil {
		return err
	}
	return nil
}

func DeleteSecretOrganization(name, id string) (string, error) {

	queryString := "DELETE FROM organization_secrets WHERE name = ?"

	statement, err := database.Db.Prepare(queryString)
	if err != nil {
		return "", err
	}
	defer statement.Close()
	_, err = statement.Exec(name)
	if err != nil {
		return "", err
	}
	return "", nil

}

func GetRegistryType() ([]*model.OrganizationRegistryType, error) {
	query := "SELECT id, name, slug, is_active FROM registry_type where is_active = 1"
	selDB, err := database.Db.Query(query)
	if err != nil {
		return []*model.OrganizationRegistryType{}, err
	}
	defer selDB.Close()
	var orgReg []*model.OrganizationRegistryType
	for selDB.Next() {
		var regType model.OrganizationRegistryType
		err = selDB.Scan(&regType.ID, &regType.Name, &regType.Slug, &regType.IsActive)

		if err != nil {
			return []*model.OrganizationRegistryType{}, err
		}

		org := &model.OrganizationRegistryType{
			ID:       regType.ID,
			Name:     regType.Name,
			Slug:     regType.Slug,
			IsActive: regType.IsActive,
		}
		orgReg = append(orgReg, org)
	}
	return orgReg, nil

}

func GetUserSecret(name, userId string) ([]*model.GetUserSecret, error) {
	var query string
	if name == "" {
		query = `select id, name, organization_id, registry_type, username, password, url, key_file_content, registry_name, is_active from organization_secrets where createdBy = ` + userId + ` and is_default = 1`
	} else {
		query = "select id, name, organization_id, registry_type, username, password, url, key_file_content, registry_name, is_active from organization_secrets where name = " + `'` + name + `'` + ` and is_default = 1`
	}
	selDB, err := database.Db.Query(query)
	if err != nil {
		return []*model.GetUserSecret{}, err
	}
	defer selDB.Close()
	var orgReg []*model.GetUserSecret
	for selDB.Next() {
		var userSec model.GetUserSecret
		err = selDB.Scan(&userSec.ID, &userSec.Name, &userSec.OrganizationID, &userSec.RegistryType, &userSec.UserName,
			&userSec.PassWord, &userSec.URL, &userSec.KeyFileContent, &userSec.RegistryName, &userSec.IsActive)

		if err != nil {
			return []*model.GetUserSecret{}, err
		}

		org := &model.GetUserSecret{
			ID:             userSec.ID,
			Name:           userSec.Name,
			OrganizationID: userSec.OrganizationID,
			RegistryType:   userSec.RegistryType,
			UserName:       userSec.UserName,
			PassWord:       userSec.PassWord,
			URL:            userSec.URL,
			KeyFileContent: userSec.KeyFileContent,
			RegistryName:   userSec.RegistryName,
			IsActive:       userSec.IsActive,
		}
		orgReg = append(orgReg, org)
	}
	return orgReg, nil

}
func GetUserSecretByOrgId(orgId string) ([]*model.GetUserSecret, error) {
	var query string
	query = "select id, name, organization_id, registry_type, username, password, url, key_file_content, registry_name, is_active from organization_secrets where organization_id = " + `'` + orgId + `'` + " and is_default = 1"
	selDB, err := database.Db.Query(query)
	if err != nil {
		return []*model.GetUserSecret{}, err
	}
	defer selDB.Close()
	var orgReg []*model.GetUserSecret
	for selDB.Next() {
		var userSec model.GetUserSecret
		err = selDB.Scan(&userSec.ID, &userSec.Name, &userSec.OrganizationID, &userSec.RegistryType, &userSec.UserName,
			&userSec.PassWord, &userSec.URL, &userSec.KeyFileContent, &userSec.RegistryName, &userSec.IsActive)

		if err != nil {
			return []*model.GetUserSecret{}, err
		}

		org := &model.GetUserSecret{
			ID:             userSec.ID,
			Name:           userSec.Name,
			OrganizationID: userSec.OrganizationID,
			RegistryType:   userSec.RegistryType,
			UserName:       userSec.UserName,
			PassWord:       userSec.PassWord,
			URL:            userSec.URL,
			KeyFileContent: userSec.KeyFileContent,
			RegistryName:   userSec.RegistryName,
			IsActive:       userSec.IsActive,
		}
		orgReg = append(orgReg, org)
	}
	return orgReg, nil

}

func GetUserSecretByRegistryType(name, userId, registryType string) (string, error) {

	query := `select id from organization_secrets where (name = ? and  registry_type = ?) and (is_active = 1 and createdBy = ?)`

	selDB, err := database.Db.Query(query, name, registryType, userId)
	if err != nil {
		return "", err
	}
	defer selDB.Close()
	var userSecId string
	for selDB.Next() {
		err = selDB.Scan(&userSecId)
		if err != nil {
			return "", err
		}
	}
	return userSecId, nil

}

func GetSecretBySecId(secId, userId string) (*model.GetUserSecret, error) {
	selDB, err := database.Db.Query(`select id, name, organization_id, registry_type, username, password, url, key_file_content, registry_name, is_active from organization_secrets where id = ? and createdBy = ?`, secId, userId)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	var userSec model.GetUserSecret
	for selDB.Next() {
		err = selDB.Scan(&userSec.ID, &userSec.Name, &userSec.OrganizationID, &userSec.RegistryType, &userSec.UserName, &userSec.PassWord, &userSec.URL, &userSec.KeyFileContent, &userSec.RegistryName, &userSec.IsActive)
		if err != nil {
			return nil, err
		}
	}
	return &userSec, err
}

func GetRegistryNameList(userId, orgId, regType string) ([]*model.GetSecRegistry, error) {

	query := `select id, name from organization_secrets where organization_id = ? and registry_type = ? and createdBy = ?`

	selDB, err := database.Db.Query(query, orgId, regType, userId)
	if err != nil {
		return []*model.GetSecRegistry{}, err
	}
	var regList []*model.GetSecRegistry
	defer selDB.Close()

	for selDB.Next() {
		var reg model.GetSecRegistry
		err = selDB.Scan(&reg.ID, &reg.Name)
		if err != nil {
			log.Println(err)
			return []*model.GetSecRegistry{}, err
		}

		res := model.GetSecRegistry{
			ID:   reg.ID,
			Name: reg.Name,
		}

		regList = append(regList, &res)
	}

	return regList, nil

}

func GetRegId(name string) (string, error) {

	query := `select id from organization_secrets where name = ?`
	selDB, err := database.Db.Query(query, name)
	if err != nil {
		return "", err
	}
	var regId string
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&regId)
		if err != nil {
			log.Println(err)
			return "", err
		}
	}

	return regId, nil

}

func UpdateRegId(appName, id string) error {

	statement, err := database.Db.Prepare("Update app set secrets_registry_id = ? where name = ?")
	if err != nil {
		return err
	}
	_, err = statement.Exec(id, appName)
	if err != nil {
		return err
	}

	return nil
}

func GetRegistryById(id string) (model.GetUserSecret, error) {

	query := `select name, organization_id, registry_type,username, password, url, key_file_content, registry_name from organization_secrets where id = ?`
	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return model.GetUserSecret{}, err
	}

	defer selDB.Close()

	var secretDetails model.GetUserSecret

	for selDB.Next() {
		err = selDB.Scan(&secretDetails.Name, &secretDetails.OrganizationID, &secretDetails.RegistryType,
			&secretDetails.UserName, &secretDetails.PassWord, &secretDetails.URL, &secretDetails.KeyFileContent,
			&secretDetails.RegistryName)
		if err != nil {
			log.Println(err)
			return model.GetUserSecret{}, err
		}
	}
	return secretDetails, nil
}

func GetOrganizationById(id string) (model.Organization, error) {

	var org model.Organization

	query := "SELECT name, slug, type, domains FROM organization WHERE id=? and is_deleted = 0"

	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return model.Organization{}, err
	}
	defer selDB.Close()

	for selDB.Next() {
		var dom []uint8
		var domains model.Domains
		err = selDB.Scan(&org.Name, &org.Slug, &org.Type, &dom)
		if err != nil {
			return model.Organization{}, err
		}
		json.Unmarshal([]byte(string(dom)), &domains)
		org.Domains = &domains
	}

	return org, nil

}

func GetAppCountByOrg(userId, roleId string) ([]*model.AppOrgCount, error) {

	query := `select COUNT(app.name), org.name from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	INNER JOIN app ON app.organization_id = org.id
	where organization_users.user_id =? and (app.status = "Active" or app.status = "Suspended") and organization_users.role_id = ? group by org.name`

	selDB, err := database.Db.Query(query, userId, roleId)
	if err != nil {
		return []*model.AppOrgCount{}, err
	}
	defer selDB.Close()

	var RegionCount []*model.AppOrgCount

	for selDB.Next() {

		var organization string
		var apps int
		err = selDB.Scan(&apps, &organization)
		if err != nil {
			return []*model.AppOrgCount{}, err
		}

		result := model.AppOrgCount{
			Organization: &organization,
			Apps:         &apps,
		}

		RegionCount = append(RegionCount, &result)
	}

	return RegionCount, nil
}

func GetOrganizationCountById(id string) (int, error) {
	var TotalOrgCount int
	selDB, err := database.Db.Query(`select count(org.name) from organization org	INNER JOIN organization_users ON org.id = organization_users.organization_id where (org.is_deleted = 0 and org.parent_orgid = "") and (organization_users.is_active = 1 and organization_users.user_id = ?);`, id)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&TotalOrgCount)
		if err != nil {
			return 0, err
		}
	}
	return TotalOrgCount, err
}

func GetSubOrganizationCountById(id string) (int, error) {
	var TotalSubOrgCount int
	selDB, err := database.Db.Query(`select count(org.name) from organization org	INNER JOIN organization_users ON org.id = organization_users.organization_id where (org.is_deleted = 0 and org.parent_orgid != "") and (organization_users.is_active = 1 and organization_users.user_id = ?);`, id)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&TotalSubOrgCount)
		if err != nil {
			return 0, err
		}
	}
	return TotalSubOrgCount, err
}

func GetBusinessUnitCountById(id string) (int, error) {
	var TotalBusinessUnitCount int
	selDB, err := database.Db.Query(`SELECT  count(name) FROM business_unit where created_by = ? and  is_active = 1;`, id)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&TotalBusinessUnitCount)
		if err != nil {
			return 0, err
		}
	}
	return TotalBusinessUnitCount, err
}

func GetOrgNameById(id string) (string, error) {

	query := `select name from organization where id = ?`
	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return "", err
	}
	var name string
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&name)
		if err != nil {
			log.Println(err)
			return "", err
		}
	}

	return name, nil

}

func GetBusinessUnitById(id string) (string, error) {

	query := `select name from business_unit where id = ?`
	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return "", err
	}
	var name string
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&name)
		if err != nil {
			log.Println(err)
			return "", err
		}
	}

	return name, nil

}
func GetWorkloadNameById(id string) (string, string, error) {

	query := `select environment_name, environment_endpoint from workload_management where id = ?`
	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return "", "", err
	}
	var workloadName string
	var workloadEndPoint string

	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&workloadName, &workloadEndPoint)
		if err != nil {
			log.Println(err)
			return "", "", err
		}
	}

	return workloadName, workloadEndPoint, nil

}

func GetworkloadByName(name, userId string) (string, string, error) {

	query := `SELECT environment_name, environment_endpoint FROM workload_management where (user_id = ? and environment_name = ?) and is_active = ?`
	selDB, err := database.Db.Query(query, userId, name, true)
	if err != nil {
		return "", "", err
	}
	var wlName string
	var wlEndPoint string
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&wlName, &wlEndPoint)
		if err != nil {
			log.Println(err)
			return "", "", err
		}
	}

	return wlName, wlEndPoint, nil

}

func DeleteBusinessUnitByOrg(subOrgId string) error {
	statement, err := database.Db.Prepare("UPDATE business_unit SET is_active = ? WHERE sub_org_id = ?;")
	if err != nil {
		return err
	}
	_, err = statement.Exec(false, subOrgId)
	if err != nil {
		return err
	}

	return nil
}

func PrentIdBySubOrganization(userId, subOrgID string) (*model.Organizations, error) {
	var orgs model.Organizations

	selDB, err := database.Db.Query(`select org.id, org.parent_orgid,org.name, org.slug, org.type, org.domains, organization_users.is_active from organization org	
	INNER JOIN organization_users ON org.id = organization_users.organization_id 
	where (org.is_deleted = 0 and org.id = ? ) and organization_users.is_active = 1 and organization_users.user_id = ?;`, subOrgID, userId)
	if err != nil {
		return &orgs, err
	}

	defer selDB.Close()

	for selDB.Next() {
		var org model.Organization
		var dom []uint8
		var domains model.Domains
		err = selDB.Scan(&org.ID, &org.ParentID, &org.Name, &org.Slug, &org.Type, &dom, &org.IsActive)
		if err != nil {
			return &orgs, err
		}

		json.Unmarshal([]byte(string(dom)), &domains)
		org.Domains = &domains
		_, err := json.Marshal(org)
		if err != nil {
			return &orgs, err
		}

		if *org.Type == "" || *org.Type == "0" {
			*org.Type = "false"
		} else {
			*org.Type = "true"
		}

		region, err := GetRegionByOrgId(*org.ID, userId)

		if err != nil {
			return &orgs, err
		}
		org.Region = region
		subOrg, err := AllSubOrganizations(userId, *org.ID)
		org.SubOrg = subOrg

		orgs.Nodes = append(orgs.Nodes, &org)
	}
	return &orgs, nil
}

func InsertOrganizationRegion(orgId, regionCode, userId string) (string, error) {

	statement, err := database.Db.Prepare("INSERT INTO organization_regions (id, organization_id, region_code, is_default) VALUES (?, ?, ?, ?)")
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, orgId, regionCode, 0)
	if err != nil {
		return "", err
	}

	orgDet, err := GetOrganizationById(orgId)
	if err != nil {
		return "", err
	}

	ccheckReg, err := CheckRegionsByRegionCodes(regionCode)
	if err != nil {
		return "", err
	}

	var clusterPath string
	var clusterRegion1 *model.ClusterDetails

	if ccheckReg != "" {

		clusterRegion, err := GetClusterDetailsByRegionCode(regionCode)
		if err != nil {
			return "", err
		}

		clusterPath = *clusterRegion.ClusterConfigPath

	} else {
		clusterRegion1, err = GetUserAddedClusterDetailsByRegionCode(regionCode, userId)
		if err != nil {
			return "", err
		}
		//====================---------------------========================
		k8sPath := "k8s_config/" + *clusterRegion1.RegionCode
		var fileSavePath string
		err = os.Mkdir(k8sPath, 0755)
		if err != nil {
			return "", err
		}

		fileSavePath = k8sPath + "/config"

		_, err = organizationInfo.GetFileFromPrivateS3kubeconfigs(*clusterRegion1.ClusterConfigURL, fileSavePath)
		if err != nil {
			helper.DeletedSourceFile("k8s_config/" + *clusterRegion1.RegionCode)
			return "", err
		}
		clusterPath = "./k8s_config/" + *clusterRegion1.RegionCode + "/config"

	}

	err = organizationInfo.CreateNamespaceInCluster(*orgDet.Slug, clusterPath)
	if err != nil {
		if ccheckReg == "" {
			helper.DeletedSourceFile("k8s_config/" + *clusterRegion1.RegionCode)
		}
		log.Println(err)
	}
	if ccheckReg == "" {
		helper.DeletedSourceFile("k8s_config/" + *clusterRegion1.RegionCode)
	}

	return "Inserted Successfully", nil
}

func CheckRegionsByRegionCodes(regionCode string) (string, error) {

	query := "SELECT region_code FROM cluster_info WHERE region_code = ? and is_active = ?; "

	selDB, err := database.Db.Query(query, regionCode, true)
	if err != nil {
		return "", err
	}
	var checkRegionCode string

	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&checkRegionCode)

	return checkRegionCode, nil

}

func GetClusterDetailsByRegionCode(regionCode string) (*model.ClusterDetails, error) {
	var clusterDetail model.ClusterDetails
	query := `select ci.region_code, ci.name,ci.cluster_config_path , ci.ebl_enabled, ci.port ,ci.provided_type, ci.cluster_type,ci.external_base_address,
	ci.external_agent_platform, ci.external_cloud_type, ci.interface,ci.route53_country_code,ci.tenant_id,ci.allocation_tag
			FROM cluster_info ci 
			where ci.region_code = ? and ci.is_active = 1;`

	selDB, err := database.Db.Query(query, regionCode)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		err := selDB.Scan(&clusterDetail.RegionCode, &clusterDetail.RegionName, &clusterDetail.ClusterConfigPath, &clusterDetail.EblEnabled, &clusterDetail.Port, &clusterDetail.ProviderType, &clusterDetail.ClusterType, &clusterDetail.ExternalBaseAddress,
			&clusterDetail.ExternalAgentPlatForm, &clusterDetail.ExternalCloudType, &clusterDetail.InterfaceType, &clusterDetail.Route53countryCode, &clusterDetail.TenantID, &clusterDetail.AllocationTag)
		if err != nil {
			return nil, err
		}

	}
	return &clusterDetail, nil
}

func GetUserAddedClusterDetailsByRegionCode(regionCode, userId string) (*model.ClusterDetails, error) {
	var userAddedRegion model.ClusterDetails
	query := `SELECT region_code,  provided_type, cluster_type, location_name, interface, cluster_config_url, ebl_enabled, port FROM cluster_info_user where (user_id = ? and is_active = 1) and region_code = ?`

	selDB, err := database.Db.Query(query, userId, regionCode)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		err := selDB.Scan(&userAddedRegion.RegionCode, &userAddedRegion.ProviderType, &userAddedRegion.ClusterType, &userAddedRegion.RegionName, &userAddedRegion.InterfaceType, &userAddedRegion.ClusterConfigURL, &userAddedRegion.EblEnabled, &userAddedRegion.Port)
		if err != nil {
			return nil, err
		}
	}
	return &userAddedRegion, nil
}
