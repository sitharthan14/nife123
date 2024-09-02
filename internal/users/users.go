package users

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	// "github.com/nifetency/nife.io/internal/github"
	"log"
	"net/http"

	// "github.com/aws/aws-sdk-go/service"

	"github.com/google/uuid"
	"gopkg.in/go-playground/validator.v9"

	"strconv"
	"time"

	hashers "github.com/meehow/go-django-hashers"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal/decode"
	"github.com/nifetency/nife.io/internal/github"
	"github.com/nifetency/nife.io/internal/stripes"
	"github.com/stripe/stripe-go/v71"

	cli_session "github.com/nifetency/nife.io/internal/cli_session"
	oragnizationUsers "github.com/nifetency/nife.io/internal/organizaiton_users"
	organizationInfo "github.com/nifetency/nife.io/internal/organization_info"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	singleSignOn "github.com/nifetency/nife.io/internal/single_sign_on"

	// _helper "github.com/nifetency/nife.io/pkg/helper"

	// decrypt "github.com/nifetency/nife.io/pkg/helper"
	// "github.com/stripe/stripe-go/v71"

	"github.com/nifetency/nife.io/pkg/jwt"
	// "github.com/stripe/stripe-go/v71/customer"
	"github.com/alecthomas/log4go"
)

type User struct {
	ID                     string      `json:"id"`
	Email                  string      `json:"email"`
	Password               string      `json:"password"`
	PhoneNumber            string      `json:"phoneNumber"`
	CompanyName            string      `json:"companyName"`
	Industry               string      `json:"industry"`
	Location               string      `json:"location"`
	CreatedAt              time.Time   `josn:"createdAt"`
	FirstName              string      `json:"firstName"`
	LastName               string      `json:"lastName"`
	SSOType                string      `json:"ssoType"`
	CustomerStripeId       string      `json:"customerStripeId"`
	StripeProductId        string      `json:"stripePriceId"`
	RoleId                 int         `json:"role_id"`
	CompanyId              string      `json:"companyId"`
	WebsiteURL             string      `json:"website_url"`
	LinkedinURL            string      `json:"linkedin_url"`
	EstimatedNumEmployees  int         `json:"estimated_num_employees"`
	FoundedYear            int         `json:"founded_year"`
	TotalFunding           interface{} `json:"total_funding"`
	LatestFundingRoundDate interface{} `json:"latest_funding_round_date"`
}

type Company struct {
	ID          string `json:"id"`
	CompanyName string `json:"companyName"`
	ExternalId  string `json:"externalId"`
}

type PlanAndPermission struct {
	Id                          string `json:"id"`
	PlanName                    string `json:"planName"`
	Apps                        int    `json:"apps"`
	WorkloadManagement          bool   `json:"workloadManagement"`
	OrganisationManagement      bool   `json:"organizationManagement"`
	InviteUserLimit             int    `json:"inviteUserLimit"`
	ApplicationHealthDashboard  bool   `json:"applicationHealthDashboard"`
	Byoh                        bool   `json:"byoh"`
	Storage                     bool   `json:"storage"`
	VersionControlPanel         bool   `json:"versionControlPanel"`
	SingleSignOn                bool   `json:"singleSignOn"`
	OrganizationCount           int    `json:"organizationCount"`
	SubOrganizationCount        int    `json:"subOrganizationCount"`
	BusinessunitCount           int    `json:"businessunitCount"`
	CustomDomain                bool   `json:"customDomain"`
	AppNotification             bool   `json:"appNotification"`
	Secret                      bool   `json:"secret"`
	MonitoringPlatform          bool   `json:"monitoringPlatform"`
	AlertsAdvisories            bool   `json:"alertsAdvisories"`
	AuditLogs                   int    `json:"auditLogs"`
	SslSecurity                 bool   `json:"sslSecurity"`
	InfrastructureConfiguration int    `json:"infrastructureConfiguration"`
	Replicas                    int    `json:"replicas"`
	K8sRegions                  int    `json:"k8sRegions"`
}

type Activity struct {
	Id         string `json:"id"`
	Type       string `json:"type"`
	UserId     string `json:"user_id"`
	Activities string `json:"activities"`
	Message    string `json:"message"`
	RefId      string `json:"ref_id"`
}

func (user *User) Create() (int64, error) {
	statement, err := database.Db.Prepare("INSERT INTO user(email,password,company_name,phone_number,location,industry,firstName,lastName, createdAt, first_time_user, role_id, company_id, company_website_url, company_linkedin_url,estimated_num_employees, founded_year, total_funding, latest_funding_round_date) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return 0, err
	}
	hashedPassword, _ := HashPassword(user.Password)
	defer statement.Close()
	res, err := statement.Exec(user.Email, hashedPassword, user.CompanyName, user.PhoneNumber, user.Location, user.Industry, user.FirstName, user.LastName, user.CreatedAt, true, user.RoleId, user.CompanyId, user.WebsiteURL, user.LinkedinURL, user.EstimatedNumEmployees, user.FoundedYear, user.TotalFunding, user.LatestFundingRoundDate)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (user *User) AccountCreate() (int64, error) {
	statement, err := database.Db.Prepare("INSERT INTO user(email,password,company_name,phone_number,location,industry,firstName,lastName,createdAt,sso_type,first_time_user,role_id, company_id, company_website_url, company_linkedin_url,estimated_num_employees, founded_year, total_funding, latest_funding_round_date) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return 0, err
	}
	defer statement.Close()
	res, err := statement.Exec(user.Email, "", user.CompanyName, user.PhoneNumber, user.Location, user.Industry, user.FirstName, user.LastName, user.CreatedAt, user.SSOType, true, user.RoleId, user.CompanyId, user.WebsiteURL, user.LinkedinURL, user.EstimatedNumEmployees, user.FoundedYear, user.TotalFunding, user.LatestFundingRoundDate)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (user *User) Authenticate() (bool, error) {
	statement, err := database.Db.Prepare("select password from user WHERE email = ?")
	if err != nil {
		return false, err
	}
	row := statement.QueryRow(user.Email)

	var hashedPassword string
	defer statement.Close()
	err = row.Scan(&hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println(err)
			return false, err
		} else {
			fmt.Println(hashedPassword)
			return false, err
		}
	}

	return CheckPasswordHash(user.Password, hashedPassword), err
}

func (company *Company) InsertCompanyDetails() error {
	statement, err := database.Db.Prepare("INSERT INTO company (id, company_name, external_id) VALUES (?,?,?)")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(company.ID, company.CompanyName, company.ExternalId)
	if err != nil {
		return err
	}
	return err
}

// GetUserIdByUsername check if a user exists in database by given username
func GetUserIdByEmail(email string) (int, error) {
	selDB, err := database.Db.Query("select id from user WHERE email = ?", email)
	if err != nil {
		return -1, err
	}
	defer selDB.Close()
	var id int
	for selDB.Next() {
		err = selDB.Scan(&id)
		if err != nil {
			return -1, err
		}
	}
	return id, nil
}

func GetUserRole(userId int) (int, string, error) {
	selDB, err := database.Db.Query("select role.id,role.name from role INNER JOIN user ON user.role_id = role.id where user.id = ?", userId)
	if err != nil {
		return -1, "", err
	}
	defer selDB.Close()
	var roleId int
	var role string
	for selDB.Next() {
		err = selDB.Scan(&roleId, &role)
		if err != nil {
			return -1, "", err
		}
	}
	return roleId, role, nil
}

func FreePlanDetails(id int) (bool, error) {

	query := "SELECT is_free_plan FROM user where id = ?"

	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return false, err
	}
	defer selDB.Close()
	var isFreePlan bool
	selDB.Next()
	err = selDB.Scan(&isFreePlan)
	if err != nil {
		return false, err
	}

	return isFreePlan, nil

}

