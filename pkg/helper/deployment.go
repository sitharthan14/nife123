package helper

import (
	appDeployments "github.com/nifetency/nife.io/internal/app_deployments"
	appRelease "github.com/nifetency/nife.io/internal/app_release"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func CreateDeploymentsRecord(deployments appDeployments.AppDeployments) error {
	statement, err := database.Db.Prepare("INSERT INTO app_deployments(id, appId, region_code, status, deployment_id, port, app_url,release_id, createdAt, updatedAt, container_id) VALUES (?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}

	_, err = statement.Exec(deployments.Id, deployments.AppId, deployments.Region_code, deployments.Status, deployments.Deployment_id, deployments.Port, deployments.App_Url, deployments.Release_id, deployments.CreatedAt, deployments.UpdatedAt, deployments.ContainerID)
	if err != nil {
		return err
	}
	return nil
}

func GetDeploymentsRecordSingle(appId string, regionCode string, statusToCheck string) (deployments *appDeployments.AppDeployments, err error) {
	var deployment appDeployments.AppDeployments
	selDB, err := database.Db.Query("SELECT id, region_code, status, deployment_id, port, appId, app_url , release_id, container_id FROM app_deployments where appId = ? and region_code = ? and status = ?", appId, regionCode, statusToCheck)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		err = selDB.Scan(&deployment.Id, &deployment.Region_code, &deployment.Status, &deployment.Deployment_id, &deployment.Port, &deployment.AppId, &deployment.App_Url, &deployment.Release_id, &deployment.ContainerID)
		if err != nil {
			return nil, err
		}
	}
	return &deployment, nil
}

func GetDeploymentsByReleaseId(appId string, releaseId string) (AppDeployments *[]appDeployments.AppDeployments, err error) {
	var deployments []appDeployments.AppDeployments
	selDB, err := database.Db.Query("SELECT id, region_code, status, deployment_id, port, appId, app_url , release_id FROM app_deployments where appId = ? and release_id = ? and status != 'destroyed'", appId, releaseId)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var deployment appDeployments.AppDeployments
		err = selDB.Scan(&deployment.Id, &deployment.Region_code, &deployment.Status, &deployment.Deployment_id, &deployment.Port, &deployment.AppId, &deployment.App_Url, &deployment.Release_id)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}

	return &deployments, nil
}
func GetDeploymentIdByReleaseId(appId string, releaseId string) (AppDeployments *[]appDeployments.AppDeployments, err error) {
	var deployments []appDeployments.AppDeployments
	selDB, err := database.Db.Query("SELECT id, region_code, status, deployment_id, port, appId, app_url , release_id FROM app_deployments where appId = ? and release_id = ? ", appId, releaseId)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var deployment appDeployments.AppDeployments
		err = selDB.Scan(&deployment.Id, &deployment.Region_code, &deployment.Status, &deployment.Deployment_id, &deployment.Port, &deployment.AppId, &deployment.App_Url, &deployment.Release_id)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}

	return &deployments, nil
}

