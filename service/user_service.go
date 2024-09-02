package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	"github.com/nifetency/nife.io/internal/users"
	//	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func UpdateUserDetails(id, phone_number, company_name, location, industry, firstName, lastName, user_id string, mode bool) (model.UpdateUser, error) {
	statement, err := database.Db.Prepare("update user set phone_number = ?, company_name = ?, location = ?, industry =?, firstName=?, lastName=?, updatedBy =?, updatedAt =?, mode=? where id = ?")
	if err != nil {
		return model.UpdateUser{}, err
	}
	_, err = statement.Exec(phone_number, company_name, location, industry, firstName, lastName, user_id, time.Now().UTC(), mode, id)
	if err != nil {
		return model.UpdateUser{}, err
	}

	now := time.Now().UTC()
	res := model.UpdateUser{
		CompanyName: &company_name,
		PhoneNumber: &phone_number,
		Location:    &location,
		Industry:    &industry,
		UpdatedAt:   &now,
	}
	return res, nil
}

func UpdateInviteUserDetails(id, phone_number, location, industry, firstName, lastName string, mode bool) (model.UpdateUser, error) {
	statement, err := database.Db.Prepare("update user set phone_number = ?, location = ?, industry =?, firstName=?, lastName=?, mode=? where id = ?")
	if err != nil {
		return model.UpdateUser{}, err
	}
	_, err = statement.Exec(phone_number, location, industry, firstName, lastName, mode, id)
	if err != nil {
		return model.UpdateUser{}, err
	}

	res := model.UpdateUser{
		PhoneNumber: &phone_number,
		Location:    &location,
		Industry:    &industry,
	}
	return res, nil
}

func GetUserDetails(id string) (users.UserRegisterRequestBody, error) {
	var user users.UserRegisterRequestBody
	query := "SELECT email FROM user where id = ?"

	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return user, err
	}
	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&user.Email)
	if err != nil {
		return user, err
	}

	return user, nil
}

func GetUserRole(id string) (string, error) {
	// var user users.UserRegisterRequestBody
	query := "SELECT role_id FROM user where id = ?"

	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return "", err
	}
	defer selDB.Close()
	var roleId string
	selDB.Next()
	err = selDB.Scan(&roleId)
	if err != nil {
		return "", err
	}

	return roleId, nil
}

func GetUserPassword(id string) (users.UserRegisterRequestBody, error) {
	var user users.UserRegisterRequestBody
	query := "SELECT password FROM user where id = ?"

	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return user, err
	}
	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&user.Password)
	if err != nil {
		return user, err
	}

	return user, nil
}

func UpdateUserPassword(id string, HashPassword string, user_id string) (model.Password, error) {
	statement, err := database.Db.Prepare("update user set password = ?, updatedBy =?, updatedAt =? where id = ?")
	if err != nil {
		return model.Password{}, err
	}
	defer statement.Close()
	_, err = statement.Exec(HashPassword, user_id, time.Now().UTC(), id)
	if err != nil {
		return model.Password{}, err
	}
	message := "new password updated successfully"
	res := model.Password{
		Message: &message,
	}
	return res, nil
}

func UpdateUserInActive(userId, organizationId string, isActive bool) error {
	statement, err := database.Db.Prepare("update organization_users set is_active = ? where user_id = ? and organization_id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(isActive, userId, organizationId)
	if err != nil {
		return err
	}
	return nil
}
func GetUserByOrganization(userId, organizationId string) (string, error) {

	query := "select id from organization_users where user_id = ? and organization_id = ?"

	selDB, err := database.Db.Query(query, userId, organizationId)
	if err != nil {
		return "", err
	}
	var id string
	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func GetById(id string) (model.GetUserByID, error) {
	var user model.GetUserByID
	query := "SELECT email, company_name, phone_number, location, industry, firstName, lastName, sso_type, is_free_plan,profile_image_url,is_active,is_delete, company_id, user_profile_created, role_id, mode, slack_webhook_url FROM user where id = ?"

	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return user, err
	}
	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&user.Email, &user.CompanyName, &user.PhoneNumber, &user.Location, &user.Industry, &user.FirstName, &user.LastName, &user.SsoType, &user.FreePlan, &user.ProfileImageURL, &user.IsActive, &user.IsDelete, &user.CompanyID, &user.UserProfileCreated, &user.RoleID, &user.Mode, &user.SlackWebhookURL)
	if err != nil {
		return user, err
	}

	return user, nil
}

