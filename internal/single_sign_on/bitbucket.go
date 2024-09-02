package singleSignOn

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func BitbucketCheckUser(accessToken string) (string, error) {
	var email string

	getUserUrl := os.Getenv("BITBUCKET_GETUSER_URL")

	payLoad := GetUserBitbucket(accessToken, getUserUrl+"/emails")

	var result map[string]interface{}
	json.Unmarshal(payLoad, &result)

	values, ok := result["values"].([]interface{})

	if ok {

		data := values[0].(map[string]interface{})
		emailStr ,_ := data["email"].(string)
		email = emailStr
	}
	
	return email, nil

}

func GetUserBitbucket(accessToken, URL string) []byte {

	fmt.Println(URL)

	req, reqerr := http.NewRequest("GET", URL, nil)
	if reqerr != nil {
		log.Println("API Request creation failed")
	}

	authorizationHeaderValue := fmt.Sprintf("Bearer %s", accessToken)
	req.Header.Set("Authorization", authorizationHeaderValue)

	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		log.Println("Request failed")
	}

	respbody, _ := ioutil.ReadAll(resp.Body)
	return respbody
}

// func GetPrimaryEmail(def map[string]interface{}) string {
// 	primary := def["primary"].(bool)
// 	if primary {
// 		email := def["email"].(string)
// 		return email
// 	}
// 	return ""
// }

func SerializingDefinition(def map[string]interface{}) (string, error) {
	userName, err := def["name"].(string)
	if !err {
		return "", fmt.Errorf("Please Add the username in the github")
	}

	return userName, nil

}