func CreateAppRelease(release appRelease.AppRelease) error {
	statement, err := database.Db.Prepare("INSERT INTO app_release(id, app_id, status,version,user_id, createdAt, image_name, port, archive_url, builder_type, routing_policy) VALUES (?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}

	_, err = statement.Exec(release.Id, release.AppId, release.Status, release.Version, release.UserId, release.CreatedAt, release.ImageName, release.Port, release.ArchiveUrl, release.BuilderType, release.RoutingPolicy)
	if err != nil {
		return err
	}
	return nil
}
func UpdateDeploymentsReleaseId(deployment_id, releaseId string) error {
	statement, err := database.Db.Prepare("UPDATE app_deployments set release_id = ? where id = ?")
	if err != nil {
		return err
	}

	_, err = statement.Exec(releaseId, deployment_id)
	if err != nil {
		return err
	}
	return nil
}

func UpdateAppReleases(release appRelease.AppRelease) error {
	statement, err := database.Db.Prepare("UPDATE app_release SET status = ?, createdAt = ?, image_name = ?, port = ?, archive_url = ? WHERE app_id = ? and version = ?")
	if err != nil {
		return err
	}

	_, err = statement.Exec(release.Status, release.CreatedAt, release.ImageName, release.Port, release.ArchiveUrl, release.AppId, release.Version)
	if err != nil {
		return err
	}
	return nil
}

func UpdateAppRelease(status, releaseId string) error {
	statement, err := database.Db.Prepare("UPDATE app_release set status = ? where id = ?")
	if err != nil {
		return err
	}
	_, err = statement.Exec(status, releaseId)
	if err != nil {
		return err
	}
	return nil
}

// func UpdateAppReleases(release appRelease.AppRelease) error {
// 	statement, err := database.Db.Prepare("UPDATE app_release SET status = ?, createdAt = ?, image_name = ?, port = ?, archive_url = ? WHERE app_id = ? and version = ?")
// 	if err != nil {
// 		return err
// 	}

// 	_, err = statement.Exec(release.Status, release.CreatedAt, release.ImageName, release.Port, release.ArchiveUrl, release.AppId, release.Version)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func GetAppRelease(appId string, statusToCheck string) (release *appRelease.AppRelease, err error) {
	var appRelease appRelease.AppRelease
	selDB, err := database.Db.Query("SELECT id, version, image_name ,port FROM app_release where app_Id = ? and status = ?", appId, statusToCheck)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		err = selDB.Scan(&appRelease.Id, &appRelease.Version, &appRelease.ImageName, &appRelease.Port)
		if err != nil {
			return nil, err
		}
	}

	return &appRelease, nil
}
func GetAppReleaseByVersion(appId string) (release *appRelease.AppRelease, err error) {
	var appRelease appRelease.AppRelease
	selDB, err := database.Db.Query("SELECT id, version, image_name ,port FROM app_release where app_Id = ? and version = 1", appId)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		err = selDB.Scan(&appRelease.Id, &appRelease.Version, &appRelease.ImageName, &appRelease.Port)
		if err != nil {
			return nil, err
		}
	}

	return &appRelease, nil
}
func GetRoutingPolicyInAppRelease(appId string, version int) (routingPol string, err error) {
	selDB, err := database.Db.Query("SELECT routing_policy FROM app_release where app_Id = ? and version = ?", appId, version)
	if err != nil {
		return "", err
	}
	defer selDB.Close()
	for selDB.Next() {
		err = selDB.Scan(&routingPol)
		if err != nil {
			return "", err
		}
	}

	return routingPol, nil
}

func GetAppDeploymentsByOrg(orgId string) (AppDeployments *[]appDeployments.AppDeployments, err error) {
	var deployments []appDeployments.AppDeployments
	selDB, err := database.Db.Query(`SELECT appId,region_code,app_url, elb_record_name, elb_record_id FROM app_deployments ad JOIN app ap ON ad.appId = ap.name 	
	Where ap.organization_id = ? and ad.status = 'running' and ap.status = 'active'`, orgId)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()
	for selDB.Next() {
		var deployment appDeployments.AppDeployments
		err = selDB.Scan(&deployment.AppId, &deployment.Region_code, &deployment.App_Url, &deployment.ELBRecordName, &deployment.ELBRecordId)
		// if err != nil {
		// 	return nil, err
		// }
		deployments = append(deployments, deployment)
	}

	return &deployments, nil
}

func UpdatePort(appName, imageName, internalPort string, replicas int) error {
	statement, err := database.Db.Prepare("UPDATE app set port = ?,imageName =?, replicas=? where name = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(internalPort, imageName, replicas, appName)
	if err != nil {
		return err
	}
	return nil
}

func UpdateAppVersion(appName string, version int) error {
	statement, err := database.Db.Prepare("UPDATE app set version = ?,status = ? where name = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(version, "Active", appName)
	if err != nil {
		return err
	}
	return nil
}

func GetLatestVersion(appName string) (int, error) {
	selDB, err := database.Db.Query("SELECT MAX(version) AS highest_version FROM app_release WHERE app_id = ?", appName)

	if err != nil {
		return 0, err
	}
	defer selDB.Close()
	var version int
	for selDB.Next() {
		err = selDB.Scan(&version)
		if err != nil {
			return 0, err
		}
	}

	return version, nil
}

func GetAppReplicas(appName string) (int, error) {
	var appReplicas int
	selDB, err := database.Db.Query("SELECT replicas FROM app where name = ? ", appName)
	if err != nil {
		return 0, err
	}
	defer selDB.Close()
	for selDB.Next() {
		err = selDB.Scan(&appReplicas)
		if err != nil {
			return 0, err
		}
	}

	return appReplicas, nil
}
