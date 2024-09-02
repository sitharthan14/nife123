package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/alecthomas/log4go"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/internal/auth"
	organizationInfo "github.com/nifetency/nife.io/internal/organization_info"
	servicelogs "github.com/nifetency/nife.io/internal/service_logs"
	_helper "github.com/nifetency/nife.io/pkg/helper"
	"github.com/nifetency/nife.io/service"
)

func (r *mutationResolver) MigrateOrganization(ctx context.Context, input model.MigrateOrganizationInput) (string, error) {
	moduleName := "MigrateOrganization"
	emptyStr := ""
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	organizationss, err := service.AllOrganizations(user.ID)
	if err != nil {
		servicelogs.ErrorLog(moduleName, "AllOrganizations", user.ID, err)
		return "", err
	}
	servicelogs.SuccessLog(moduleName, "AllOrganizations", "Fetching all the organization to get the count of the organization", user.ID)

	if len(organizationss.Nodes) <= 1 {
		return "", fmt.Errorf("There only one Organization in user account. Create a new organization and try to migrate")
	}

	orgDetails, err := service.GetOrganizationById(*input.OrganizationIDFrom)
	if err != nil {
		servicelogs.ErrorLog(moduleName, "GetOrganizationById", user.ID, err)
		return "", err
	}
	servicelogs.SuccessLog(moduleName, "GetOrganizationById", "Fetching the organization details using organization Id: "+*input.OrganizationIDFrom, user.ID)

	appsUnderOrgAndOrgBU, err := service.GetAppOnlyInOrganization(user.ID, emptyStr, *orgDetails.Slug)
	servicelogs.SuccessLog(moduleName, "GetAppOnlyInOrganization", "Fetching the Apps inside the organization: "+*orgDetails.Slug, user.ID)

	for _, appss := range appsUnderOrgAndOrgBU.Nodes {
		if appss.Status == "New" {
			err = service.MigrateApps(*input.OrganizationIDTo, appss.Name)
			if err != nil {
				servicelogs.ErrorLog(moduleName, "MigrateApps", user.ID, err)
				return "", err
			}
			servicelogs.SuccessLog(moduleName, "MigrateApps", "Migrating the Apps in New State from the organization: "+*input.OrganizationIDFrom+" to "+*input.OrganizationIDTo, user.ID)
		} else {
			var regionCode string
			for _, reg := range appss.Regions {
				regionCode = *reg.Code
			}
			r.DeleteApp(ctx, appss.Name, regionCode)
			var businessUnitId string
			if *appss.BusinessUnitID != "" {
				businessUnitId = *appss.BusinessUnitID
			} else {
				businessUnitId = emptyStr
			}
			randomNumber := organizationInfo.RandomNumber4Digit()
			randno := strconv.Itoa(int(randomNumber))

			index := strings.LastIndex(appss.Name, "-")
			if index != -1 {
				appss.Name = appss.Name[:index]
			}
			newAppName := appss.Name + "-" + randno

			r.CreateApp(ctx, model.CreateAppInput{Name: newAppName, Runtime: emptyStr, OrganizationID: *input.OrganizationIDTo, SubOrganizationID: emptyStr, BusinessUnitID: businessUnitId, WorkloadManagementID: *appss.WorkloadManagementID})

			internalPort, _ := _helper.GetInternalPort(appss.Config.Definition)
			externalPort, _ := _helper.GetExternalPort(appss.Config.Definition)

			build, _ := json.Marshal(appss.Config.Build)

			memoryManage := _helper.GetResourceRequirement(appss.Config.Definition)
			memoryRes := "{\"RequestRequirement\":{\"cpu\":\"" + *memoryManage.RequestRequirement.CPU + "\",\"Memory\":\"" + *memoryManage.RequestRequirement.Memory + "\"},\"LimitRequirement\":{\"cpu\":\"" + *memoryManage.LimitRequirement.CPU + "\",\"Memory\":\"" + *memoryManage.LimitRequirement.Memory + "\"}}"

			r.UpdateApp(ctx, model.UpdateAppInput{AppID: newAppName, InternalPort: strconv.Itoa(int(internalPort)), ExternalPort: strconv.Itoa(int(externalPort)), Build: string(build), RoutingPolicy: "Latency", Resource: memoryRes, Replicas: 1})
			r.DeployImage(ctx, model.DeployImageInput{AppID: newAppName, Image: *appss.ImageName, Definition: appss.Config.Definition, Strategy: nil, Services: []map[string]interface{}{}, EnvArgs: nil, EnvMapArgs: nil, ArchiveURL: &emptyStr})
		}
	}

	//--------Get sub organization-------
	subOrganizationsUnderFromOrg, err := service.SubOrganizationsById(user.ID, *input.OrganizationIDFrom)
	if err != nil {
		servicelogs.ErrorLog(moduleName, "SubOrganizationsById", user.ID, err)
		return "", err
	}
	servicelogs.SuccessLog(moduleName, "SubOrganizationsById", "Fetching Sub-Organiztions under the Organization: "+*input.OrganizationIDFrom, user.ID)

	//---------Apps under the Sub Organization
	var AppInSubOrgArr []model.AppInSubOrg
	for _, appsSubOrg := range subOrganizationsUnderFromOrg.Nodes {
		appsUnderSubOrg, err := service.AllSubApps(user.ID, emptyStr, *appsSubOrg.Slug)
		if err != nil {
			servicelogs.ErrorLog(moduleName, "AllSubApps", user.ID, err)
			return "", err
		}
		servicelogs.SuccessLog(moduleName, "AllSubApps", "Fetching all the Apps under the Sub-Organiztions: "+*appsSubOrg.Slug, user.ID)

		result1 := model.AppInSubOrg{
			SubOrgID:      appsSubOrg.ID,
			SubOrgName:    appsSubOrg.Name,
			AppsInSubOrgs: appsUnderSubOrg.Nodes,
		}
		AppInSubOrgArr = append(AppInSubOrgArr, result1)
	}
	for _, updateApps := range AppInSubOrgArr {
		for _, updateApp1 := range updateApps.AppsInSubOrgs {
			err = service.MigrateApps(*input.OrganizationIDTo, updateApp1.Name)
			if err != nil {
				servicelogs.ErrorLog(moduleName, "MigrateApps", user.ID, err)
				return "", err
			}
			servicelogs.SuccessLog(moduleName, "MigrateApps", "Migrating the Apps under the Sub-Organization from the organization: "+*input.OrganizationIDFrom+" to "+*input.OrganizationIDTo, user.ID)
		}
	}

	//----------Get Businessunit from Parent Organization-------
	businessUnitUnderFromOrg, err := service.GetBusinessUnitByOrgIdOrSubOrgId(*input.OrganizationIDFrom, emptyStr)
	if err != nil {
		servicelogs.ErrorLog(moduleName, "GetBusinessUnitByOrgIdOrSubOrgId", user.ID, err)
		return "", err
	}
	servicelogs.SuccessLog(moduleName, "GetBusinessUnitByOrgIdOrSubOrgId", "Fetching all the Business Unit inside the Organization: "+*orgDetails.Slug, user.ID)

	for _, moveBs := range businessUnitUnderFromOrg {
		err = service.MoveBusinessUnit(*input.OrganizationIDTo, *moveBs.ID, user.ID)
		if err != nil {
			servicelogs.ErrorLog(moduleName, "MoveBusinessUnit", user.ID, err)
			return "", err
		}
		servicelogs.SuccessLog(moduleName, "MoveBusinessUnit", "Migrating all the Business Unit under the Organization: "+*input.OrganizationIDFrom+" to "+*input.OrganizationIDTo, user.ID)
	}
	//-----------Get Businessunit from Sub Organization-------
	var businessUnitListInSubOrgArr []model.BusinessUnitListInSubOrg
	for _, subOrg := range subOrganizationsUnderFromOrg.Nodes {
		businessUnitUnderFromOrgAndSubOrg, err := service.GetBusinessUnitByOrgIdOrSubOrgId(*input.OrganizationIDFrom, *subOrg.ID)
		if err != nil {
			servicelogs.ErrorLog(moduleName, "GetBusinessUnitByOrgIdOrSubOrgId", user.ID, err)
			return "", err
		}
		servicelogs.SuccessLog(moduleName, "GetBusinessUnitByOrgIdOrSubOrgId", "Fetching all the Business Unit inside the Sub-Organization: "+*subOrg.Slug, user.ID)

		result := model.BusinessUnitListInSubOrg{
			SubOrgID:                subOrg.ID,
			SubOrgName:              subOrg.Name,
			BusinessUnitUnderSubOrg: businessUnitUnderFromOrgAndSubOrg,
		}

		businessUnitListInSubOrgArr = append(businessUnitListInSubOrgArr, result)
	}
	var appInBusinessUnitArr []model.AppInBusinessUnit
	for _, buMove := range businessUnitListInSubOrgArr {
		for _, moveBu := range buMove.BusinessUnitUnderSubOrg {
			//-------------Apps under the business unit

			appsUnderBusinessUnit, _ := service.AllBusinessUnitApps(user.ID, emptyStr, *moveBu.Name)
			servicelogs.SuccessLog(moduleName, "AllBusinessUnitApps", "Fetching all the Apps inside the Business Unit: "+*moveBu.Name, user.ID)

			result2 := model.AppInBusinessUnit{
				BusinessUnitID:     moveBu.ID,
				BusinessUnitName:   moveBu.Name,
				AppsInBusinessUnit: appsUnderBusinessUnit.Nodes,
			}
			appInBusinessUnitArr = append(appInBusinessUnitArr, result2)

			err = service.MoveBusinessUnit(*input.OrganizationIDTo, *moveBu.ID, user.ID)
			if err != nil {
				servicelogs.ErrorLog(moduleName, "MoveBusinessUnit", user.ID, err)
				return "", err
			}
			servicelogs.SuccessLog(moduleName, "MoveBusinessUnit", "Migrating all Business Unit from Organization: "+*input.OrganizationIDFrom+" to "+*input.OrganizationIDTo, user.ID)
		}
	}
	for _, appsUnderBU := range appInBusinessUnitArr {
		for _, appsUnderBu1 := range appsUnderBU.AppsInBusinessUnit {
			err = service.MigrateApps(*input.OrganizationIDTo, appsUnderBu1.Name)
			if err != nil {
				return "", err
			}
			servicelogs.SuccessLog(moduleName, "MigrateApps", "Migrating all Apps inside the Business Unit from Organization: "+*input.OrganizationIDFrom+" to "+*input.OrganizationIDTo, user.ID)
		}
	}

	// ---------------- After moving everything inside the sub org
	for _, moveSubOrg := range subOrganizationsUnderFromOrg.Nodes {
		err = service.MoveSubOrg(*input.OrganizationIDTo, *moveSubOrg.ID)
		if err != nil {
			servicelogs.ErrorLog(moduleName, "MoveSubOrg", user.ID, err)
			return "", err
		}
		servicelogs.SuccessLog(moduleName, "MoveSubOrg", "Migrating all the Sub-Organiation from Organization: "+*input.OrganizationIDFrom+" to "+*input.OrganizationIDTo, user.ID)
	}
	// ---------------Get Workload from the Organization
	workloaddetUnderOrg, err := service.GetWorkLoadManagementByOrgIdSubOrgBusinessU(user.ID, *input.OrganizationIDFrom, emptyStr, emptyStr)
	if err != nil {
		servicelogs.ErrorLog(moduleName, "GetWorkLoadManagementByOrgIdSubOrgBusinessU", user.ID, err)
		return "", err
	}
	servicelogs.SuccessLog(moduleName, "GetWorkLoadManagementByOrgIdSubOrgBusinessU", "Fetching the Workload based on the Orgnization: "+*orgDetails.Slug, user.ID)

	for _, wl := range workloaddetUnderOrg {
		err = service.MoveWorkload(*input.OrganizationIDTo, *wl.OrganizationID, user.ID)
		if err != nil {
			servicelogs.ErrorLog(moduleName, "MoveWorkload", user.ID, err)
			return "", err
		}
		servicelogs.SuccessLog(moduleName, "MoveWorkload", "Migrating all the Workloads from Organization: "+*input.OrganizationIDFrom+" to "+*input.OrganizationIDTo, user.ID)
	}

	orgInstance, err := service.GetHostDetails(user.ID, *input.OrganizationIDFrom)
	if err != nil {
		servicelogs.ErrorLog(moduleName, "GetHostDetails", user.ID, err)
		return "", err
	}
	servicelogs.SuccessLog(moduleName, "GetHostDetails", "Fetching all the Instance inside the Organization: "+*orgDetails.Slug, user.ID)

	for _, instance := range orgInstance {

		err = service.UpdateInstanceOrganization(*instance.ID, *input.OrganizationIDTo)
		if err != nil {
			servicelogs.ErrorLog(moduleName, "UpdateInstanceOrganization", user.ID, err)
			return "", err
		}
		servicelogs.SuccessLog(moduleName, "UpdateInstanceOrganization", " Migrating all the Instance from Organization: "+*input.OrganizationIDFrom+" to "+*input.OrganizationIDTo, user.ID)
	}

	orgDetailsNew, err := service.GetOrganizationById(*input.OrganizationIDTo)
	if err != nil {
		servicelogs.ErrorLog(moduleName, "GetOrganizationById", user.ID, err)
		return "", err
	}
	servicelogs.SuccessLog(moduleName, "GetOrganizationById", "Fetching the details of the Organization: "+*orgDetails.Slug, user.ID)

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		servicelogs.ErrorLog(moduleName, "CreateWorkloadManagement", user.ID, err)
		return "", err
	}
	servicelogs.SuccessLog(moduleName, "CreateWorkloadManagement", "GetById, Message: Get user details for activity table by", user.ID)

	AddOperation := service.Activity{
		Type:       "ORGANIZATION",
		UserId:     user.ID,
		Activities: "MIGRATED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Migrated Organization `" + *orgDetails.Name + "` To `" + *orgDetailsNew.Name + "`",
		RefId:      *input.OrganizationIDFrom,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	err = service.SendSlackNotification(user.ID, AddOperation.Message)
	if err != nil {
		log4go.Error("Module: MigrateOrganization, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	return "Successfully Migrated", nil
}
