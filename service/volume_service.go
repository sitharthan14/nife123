package service

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func CreateVolumes(vol *model.DuploVolumeInput) error {
	statement, err := database.Db.Prepare("INSERT INTO volumes (id,app_id,access_mode,name,path,container_path,host_path,size,created_at,volume_type_id) VALUES (?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id,&vol.AppID ,&vol.AccessMode, &vol.Name, &vol.Path, &vol.ContainerPath, &vol.HostPath, &vol.Size, time.Now(), &vol.VolumeTypeID)
	if err != nil {
		return err
	}
	return nil
}

func UpdateVolume(appName string, size string) error {
	statement, err := database.Db.Prepare("Update volumes set size = ? where app_id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(size, appName)
	if err != nil {
		return err
	}
	return nil
}

// func InsertVolIdInApp(id , appName string) error {

// 	statement, err := database.Db.Prepare("update app set volume_id = ? WHERE name = ?")
// 	if err != nil {
// 		return  err
// 	}
// 	_, err = statement.Exec(id, appName)
// 	if err != nil {
// 		return  err
// 	}
// 	return  nil
// }

func GetVolumeDetailsByAppName(appId string) ([]*model.DuploVolumeInput,error) {

	var volume []*model.DuploVolumeInput

	query := `SELECT volumes.app_id , volumes.access_mode, volumes.name, volumes.path, volumes.size FROM volumes where volumes.app_id = ?`

	selDB, err := database.Db.Query(query, appId)
	if err != nil {
		return []*model.DuploVolumeInput{}, err
	}
	defer selDB.Close()

	selDB.Next()
	var duplo model.DuploVolumeInput
	err = selDB.Scan(&duplo.AppID, &duplo.AccessMode,&duplo.Name, &duplo.Path, &duplo.Size)
	if err != nil && err != sql.ErrNoRows  {
		return nil, nil
	}
	volume = append(volume, &duplo)

	return volume, nil
}

func GetVolumeByAppName(appId string) (*model.DuploVolumeInput,error) {


	query := `SELECT access_mode,app_id ,name, path, size FROM volumes where volumes.app_id = ?;`

	selDB, err := database.Db.Query(query, appId)
	if err != nil {
		return &model.DuploVolumeInput{}, err
	}
	defer selDB.Close()
	var volume model.DuploVolumeInput

	selDB.Next()
	err = selDB.Scan(&volume.AccessMode, &volume.AppID, &volume.Name, &volume.Path, &volume.Size)
	if err != nil && err != sql.ErrNoRows  {
		return nil, nil
	}

	return &volume, nil
}

func GetVolumeType()([]*model.VolumeType, error){
	query := "SELECT id, name, is_read, is_host_volume, description FROM volume_type" 
	
	selDB, err := database.Db.Query(query)
	if err != nil {
		return []*model.VolumeType{}, err
	}	
	defer selDB.Close()
	result := []*model.VolumeType{}
	for selDB.Next() {
	var vol model.VolumeType
   	err = selDB.Scan(&vol.ID,&vol.Name,&vol.IsRead,&vol.IsHostVolume,&vol.Description)
	if err != nil {
		return  []*model.VolumeType{}, err
	}
	result = append(result, &vol)
}

	return result, nil
}


func GetAppIdByName(appName string) (string, error) {
	selDB, err := database.Db.Query("SELECT id FROM app where name = ?",appName )
	if err != nil {
		return "", err
	}
	defer selDB.Close()
	var id string
	for selDB.Next() {
		err = selDB.Scan(&id)
		if err != nil {
			return "", err
		}
	}
	return id, nil
}