func GetInviteUserByCompanyId(companyId string) ([]*model.GetUserByID, error) {

	query := `SELECT id, email, company_name, phone_number, location, industry, firstName, lastName, sso_type, is_free_plan,profile_image_url , company_id, is_active, is_delete, role_id, user_profile_created
	FROM user where company_id = ? and is_delete = 0`

	selDB, err := database.Db.Query(query, companyId)
	if err != nil {
		return []*model.GetUserByID{}, err
	}
	defer selDB.Close()
	result := []*model.GetUserByID{}
	for selDB.Next() {
		var inviteUser model.GetUserByID

		err = selDB.Scan(&inviteUser.ID, &inviteUser.Email, &inviteUser.CompanyName, &inviteUser.PhoneNumber, &inviteUser.Location, &inviteUser.Industry, &inviteUser.FirstName, &inviteUser.LastName, &inviteUser.SsoType, &inviteUser.FreePlan, &inviteUser.ProfileImageURL, &inviteUser.CompanyID, &inviteUser.IsActive, &inviteUser.IsDelete, &inviteUser.RoleID, &inviteUser.UserProfileCreated)
		if err != nil {
			return []*model.GetUserByID{}, err
		}

		result = append(result, &inviteUser)
	}

	return result, nil
}

func GetInviteUserCountByCompanyId(companyId string) (int, error) {

	query := `SELECT count(email) FROM user where company_id = ? and is_delete = 0`

	selDB, err := database.Db.Query(query, companyId)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()
	var inviteUserCount int
	for selDB.Next() {

		err = selDB.Scan(&inviteUserCount)
		if err != nil {
			return 0, err
		}
	}

	return inviteUserCount - 1, nil
}

