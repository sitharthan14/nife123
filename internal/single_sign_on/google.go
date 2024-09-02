package singleSignOn

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func GetGoogleData(accessToken string)[]byte {
	googleToken := os.Getenv("GOOGLE_ACCESSTOKEN_URL")

	resp, err := http.Get(googleToken + accessToken)
	if err != nil {
		log.Println(err)
	}

	defer resp.Body.Close()

	content, _ := ioutil.ReadAll(resp.Body)
	return content
}