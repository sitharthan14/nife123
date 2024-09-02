package duplo

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func UpdateRegionStatus(appName string, regions []*model.Region) error {
	statement, err := database.Db.Prepare("Update app set regions = ? where name = ?")
	if err != nil {
		return err
	}
	regionsJSON, _ := json.Marshal(regions)
	_, err = statement.Exec(regionsJSON, appName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func UpdateDeploymentsRecord(status, appId, deployment_id string, updatedAt time.Time) error {
	statement, err := database.Db.Prepare("UPDATE app_deployments set status = ?, updatedAt = ? where appId = ? and deployment_id = ?")
	if err != nil {
		return err
	}

	_, err = statement.Exec(status, updatedAt, appId, deployment_id)
	if err != nil {
		return err
	}
	return nil
}


func UpdateVersions(appName string, vers int) error {
	statement, err := database.Db.Prepare("Update app set version = ? where name = ?")
	if err != nil {
		return err
	}
	_, err = statement.Exec(vers, appName)
	if err != nil {
		return err
	}
	return nil
}


func UpdateImageandPort(appName, imageName string, internalPort int) error {

	queryString := `UPDATE app  
	INNER JOIN  
	app_release  
	ON app_release.app_id = app.name  
	SET app.imageName = ?, app.port = ? ,app_release.image_name = ?,app_release.port = ? where app_release.app_id = ? and app_release.status='active';`

	statement, err := database.Db.Prepare(queryString)
	if err != nil {
		return err
	}

	_, err = statement.Exec(imageName, internalPort ,imageName, internalPort ,appName)
	if err != nil {
		return err
	}
	return nil
}