package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/log4go"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal/auth"
	"github.com/nifetency/nife.io/internal/decode"
	inviteuser "github.com/nifetency/nife.io/internal/invite_user"
	oragnizationUsers "github.com/nifetency/nife.io/internal/organizaiton_users"
	"github.com/nifetency/nife.io/internal/stripes"
	"github.com/nifetency/nife.io/internal/users"
	_helper "github.com/nifetency/nife.io/pkg/helper"
	"github.com/nifetency/nife.io/pkg/jwt"
	"github.com/nifetency/nife.io/service"
)

func (r *mutationResolver) UpdateUser(ctx context.Context, input *model.UpdateUserInput) (*model.UpdateUser, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.UpdateUser{}, fmt.Errorf("Access Denied")
	}

	if !service.ValidateFields(*input.FirstName, *input.LastName, *input.LastName, *input.Industry) {
		return nil, fmt.Errorf("The required fields cannot be empty or have spaces in them")
	}
	phoneNumber := decode.DePwdCode(*input.PhoneNumber)
	phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")

	if !service.IsValidMobileNumber(phoneNumber) {
		return nil, fmt.Errorf("Invalid phone number")
	}

	get, _ := service.GetUserDetails(*input.ID)
	if get.Email == "" {
		return &model.UpdateUser{}, fmt.Errorf("invalid User Id")
	}

	role, err := service.GetUserRole(*input.ID)
	if err != nil {
		log4go.Error("Module: UpdateUser, MethodName: GetUserRole, Message: %s user:%s", err.Error(), user.ID)
		return &model.UpdateUser{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UpdateUser, MethodName: GetUserRole, Message: Fetching user role by userid is successfully completed, user: %s", user.ID)

	if role == "1" {

		update, _ := service.UpdateUserDetails(*input.ID, *input.PhoneNumber, *input.CompanyName, *input.Location, *input.Industry, *input.FirstName, *input.LastName, user.ID, *input.Mode)
		return &update, nil

	} else {
		mode := false
		if input.Mode == nil {
			input.Mode = &mode
		}
		inviteUserUpdate, _ := service.UpdateInviteUserDetails(*input.ID, *input.PhoneNumber, *input.Location, *input.Industry, *input.FirstName, *input.LastName, *input.Mode)
		return &inviteUserUpdate, nil
	}
}

func (r *mutationResolver) ChangePassword(ctx context.Context, input model.ChangePassword) (*model.Password, error) {
	user := auth.ForContext(ctx)
	var check bool
	if user == nil {
		return &model.Password{}, fmt.Errorf("Access Denied")
	}

	input.Oldpassword = decode.DePwdCode(input.Oldpassword)
	input.NewPassword = decode.DePwdCode(input.NewPassword)

	if input.Oldpassword == "" || input.NewPassword == "" {
		return &model.Password{}, fmt.Errorf("Something Went Wrong with password decode")
	}

	var message string
	get, _ := service.GetUserPassword(input.ID)
	if get.Password == "" {
		return &model.Password{}, fmt.Errorf("invalid user id")
	} else {
		check = users.CheckPasswordHash(input.Oldpassword, get.Password)
		if !check {
			return &model.Password{}, fmt.Errorf("Old password doesn't match")
		}
		if check {
			newHash, _ := users.HashPassword(input.NewPassword)
			newPassword, _ := service.UpdateUserPassword(input.ID, newHash, user.ID)
			message = *newPassword.Message
		}
	}
	now := time.Now().UTC()
	return &model.Password{
		Message:   &message,
		UpdatedAt: &now,
	}, nil
}

func (r *mutationResolver) ActiveUser(ctx context.Context, isActive *bool, isDelete *bool) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	if *isActive {
		err := service.UpdateUserActive(user.ID, *isActive)
		if err != nil {
			log4go.Error("Module: ActiveUser, MethodName: UpdateUserActive, Message: %s user:%s", err.Error(), user.ID)
			return "", fmt.Errorf(err.Error())
		}
		log4go.Info("Module: ActiveUser, MethodName: UpdateUserActive, Message: Successfully Updated the user active , user: %s", user.ID)

	} else if !*isActive {
		err := service.UpdateUserActive(user.ID, *isActive)
		if err != nil {
			log4go.Error("Module: ActiveUser, MethodName: UpdateUserActive, Message: %s user:%s", err.Error(), user.ID)
			return "", fmt.Errorf(err.Error())
		}
		log4go.Info("Module: ActiveUser, MethodName: UpdateUserActive, Message: Successfully Updated the user active , user: %s", user.ID)
	}

	if *isDelete {
		err := service.UpdateUserDetele(user.ID, *isDelete)
		if err != nil {
			log4go.Error("Module: ActiveUser, MethodName: UpdateUserDetele, Message: %s user:%s", err.Error(), user.ID)
			return "", fmt.Errorf(err.Error())
		}
		log4go.Info("Module: ActiveUser, MethodName: UpdateUserDetele, Message: Successfully Updated the user inactive , user: %s", user.ID)
	} else if !*isDelete {
		err := service.UpdateUserDetele(user.ID, *isDelete)
		if err != nil {
			log4go.Error("Module: ActiveUser, MethodName: UpdateUserDetele, Message: %s user:%s", err.Error(), user.ID)
			return "", fmt.Errorf(err.Error())
		}
		log4go.Info("Module: ActiveUser, MethodName: UpdateUserDetele, Message: Successfully Updated the user inactive , user: %s", user.ID)
	}
	return "Updated Successfully", nil
}

