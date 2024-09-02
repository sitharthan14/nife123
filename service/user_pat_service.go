package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func InsertUserPAT(userId string, patDetails model.UserPat) error {
	statement, err := database.Db.Prepare("INSERT INTO user_pat(id, type , pat_token, user_id, created_at, updated_at, is_active) VALUES(?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, patDetails.Type, patDetails.PatToken, userId, time.Now(), time.Now(), true)
	if err != nil {
		return err
	}

	return nil
}

func UpdateUserPAT(userId string, patDetails model.UserPat) error {
	statement, err := database.Db.Prepare("update user_pat set type = ?, pat_token = ?, updated_at= ? where user_id = ? and is_active = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(patDetails.Type, patDetails.PatToken, time.Now(), userId, true)
	if err != nil {
		return err
	}

	return nil
}

func DeleteUserPAT(userId, patId string) error {
	statement, err := database.Db.Prepare("update user_pat set is_active = ? where user_id = ? and id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(false, userId, patId)
	if err != nil {
		return err
	}

	return nil
}

func GetUserPAT(userId string) ([]*model.GetUserPat, error) {

	query := `SELECT id, type, pat_token, user_id, created_at, updated_at FROM user_pat where user_id = ? and is_active = ?`

	selDB, err := database.Db.Query(query, userId, true)
	if err != nil {
		return []*model.GetUserPat{}, err
	}
	defer selDB.Close()
	result := []*model.GetUserPat{}
	for selDB.Next() {
		var userPAT model.GetUserPat

		err = selDB.Scan(&userPAT.ID, &userPAT.Type, &userPAT.PatToken, &userPAT.UserID, &userPAT.CreatedAt, &userPAT.UpdatedAt)
		if err != nil {
			return []*model.GetUserPat{}, err
		}

		result = append(result, &userPAT)
	}

	return result, nil
}
