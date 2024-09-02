package tokenverification

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal/users"
	"github.com/nifetency/nife.io/pkg/jwt"
)

type TokenDetails struct {
	AccessToken string `json:"accessToken"`
}

func VerifyAccessToken(w http.ResponseWriter, r *http.Request) {
	var dataBody TokenDetails

	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	//-----------validating access token format

	err = jwt.ValidateJWTTokenFormat(dataBody.AccessToken)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	email, stripeProductId, firstName, lastName, comapanyName, roleId, customerStripeId, err := jwt.ParseToken(dataBody.AccessToken)

	if err != nil {
		log.Println(err)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	checkEmail, err := users.GetUserIdByEmail(email)

	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	if checkEmail == 0 {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Token not valid"})
		return
	}

	generateRefreshToken, err := jwt.GenerateRefreshToken(email, stripeProductId, firstName, lastName, comapanyName, roleId, customerStripeId)

	if err != nil {
		log.Println(err)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"refresh_token": generateRefreshToken,
	})
}