func UpdateFreePlanFlag(id int) error {
	statement, err := database.Db.Prepare("UPDATE user set is_free_plan = ? where id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer statement.Close()
	_, err = statement.Exec(false, id)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func GetCustomerStripeId(id int) (string, error) {

	query := "SELECT customer_stripes_id FROM user where id = ?"

	selDB, err := database.Db.Query(query, id)
	if err != nil {
		return "", err
	}
	defer selDB.Close()
	var customerStripesId string
	selDB.Next()
	err = selDB.Scan(&customerStripesId)
	if err != nil {
		return "", err
	}

	return customerStripesId, nil

}

func GetFirsttimeUser(email string) (bool, error) {
	statement, err := database.Db.Prepare("select first_time_user from user WHERE email = ?")
	if err != nil {
		log.Println(err)
	}
	row := statement.QueryRow(email)

	var firstTimeUser bool
	defer statement.Close()
	err = row.Scan(&firstTimeUser)
	if err != nil {
		return false, err
	}

	return firstTimeUser, nil
}

func IsEmailVerified(email string) (bool, error) {
	statement, err := database.Db.Prepare("select is_email_verified from user WHERE email = ?")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(email)

	var verifyEmail bool
	defer statement.Close()
	err = row.Scan(&verifyEmail)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return false, err
	}

	return verifyEmail, nil
}

// GetUserIdByUsername check if a user exists in database by given username
func ChangeFirstUserOption(email string) (int, error) {
	statement, err := database.Db.Prepare("UPDATE user set first_time_user = ? where email = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer statement.Close()
	_, err = statement.Exec(false, email)
	if err != nil {
		log.Println(err)
		return 0, err
	}

	return 1, nil
}

func GetStripeIdByuserId(id int) (string, error) {
	statement, err := database.Db.Prepare("select customer_stripes_id from user WHERE id = ?")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(id)

	var stripesId string
	defer statement.Close()
	err = row.Scan(&stripesId)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return "", err
	}

	return stripesId, nil
}

func GetEmailByCompanyName(email string) (string, error) {
	statement, err := database.Db.Prepare("select company_name from user WHERE email = ?")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(email)

	var companyName string
	defer statement.Close()
	err = row.Scan(&companyName)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return "", err
	}

	return companyName, nil
}

func GetAdminByCompanyName(companyName string, roleId int) (string, string, error) {
	statement, err := database.Db.Prepare("select firstName , email from user WHERE company_name = ? and role_id = ?")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(companyName, roleId)

	var email string
	var name string
	defer statement.Close()
	err = row.Scan(&email, &name)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return "", "", err
	}

	return name, email, nil
}

func GetAdminByemail(emails string, roleId int) (string, string, error) {
	statement, err := database.Db.Prepare("select firstName , email from user WHERE email = ? and role_id = ?")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(emails, roleId)

	var email string
	var name string
	defer statement.Close()
	err = row.Scan(&email, &name)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return "", "", err
	}

	return name, email, nil
}

func GetAdminByCompanyNameAndEmail(companyName string) (string, error) {
	statement, err := database.Db.Prepare("select external_id from company WHERE company_name = ?")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(companyName)

	var email string
	defer statement.Close()
	err = row.Scan(&email)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return "", err
	}

	return email, nil
}

func GetCompanyNameByEmail(email string) (string, error) {
	statement, err := database.Db.Prepare("select company_name from company WHERE external_id = ?")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(email)

	var companyName string
	defer statement.Close()
	err = row.Scan(&companyName)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return "", err
	}

	return companyName, nil
}

func CheckCompanyExists(companyName string) (string, error) {
	statement, err := database.Db.Prepare("select company_name from user WHERE company_name = ?")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(companyName)

	// var companyName string
	defer statement.Close()
	err = row.Scan(&companyName)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
			return "", err
		}
		return "", err
	}

	return companyName, nil
}

func CheckOrganizationExists(companyName string) (string, error) {
	statement, err := database.Db.Prepare("select name from organization WHERE name = ?;")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(companyName)

	// var companyName string
	defer statement.Close()
	err = row.Scan(&companyName)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
			return "", err
		}
		return "", err
	}

	return companyName, nil
}

// GetUserByID check if a user exists in database and return the user object.
func GetEmailById(userId string) (User, error) {
	statement, err := database.Db.Prepare("select email, firstName, lastName, company_name, role_id from user WHERE id = ?")
	if err != nil {
		log.Println(err)
	}
	row := statement.QueryRow(userId)

	var email string
	var firstName string
	var lastName string
	var companyName string
	var roleId int
	defer statement.Close()
	err = row.Scan(&email, &firstName, &lastName, &companyName, &roleId)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return User{}, err
	}

	return User{ID: userId, Email: email, FirstName: firstName, LastName: lastName, CompanyName: companyName, RoleId: roleId}, nil
}

func GetUserNameByEmail(email string) (User, error) {
	statement, err := database.Db.Prepare("select firstName, lastName, company_name, role_id, customer_stripes_id, createdAt from user WHERE email = ?")
	if err != nil {
		log.Println(err)
	}
	row := statement.QueryRow(email)

	var firstName string
	var lastName string
	var companyName string
	var roleId int
	var customerStripesId string
	var createdAt time.Time
	defer statement.Close()
	err = row.Scan(&firstName, &lastName, &companyName, &roleId, &customerStripesId, &createdAt)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return User{}, err
	}

	return User{FirstName: firstName, LastName: lastName, CompanyName: companyName, RoleId: roleId, CustomerStripeId: customerStripesId, CreatedAt: createdAt}, nil
}

func GetUserCreatedTime(email string) (User, error) {
	statement, err := database.Db.Prepare("select createdAt from user WHERE email = ?")
	if err != nil {
		log.Println(err)
	}
	defer statement.Close()
	row := statement.QueryRow(email)
	var createdAt time.Time
	err = row.Scan(&createdAt)

	return User{CreatedAt: createdAt}, nil
}

func GetUsersNameByEmail(email string) (User, error) {
	statement, err := database.Db.Prepare("select firstName, lastName, company_name, role_id, createdAt from user WHERE email = ?")
	if err != nil {
		log.Println(err)
	}
	row := statement.QueryRow(email)

	var firstName string
	var lastName string
	var companyName string
	var roleId int
	var createdAt time.Time
	defer statement.Close()
	err = row.Scan(&firstName, &lastName, &companyName, &roleId, &createdAt)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return User{}, err
	}

	return User{FirstName: firstName, LastName: lastName, CompanyName: companyName, RoleId: roleId, CreatedAt: createdAt}, nil
}

func GetIdByCompanyName(compName string) (int, error) {

	selDB, err := database.Db.Query("select id from user WHERE company_name = ?", compName)
	if err != nil {
		return -1, err
	}
	defer selDB.Close()
	var id int
	for selDB.Next() {
		err = selDB.Scan(&id)
		if err != nil {
			return -1, err
		}
	}
	return id, nil
}

func GetCompanyNameById(id string) (string, error) {

	selDB, err := database.Db.Query("SELECT company_name FROM user where id = ?", id)
	if err != nil {
		return "", err
	}
	defer selDB.Close()
	var companyName string
	for selDB.Next() {
		err = selDB.Scan(&companyName)
		if err != nil {
			return "", err
		}
	}
	return companyName, nil
}

// HashPassword hashes given password
func HashPassword(password string) (string, error) {
	hashers.DefaultHasher = "md5"
	hashedPassword, err := hashers.MakePassword(password)
	return hashedPassword, err
}

// CheckPassword hash compares raw password with it's hashed values
func CheckPasswordHash(password, hash string) bool {
	ok, err := hashers.CheckPassword(password, hash)
	return ok && err == nil
}

func CheckSSOEmail(ssoEmail string) (int, error) {
	statement, err := database.Db.Prepare("select id from user WHERE email = ?")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(ssoEmail)

	var Id int
	defer statement.Close()
	err = row.Scan(&Id)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return 0, err
	}

	return Id, nil

}

