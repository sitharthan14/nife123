package uilogs

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nifetency/nife.io/helper"
)

type ClientSideLogs struct {
	Message string `json:"message"`
	Level   string `json:"level"`

	UserId string `json:"userId"`
}

func UILogs(w http.ResponseWriter, r *http.Request) {
	var dataBody ClientSideLogs

	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	if !LogFileExists("internal/ui_logs/UI-Logs") {

		_, err := os.Create("internal/ui_logs/UI-Logs")
		if err != nil {
			log.Println(err)
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return

		}
	}

	f, err := os.OpenFile("internal/ui_logs/UI-Logs", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	dt := time.Now()

	data := fmt.Sprintln(dt.String() + " UserId: " + dataBody.UserId + " Level: " + dataBody.Level + " Message: " + dataBody.Message)

	_, err = f.Write([]byte(data))

	if err != nil {
		log.Fatal(err)
	}

	helper.RespondwithJSON(w, http.StatusOK, map[string]string{"message": "Succesfully Inserted Logs"})
}

func LogFileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
