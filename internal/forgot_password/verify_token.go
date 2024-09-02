package forgotpassword

import (
	"net/http"

	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/pkg/jwt"
)

func VerifyToken(w http.ResponseWriter, r *http.Request) {
	query_token := r.URL.Query().Get("token")

	email,_, _,_,_,_, _,err := jwt.ParseToken(query_token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if email == "" {
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"statusCode": http.StatusOK,
		"message":    "Verified succesfully",
	})

}