func InsertCustomerStripesId(custId, email string) error {
	statement, err := database.Db.Prepare("UPDATE user set customer_stripes_id = ? where email = ?")
	if err != nil {
		log.Println(err)
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(custId, email)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func UpdateRoleId(email string, roleId int) error {
	statement, err := database.Db.Prepare("UPDATE user set role_id = ? where email = ?")
	if err != nil {
		log.Println(err)
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(roleId, email)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func CheckEmailVerify(email string) error {
	statement, err := database.Db.Prepare("UPDATE user set is_email_verified = ? where email = ?")
	if err != nil {
		log.Println(err)
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(true, email)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func UpdateCompanyIdToUser(email string) error {
	statement, err := database.Db.Prepare("UPDATE user set company_id = ? where id = ?")
	if err != nil {
		log.Println(err)
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(true, email)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func GetCompanyDetails(compName string) ([]Company, error) {
	// var data Company
	selDB, err := database.Db.Query("select id, company_name, external_id from company WHERE company_name = ?", compName)
	if err != nil {
		return []Company{}, err
	}
	defer selDB.Close()
	companyDetail := []Company{}
	for selDB.Next() {
		var comp Company
		err = selDB.Scan(&comp.ID, &comp.CompanyName, &comp.ExternalId)
		if err != nil {
			return []Company{}, err
		}
		companyDetail = append(companyDetail, comp)
	}
	return companyDetail, nil
}

func GetCustomerPermissionByPlan(planName string) (PlanAndPermission, error) {

	selDB, err := database.Db.Query(`SELECT id, plan_name, apps, workload_management, organisation_management, invite_user_limit, application_health_dashboard, byoh, storage, version_control_panel, single_sign_on, organization_count, sub_organization_count, businessunit_count, custom_domain, app_notification, secret, monitoring_platform, alerts_advisories, audit_logs, ssl_security, infrastructure_configuration, replicas, k8s_region
	FROM plans_permissions where plan_name = ?;`, planName)
	if err != nil {
		return PlanAndPermission{}, err
	}
	defer selDB.Close()
	var planAndPermissions PlanAndPermission
	for selDB.Next() {
		err = selDB.Scan(&planAndPermissions.Id, &planAndPermissions.PlanName, &planAndPermissions.Apps, &planAndPermissions.WorkloadManagement, &planAndPermissions.OrganisationManagement, &planAndPermissions.InviteUserLimit, &planAndPermissions.ApplicationHealthDashboard, &planAndPermissions.Byoh, &planAndPermissions.Storage, &planAndPermissions.VersionControlPanel, &planAndPermissions.SingleSignOn, &planAndPermissions.OrganizationCount, &planAndPermissions.SubOrganizationCount, &planAndPermissions.BusinessunitCount, &planAndPermissions.CustomDomain, &planAndPermissions.AppNotification, &planAndPermissions.Secret, &planAndPermissions.MonitoringPlatform, &planAndPermissions.AlertsAdvisories, &planAndPermissions.AuditLogs, &planAndPermissions.SslSecurity, &planAndPermissions.InfrastructureConfiguration, &planAndPermissions.Replicas, &planAndPermissions.K8sRegions)
		if err != nil {
			return PlanAndPermission{}, err
		}
	}
	return planAndPermissions, nil
}

// Login godoc
// @Summary Authenticate user
// @Description Authenticate user to create JWT token
// @Tags  login
// @Accept  json
// @Produce  json
// @Param dataBody body LoginRequestBody true "Create Token"
// @Success 200 {object} TokenResponseBody
// @Failure 400 {object} TokenErrorBody
// @Failure 500 {object} TokenErrorBody
// @Router /api/v1/login [post]
var validate *validator.Validate

func validateStruct(databody LoginRequestBody) error {
	validate = validator.New()
	err := validate.Struct(databody)
	if err != nil {
		return err
	}
	return nil
}

func Login(w http.ResponseWriter, r *http.Request) {

	// key := os.Getenv("ENCRYPTION_KEY")

	var dataBody LoginRequestBody
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	err = validateStruct(dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Email or Password field is missing."})
		return
	}

	var user User
	user.Email = dataBody.Data.Attributes.Email
	//  user.Password = dataBody.Data.Attributes.Password

	user.Password = decode.DePwdCode(dataBody.Data.Attributes.Password)

	correct, err := user.Authenticate()
	// if err != nil {
	// 	log4go.Error("Module: Login, MethodName: Authenticate, Message: %s user:%s", err.Error(), user.Email)
	// 	helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
	// 	return
	// }

	if !correct {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid User"})
		return
	}
	log4go.Info("Module: Login, MethodName: Authenticate, Message: Authentication successful, user: %s", user.Email)

	checkUser, err := GetUserIdByEmail(user.Email)
	if err != nil {
		log4go.Error("Module: Login, MethodName: GetUserIdByEmail, Message: %s user:%s", err.Error(), user.Email)
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: Login, MethodName: GetUserIdByEmail, Message: Fetching user Id by Email successful, user: %s", user.Email)

	roleId, role, err := GetUserRole(checkUser)

	if err != nil {
		log4go.Error("Module: Login, MethodName: GetUserRole, Message: %s user:%s", err.Error(), user.Email)
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: Login, MethodName: GetUserRole, Message: Fetching role of the user is successfully completed, user: %s", user.Email)

	rolePermission, err := GetPermission(roleId)
	if err != nil {
		log4go.Error("Module: Login, MethodName: GetPermission, Message: %s user:%s", err.Error(), user.Email)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: Login, MethodName: GetPermission, Message: Permission based on role fetched successfully, user: %s", user.Email)

	if roleId == 1 {

		verifyEmail, err := IsEmailVerified(user.Email)
		if err != nil {
			log4go.Error("Module: Login, MethodName: IsEmailVerified, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: IsEmailVerified, Message: Email verification is successfully completed , user: %s", user.Email)

		if !verifyEmail {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Email not verified!", "userId": strconv.Itoa(checkUser)})
			return
		}

		freePlan, err := FreePlanDetails(checkUser)

		if err != nil {
			log4go.Error("Module: Login, MethodName: FreePlanDetails, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: FreePlanDetails, Message: Checking free plan or not, successfully completed, user: %s", user.Email)

		if freePlan {
			custStripe := stripes.GetStripeCustomerDetails(user.Email)
			if custStripe != "" {
				checkPlanName, err := stripes.GetCustPlanName(custStripe)
				if err != nil {
					return
				}
				fmt.Println(checkPlanName)
				if checkPlanName != "free plan" {
					UpdateFreePlanFlag(checkUser)
					freePlan = false
				}
			}
		}

		custStripeId := ""
		productId := ""
		var activeSubcription stripe.SubscriptionStatus

		if !freePlan {

			custStripeId = stripes.GetStripeCustomerDetails(user.Email)

			if custStripeId == "" {
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Cannot find customer in payment partner with provided email!", "UserId": strconv.Itoa(checkUser)})
				return
			}

			err = InsertCustomerStripesId(custStripeId, user.Email)

			if err != nil {
				log4go.Error("Module: Login, MethodName: InsertCustomerStripesId, Message: %s user:%s", err.Error(), user.Email)
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: Login, MethodName: InsertCustomerStripesId, Message: Customer stripe Id successfully inserted, user: %s", user.Email)

			activeSubcription, productId = stripes.ListSubscription(custStripeId)
			if err != nil {
				log4go.Error("Module: Login, MethodName: InsertCustomerStripesId, Message: %s user:%s", err.Error(), user.Email)
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
				return
			}

			if activeSubcription != "active" {
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Cannot find active subscription for the provided email!"})
				return
			}

			if productId == "" {
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Cannot find active subscription for the provided email!"})
				return

			}

		}

		checkCompany, err := GetEmailByCompanyName(user.Email)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: Login, MethodName: GetEmailByCompanyName, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: GetEmailByCompanyName, Message: Fetching Company Name by Email successfully completed, user: %s", user.Email)

		firstTimeUser, err := GetFirsttimeUser(user.Email)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: Login, MethodName: GetFirsttimeUser, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: GetFirsttimeUser, Message: Checking first time user is successfully completed, user: %s", user.Email)

		if roleId == 1 && firstTimeUser {
			err = updateUserProfileCreated(strconv.Itoa(checkUser), true)
		}

		if checkCompany != "" && firstTimeUser {

			var comName string
			re, err := regexp.Compile(`[^\w]`)
			if err != nil {
				fmt.Println(err)
			}
			comName = re.ReplaceAllString(checkCompany, "")
			comName = strings.ToLower(comName)

			checkOrg, err := CheckOrganizationBySlug(comName)
			if checkOrg.Slug != nil {
				randomNumber := organizationInfo.RandomNumber4Digit()
				randno := strconv.Itoa(int(randomNumber))
				checkCompany = checkCompany + "-" + randno
			}

			_, err = organizationInfo.CreateOrgainzation(checkCompany, strconv.Itoa(int(checkUser)), "1")
			if err != nil {
				log.Println(err)
				log4go.Error("Module: Login, MethodName: CreateOrgainzation, Message: %s user:%s", err.Error(), user.Email)
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: Login, MethodName: CreateOrgainzation, Message: Organization Created Successfully for the first time user, user: %s", user.Email)

		}

		if firstTimeUser {
			_, err = ChangeFirstUserOption(user.Email)
			if err != nil {
				log.Println(err)
				log4go.Error("Module: Login, MethodName: ChangeFirstUserOption, Message: %s user:%s", err.Error(), user.Email)
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: Login, MethodName: ChangeFirstUserOption, Message: Updating first time user is successfully completed, user: %s", user.Email)
		}
		var userDet User
		// var custstripe string
		if freePlan {
			userDet, err = GetUsersNameByEmail(dataBody.Data.Attributes.Email)
			if err != nil {
				log4go.Error("Module: Login, MethodName: GetUserNameByEmail, Message: %s user:%s", err.Error(), user.Email)
				helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: Login, MethodName: GetUserNameByEmail, Message: Fetching user name by email is successfully completed, user: %s", user.Email)
		} else {

			userDet, err = GetUserNameByEmail(dataBody.Data.Attributes.Email)
			if err != nil {
				log4go.Error("Module: Login, MethodName: GetUserNameByEmail, Message: %s user:%s", err.Error(), user.Email)
				helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
				return
			}
			// custstripe = userDet.CustomerStripeId
			log4go.Info("Module: Login, MethodName: GetUserNameByEmail, Message: Fetching user name by email is successfully completed, user: %s", user.Email)
		}

		accessToken, refreshToken, err := GenerateAccessAndRefreshToken(user.Email, productId, false, userDet.FirstName, userDet.LastName, userDet.CompanyName, userDet.RoleId, userDet.CustomerStripeId)
		if err != nil {
			log4go.Error("Module: Login, MethodName: GenerateAccessAndRefreshToken, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: GenerateAccessAndRefreshToken, Message: Generating Access Token and Refresh Token successfully completed, user: %s", user.Email)

		getUserId, err := GetUserIdByEmail(user.Email)
		if err != nil {
			log4go.Error("Module: Login, MethodName: GetUserIdByEmail, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: GetUserIdByEmail, Message: Fetching user Id by Email successful, user: %s", user.Email)

		var planAndPermission PlanAndPermission
		if !freePlan {
			planName, err := stripes.GetCustPlanName(custStripeId)
			if err != nil {
				log4go.Error("Module: Login, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.Email)
				helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: Login, MethodName: GetCustPlanName, Message: Get user plan with ProductId:"+productId+", user: %s", user.Email)

			planAndPermission, err = GetCustomerPermissionByPlan(planName)
			if err != nil {
				log4go.Error("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
				helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+productId+", user: %s", user.Email)
		}
		if freePlan {
			planAndPermission, err = GetCustomerPermissionByPlan("free plan")
			if err != nil {
				log4go.Error("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
				helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+productId+", user: %s", user.Email)
		}

		if freePlan {
			custStripeIds := stripes.GetStripeCustomerDetails(user.Email)

			if custStripeIds == "" {
				// CHECK 14 DAYS FREE PLAN IS EXPIRED OR NOT
				layout := "2006-01-02 15:04:05"
				userAccountCreatedDate := userDet.CreatedAt.Format("2006-01-02 15:04:05")
				createdDate, err := time.Parse(layout, userAccountCreatedDate)
				if err != nil {
					fmt.Println("Error parsing date:", err)
					return
				}
				deadline := createdDate.Add(14 * 24 * time.Hour)
				currentDate := time.Now()
				isBeforeDeadline := currentDate.Before(deadline)

				if !isBeforeDeadline {
					helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": "Your 14-day trial period has ended. Please upgrade your plan", "userId": userDet.ID})
					return
				}
			}
		}

		userId := strconv.Itoa(getUserId)

		AddOperation := Activity{
			Type:       "LOGIN",
			UserId:     strconv.Itoa(checkUser),
			Activities: "LOGGED IN",
			Message:    user.Email + " has logged into Nife platform",
			RefId:      user.Email,
		}

		_, err = InsertActivity(AddOperation)
		if err != nil {
			fmt.Println(err)
			return
		}

		tokenData := buildTokenData(accessToken, refreshToken, userId, custStripeId, role, rolePermission, false, false, firstTimeUser, &planAndPermission)
		helper.RespondwithJSON(w, http.StatusOK, tokenData)
	} else {
		// Invite User login process
		compName, err := GetEmailByCompanyName(dataBody.Data.Attributes.Email)
		if err != nil {
			log4go.Error("Module: Login, MethodName: GetEmailByCompanyName, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: GetEmailByCompanyName, Message: Fetching Company Name by Email successfully completed, user: %s", user.Email)

		companyDetails, err := GetCompanyDetails(compName)
		if err != nil {
			return
		}
		var emailAd string
		for _, val := range companyDetails {
			emailAd = val.ExternalId
		}
		adminEmail, adminName, err := GetAdminByemail(emailAd, 1)
		if err != nil {
			log4go.Error("Module: Login, MethodName: GetAdminByCompanyName, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: GetAdminByCompanyName, Message: Get Admin by comapny name is successfully completed, user: %s", user.Email)
		fmt.Println(adminName)
		custStripeId := ""
		productId := ""
		var activeSubcription stripe.SubscriptionStatus

		custStripeId = stripes.GetStripeCustomerDetails(adminEmail)

		if custStripeId == "" {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Cannot find customer in payment partner with provided email!"})
			return
		}
		log4go.Info("Module: Login, MethodName: GetStripeCustomerDetails, Message: Fetch customer stripe details successfully completed, user: %s", user.Email)

		activeSubcription, productId = stripes.ListSubscription(custStripeId)

		if activeSubcription != "active" {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Cannot find active subscription for the provided email!"})
			return
		}

		getIsInvited, profileCreated, err := GetIsInvitedByEmail(dataBody.Data.Attributes.Email)

		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		err = oragnizationUsers.UpdateTempPwdandIsUserInvite(dataBody.Data.Attributes.Email, false)

		if err != nil {
			log4go.Error("Module: Login, MethodName: UpdateTempPwdandIsUserInvite, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: UpdateTempPwdandIsUserInvite, Message: Temporary Password is updated successfully to the user, user: %s", user.Email)

		// _, err = ChangeFirstUserOption(dataBody.Data.Attributes.Email)
		// 	if err != nil {
		// 		log.Println(err)
		// 		log4go.Error("Module: Login, MethodName: ChangeFirstUserOption, Message: %s user:%s", err.Error(), user.Email)
		// 		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		// 		return
		// 	}
		// 	log4go.Info("Module: Login, MethodName: ChangeFirstUserOption, Message: Updating first time user is successfully completed, user: %s", user.Email)

		// 	err = CheckEmailVerify(dataBody.Data.Attributes.Email)

		// if err != nil {
		// 	helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		// 	return
		// }

		err = InsertCustomerStripesId(custStripeId, dataBody.Data.Attributes.Email)
		if err != nil {
			log4go.Error("Module: Login, MethodName: InsertCustomerStripesId, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}

		userDet, err := GetUserNameByEmail(dataBody.Data.Attributes.Email)
		if err != nil {
			log4go.Error("Module: Login, MethodName: GetUserNameByEmail, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: GetUserNameByEmail, Message: Fetching user name by email is successfully completed, user: %s", user.Email)

		accessToken, refreshToken, err := GenerateAccessAndRefreshToken(user.Email, productId, false, userDet.FirstName, userDet.LastName, userDet.CompanyName, roleId, custStripeId)
		if err != nil {
			log4go.Error("Module: Login, MethodName: GenerateAccessAndRefreshToken, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: GenerateAccessAndRefreshToken, Message: Generating Access Token and Refresh Token successfully completed, user: %s", user.Email)

		getUserId, err := GetUserIdByEmail(user.Email)
		if err != nil {
			helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
			return
		}

		var planAndPermission PlanAndPermission

		planName, err := stripes.GetCustPlanName(custStripeId)
		if err != nil {
			log4go.Error("Module: Login, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: GetCustPlanName, Message: Get user plan with ProductId:"+productId+", user: %s", user.Email)

		planAndPermission, err = GetCustomerPermissionByPlan(planName)
		if err != nil {
			log4go.Error("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
			helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+productId+", user: %s", user.Email)

		userId := strconv.Itoa(getUserId)
		tokenData := buildTokenData(accessToken, refreshToken, userId, custStripeId, role, rolePermission, getIsInvited, profileCreated, false, &planAndPermission)
		helper.RespondwithJSON(w, http.StatusOK, tokenData)

	}
}

// RefreshToken godoc
// @Summary Reauthenticate user using refresh_token
// @Description Reauthenticate user by renewing JWT token
// @Tags  refreshToken
// @Accept  json
// @Produce  json
// @Param dataBody body RefreshTokenRequestBody true "Refresh Token"
// @Success 200 {object} TokenResponseBody
// @Failure 400 {object} TokenErrorBody
// @Failure 401 {object} TokenErrorBody
// @Router /api/v1/refreshToken [post]
func RefreshToken(w http.ResponseWriter, r *http.Request) {
	var dataBody RefreshTokenRequestBody
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	token := dataBody.RefreshToken
	email, stripesProductId, firstName, lastName, companyName, roleId, customerStripeId, err := jwt.ParseToken(token)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
		return
	}
	accessToken, refreshToken, err := GenerateAccessAndRefreshToken(email, stripesProductId, false, firstName, lastName, companyName, roleId, customerStripeId)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}
	getUserId, err := GetUserIdByEmail(email)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
		return
	}

	userId := strconv.Itoa(getUserId)

	tokenData := buildTokenData(accessToken, refreshToken, userId, "", "", nil, false, false, false, nil)
	helper.RespondwithJSON(w, http.StatusOK, tokenData)
}

func UserRegister(w http.ResponseWriter, r *http.Request) {
	var dataBody UserRegisterRequestBody
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	user := User{Email: dataBody.Email, Password: dataBody.Password, PhoneNumber: dataBody.PhoneNumber,
		Location: dataBody.Location, Industry: dataBody.Industry, CompanyName: dataBody.CompanyName}

	//

	userId, err := user.Create()
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	accessToken, _, err := GenerateAccessAndRefreshToken(user.Email, "", false, user.FirstName, user.LastName, user.CompanyName, user.RoleId, "")
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	_, err = cli_session.UpdateCLISession(accessToken, int(userId), dataBody.SessionId)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"user_Id":      userId,
		"access_token": accessToken,
		//"customer_Stripe_Id": result.ID,
	})

}

func validateStructReg(databody UserRegisterRequestBody) error {
	validate = validator.New()
	err := validate.Struct(databody)
	if err != nil {
		return err
	}
	return nil
}

func UserRegisterV2(w http.ResponseWriter, r *http.Request) {

	var dataBody UserRegisterRequestBody
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	err = validateStructReg(dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Mandatory fields are empty"})
		return
	}

	password := decode.DePwdCode(dataBody.Password)

	if password == "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid Credentials"})
		return

	}

	checkUser, _ := GetUserIdByEmail(dataBody.Email)

	if checkUser != 0 {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Email already register! Login directly.."})
		return
	}
	if dataBody.CompanyName == "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Company name should not be empty!"})
		return

	}

	if dataBody.Location == "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Location field should not be empty!"})
		return

	}

	checkCompany, err := CheckCompanyExists(dataBody.CompanyName)

	if checkCompany != "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Try with different company name!"})
		// randomNumber := organizationInfo.RandomNumber4Digit()
		// randomNum := strconv.Itoa(int(randomNumber))
		// dataBody.CompanyName = dataBody.CompanyName + "-" + randomNum
		return
	}
	CheckOrganization, err := CheckOrganizationExists(dataBody.CompanyName)

	if CheckOrganization != "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Try with different company name!"})
		return
	}

	//-------------------------------------------------

	apolloDetails, err := GetUserDetailsInApolloAPI(dataBody.Email)
	if err != nil {
		return
	}

	//
	compId := uuid.NewString()

	user := User{Email: dataBody.Email, Password: password, PhoneNumber: dataBody.PhoneNumber,
		Location: dataBody.Location, Industry: apolloDetails.Person.Organization.Industry,
		CompanyName: dataBody.CompanyName, FirstName: dataBody.FirstName,
		LastName: dataBody.LastName, CreatedAt: time.Now(), RoleId: 1, CompanyId: compId,
		WebsiteURL: apolloDetails.Person.Organization.WebsiteURL, LinkedinURL: apolloDetails.Person.Organization.LinkedinURL,
		EstimatedNumEmployees: apolloDetails.Person.Organization.EstimatedNumEmployees, FoundedYear: apolloDetails.Person.Organization.FoundedYear,
		TotalFunding: apolloDetails.Person.Organization.TotalFunding, LatestFundingRoundDate: apolloDetails.Person.Organization.LatestFundingRoundDate}

	//

	userId, err := user.Create()
	if err != nil {
		log4go.Error("Module: UserRegisterV2, MethodName: Create, Message: %s user:%s", err.Error(), user.Email)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: UserRegisterV2, MethodName: Create, Message: User registration details inserted successfully, user: %s", user.Email)
	company := Company{ID: compId, CompanyName: dataBody.CompanyName, ExternalId: dataBody.Email}

	err = company.InsertCompanyDetails()
	if err != nil {
		log4go.Error("Module: UserRegisterV2, MethodName: InsertCompanyDetails, Message: %s user:%s", err.Error(), user.Email)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: UserRegisterV2, MethodName: InsertCompanyDetails, Message: Inserted company details, user: %s", user.Email)

	companyDet, err := GetCompanyDetails(dataBody.CompanyName)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	// org, err := organizationInfo.CreateOrgainzation(dataBody.CompanyName, strconv.Itoa(int(userId)))
	// if err != nil {
	// 	log.Println(err)
	// 	return
	// }

	accessToken, _, err := GenerateAccessAndRefreshToken(user.Email, "", false, user.FirstName, user.LastName, user.CompanyName, user.RoleId, "")
	if err != nil {
		log4go.Error("Module: UserRegisterV2, MethodName: GenerateAccessAndRefreshToken, Message: %s user:%s", err.Error(), user.Email)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: UserRegisterV2, MethodName: GenerateAccessAndRefreshToken, Message: Generating Access Token and Refresh Token successfully completed, user: %s", user.Email)

	// _, err = cli_session.UpdateCLISession(accessToken, int(userId), dataBody.SessionId)
	// if err != nil {
	// 	helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
	// 	return
	// }

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"user_Id":      userId,
		"access_token": accessToken,
		"company":      companyDet,
	})

}

// Generate accessToken and refreshToken from email
func GenerateAccessAndRefreshToken(email string, productId string, isResetPassword bool, firstName string, lastName string, companyName string, roleId int, customerStripeId string) (string, string, error) {
	accessToken, err := jwt.GenerateAccessToken(email, productId, isResetPassword, firstName, lastName, companyName, roleId, customerStripeId)
	refreshToken, err := jwt.GenerateRefreshToken(email, productId, firstName, lastName, companyName, roleId, customerStripeId)
	return accessToken, refreshToken, err
}

// Build response data using accessToken and refreshToken
func buildTokenData(accessToken, refreshToken, userId, custStripeId, role string, rolePermission []RolePermission, getIsInvited, profileCreated, firstTimeUser bool, planAndPermission *PlanAndPermission) map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"attributes": map[string]interface{}{
				"access_token":       accessToken,
				"refresh_token":      refreshToken,
				"userId":             userId,
				"customerStripeId":   custStripeId,
				"role":               role,
				"permissions":        rolePermission,
				"isUserInvited":      getIsInvited,
				"userProfileCreated": profileCreated,
				"FirstTimeUser":      firstTimeUser,
				"PlanAndPermission":  planAndPermission,
			},
		},
	}
}

func SSOSignIn(w http.ResponseWriter, r *http.Request) {

	var loginDetails SsoLoginDetails

	err := json.NewDecoder(r.Body).Decode(&loginDetails)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	var SSOEmail string

	accessToken := decode.DePwdCode(loginDetails.AccessToken)
	// accessToken := loginDetails.AccessToken

	if loginDetails.SSOType == "google" {
		content := singleSignOn.GetGoogleData(accessToken)
		response := singleSignOn.FindEmail(content)
		if response.Id == nil {
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "something went wrong! check email or password"})
			return
		}
		SSOEmail = fmt.Sprintf("%v", response.Email)

	}

	if loginDetails.SSOType == "github" {
		gitEmail, err := github.GithubCheckUser(accessToken)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: SSOSignIn, MethodName: GithubCheckUser, Message: %s user:%s", err.Error())
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		SSOEmail = gitEmail
		log4go.Info("Module: SSOSignIn, MethodName: GithubCheckUser, Message: SSO Github login is successfully completed, user: %s", SSOEmail)

	}

	if loginDetails.SSOType == "gitlab" {
		gitLabEmail, err := singleSignOn.GitlabCheckUser(accessToken)
		if err != nil {
			log4go.Error("Module: SSOSignIn, MethodName: GitlabCheckUser, Message: %s user:%s", err.Error())
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		SSOEmail = gitLabEmail
		log4go.Info("Module: SSOSignIn, MethodName: GitlabCheckUser, Message: SSO Gitlab login is successfully completed, user: %s", SSOEmail)
	}

	if loginDetails.SSOType == "bitbucket" {
		bitbucketEmail, err := singleSignOn.BitbucketCheckUser(accessToken)
		if err != nil {
			log4go.Error("Module: SSOSignIn, MethodName: BitbucketCheckUser, Message: %s user:%s", err.Error())
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		SSOEmail = bitbucketEmail
		log4go.Info("Module: SSOSignIn, MethodName: BitbucketCheckUser, Message: SSO Bitbucket login is successfully completed, user: %s", SSOEmail)
	}

	checkUser, err := CheckSSOEmail(SSOEmail)

	if checkUser == 0 {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid User"})
		return
	}
	log4go.Info("Module: SSOSignIn, MethodName: CheckSSOEmail, Message: Check user exist reached successfully, user: %s", SSOEmail)

	roleId, role, err := GetUserRole(checkUser)

	if err != nil {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}

	if roleId == 2 {
		compName, err := GetEmailByCompanyName(SSOEmail)
		if err != nil {
			log4go.Error("Module: SSOSignIn, MethodName: GetEmailByCompanyName, Message: %s user:%s", err.Error(), SSOEmail)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: SSOSignIn, MethodName: GetEmailByCompanyName, Message: Fetching Company Name by Email successfully completed, user: %s", SSOEmail)

		adminEmail, err := GetAdminByCompanyNameAndEmail(compName)
		if err != nil {
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		if adminEmail == SSOEmail {
			UpdateRoleId(SSOEmail, 1)
		}

		roleId, role, err = GetUserRole(checkUser)

		if err != nil {
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
	}

	rolePermission, err := GetPermission(roleId)
	if err != nil {
		log4go.Error("Module: SSOSignIn, MethodName: GetPermission, Message: %s user:%s", err.Error(), SSOEmail)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: SSOSignIn, MethodName: GetPermission, Message: Permission based on role fetched successfully, user: %s", SSOEmail)

	if roleId == 1 {
		freePlan, err := FreePlanDetails(checkUser)

		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		if freePlan {
			custStripe := stripes.GetStripeCustomerDetails(SSOEmail)
			if custStripe != "" {
				checkPlanName, err := stripes.GetCustPlanName(custStripe)
				if err != nil {
					return
				}
				fmt.Println(checkPlanName)
				if checkPlanName != "free plan" {
					UpdateFreePlanFlag(checkUser)
					freePlan = false
				}

			}
		}

		custStripeId := ""
		productId := ""
		var activeSubcription stripe.SubscriptionStatus
		var planAndPermission PlanAndPermission
		if !freePlan {

			custStripeId = stripes.GetStripeCustomerDetails(SSOEmail)

			if custStripeId == "" {
				// helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Cannot find customer in payment partner with provided email!"})
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "User haven't subscribe for any plan", "user_id": strconv.Itoa(checkUser)})
				return
			}
			log4go.Info("Module: SSOSignIn, MethodName: GetStripeCustomerDetails, Message: Fetch customer stripe details successfully completed, user: %s", SSOEmail)

			err = InsertCustomerStripesId(custStripeId, SSOEmail)

			if err != nil {
				log4go.Error("Module: SSOSignIn, MethodName: InsertCustomerStripesId, Message: %s user:%s", err.Error(), SSOEmail)
				log.Println(err)
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: SSOSignIn, MethodName: InsertCustomerStripesId, Message: Customer stripe Id successfully inserted, user: %s", SSOEmail)

			activeSubcription, productId = stripes.ListSubscription(custStripeId)

			if activeSubcription != "active" {
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Cannot find active subscription for the provided email!", "user_id": strconv.Itoa(checkUser)})
				return
			}
			log4go.Info("Module: SSOSignIn, MethodName: ListSubscription, Message: Check Active Subcription or Not reached successfully, user: %s", SSOEmail)

			planName, err := stripes.GetCustPlanName(custStripeId)
			if err != nil {
				log4go.Error("Module: Login, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), SSOEmail)
				helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: Login, MethodName: GetCustPlanName, Message: Get user plan with ProductId:"+productId+", user: %s", SSOEmail)

			planAndPermission, err = GetCustomerPermissionByPlan(planName)
			if err != nil {
				log4go.Error("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), SSOEmail)
				helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+productId+", user: %s", SSOEmail)

		}
		if freePlan {
			planAndPermission, err = GetCustomerPermissionByPlan("free plan")
			if err != nil {
				log4go.Error("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), SSOEmail)
				helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: Login, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:"+productId+", user: %s", SSOEmail)
		}

		firstTimeUser, err := GetFirsttimeUser(SSOEmail)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: SSOSignIn, MethodName: GetFirsttimeUser, Message: %s user:%s", err.Error(), SSOEmail)
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: SSOSignIn, MethodName: GetFirsttimeUser, Message: Checking first time user is successfully completed, user: %s", SSOEmail)

		checkCompany, err := GetEmailByCompanyName(SSOEmail)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: SSOSignIn, MethodName: GetEmailByCompanyName, Message: %s user:%s", err.Error(), SSOEmail)
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: SSOSignIn, MethodName: GetEmailByCompanyName, Message: Fetching Company Name by Email successfully completed, user: %s", SSOEmail)

		if checkCompany != "" && firstTimeUser {

			var comName string
			re, err := regexp.Compile(`[^\w]`)
			if err != nil {
				fmt.Println(err)
			}
			comName = re.ReplaceAllString(checkCompany, "")
			comName = strings.ToLower(comName)

			checkOrg, err := CheckOrganizationBySlug(comName)
			if checkOrg.Slug != nil {
				randomNumber := organizationInfo.RandomNumber4Digit()
				randno := strconv.Itoa(int(randomNumber))
				checkCompany = checkCompany + "-" + randno
			}
			_, err = organizationInfo.CreateOrgainzation(checkCompany, strconv.Itoa(int(checkUser)), "1")
			if err != nil {
				log.Println(err)
				log4go.Error("Module: SSOSignIn, MethodName: CreateOrgainzation, Message: %s user:%s", err.Error(), SSOEmail)
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: SSOSignIn, MethodName: CreateOrgainzation, Message: Organization Created Successfully for the first time user, user: %s", SSOEmail)

		}

		if firstTimeUser {
			_, err = ChangeFirstUserOption(SSOEmail)
			if err != nil {
				log.Println(err)
				log4go.Error("Module: SSOSignIn, MethodName: ChangeFirstUserOption, Message: %s user:%s", err.Error(), SSOEmail)
				helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
				return
			}
			log4go.Info("Module: SSOSignIn, MethodName: ChangeFirstUserOption, Message: Updating first time user is successfully completed, user: %s", SSOEmail)
		}
		userDet, err := GetUserCreatedTime(SSOEmail)
		if err != nil {
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		if freePlan {
			if freePlan {
				custStripeIds := stripes.GetStripeCustomerDetails(SSOEmail)

				if custStripeIds == "" {
					// CHECK 14 DAYS FREE PLAN IS EXPIRED OR NOT
					layout := "2006-01-02 15:04:05"
					userAccountCreatedDate := userDet.CreatedAt.Format("2006-01-02 15:04:05")
					createdDate, err := time.Parse(layout, userAccountCreatedDate)
					if err != nil {
						fmt.Println("Error parsing date:", err)
						return
					}
					deadline := createdDate.Add(14 * 24 * time.Hour)
					currentDate := time.Now()
					isBeforeDeadline := currentDate.Before(deadline)

					if !isBeforeDeadline {
						helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": "Your 14-day trial period has ended. Please upgrade your plan", "userId": userDet.ID})
						return
					}
				}
			}
		}

		accessToken, refreshToken, err := GenerateAccessAndRefreshToken(SSOEmail, productId, false, "", "", "", roleId, custStripeId)
		if err != nil {
			log4go.Error("Module: SSOSignIn, MethodName: GenerateAccessAndRefreshToken, Message: %s user:%s", err.Error(), SSOEmail)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: SSOSignIn, MethodName: GenerateAccessAndRefreshToken, Message: Generating Access Token and Refresh Token successfully completed, user: %s", SSOEmail)

		userId := strconv.Itoa(checkUser)

		AddOperation := Activity{
			Type:       "LOGIN",
			UserId:     strconv.Itoa(checkUser),
			Activities: "LOGGED IN",
			Message:    SSOEmail + " has logged into Nife platform",
			RefId:      SSOEmail,
		}

		_, err = InsertActivity(AddOperation)
		if err != nil {
			fmt.Println(err)
			return
		}

		tokenData := buildTokenData(accessToken, refreshToken, userId, custStripeId, role, rolePermission, false, false, firstTimeUser, &planAndPermission)

		helper.RespondwithJSON(w, http.StatusOK, tokenData)
	} else {
		compName, err := GetEmailByCompanyName(SSOEmail)
		if err != nil {
			log4go.Error("Module: SSOSignIn, MethodName: GetEmailByCompanyName, Message: %s user:%s", err.Error(), SSOEmail)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: SSOSignIn, MethodName: GetEmailByCompanyName, Message: Fetching Company Name by Email successfully completed, user: %s", SSOEmail)

		adminEmail, adminName, err := GetAdminByCompanyName(compName, 1)
		if err != nil {
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		fmt.Println(adminName)
		custStripeId := ""
		productId := ""
		var activeSubcription stripe.SubscriptionStatus

		custStripeId = stripes.GetStripeCustomerDetails(adminEmail)

		if custStripeId == "" {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Cannot find customer in payment partner with provided email!"})
			return
		}
		log4go.Info("Module: SSOSignIn, MethodName: GetStripeCustomerDetails, Message: Fetch customer stripe details successfully completed, user: %s", SSOEmail)

		activeSubcription, productId = stripes.ListSubscription(custStripeId)

		if activeSubcription != "active" {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Cannot find active subscription for the provided email!"})
			return
		}
		getIsInvited, profileCreated, err := GetIsInvitedByEmail(SSOEmail)

		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		err = oragnizationUsers.UpdateTempPwdandIsUserInvite(SSOEmail, false)

		if err != nil {
			log4go.Error("Module: SSOSignIn, MethodName: UpdateTempPwdandIsUserInvite, Message: %s user:%s", err.Error(), SSOEmail)
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: SSOSignIn, MethodName: UpdateTempPwdandIsUserInvite, Message: Temporary Password is updated successfully to the user, user: %s", SSOEmail)

		accessToken, refreshToken, err := GenerateAccessAndRefreshToken(SSOEmail, productId, false, "", "", "", 0, "")
		if err != nil {
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		getUserId, err := GetUserIdByEmail(SSOEmail)
		if err != nil {
			log4go.Error("Module: SSOSignIn, MethodName: GetUserIdByEmail, Message: %s user:%s", err.Error(), SSOEmail)
			helper.RespondwithJSON(w, http.StatusUnauthorized, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: SSOSignIn, MethodName: GetUserIdByEmail, Message: Fetching user Id by Email successful, user: %s", SSOEmail)

		userId := strconv.Itoa(getUserId)
		tokenData := buildTokenData(accessToken, refreshToken, userId, custStripeId, role, rolePermission, getIsInvited, profileCreated, false, nil)
		helper.RespondwithJSON(w, http.StatusOK, tokenData)

	}
}

func SSOSignUp(w http.ResponseWriter, r *http.Request) {
	var dataBody UserRegisterRequestBody
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	if dataBody.CompanyName == "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "company name should not be empty!"})
		return

	}

	// checkCompany, err := GetIdByCompanyName(dataBody.CompanyName)
	// if err != nil {
	// 	log4go.Error("Module: SSOSignUp, MethodName: GetIdByCompanyName, Message: %s user:%s", err.Error(), dataBody.Email)
	// 	log.Println(err)
	// 	return
	// }
	// log4go.Info("Module: SSOSignUp, MethodName: GetIdByCompanyName, Message: Fetch user Id by the company name is completed succesfully, user: %s", dataBody.Email)

	// if checkCompany != 0 {
	// 	helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Try with different company name!"})
	// 	return
	// }

	var SSOEmail string

	password := decode.DePwdCode(dataBody.Password)
	// password := dataBody.Password

	if dataBody.SsoType == "google" {
		content := singleSignOn.GetGoogleData(password)
		response := singleSignOn.FindEmail(content)
		if response.Id == nil {
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "something went wrong! check email or password"})
			return
		}
		SSOEmail = fmt.Sprintf("%v", response.Email)
	}

	if dataBody.SsoType == "github" {
		gitEmail, err := github.GithubCheckUser(password)
		if err != nil {
			log4go.Error("Module: SSOSignUp, MethodName: GithubCheckUser, Message: %s user:%s", err.Error(), dataBody.Email)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		SSOEmail = gitEmail
		log4go.Info("Module: SSOSignUp, MethodName: GithubCheckUser, Message: SSO Github login is successfully completed, user: %s", SSOEmail)
	}

	if dataBody.SsoType == "gitlab" {
		gitLabEmail, err := singleSignOn.GitlabCheckUser(password)
		if err != nil {
			log4go.Error("Module: SSOSignUp, MethodName: GitlabCheckUser, Message: %s user:%s", err.Error(), dataBody.Email)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		SSOEmail = gitLabEmail
		log4go.Info("Module: SSOSignUp, MethodName: GitlabCheckUser, Message: SSO Gitlab login is successfully completed, user: %s", SSOEmail)
	}

	if dataBody.SsoType == "bitbucket" {
		gitEmail, err := singleSignOn.BitbucketCheckUser(password)
		if err != nil {
			log4go.Error("Module: SSOSignUp, MethodName: BitbucketCheckUser, Message: %s user:%s", err.Error(), dataBody.Email)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		SSOEmail = gitEmail
		log4go.Info("Module: SSOSignUp, MethodName: BitbucketCheckUser, Message: SSO Bitbucket login is successfully completed, user: %s", SSOEmail)
	}

	if SSOEmail == "<nil>" || SSOEmail == "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "something went wrong! check email or password"})
		return
	}

	checkUser, _ := CheckSSOEmail(SSOEmail)

	if checkUser != 0 {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "User already register"})
		return
	}
	log4go.Info("Module: SSOSignUp, MethodName: CheckSSOEmail, Message: Check user exist reached successfully, user: %s", SSOEmail)
	checkCompany, err := GetIdByCompanyName(dataBody.CompanyName)
	if err != nil {
		log4go.Error("Module: SSOSignUp, MethodName: GetIdByCompanyName, Message: %s user:%s", err.Error(), dataBody.Email)
		log.Println(err)
		return
	}
	log4go.Info("Module: SSOSignUp, MethodName: GetIdByCompanyName, Message: Fetch user Id by the company name is completed succesfully, user: %s", dataBody.Email)

	if checkCompany != 0 {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Try with different company name!"})
		return
	}
	compId := uuid.NewString()

	apolloDetails, err := GetUserDetailsInApolloAPI(dataBody.Email)
	if err != nil {
		return
	}

	user := User{Email: SSOEmail, Password: "", PhoneNumber: dataBody.PhoneNumber,
		Location: dataBody.Location, Industry: dataBody.Industry,
		CompanyName: dataBody.CompanyName, FirstName: dataBody.FirstName,
		LastName: dataBody.LastName, CreatedAt: time.Now(), SSOType: dataBody.SsoType, RoleId: 1, CompanyId: compId,
		WebsiteURL: apolloDetails.Person.Organization.WebsiteURL, LinkedinURL: apolloDetails.Person.Organization.LinkedinURL,
		EstimatedNumEmployees: apolloDetails.Person.Organization.EstimatedNumEmployees, FoundedYear: apolloDetails.Person.Organization.FoundedYear,
		TotalFunding: apolloDetails.Person.Organization.TotalFunding, LatestFundingRoundDate: apolloDetails.Person.Organization.LatestFundingRoundDate}

	//

	userId, err := user.AccountCreate()
	if err != nil {
		log4go.Error("Module: SSOSignUp, MethodName: AccountCreate, Message: %s user:%s", err.Error(), SSOEmail)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: SSOSignUp, MethodName: AccountCreate, Message: Account created successfully, user: %s", SSOEmail)

	company := Company{ID: compId, CompanyName: dataBody.CompanyName, ExternalId: dataBody.Email}
	err = company.InsertCompanyDetails()
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	accessToken, _, err := GenerateAccessAndRefreshToken(user.Email, "", false, user.FirstName, user.LastName, user.CompanyName, user.RoleId, "")
	if err != nil {
		log4go.Error("Module: SSOSignUp, MethodName: GenerateAccessAndRefreshToken, Message: %s user:%s", err.Error(), SSOEmail)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: SSOSignUp, MethodName: GenerateAccessAndRefreshToken, Message: Generating Access Token and Refresh Token successfully completed, user: %s", SSOEmail)
	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"user_Id":      userId,
		"access_token": accessToken,
	})

}

func UpdateProfileImage(imageName string, userId int) error {

	statement, err := database.Db.Prepare("UPDATE user set profile_image_url = ? where id = ?")
	if err != nil {
		log.Println(err)
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(imageName, userId)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func GetPermission(roleId int) ([]RolePermission, error) {

	query := `SELECT pm.module, pm.title, pm.create,pm.view, pm.delete, pm.update
	FROM permission pm
	JOIN role_permission
	  ON role_permission.permission_id = pm.id
	JOIN role
	  ON role.id = role_permission.role_id where role.id = ?`

	statement, err := database.Db.Query(query, roleId)
	if err != nil {
		return []RolePermission{}, err
	}
	defer statement.Close()

	permission := []RolePermission{}

	for statement.Next() {
		var rolepm RolePermission
		if err = statement.Scan(&rolepm.Module, &rolepm.Title, &rolepm.Create, &rolepm.View, &rolepm.Delete, &rolepm.Update); err != nil {
			return []RolePermission{}, err
		}
		permission = append(permission, rolepm)
	}
	return permission, nil
}

func GetIsInvitedByEmail(email string) (bool, bool, error) {

	selDB, err := database.Db.Query("SELECT is_user_invited, user_profile_created FROM user where email = ?", email)
	if err != nil {
		return false, false, err
	}
	defer selDB.Close()
	var userInvited bool
	var profileCreated bool
	for selDB.Next() {
		err = selDB.Scan(&userInvited, &profileCreated)
		if err != nil {
			return false, false, err
		}
	}
	return userInvited, profileCreated, nil
}

func InsertActivity(activity Activity) (string, error) {

	statement, err := database.Db.Prepare("INSERT INTO activity (id, type, user_id, activities, message, ref_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, activity.Type, activity.UserId, activity.Activities, activity.Message, activity.RefId, time.Now())
	if err != nil {
		return "", err
	}
	return "", nil
}

func CheckOrganizationBySlug(slug string) (*model.Organization, error) {
	var org model.Organization
	constraintString := slug
	query := "SELECT id, parent_orgid, name, slug, type, domains FROM organization WHERE slug=? and is_deleted = 0"
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

func updateUserProfileCreated(userid string, userProfileCreated bool) error {
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

func UserRegisterOnBoard(w http.ResponseWriter, r *http.Request) {

	var dataBody UserRegisterRequestBody
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	dataBody.FirstName = "Nife"
	dataBody.LastName = "User"
	dataBody.Password = "b03b1ee0b28f77129e0800b20a38f495"
	dataBody.Industry = "Undefined"
	dataBody.Location = "New Delhi, India"
	dataBody.PhoneNumber = "183aec5711d714150218ec782d8e90c2"

	companyName, err := extractUsername(dataBody.Email)

	dataBody.CompanyName = companyName

	err = validateStructReg(dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Mandatory fields are empty"})
		return
	}

	password := decode.DePwdCode(dataBody.Password)

	if password == "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid Credentials"})
		return

	}

	checkUser, _ := GetUserIdByEmail(dataBody.Email)

	if checkUser != 0 {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Email already register! Login directly.."})
		return
	}
	if dataBody.CompanyName == "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Company name should not be empty!"})
		return

	}

	if dataBody.Location == "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Location field should not be empty!"})
		return

	}

	checkCompany, err := CheckCompanyExists(dataBody.CompanyName)

	if checkCompany != "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Try with different company name!"})
		return
	}
	CheckOrganization, err := CheckOrganizationExists(dataBody.CompanyName)

	if CheckOrganization != "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Try with different company name!"})
		return
	}

	//
	compId := uuid.NewString()

	user := User{Email: dataBody.Email, Password: password, PhoneNumber: dataBody.PhoneNumber,
		Location: dataBody.Location, Industry: dataBody.Industry,
		CompanyName: dataBody.CompanyName, FirstName: dataBody.FirstName,
		LastName: dataBody.LastName, CreatedAt: time.Now(), RoleId: 1, CompanyId: compId}

	userId, err := user.Create()
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	company := Company{ID: compId, CompanyName: dataBody.CompanyName, ExternalId: dataBody.Email}

	err = company.InsertCompanyDetails()
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	companyDet, err := GetCompanyDetails(dataBody.CompanyName)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	err = UpdateEmailVerify(dataBody.Email)
	if err != nil {
		return
	}
	accessToken, _, err := GenerateAccessAndRefreshToken(user.Email, "", false, user.FirstName, user.LastName, user.CompanyName, user.RoleId, "")
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	requestData := map[string]interface{}{
		"data": map[string]interface{}{
			"attributes": map[string]interface{}{
				"email":    dataBody.Email,
				"password": dataBody.Password,
			},
		},
	}

	// Convert the data to JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}
	endPoints := os.Getenv("LOGIN_API_ENDPOINT")

	response, err := http.Post(endPoints, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, "Error making request", http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	_, err = ioutil.ReadAll(response.Body)
	if err != nil {
		http.Error(w, "Error reading response", http.StatusInternalServerError)
		return
	}

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"user_Id":      userId,
		"access_token": accessToken,
		"company":      companyDet,
	})

}

func extractUsername(email string) (string, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid email format")
	}
	return parts[0], nil
}

// is_email_verified

func UpdateEmailVerify(email string) error {
	statement, err := database.Db.Prepare("UPDATE user set is_email_verified = ?, is_free_plan = ? where email = ?")
	if err != nil {
		log.Println(err)
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(true, true, email)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
