package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	//"github.com/go-chi/chi"
	"github.com/gorilla/mux"
	"github.com/nifetency/nife.io/helper"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	"github.com/nifetency/nife.io/utils"
	uuid "github.com/satori/go.uuid"
)

type CLISessionRequestBody struct {
	Name   string `json:"name"`
	SignUp bool   `json:"signup"`
}

func CLIUserSession(w http.ResponseWriter, r *http.Request) {

	var dataBody CLISessionRequestBody
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	id := uuid.NewV4().String()
	_, err = insertCLISession(id, dataBody.Name, dataBody.SignUp)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	authURL := fmt.Sprintf("%s?id=%s", utils.GetEnv("WEB_AUTH_URL", ""), id)

	helper.RespondwithJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       id,
		"auth_url": authURL,
	})
}

func GETCLIUserSession(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)
	accessToken, email, err := getCLISession(id["id"])
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	helper.RespondwithJSON(w, http.StatusOK, map[string]string{"access_token": accessToken, "email": email})
}

func insertCLISession(id, name string, signup bool) (bool, error) {
	statement, err := database.Db.Prepare("INSERT INTO cli_session(id,name,signup,access_token) VALUES(?,?,?,?)")
	if err != nil {
		return false, err
	}
	_, err = statement.Exec(id, name, signup, "")
	if err != nil {
		return false, err
	}
	return true, nil
}

func UpdateCLISession(access_token string, userId int, id string) (bool, error) {
	statement, err := database.Db.Prepare("UPDATE cli_session SET access_token = ?, user_id =? where id = ?")
	if err != nil {
		return false, err
	}
	_, err = statement.Exec(access_token, userId, id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func getCLISession(id string) (string, string, error) {
	statement, err := database.Db.Prepare("select cs.access_token , u.email from user u join cli_session cs  on u.id = cs.user_id where cs.id = ?")
	if err != nil {
		return "", "", err
	}
	row := statement.QueryRow(id)

	var accessToken string
	var email string
	err = row.Scan(&accessToken, &email)
	if err != nil {
		fmt.Println(err)
		if err != sql.ErrNoRows {
			return "", "", err
		}
		return "", "", err
	}

	return accessToken, email, nil
}
