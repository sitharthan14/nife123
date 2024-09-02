package singleSignOn

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type Gitlab struct {
	Id string `josn:"id"`
	Email string `json:"email"`
}

func GitlabCheckUser(accessToken string) (string, error) {

	getUserUrl := os.Getenv("GITLAB_GETUSER_URL")

	payLoad := GetUserGitlab(accessToken, getUserUrl+"/emails")

	var result []Gitlab
	json.Unmarshal(payLoad, &result)

	for _,i := range result {
		return i.Email, nil
	} 

	return "", nil

}

func GetUserGitlab(accessToken, URL string) []byte {

	s := URL+"?access_token="+accessToken

	response, err := http.Get(s)

    if err != nil {
        fmt.Print(err.Error())
        return []byte(err.Error())
    }

    responseData, err := ioutil.ReadAll(response.Body)
    if err != nil {
        log.Fatal(err)
    }
    

	return responseData
}
