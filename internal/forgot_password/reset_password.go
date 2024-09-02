package forgotpassword

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alecthomas/log4go"
	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal/users"
	"github.com/nifetency/nife.io/pkg/jwt"
	"github.com/nifetency/nife.io/service"
)

type UserDetails struct {
	AccessToken string `json:"accessToken"`
	PassWord    string `json:"password"`
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
	var userdetails UserDetails
	err := json.NewDecoder(r.Body).Decode(&userdetails)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	if userdetails.PassWord == ""{
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Password field should not be empty"})
		return
	}

	email, _,_,_,_,_,_,err := jwt.ParseToken(userdetails.AccessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	checkUser, _ := users.GetUserIdByEmail(email)
	if checkUser == 0 {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Email does not Exist!"})
		return
	}
	log4go.Info("Module: ResetPassword, MethodName: GetUserIdByEmail, Message:successfully reached, user: %s", email)

	encryptPwd, _ := users.HashPassword(userdetails.PassWord)

	id := strconv.Itoa(checkUser)

	_, err = service.UpdateUserPassword(id, encryptPwd, id)
	if err != nil {
		log4go.Error("Module: ResetPassword, MethodName: UpdateUserPassword, Message: %s user:%s", err.Error(), email)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: ResetPassword, MethodName: UpdateUserPassword, Message:successfully reached, user: %s", email)

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"statusCode": http.StatusOK,
		"message":    "Reset password successfully!",
	})

}
