package singleSignOn

import (
	"encoding/json"
	
)

type SingleSignOnDetails struct {
	Email interface{} `json:"email"`
	Id    interface{}      `json:"id"`
	UserName interface{}   `json:"userName"`
}

func FindEmail(content []byte) SingleSignOnDetails {

	var details map[string]interface{}
	json.Unmarshal([]byte(string(content)), &details)

	var user SingleSignOnDetails

	for key, value := range details {
		switch field := key; field {
		case "email":
			user.Email = value
		case "id":
			user.Id = value
		case "login":
			user.UserName = value

		}
	}
    res := &SingleSignOnDetails{ 
		Email:user.Email,
		Id: user.Id,
		UserName: user.UserName,
	}	

	return *res

}