func UpdateUserActive(userid string, isactive bool) error {
	statement, err := database.Db.Prepare("update user set is_active = ? where id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(isactive, userid)
	if err != nil {
		return err
	}
	return nil
}

func UpdateUserDetele(userid string, isDelete bool) error {
	statement, err := database.Db.Prepare("update user set is_delete = ? where id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(isDelete, userid)
	if err != nil {
		return err
	}
	return nil
}

func UserProfileCreated(userid string, userProfileCreated bool) error {
	statement, err := database.Db.Prepare("update user set user_profile_created = ? where id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(userProfileCreated, userid)
	if err != nil {
		return err
	}
	return nil
}

func UpdateImageByCompanyId(logoURL, companyid string) error {
	statement, err := database.Db.Prepare("UPDATE company SET logo = ? WHERE id = ?;")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(&logoURL, &companyid)
	if err != nil {
		return err
	}
	return nil
}

func GetCompanyLogo(companyId string) (string, error) {
	query := "SELECT logo FROM company where id = ?"

	selDB, err := database.Db.Query(query, companyId)
	if err != nil {
		return "", err
	}
	var companyLogo string
	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&companyLogo)
	if err != nil {
		return "", err
	}
	return companyLogo, nil
}

func UpdateInviteUserInactive(userId string) error {
	statement, err := database.Db.Prepare("UPDATE user SET is_active = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(false, userId)
	if err != nil {
		return err
	}
	return nil
}

func GetUserActivitiesByUserId(userId string, first int) ([]*model.UserActivities, error) {

	query := `SELECT id, type, user_id, activities, message, is_read, created_at, ref_id FROM activity where user_id = ? and type != "LOGIN" ORDER BY created_at DESC LIMIT ?;`

	selDB, err := database.Db.Query(query, userId, first)
	if err != nil {
		return []*model.UserActivities{}, err
	}

	defer selDB.Close()
	result := []*model.UserActivities{}
	for selDB.Next() {
		var activities model.UserActivities

		err = selDB.Scan(&activities.ID, &activities.Type, &activities.UserID, &activities.Activities, &activities.Message, &activities.IsRead, &activities.CreatedAt, &activities.ReferenceID)
		if err != nil {
			return []*model.UserActivities{}, err
		}
		// fmt.Println(string(*activities.ReferenceID))

		if *activities.Type == "ORGANIZATION" && *activities.Activities != "DELETED" {
			orgDet, err := GetOrganization(*activities.ReferenceID, "")
			if err != nil {
				return []*model.UserActivities{}, err
			}
			activities.OrganizationName = orgDet.Name
			// *activities.SubOrganizationName = ""
		}

		if *activities.Type == "SUB ORGANIZATION" && *activities.Activities != "DELETED" {
			subOrgDet, err := GetOrganization(*activities.ReferenceID, "")
			if err != nil {
				return []*model.UserActivities{}, err
			}
			// *activities.OrganizationName = ""
			activities.SubOrganizationName = subOrgDet.Name
		}
		if *activities.Type == "APP" {
			_, _, orgId, subOrgId, err := GetAppByAppId(*activities.ReferenceID)
			if err != nil {
				// return []*model.UserActivities{}, err
			}
			orgDet, err := GetOrganization(orgId, "")
			if err != nil {
				// return []*model.UserActivities{}, err
			}
			var subOrgDet *model.Organization
			activities.SubOrganizationName = nil
			if subOrgId != "" {
				subOrgDet, err = GetOrganization(subOrgId, "")
				if err != nil {
					// return []*model.UserActivities{}, err
				}
				activities.SubOrganizationName = subOrgDet.Name
			}
			activities.OrganizationName = orgDet.Name
		}
		result = append(result, &activities)
	}

	return result, nil
}

func GetUserActivitiesByUserIdAndDates(userId, startDate, endDate string) ([]*model.UserActivities, error) {
	query := `SELECT id, type, user_id, activities, message, is_read, created_at, ref_id FROM activity
	where (created_at <= ? and created_at>= ?) and user_id = ? and type != "LOGIN" order by created_at desc;`

	selDB, err := database.Db.Query(query, endDate, startDate, userId)
	if err != nil {
		return []*model.UserActivities{}, err
	}
	defer selDB.Close()
	result := []*model.UserActivities{}
	for selDB.Next() {
		var activities model.UserActivities

		err = selDB.Scan(&activities.ID, &activities.Type, &activities.UserID, &activities.Activities, &activities.Message, &activities.IsRead, &activities.CreatedAt, &activities.ReferenceID)
		if err != nil {
			return []*model.UserActivities{}, err
		}
		// fmt.Println(string(*activities.ReferenceID))

		if *activities.Type == "ORGANIZATION" && *activities.Activities != "DELETED" {
			orgDet, err := GetOrganization(*activities.ReferenceID, "")
			if err != nil {
				return []*model.UserActivities{}, err
			}
			activities.OrganizationName = orgDet.Name
			// *activities.SubOrganizationName = ""
		}

		if *activities.Type == "SUB ORGANIZATION" && *activities.Activities != "DELETED" {
			subOrgDet, err := GetOrganization(*activities.ReferenceID, "")
			if err != nil {
				return []*model.UserActivities{}, err
			}
			// *activities.OrganizationName = ""
			activities.SubOrganizationName = subOrgDet.Name
		}
		if *activities.Type == "APP" {
			_, _, orgId, subOrgId, err := GetAppByAppId(*activities.ReferenceID)
			if err != nil {
				// return []*model.UserActivities{}, err
			}
			orgDet, err := GetOrganization(orgId, "")
			if err != nil {
				// return []*model.UserActivities{}, err
			}
			var subOrgDet *model.Organization
			activities.SubOrganizationName = nil
			if subOrgId != "" {
				subOrgDet, err = GetOrganization(subOrgId, "")
				if err != nil {
					// return []*model.UserActivities{}, err
				}
				activities.SubOrganizationName = subOrgDet.Name
			}
			activities.OrganizationName = orgDet.Name
		}
		activeCount, err := GetAppActiveAndDeleteCount(startDate, endDate, userId, "Active")
		if err != nil {
			return []*model.UserActivities{}, err
		}
		deletedCount, err := GetAppActiveAndDeleteCount(startDate, endDate, userId, "Terminated")
		if err != nil {
			return []*model.UserActivities{}, err
		}

		appCount := model.AppCountsDetails{
			ActiveApps:  &activeCount,
			DeletedApps: &deletedCount,
		}
		activities.AppsCount = &appCount

		result = append(result, &activities)
	}

	return result, nil
}

//--------------------------------------------------------------------------------------------------------------------------

// func GetUserActivitiesByUserIdAndDates(userId, startDate, endDate string) ([]*model.UserActivities, error) {
// 	startTime := time.Now()

// 	query := `SELECT id, type, user_id, activities, message, is_read, created_at, ref_id FROM activity
// 	WHERE (created_at <= ? and created_at >= ?) AND user_id = ? AND type != "LOGIN" ORDER BY created_at DESC;`

// 	selDB, err := database.Db.Query(query, endDate, startDate, userId)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer selDB.Close()

// 	type ActivityResult struct {
// 		activities *model.UserActivities
// 		err        error
// 	}

// 	activityChan := make(chan *ActivityResult)
// 	var wg sync.WaitGroup

// 	for selDB.Next() {
// 		var activities model.UserActivities
// 		err := selDB.Scan(&activities.ID, &activities.Type, &activities.UserID, &activities.Activities, &activities.Message, &activities.IsRead, &activities.CreatedAt, &activities.ReferenceID)
// 		if err != nil {
// 			return nil, err
// 		}

// 		wg.Add(1)
// 		go func(a model.UserActivities) {
// 			defer wg.Done()

// 			if *a.Type == "ORGANIZATION" && *a.Activities != "DELETED" {
// 				orgDet, err := GetOrganization(*a.ReferenceID, "")
// 				if err != nil {
// 					activityChan <- &ActivityResult{nil, err}
// 					return
// 				}
// 				a.OrganizationName = orgDet.Name
// 			}

// 			if *a.Type == "SUB ORGANIZATION" && *a.Activities != "DELETED" {
// 				subOrgDet, err := GetOrganization(*a.ReferenceID, "")
// 				if err != nil {
// 					activityChan <- &ActivityResult{nil, err}
// 					return
// 				}
// 				a.SubOrganizationName = subOrgDet.Name
// 			}

// 			if *a.Type == "APP" {
// 				_, _, orgId, subOrgId, err := GetAppByAppId(*a.ReferenceID)
// 				if err != nil {
// 					return
// 				}
// 				orgDet, err := GetOrganization(orgId, "")
// 				if err != nil {
// 					return
// 				}
// 				var subOrgDet *model.Organization
// 				a.SubOrganizationName = nil
// 				if subOrgId != "" {
// 					subOrgDet, err = GetOrganization(subOrgId, "")
// 					if err != nil {
// 						return
// 					}
// 					a.SubOrganizationName = subOrgDet.Name
// 				}
// 				a.OrganizationName = orgDet.Name
// 			}

// 			activeCount, err := GetAppActiveAndDeleteCount(startDate, endDate, userId, "Active")
// 			if err != nil {
// 				activityChan <- &ActivityResult{nil, err}
// 				return
// 			}

// 			deletedCount, err := GetAppActiveAndDeleteCount(startDate, endDate, userId, "Terminated")
// 			if err != nil {
// 				activityChan <- &ActivityResult{nil, err}
// 				return
// 			}

// 			appCount := model.AppCountsDetails{
// 				ActiveApps:  &activeCount,
// 				DeletedApps: &deletedCount,
// 			}
// 			a.AppsCount = &appCount

// 			activityChan <- &ActivityResult{&a, nil}
// 		}(activities)
// 	}

// 	go func() {
// 		wg.Wait()
// 		close(activityChan)
// 	}()

// 	result := []*model.UserActivities{}
// 	for activityResult := range activityChan {
// 		if activityResult.err != nil {
// 			return nil, activityResult.err
// 		}
// 		if activityResult.activities != nil {
// 			result = append(result, activityResult.activities)
// 		}
// 	}

// 	endTime := time.Now()
// 	elapsed := endTime.Sub(startTime)
// 	fmt.Println(elapsed)
// 	return result, nil
// }

//--------------------------------------------------------------------------------------------------------------------------

func GetAppActiveAndDeleteCount(startDate, endDate, userId, status string) (int, error) {
	query := `SELECT COUNT(name) FROM app where (createdAt >= ? and createdAt<= ?) and (createdBy = ? and status = ?);`

	selDB, err := database.Db.Query(query, startDate, endDate, userId, status)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()
	var count int
	for selDB.Next() {

		err = selDB.Scan(&count)
		if err != nil {
			return 0, err
		}
	}

	return count, nil
}

func UpdateActivityAsRead(id string, isRead bool) error {
	statement, err := database.Db.Prepare("UPDATE activity SET is_read = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(isRead, id)
	if err != nil {
		return err
	}
	return nil
}

func InsertByohDetails(userId string, byoh model.ByohRequest) error {
	statement, err := database.Db.Prepare("INSERT INTO requested_region(id, organization_id ,user_name, password, ip_address, name, region, status, created_by, created_at) VALUES(?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, byoh.OrganizationID, byoh.UserName, byoh.Password, byoh.IPAddress, byoh.Name, byoh.Region, "Pending", userId, time.Now())
	if err != nil {
		return err
	}

	return nil
}

func GetTotalDeploymentCountByUser(userId, startDate, endDate string, deployAndRedeploy bool) (int, error) {
	var query string
	// '2023-08-29''2023-09-06'
	if deployAndRedeploy {
		query = `SELECT COUNT(id) FROM activity WHERE (type = "APP"  AND activities = "DEPLOYED")  AND (user_id = ? AND DATE(created_at) BETWEEN ? AND ?);
		`
	} else {
		query = `SELECT COUNT(id) FROM activity WHERE (type = "APP"  AND activities = "REDEPLOYED")  AND (user_id = ? AND DATE(created_at) BETWEEN ? AND ?);
		`
	}
	selDB, err := database.Db.Query(query, userId, startDate, endDate)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()
	var deploymentCount int
	for selDB.Next() {

		err = selDB.Scan(&deploymentCount)
		if err != nil {
			return 0, err
		}
	}

	return deploymentCount, nil
}

func GetDeploymentDetailsByUser(userDet []*model.GetUserByID, startDate, endDate string) ([]*model.UserDeploymentDetailCount, error) {

	result := []*model.UserDeploymentDetailCount{}
	for _, userId := range userDet {
		var userRecords *model.UserDeploymentDetailCount
		totalDeploymentCount, err := GetTotalDeploymentCountByUser(*userId.ID, startDate, endDate, true)
		if err != nil {
			return nil, fmt.Errorf(err.Error())
		}
		totalReDeploymentCount, err := GetTotalDeploymentCountByUser(*userId.ID, startDate, endDate, false)
		if err != nil {
			return nil, fmt.Errorf(err.Error())
		}

		deploymentByDate, err := GetdeploymentCountByDate(*userId.ID, startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf(err.Error())
		}

		reDeploymentByDate, err := GetRedeploymentCountByDate(*userId.ID, startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf(err.Error())
		}

		userRecords = &model.UserDeploymentDetailCount{
			UserName:        userId.FirstName,
			Email:           userId.Email,
			CompanyName:     &userId.CompanyName,
			RoleID:          &userId.RoleID,
			TotalDeployed:   &totalDeploymentCount,
			TotalReDeployed: &totalReDeploymentCount,
			DeployData:      deploymentByDate,
			ReDeployData:    reDeploymentByDate,
		}
		result = append(result, userRecords)
	}

	return result, nil
}

func GetdeploymentCountByDate(userId, startDate, endDate string) ([]*model.DeploymentCountByDate, error) {
	query := `SELECT DATE_FORMAT(created_at, '%y-%m-%d') AS formatted_date, COUNT(id) AS count FROM activity WHERE type = "APP" AND activities = "DEPLOYED"  AND user_id = ?
	AND DATE(created_at) BETWEEN ? AND ?
  GROUP BY formatted_date
  ORDER BY formatted_date;`
	selDB, err := database.Db.Query(query, userId, startDate, endDate)
	if err != nil {
		return []*model.DeploymentCountByDate{}, err
	}
	defer selDB.Close()
	result := []*model.DeploymentCountByDate{}
	for selDB.Next() {
		var getAllDate model.DeploymentCountByDate
		err = selDB.Scan(&getAllDate.Date, &getAllDate.Deployed)
		if err != nil {
			return []*model.DeploymentCountByDate{}, err
		}
		var (
			layoutISO = "06-01-02"
			layoutUS  = "Jan 2"
		)
		date := getAllDate.Date
		t, _ := time.Parse(layoutISO, *date)
		datess := t.Format(layoutUS)
		getAllDate.Date = &datess

		result = append(result, &getAllDate)
	}

	return result, nil
}

func GetRedeploymentCountByDate(userId, startDate, endDate string) ([]*model.ReDeploymentCountByDate, error) {
	query := `SELECT DATE_FORMAT(created_at, '%y-%m-%d') AS formatted_date, COUNT(id) AS count FROM activity WHERE type = "APP" AND activities = "REDEPLOYED"  AND user_id = ?
	AND DATE(created_at) BETWEEN ? AND ?
  GROUP BY formatted_date
  ORDER BY formatted_date;`
	selDB, err := database.Db.Query(query, userId, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	reDepResult := []*model.ReDeploymentCountByDate{}
	for selDB.Next() {
		var reDepRes model.ReDeploymentCountByDate
		err = selDB.Scan(&reDepRes.Date, &reDepRes.ReDeployed)
		if err != nil {
			return nil, err
		}

		var (
			layoutISO = "06-01-02"
			layoutUS  = "Jan 2"
		)
		date := reDepRes.Date
		t, _ := time.Parse(layoutISO, *date)
		datess := t.Format(layoutUS)
		reDepRes.Date = &datess

		reDepResult = append(reDepResult, &reDepRes)
	}

	return reDepResult, nil
}

func GetInviteUserByOrganizationId(organizationId string) ([]*model.GetUserByID, error) {

	query := `SELECT user.id, user.email, user.company_name, user.phone_number, user.location, user.industry, user.firstName, user.lastName, user.sso_type, user.is_free_plan, user.profile_image_url, user.is_active, user.is_delete, user.company_id, user.user_profile_created, user.role_id, user.mode
	FROM organization_users 
	inner join user on user.id = organization_users.user_id
	where organization_id = ?`

	selDB, err := database.Db.Query(query, organizationId)
	if err != nil {
		return []*model.GetUserByID{}, err
	}
	defer selDB.Close()
	result := []*model.GetUserByID{}
	for selDB.Next() {
		var inviteUser model.GetUserByID

		err = selDB.Scan(&inviteUser.ID, &inviteUser.Email, &inviteUser.CompanyName, &inviteUser.PhoneNumber, &inviteUser.Location, &inviteUser.Industry, &inviteUser.FirstName, &inviteUser.LastName, &inviteUser.SsoType, &inviteUser.FreePlan, &inviteUser.ProfileImageURL, &inviteUser.IsActive, &inviteUser.IsDelete, &inviteUser.CompanyID, &inviteUser.UserProfileCreated, &inviteUser.RoleID, &inviteUser.Mode)
		if err != nil {
			return []*model.GetUserByID{}, err
		}

		result = append(result, &inviteUser)
	}

	return result, nil
}

func isNotEmptyOrSpaces(input string) bool {
	return strings.TrimSpace(input) != ""
}

func ValidateFields(firstName, lastName, location, industry string) bool {
	return isNotEmptyOrSpaces(firstName) &&
		isNotEmptyOrSpaces(lastName) &&
		isNotEmptyOrSpaces(location) &&
		isNotEmptyOrSpaces(industry)
}

func IsValidMobileNumber(number string) bool {
	code := `^(?:\+\d{1,4}\s?)?\(?\d{1,4}\)?[\s.-]?\d{1,10}$`
	re := regexp.MustCompile(code)

	return re.MatchString(number)
}

func CheckUserRole(userId string) (int, error) {

	var usersId int
	usersId, _ = strconv.Atoi(userId)
	roleId, _, err := users.GetUserRole(usersId)
	if err != nil {
		return 0, err
	}
	if roleId == 3 {
		return 0, fmt.Errorf("Viewer role is restricted to view-only access")
	}
	return roleId, nil
}

func UpdateUserSlackWebhookURL(userid, slackUrl string) error {
	statement, err := database.Db.Prepare("update user set slack_webhook_url = ? where id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(slackUrl, userid)
	if err != nil {
		return err
	}
	return nil
}

func GetUserSlackWebhookURL(userId string) (string, error) {

	query := "select slack_webhook_url from user where id = ?"

	selDB, err := database.Db.Query(query, userId)
	if err != nil {
		return "", err
	}
	var slackUrl string
	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&slackUrl)
	if err != nil {
		return "", err
	}
	return slackUrl, nil
}