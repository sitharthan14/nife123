package github

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func GithubCheckUser(accessToken string) (string,error) {

	getUserUrl := os.Getenv("GITHUB_GETUSER_URL")

	payLoad := GetUserGithub(accessToken, getUserUrl+"/emails")

	var result []map[string]interface{}
	json.Unmarshal(payLoad, &result)

	email := ""
	for _, j := range result {
		email = GetPrimaryEmail(j)
		if email != "" {
			break
		}
	}

	response := GetUserGithub(accessToken, getUserUrl)

	fmt.Println(string(response))
	var serializePayload map[string]interface{}
	json.Unmarshal(response, &serializePayload)
	return email, nil


}

func GetUserGithub(accessToken, URL string) []byte {

	fmt.Println(URL)

	req, reqerr := http.NewRequest("GET", URL, nil)
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

func getAccessToken(responseString string) (string, error) {
	tokenString := strings.Split(responseString, "&")
	accessToken := strings.Split(tokenString[0], "=")
	if accessToken[0] == "error" {
		return "", fmt.Errorf("Invalid Code")
	}
	return accessToken[1], nil
}

func GetPrimaryEmail(def map[string]interface{}) string {
	primary := def["primary"].(bool)
	if primary {
		email := def["email"].(string)
		return email
	}
	return ""
}

func SerializingDefinition(def map[string]interface{}) (string, error) {
	userName, err := def["name"].(string)
	if !err {
		return "", fmt.Errorf("Please Add the username in the github")
	}

	return userName, nil

}

func GithubLoginHandler(w http.ResponseWriter, r *http.Request) {

	githubClientID := os.Getenv("GITHUB_CLIENT_ID")

	redirect_uri := os.Getenv("GITHUB_REDIRECT_URI")

	// Create the dynamic redirect URL for login
	redirectURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s",
		githubClientID, redirect_uri,
	)
	fmt.Println(redirectURL)
	http.Redirect(w, r, redirectURL, 301)
}
