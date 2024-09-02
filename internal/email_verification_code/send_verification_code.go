package emailverificationcode

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"net/http"

	"time"

	"github.com/alecthomas/log4go"
	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	"github.com/nifetency/nife.io/internal/users"
)

func SendVerificationCode(w http.ResponseWriter, r *http.Request) {
	var userId model.User

	err := json.NewDecoder(r.Body).Decode(&userId)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	if userId.ID == ""{
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "User Id cannot be empty"})
		return
	}

	email, err := users.GetEmailById(userId.ID)

	if email.Email == "" {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": "Unable to find the user with the provided User Id"})
		log4go.Error("Module: SendVerificationCode, MethodName: GetEmailById, Message: %s user:%s", err.Error(), userId.Email)
		return
	}
	log4go.Info("Module: SendVerificationCode, MethodName: GetEmailById, Message:successfully reached, user: %s", userId.Email)

	randomNumber := RandomNumber6Digit()
	err = OtpforRegister(email.Email, int(randomNumber))

	if err != nil {
		helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}

	err = InsertVerificationCode(int(randomNumber), userId.ID)
	if err != nil {
		log4go.Error("Module: SendVerificationCode, MethodName: InsertVerificationCode, Message: %s user:%s", err.Error(), userId.Email)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: SendVerificationCode, MethodName: InsertVerificationCode, Message:successfully reached, user: %s", userId.Email)

	randomNo := strconv.Itoa(int(randomNumber))

	helper.RespondwithJSON(w, http.StatusAccepted, map[string]string{
		"message": "Otp sent successfully",
		"code":    randomNo,
	})

}

type OtpDetails struct {
	Code   int    `json:"code"`
	UserId string `json:"userId"`
}

func VerificationCode(w http.ResponseWriter, r *http.Request) {

	var Details OtpDetails
	err := json.NewDecoder(r.Body).Decode(&Details)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	verfiy, err := GetOtpDetails(Details.Code, Details.UserId)

	if err != nil {
		log.Println(err)
		log4go.Error("Module: VerificationCode, MethodName: GetOtpDetails, Message: %s user:%s", err.Error(), Details.UserId)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	log4go.Info("Module: VerificationCode, MethodName: GetOtpDetails, Message:successfully reached, user: %s", Details.UserId)

	if verfiy == "Verified successfully" {

		email, err := users.GetEmailById(Details.UserId)

		if err != nil {
			log4go.Error("Module: VerificationCode, MethodName: GetEmailById, Message: %s user:%s", err.Error(), Details.UserId)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: VerificationCode, MethodName: GetEmailById, Message:successfully reached, user: %s", Details.UserId)

		accessToken, refreshToken, err := users.GenerateAccessAndRefreshToken(email.Email, "", false, email.FirstName, email.LastName, email.CompanyName, email.RoleId, "")
		if err != nil {
			log4go.Error("Module: VerificationCode, MethodName: GenerateAccessAndRefreshToken, Message: %s user:%s", err.Error(), Details.UserId)
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}
		log4go.Info("Module: VerificationCode, MethodName: GenerateAccessAndRefreshToken, Message:successfully reached, user: %s", Details.UserId)

		err = users.CheckEmailVerify(email.Email)

		if err != nil {
			helper.RespondwithJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
			return
		}

		helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
		})
	}

}

func GetOtpDetails(code int, userId string) (string, error) {

	query := `select code, expiry_time from verify_code where user_id = ?`

	selDB, err := database.Db.Query(query, userId)
	if err != nil {
		return "", err
	}
	defer selDB.Close()

	var otpCode int
	var expTime time.Time

	for selDB.Next() {
		err = selDB.Scan(&otpCode, &expTime)
		if err != nil {
			return "", err
		}
	}

	if otpCode == 0 {
		return "", fmt.Errorf("Account Not Available")

	}

	diff := expTime.Sub(time.Now())

	expireTime := diff.Minutes()

	if expireTime > 10 || expireTime < 0 {
		err = DeleteExpireCode(code)
		if err != nil {
			return "", fmt.Errorf("OTP Mismatch")
		}
		return "", fmt.Errorf("Your OTP is Expired")
	}

	if otpCode != code {
		return "", fmt.Errorf("OTP Mismatch")
	}

	return "Verified successfully", nil

}

func InsertVerificationCode(code int, userId string) error {
	statement, err := database.Db.Prepare("INSERT INTO verify_code(id,user_id,type,code,expiry_time,createdAt) VALUES(?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	id := uuid.NewString()

	dt := time.Now()
	expTime := (dt.Add(time.Minute * 5))

	_, err = statement.Exec(id, userId, "Register", code, expTime, time.Now())
	if err != nil {
		return err
	}
	return nil
}

func DeleteExpireCode(code int) error {
	statement, err := database.Db.Prepare("DELETE FROM verify_code WHERE (code = ?)")
	if err != nil {
		return err
	}

	_, err = statement.Exec(code)
	if err != nil {
		return err
	}
	return nil
}
