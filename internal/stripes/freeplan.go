package stripes

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/alecthomas/log4go"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"

	"github.com/nifetency/nife.io/helper"
)


type FreePlanDetails struct {
	UserId string  `json:"userId"`
	IsFreePlan bool `json:"isfreePlan"`

}


func EnableFreePlan(w http.ResponseWriter, r *http.Request){

	var dataBody FreePlanDetails
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
	}
	
	email ,err := GetuseremailById(dataBody.UserId)
		if email == ""{
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Cannot find the user"})
		return 
	}
	log4go.Info("Module: EnableFreePlan, MethodName: GetuseremailById, Message: Fetching user email to check the user is register or not, user: %s", dataBody.UserId)

	// if stripeId == ""{
	// 	helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "The user has already selected a plan.	"})
	// 	return 
	// }

    err = UpdateFreePlan(dataBody.UserId, dataBody.IsFreePlan)

	if err != nil {
		log.Println(err)
		log4go.Info("Module: EnableFreePlan, MethodName: UpdateFreePlan, Message: %s user:%s", err.Error(), dataBody.UserId)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: EnableFreePlan, MethodName: UpdateFreePlan, Message:successfully reached, user: %s", dataBody.UserId)

	helper.RespondwithJSON(w, http.StatusOK, map[string]string{"message": "Updated Succesfully"})

}





func UpdateFreePlan(userId string,isFreeplan bool) (error) {
	statement, err := database.Db.Prepare("UPDATE user SET is_free_plan = ? WHERE id = ?")
	if err != nil {
		return err
	}
	_, err = statement.Exec(isFreeplan, userId)
	if err != nil {
		return err
	}	
	return nil
}


func GetuseremailById(userId string) (string, error) {
	statement, err := database.Db.Prepare("select email from user WHERE id = ?")
	if err != nil {
		log.Println(err)
	}
	row := statement.QueryRow(userId)

	var email string

	err = row.Scan(&email)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Print(err)
		}
		return  "", err
	}

	return email, nil
}