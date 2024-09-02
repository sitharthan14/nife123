package forgotpassword

import (
	"encoding/json"

	"log"
	"net/http"
	"os"

	"github.com/alecthomas/log4go"
	"github.com/sendgrid/sendgrid-go"

	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal/users"
)

type UserEmail struct {
	Email string `json:"email"`
}

func ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var usermail UserEmail
	err := json.NewDecoder(r.Body).Decode(&usermail)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	checkUser, _ := users.GetUserIdByEmail(usermail.Email)
	if checkUser == 0 {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"statusCode": http.StatusBadRequest,
			"message":    "Email doesn't exist",
		})
		return
	}

	accessToken, _, err := users.GenerateAccessAndRefreshToken(usermail.Email, "", true, "", "", "", 0,"")
	if err != nil {
		log4go.Error("Module: ForgotPassword, MethodName: GenerateAccessAndRefreshToken, Message: %s user:%s", err.Error(), usermail.Email)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: ForgotPassword, MethodName: GenerateAccessAndRefreshToken, Message:successfully reached, user: %s", usermail.Email)

	apiKey := os.Getenv("SENDGRID_API_KEY")

	request := sendgrid.GetRequest(apiKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"

	userName := os.Getenv("SENDGRID_USER_NAME")
	templateId := os.Getenv("FORGET_PASSWORD_TEMPLATE_ID")
	webResetURL := os.Getenv("WEB_RESET_URL")

	request.Body = []byte(` {
		"from": {
			"email": "` + userName + `"
		},    
		"personalizations": [
		  {
			"to": [
				{
					"email": "` + usermail.Email + `"
				}
			],
			"dynamic_template_data":{ 
				"url": "` + webResetURL + "?token=" + accessToken + `"
			}
			
		  }
		],
		"template_id":"` + templateId + `"
	  }`)
	_, err = sendgrid.API(request)
	if err != nil {
		log.Println(err)
		return
	}
	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"statusCode": http.StatusOK,
		"message":    "Email sent successfully",
		"access_token": accessToken,
	})
}
