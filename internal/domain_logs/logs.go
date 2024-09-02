package domainlogs

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nifetency/nife.io/helper"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

func DomainLog(w http.ResponseWriter, r *http.Request) {
	var dataBody DomainLogModel
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	statement, err := database.Db.Prepare("Insert into domain_logs(url,organization_id,user_id,app_id,ip_address,latitude,longitude,createdAt)values(?,?,?,?,?,?,?,?)")
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	_, err = statement.Exec(dataBody.Url, dataBody.OrganizationId, dataBody.UserId,dataBody.AppId,dataBody.IpAddress, dataBody.Latitude, dataBody.Latitude, time.Now().UTC())
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"statusCode": http.StatusOK,
		"message":    "inserted successfully",
	})

}
