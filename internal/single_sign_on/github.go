package singleSignOn

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func GetGithubData(accessToken string) []byte {
	githubToken := os.Getenv("GITHUB_ACCESSTOKEN_URL")
	req, reqerr := http.NewRequest("GET", githubToken, nil)
	if reqerr != nil {
		log.Println("API Request creation failed")
	}

	authorizationHeaderValue := fmt.Sprintf("token %s", accessToken)
	req.Header.Set("Authorization", authorizationHeaderValue)

	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		log.Println("Request failed")
	}

	respbody, _ := ioutil.ReadAll(resp.Body)

	return respbody
}