func (r *mutationResolver) InviteUser(ctx context.Context, input *model.InviteUser) (*model.InviteUserOutputMessage, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.InviteUserOutputMessage{}, fmt.Errorf("Access Denied")
	}
	//---------plans and permissions
	getPermission, err := stripes.GetCustPlanName(user.CustomerStripeId)
	if err != nil {
		log4go.Error("Module: InviteUser, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: InviteUser, MethodName: GetCustPlanName, Message: Checking for permission to create workload, user: %s", user.ID)

	permissions, err := users.GetCustomerPermissionByPlan(getPermission)
	if err != nil {
		log4go.Error("Module: InviteUser, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: InviteUser, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:, user: %s", user.Email)

	userss, err := service.GetById(user.ID)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: InviteUser, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: InviteUser, MethodName: GetById, Message: successfully reached, user: %s", user.ID)

	inviteUserCount, err := service.GetInviteUserCountByCompanyId(*userss.CompanyID)
	if err != nil {
		log4go.Error("Module: InviteUser, MethodName: GetInviteUserCountByCompanyId, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	limit := strconv.Itoa(permissions.InviteUserLimit)
	log4go.Info("Module: InviteUser, MethodName: GetInviteUserCountByCompanyId, Message: Checking for permission to create invite user, Permission : "+limit+", user: %s", user.ID)

	if inviteUserCount >= permissions.InviteUserLimit {
		if permissions.InviteUserLimit == 0 {
			return nil, fmt.Errorf("Upgrade your plan to unlock Invite user feature")

		}
		return nil, fmt.Errorf("You've reached your maximum limit of current plan. Please upgrade your plan to invite users")
	}
	//-----------
	if len(input.Organization) == 0 {
		defOrg, err := oragnizationUsers.GetDefaultOrganization(user.ID)
		if err != nil {
			log4go.Error("Module: InviteUser, MethodName: GetDefaultOrganization, Message: %s user:%s", err.Error(), user.ID)
			return &model.InviteUserOutputMessage{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: InviteUser, MethodName: GetDefaultOrganization, Message: Get default organization of the user is successfully completed, user: %s", user.ID)

		input.Organization = append(input.Organization, &defOrg)
	}

	GetUserName, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: InviteUser, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return &model.InviteUserOutputMessage{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: InviteUser, MethodName: GetById, Message:successfully reached, user: %s", user.ID)

	companyNameOrg, err := users.GetCompanyNameByEmail(user.Email)
	if err != nil {
		return nil, err
	}

	firstName := GetUserName.FirstName
	lastName := GetUserName.LastName

	userId, err := users.GetUserIdByEmail(*input.UserEmail)
	if err != nil {
		log4go.Error("Module: InviteUser, MethodName: GetUserIdByEmail, Message: %s user:%s", err.Error(), user.ID)
		return &model.InviteUserOutputMessage{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: InviteUser, MethodName: GetUserIdByEmail, Message: Fetch user id by email is successfully completed, user: %s", user.ID)

	if userId != 0 {
		if err = oragnizationUsers.InviteUserAddOrgs(input.Organization, int(userId), 2); err != nil {
			return &model.InviteUserOutputMessage{}, fmt.Errorf(err.Error())
		}

		msgstring := "User Invited Successfully"
		return &model.InviteUserOutputMessage{
			Message: &msgstring,
			UserID:  &userId,
		}, nil
	} else {
		password, err := _helper.TemporaryPassword()

		if err != nil {
			return &model.InviteUserOutputMessage{}, fmt.Errorf(err.Error())
		}

		GetUserName.CompanyName = companyNameOrg
		createUser := users.User{
			Email:       *input.UserEmail,
			Password:    password,
			CompanyName: GetUserName.CompanyName,
			CreatedAt:   time.Now(),
			RoleId:      2,
			CompanyId:   *GetUserName.CompanyID,
		}
		userId, err := createUser.Create()

		if err != nil {
			return &model.InviteUserOutputMessage{}, fmt.Errorf(err.Error())
		}

		if err = oragnizationUsers.InviteUserAddOrgs(input.Organization, int(userId), 2); err != nil {
			return &model.InviteUserOutputMessage{}, err
		}

		if err = oragnizationUsers.UpdateTempPwdandIsUserInvite(*input.UserEmail, true); err != nil {
			return &model.InviteUserOutputMessage{}, err

		}

		orgs := ""

		if len(input.Organization) > 1 {
			orgs = strconv.Itoa(len(input.Organization))
		} else {
			orgsDetails, err := service.GetOrganizationById(*input.Organization[0])
			if err != nil {
				return &model.InviteUserOutputMessage{}, err
			}
			orgs = *orgsDetails.Slug
		}

		err = inviteuser.SentInvite(user.Email, *input.UserEmail, password, orgs, *firstName+*lastName)
		if err != nil {
			log4go.Error("Module: InviteUser, MethodName: SentInvite, Message: %s user:%s", err.Error(), user.ID)
			return &model.InviteUserOutputMessage{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: InviteUser, MethodName: SentInvite, Message: Email sent successfully to the invited user, user: %s", user.ID)
	}

	userDet, err := users.GetUserIdByEmail(*input.UserEmail)
	if err != nil {
		log4go.Error("Module: InviteUser, MethodName: GetUserIdByEmail, Message: %s user:%s", err.Error(), user.ID)
		return &model.InviteUserOutputMessage{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: InviteUser, MethodName: GetUserIdByEmail, Message: Fetch user id by email is successfully completed, user: %s", user.ID)

	msgstring := "User Invited Successfully"
	return &model.InviteUserOutputMessage{
		Message: &msgstring,
		UserID:  &userDet,
	}, nil
}

func (r *mutationResolver) AddInviteUserRole(ctx context.Context, email string, roleID int) (*string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	err := service.UpdateRoleByEmail(email, roleID)
	if err != nil {
		return nil, err
	}
	message := "successfully updated"
	return &message, nil
}

func (r *mutationResolver) RemoveUserOrg(ctx context.Context, organizationID *string, userID *string) (*string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	idCheck, err := service.GetUserByOrganization(*userID, *organizationID)
	if err != nil {
		log4go.Error("Module: RemoveUserOrg, MethodName: GetUserByOrganization, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: RemoveUserOrg, MethodName: GetUserByOrganization, Message: checking the organization is mapped with the user, user: %s", user.ID)

	if idCheck == "" {
		return nil, fmt.Errorf("The user is not mapped with the organization you have seleccted")
	}

	err = service.UpdateUserInActive(*userID, *organizationID, false)
	if err != nil {
		log4go.Error("Module: RemoveUserOrg, MethodName: UpdateUserInActive, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: RemoveUserOrg, MethodName: UpdateUserInActive, Message: Updating user as inactive is successfully completed, user: %s", user.ID)

	message := "Selected organization removed for the User"
	return &message, nil
}

func (r *mutationResolver) UserProfileUpdated(ctx context.Context, userID *string, userProfileCreated *bool) (*string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	err := service.UserProfileCreated(*userID, *userProfileCreated)
	if err != nil {
		log4go.Error("Module: UserProfileUpdated, MethodName: UserProfileCreated, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UserProfileUpdated, MethodName: UserProfileCreated, Message: User profile is successfully updated, user: %s", user.ID)

	message := "Updated Successfully"
	return &message, nil
}

func (r *mutationResolver) AddUserToOrg(ctx context.Context, input *model.AddUser) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	if len(input.OrganizationID) == 0 {
		defOrg, err := oragnizationUsers.GetDefaultOrganization(user.ID)
		if err != nil {
			log4go.Error("Module: AddUserToOrg, MethodName: GetDefaultOrganization, Message: %s user:%s", err.Error(), user.ID)
			return "", fmt.Errorf(err.Error())
		}
		log4go.Info("Module: AddUserToOrg, MethodName: GetDefaultOrganization, Message: Get default organization of the user is successfully completed, user: %s", user.ID)
		input.OrganizationID = append(input.OrganizationID, &defOrg)
	}

	if *input.UserID != 0 {
		if err := oragnizationUsers.InviteUserAddOrgs(input.OrganizationID, *input.UserID, 2); err != nil {
			return "", err
		}
	}
	return "User Successfully Invited", nil
}

func (r *mutationResolver) UploadCompanyLogo(ctx context.Context, input *model.Image) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	userCompany, err := service.GetById(user.ID)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: UploadCompanyLogo, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UploadCompanyLogo, MethodName: GetById, Message: Company logo is successfully uploaded, user: %s", user.ID)

	err = service.UpdateImageByCompanyId(input.LogoURL, *userCompany.CompanyID)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: UploadCompanyLogo, MethodName: UpdateImageByCompanyId, Message: %s user:%s", err.Error(), user.ID)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UploadCompanyLogo, MethodName: UpdateImageByCompanyId, Message: Update Image url to the company is successfully completed, user: %s", user.ID)

	return "Uploaded Successfully", err
}

func (r *mutationResolver) RemoveInviteuser(ctx context.Context, userID *string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	if user.RoleId == 1 {
		userCompany, err := users.GetCompanyNameById(*userID)
		if err != nil {
			return "", fmt.Errorf(err.Error())
		}
		if user.CompanyName != userCompany {
			return "You haven't Invited this User", nil
		}
		err = service.UpdateInviteUserInactive(*userID)
		if err != nil {
			log4go.Error("Module: RemoveInviteuser, MethodName: UpdateInviteUserInactive, Message: %s user:%s", err.Error(), user.ID)
			return "", fmt.Errorf(err.Error())
		}
		log4go.Info("Module: RemoveInviteuser, MethodName: UpdateInviteUserInactive, Message: Updating invite user as inactive is successfully completed, user: %s", user.ID)
	} else {
		return "User do not have the permission to remove", nil
	}

	return "Removed the User", nil
}

func (r *mutationResolver) NotificationInfo(ctx context.Context, input *model.Notification) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	for _, i := range input.ID {
		err := service.UpdateActivityAsRead(*i, *input.IsRead)
		if err != nil {
			log.Println(err)
			return "", fmt.Errorf(err.Error())
		}
	}
	return "Successfully Updated", nil
}

func (r *mutationResolver) UserRequestingByoh(ctx context.Context, input *model.ByohRequest) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	getPermission, err := stripes.GetCustPlanName(user.CustomerStripeId)
	if err != nil {
		log4go.Error("Module: CreateWorkloadManagement, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.ID)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: CreateWorkloadManagement, MethodName: GetCustPlanName, Message: Checking for permission to create workload, user: %s", user.ID)

	permissions, err := users.GetCustomerPermissionByPlan(getPermission)
	if err != nil {
		log4go.Error("Module: CreateWorkloadManagement, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
		return "", fmt.Errorf(err.Error())
	}
	log4go.Info("Module: CreateWorkloadManagement, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:, user: %s", user.Email)

	if permissions.Byoh == false {
		return "", fmt.Errorf("Please upgrade your plan to get access")
	}

	err = service.InsertByohDetails(user.ID, *input)
	if err != nil {
		log.Println(err)
		return "", fmt.Errorf(err.Error())
	}
	userDet, err := users.GetEmailById(user.ID)
	if err != nil {
		return "", err
	}
	emails := []string{"nida@nife.io", "afzal.jan@nife.io", "srinivasannathan@nife.io", "jigar@nife.io", "revathi@nife.io"}
	for _, email := range emails {
		helper.RequestingBYOH(email, user.Email, *input.UserName, *input.Password, *input.IPAddress, *input.Region, userDet.FirstName+" "+userDet.LastName)
	}

	return "Successfully Updated", nil
}

func (r *mutationResolver) RequestingPicoNets(ctx context.Context, appName *string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	name := user.FirstName + " " + user.LastName

	err := helper.RequestingMarketPlaceApp(*appName, user.Email, name)
	if err != nil {
		log.Println(err)
		return "", fmt.Errorf(err.Error())
	}

	return "", nil
}

func (r *mutationResolver) SetUserTokenExpireTime(ctx context.Context, expireTime *int) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	token, err := jwt.GenerateToken(user.Email, user.StripeProductId, strconv.Itoa(*expireTime), user.FirstName, user.LastName, user.CompanyName, user.RoleId, user.CustomerStripeId)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (r *mutationResolver) UpdateUserwebhookURLSlack(ctx context.Context, webhookURL *string) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	err := service.UpdateUserSlackWebhookURL(user.ID, *webhookURL)
	if err != nil {
		return "", err
	}

	return "Added Successfully", nil
}

func (r *queryResolver) GetUserByID(ctx context.Context) (*model.GetUserByID, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.GetUserByID{}, fmt.Errorf("Access Denied")
	}

	getEmail, _ := service.GetUserDetails(user.ID)
	if getEmail.Email == "" {
		return &model.GetUserByID{}, fmt.Errorf("There is no user present in the Id: %s", user.ID)
	}

	getId, _ := service.GetById(user.ID)
	CompanyLogo, _ := service.GetCompanyLogo(*getId.CompanyID)
	response := model.GetUserByID{
		Email:           getId.Email,
		CompanyName:     getId.CompanyName,
		PhoneNumber:     getId.PhoneNumber,
		Location:        getId.Location,
		Industry:        getId.Industry,
		FirstName:       getId.FirstName,
		LastName:        getId.LastName,
		SsoType:         getId.SsoType,
		FreePlan:        getId.FreePlan,
		ProfileImageURL: getId.ProfileImageURL,
		Companylogo:     CompanyLogo,
		Mode:            getId.Mode,
		SlackWebhookURL: getId.SlackWebhookURL,
	}
	return &response, nil
}

func (r *queryResolver) CurrentUser(ctx context.Context) (*model.CurrentUserEmail, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.CurrentUserEmail{}, fmt.Errorf("Access Denied")
	}
	email, err := users.GetEmailById(user.ID)
	if err != nil {
		log.Println(err)
		return &model.CurrentUserEmail{}, fmt.Errorf(err.Error())
	}
	result := model.CurrentUserEmail{
		Email:     email.Email,
		FirstName: email.FirstName,
		LastName:  email.LastName,
	}
	return &result, nil
}

func (r *queryResolver) GetUserByAdmin(ctx context.Context) ([]*model.GetUserByID, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.GetUserByID{}, fmt.Errorf("Access Denied")
	}

	userss, err := service.GetById(user.ID)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: GetUserByAdmin, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return []*model.GetUserByID{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetUserByAdmin, MethodName: GetById, Message:successfully reached, user: %s", user.ID)

	adminUsers, err := service.GetInviteUserByCompanyId(*userss.CompanyID)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: GetUserByAdmin, MethodName: GetInviteUserByCompanyId, Message: %s user:%s", err.Error(), user.ID)
		return []*model.GetUserByID{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetUserByAdmin, MethodName: GetInviteUserByCompanyId, Message: Fetching invite user by company name is successfully completed, user: %s", user.ID)

	for _, adminuser := range adminUsers {
		orgDet, err := service.AllOrganizations(*adminuser.ID)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: GetUserByAdmin, MethodName: AllOrganizations, Message: %s user:%s", err.Error(), user.ID)
			return []*model.GetUserByID{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: GetUserByAdmin, MethodName: AllOrganizations, Message: Fetching all organization based on admin Id is successfully completed, user: %s", user.ID)

		adminuser.Organization = append(adminuser.Organization, orgDet)

	}

	return adminUsers, nil
}

func (r *queryResolver) GetUserByAdminAndOrganization(ctx context.Context, organizationID *string) ([]*model.GetUserByID, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.GetUserByID{}, fmt.Errorf("Access Denied")
	}

	return nil, nil
}

func (r *queryResolver) UserActivities(ctx context.Context, first *int) ([]*model.UserActivities, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.UserActivities{}, fmt.Errorf("Access Denied")
	}

	// getPermission, err := stripes.GetCustPlanName(user.CustomerStripeId)
	// if err != nil {
	// 	log4go.Error("Module: CreateWorkloadManagement, MethodName: GetCustPlanName, Message: %s user:%s", err.Error(), user.ID)
	// 	return nil, fmt.Errorf(err.Error())
	// }
	// log4go.Info("Module: CreateWorkloadManagement, MethodName: GetCustPlanName, Message: Checking for permission to create workload, user: %s", user.ID)

	// permissions, err := users.GetCustomerPermissionByPlan(getPermission)
	// if err != nil {
	// 	log4go.Error("Module: CreateWorkloadManagement, MethodName: GetCustomerPermissionByPlan, Message: %s user:%s", err.Error(), user.Email)
	// 	return nil, fmt.Errorf(err.Error())
	// }
	// log4go.Info("Module: CreateWorkloadManagement, MethodName: GetCustomerPermissionByPlan, Message: Get user permission with plan:, user: %s", user.Email)

	// if permissions.AppNotification == false {
	// 	return nil, fmt.Errorf("Please upgrade your plan to get access for Notifications")
	// }

	activities, err := service.GetUserActivitiesByUserId(user.ID, *first)
	if err != nil {
		log4go.Error("Module: UserActivities, MethodName: GetUserActivitiesByUserId, Message: %s user:%s", err.Error(), user.ID)
		return []*model.UserActivities{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UserActivities, MethodName: GetUserActivitiesByUserId, Message: Fetching activities for the user , user: %s", user.ID)
	return activities, nil
}

func (r *queryResolver) UserActivitiesByDate(ctx context.Context, startDate *string, endDate *string) ([]*model.UserActivities, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return []*model.UserActivities{}, fmt.Errorf("Access Denied")
	}
	activities, err := service.GetUserActivitiesByUserIdAndDates(user.ID, *startDate, *endDate)
	if err != nil {
		log4go.Error("Module: UserActivities, MethodName: GetUserActivitiesByUserIdAndDates, Message: %s user:%s", err.Error(), user.ID)
		return []*model.UserActivities{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: UserActivities, MethodName: GetUserActivitiesByUserIdAndDates, Message: Fetching activities for the user based on Start date: "+*startDate+" and End date: "+*endDate+" , user: %s", user.ID)
	return activities, nil
}

func (r *queryResolver) GetInviteUserCountByAdminUser(ctx context.Context) (*int, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	userss, err := service.GetById(user.ID)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: GetInviteUserCountByAdminUser, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetInviteUserCountByAdminUser, MethodName: GetById, Message: successfully reached, user: %s", user.ID)

	inviteUserCount, err := service.GetInviteUserCountByCompanyId(*userss.CompanyID)
	if err != nil {
		log4go.Error("Module: GetInviteUserCountByAdminUser, MethodName: GetInviteUserCountByCompanyId, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: GetInviteUserCountByAdminUser, MethodName: GetInviteUserCountByCompanyId, Message: Fetching invited users count by company name:"+user.CompanyName+" , user: %s", user.ID)

	return &inviteUserCount, nil
}

func (r *queryResolver) UserDeploymentCountDetails(ctx context.Context, startDate *string, endDate *string) ([]*model.UserDeploymentDetailCount, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	userss, err := service.GetById(user.ID)
	if err != nil {
		log.Println(err)
		return []*model.UserDeploymentDetailCount{}, fmt.Errorf(err.Error())
	}

	adminUsers, err := service.GetInviteUserByCompanyId(*userss.CompanyID)
	if err != nil {
		log.Println(err)
		return []*model.UserDeploymentDetailCount{}, fmt.Errorf(err.Error())
	}
	getUserDeplymentDet, err := service.GetDeploymentDetailsByUser(adminUsers, *startDate, *endDate)
	if err != nil {
		return []*model.UserDeploymentDetailCount{}, fmt.Errorf(err.Error())
	}

	return getUserDeplymentDet, nil
}

func (r *queryResolver) GetUserByOrganizationID(ctx context.Context, organizationID *string) ([]*model.GetUserByID, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	invitedUser, err := service.GetInviteUserByOrganizationId(*organizationID)
	if err != nil {
		return []*model.GetUserByID{}, err
	}

	return invitedUser, nil
}
