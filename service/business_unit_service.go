package service

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func CreateBusinessUnit(businessUnit model.BusinessUnitInput, user_id string) error {
	statement, err := database.Db.Prepare("INSERT INTO business_unit(id, org_id, sub_org_id, name, created_by, is_active, created_at) VALUES(?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	id := uuid.NewString()
	name := strings.ToUpper(*businessUnit.Name)
	defer statement.Close()
	_, err = statement.Exec(id, businessUnit.OrgID, businessUnit.SubOrg, name, user_id, true, time.Now())
	if err != nil {
		return err
	}
	return nil
}

func UpdateBusinessUnit(businessUnit model.BusinessUnitInput, user_id string) error {
	statement, err := database.Db.Prepare(`UPDATE business_unit SET name = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(businessUnit.Name, businessUnit.ID)
	if err != nil {
		return err
	}

	return nil
}

func GetBusinessUnit(orgId, subOrgId string) ([]*model.GetBusinessUnit, error) {
	var queryString string

	if subOrgId == "" {
		querystring := `SELECT id, name, is_active FROM business_unit where org_id =` + `"` + orgId + `"`
		queryString = querystring
	} else {
		querystring := `SELECT id, name, is_active FROM business_unit where org_id =` + `"` + orgId + `" and sub_org_id =` + `"` + subOrgId + `"`
		queryString = querystring
	}
	selDB, err := database.Db.Query(queryString)
	if err != nil {
		return nil, err
	}
	var result []*model.GetBusinessUnit
	defer selDB.Close()
	for selDB.Next() {
		var businessUnit model.GetBusinessUnit

		err := selDB.Scan(&businessUnit.ID, &businessUnit.Name, &businessUnit.IsActive)
		if err != nil {
			return []*model.GetBusinessUnit{}, err
		}
		result = append(result, &businessUnit)
	}
	return result, nil
}

func DeleteBusinessUnit(id string) error {
	statement, err := database.Db.Prepare("UPDATE business_unit SET is_active = ? WHERE id = ?;")
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

func GetAllBusinessUnit(userId string) ([]*model.ListBusinessUnit, error) {

	querystring := `SELECT id,org_id,sub_org_id,name FROM business_unit where created_by = ` + userId + ` and is_active = 1`

	selDB, err := database.Db.Query(querystring)
	if err != nil {
		return nil, err
	}
	var result []*model.ListBusinessUnit
	defer selDB.Close()
	for selDB.Next() {
		var businessUnit model.ListBusinessUnit

		err := selDB.Scan(&businessUnit.ID, &businessUnit.OrgID, &businessUnit.SubOrgID, &businessUnit.Name)
		if err != nil {
			return []*model.ListBusinessUnit{}, err
		}
		orgName, err := GetOrgNameById(*businessUnit.OrgID)
		if err != nil {
			return []*model.ListBusinessUnit{}, err
		}
		orgSubName, err := GetOrgNameById(*businessUnit.SubOrgID)
		if err != nil {
			return []*model.ListBusinessUnit{}, err
		}
		businessUnit.OrgName = &orgName
		businessUnit.SubOrgName = &orgSubName

		result = append(result, &businessUnit)
	}
	return result, nil
}

func GetBusinessUnitByName(userId, name string) (*model.ListBusinessUnit, error) {

	query := `SELECT id,org_id,sub_org_id,name FROM business_unit where created_by = ? and is_active = 1 and name = ? `

	selDB, err := database.Db.Query(query, userId, name)
	if err != nil {
		return &model.ListBusinessUnit{}, err
	}

	defer selDB.Close()
	var businessUnit model.ListBusinessUnit

	for selDB.Next() {

		err := selDB.Scan(&businessUnit.ID, &businessUnit.OrgID, &businessUnit.SubOrgID, &businessUnit.Name)
		if err != nil {
			return &model.ListBusinessUnit{}, err
		}
	}
	return &businessUnit, nil
}


func GetBusinessUnitByOrgIdOrSubOrgId(orgId, subOrgId string) ([]*model.ListBusinessUnit, error) {

	querystring := `SELECT id, org_id, sub_org_id, name FROM business_unit where org_id = ? and sub_org_id = ?`

	selDB, err := database.Db.Query(querystring, orgId, subOrgId)
	if err != nil {
		return nil, err
	}
	var result []*model.ListBusinessUnit
	defer selDB.Close()
	for selDB.Next() {
		var businessUnit model.ListBusinessUnit

		err := selDB.Scan(&businessUnit.ID, &businessUnit.OrgID, &businessUnit.SubOrgID, &businessUnit.Name)
		if err != nil {
			return []*model.ListBusinessUnit{}, err
		}
		orgName, err := GetOrgNameById(*businessUnit.OrgID)
		if err != nil {
			return []*model.ListBusinessUnit{}, err
		}
		orgSubName, err := GetOrgNameById(*businessUnit.SubOrgID)
		if err != nil {
			return []*model.ListBusinessUnit{}, err
		}
		businessUnit.OrgName = &orgName
		businessUnit.SubOrgName = &orgSubName

		result = append(result, &businessUnit)
	}
	return result, nil
}