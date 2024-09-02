package helper

import (
	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func GetRegionByCode(input string, requestType, userId string) (*model.Region, error) {

	ccheckReg, err := CheckRegionsByRegionCode(input)
	if err != nil {
		return nil, err
	}
	var region model.Region
	query := ""

	if ccheckReg != "" {

		if requestType == "code" {
			query = "SELECT code, name, latitude, longitude FROM regions where code = ? limit 1"
		} else {
			query = "SELECT code, name, latitude, longitude FROM regions where is_default = ? limit 1"
		}
		selDB, err := database.Db.Query(query, input)
		if err != nil {
			return nil, err
		}
		defer selDB.Close()
		selDB.Next()
		err = selDB.Scan(&region.Code, &region.Name, &region.Latitude, &region.Longitude)
		if err != nil {
			return nil, err
		}
	} else {
		query = "SELECT region_code, location_name FROM cluster_info_user where (region_code = ? and is_active = 1) and user_id = ?"
		selDB, err := database.Db.Query(query, input, userId)
		if err != nil {
			return nil, err
		}
		defer selDB.Close()
		selDB.Next()
		err = selDB.Scan(&region.Code, &region.Name)
		if err != nil {
			return nil, err
		}

	}
	return &region, nil

}

func CheckRegionsByRegionCode(regionCode string) (string, error) {

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
