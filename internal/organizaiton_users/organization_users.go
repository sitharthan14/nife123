package oragnizationUsers

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func GetOrgUserDetails(orgId string) ([]OrganizationUsers, error) {

	var orgUsers []OrganizationUsers
	selDB, err := database.Db.Query(`select u.firstName, u.lastName, ou.user_id,
	ou.joined_at, u.email, ou.id, ou.role_id
	from organization_users ou join user u on ou.user_id = u.id where ou.organization_id = ?`, orgId)
	if err != nil {
		return orgUsers, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var orgUser OrganizationUsers
		err = selDB.Scan(&orgUser.FirstName, &orgUser.LastName, &orgUser.UserId, &orgUser.JoinedAt, &orgUser.UserEmail, &orgUser.Id, &orgUser.RoleId)
		if err != nil {
			return nil, err
		}
		orgUsers = append(orgUsers, orgUser)
	}
	return orgUsers, nil
}

func AddUserToOrg(orgId, userId string, roleId int) error {
	id := uuid.New()
	statement, err := database.Db.Prepare("INSERT INTO organization_users (id,user_id,organization_id,joined_at,is_active, role_id) VALUES(?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	_, err = statement.Exec(id, userId, orgId, time.Now(), 1, roleId)
	if err != nil {
		return err
	}
	return nil
}

func GetRoleIdByUserId(userId int) (int, error) {
	statement, err := database.Db.Prepare("select role_id from user WHERE id = ?")
	if err != nil {
		return 0, err
	}
	row := statement.QueryRow(userId)

	var RoleId int

	err = row.Scan(&RoleId)
	if err != nil {
		return 0, err
	}

	return RoleId, nil
}

func UpdateTempPwdandIsUserInvite(email string, isUserInvite bool) error {
	statement, err := database.Db.Prepare("UPDATE user SET temporary_password = ? , is_user_invited = ? where email = ?")
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	_, err = statement.Exec(true, isUserInvite, email)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func GetDefaultOrganization(userId string) (string, error) {

	statement, err := database.Db.Prepare(`SELECT organization.id FROM organization_users
	INNER JOIN organization ON organization.id = organization_users.organization_id 
	WHERE organization.type = ? AND organization_users.user_id = ? `)
	if err != nil {
		return "", err
	}
	row := statement.QueryRow(true, userId)
	var orgId string
	err = row.Scan(&orgId)
	if err != nil {
		return "", err
	}

	return orgId, nil
}

func GetOrgByUserId(userId int, orgId string) (string, error) {
	statement, err := database.Db.Query("SELECT id FROM organization_users where user_id = ? && organization_id = ?", userId, orgId)
	if err != nil {
		return "", err
	}

	defer statement.Close()

	var Id string

	for statement.Next() {
		err = statement.Scan(&Id)
		if err != nil {
			return "", err
		}
	}

	return Id, nil
}

func InviteUserAddOrgs(organization []*string, userId, roleId int) error {

	count := 0

	for _, orgId := range organization {
		checkOrg, err := GetOrgByUserId(userId, *orgId)
		if err != nil {
			return err
		}
		if checkOrg == "" {
			err = AddUserToOrg(*orgId, strconv.Itoa(userId), roleId)
			if err != nil {
				return err
			}
		} else {
			count += 1
		}
	}
	if count > 0 {
		return fmt.Errorf("User Already Invited")
	}

	return nil
}
