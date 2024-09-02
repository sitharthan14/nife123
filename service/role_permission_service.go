package service

import (
	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func GetUserPermission(userId string) ([]*model.Permission, error) {
	query := `select pm.id, pm.module, pm.title, pm.create, pm.view,pm.delete, pm.update, pm.is_active, pm.created_at
     from permission pm
	 INNER JOIN role_permission rp ON pm.id = rp.permission_id
	 INNER JOIN user ON user.role_id = rp.role_id
	 where user.id = ?`

	selDB, err := database.Db.Query(query, userId)
	if err != nil {
		return []*model.Permission{}, err
	}
	defer selDB.Close()
	result := []*model.Permission{}
	for selDB.Next() {
		var permission model.Permission
		err = selDB.Scan(&permission.ID, &permission.Module, &permission.Title, &permission.Create, &permission.View, &permission.Delete, &permission.Update, &permission.IsActive, &permission.CreatedAt)
		if err != nil {
			return []*model.Permission{}, err
		}
		result = append(result, &permission)
	}
	return result, nil
}

func UpdateRole(userid string, roleId int) error {
	statement, err := database.Db.Prepare("update user set role_id = ? where id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(roleId, userid)
	if err != nil {
		return err
	}
	return nil
}
func UpdateRoleByEmail(email string, roleId int) error {
	statement, err := database.Db.Prepare("update user set role_id = ? where email = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(roleId, email)
	if err != nil {
		return err
	}
	return nil
}

//---------plans and permissions
func GetCustomerPermissionByPlans(planName string) (*model.PlanAndPermission, error) {

	selDB, err := database.Db.Query(`SELECT id, plan_name, apps, workload_management, organisation_management, invite_user_limit, application_health_dashboard, byoh, storage, version_control_panel, single_sign_on, organization_count, sub_organization_count, businessunit_count, custom_domain, app_notification, secret, monitoring_platform, alerts_advisories, audit_logs, ssl_security, infrastructure_configuration, replicas, k8s_region
	FROM plans_permissions where plan_name = ?;`, planName)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	var planAndPermissions model.PlanAndPermission
	for selDB.Next() {
		err = selDB.Scan(&planAndPermissions.ID, &planAndPermissions.PlanName, &planAndPermissions.Apps, &planAndPermissions.WorkloadManagement, &planAndPermissions.OrganizationManagement, &planAndPermissions.InviteUserLimit, &planAndPermissions.ApplicationHealthDashboard, &planAndPermissions.Byoh, &planAndPermissions.Storage, &planAndPermissions.VersionControlPanel, &planAndPermissions.SingleSignOn, &planAndPermissions.OrganizationCount, &planAndPermissions.SubOrganizationCount, &planAndPermissions.BusinessunitCount, &planAndPermissions.CustomDomain, &planAndPermissions.AppNotification, &planAndPermissions.Secret, &planAndPermissions.MonitoringPlatform, &planAndPermissions.AlertsAdvisories, &planAndPermissions.AuditLogs, &planAndPermissions.SslSecurity, &planAndPermissions.InfrastructureConfiguration, &planAndPermissions.Replicas, &planAndPermissions.K8sRegions)
		if err != nil {
			return &model.PlanAndPermission{}, err
		}
	}
	return &planAndPermissions, nil
}

//---------------
