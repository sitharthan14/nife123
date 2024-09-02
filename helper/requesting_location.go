package helper

import (
	"fmt"
	"log"
	"os"

	"github.com/sendgrid/sendgrid-go"
)

func RequestingMarketPlaceApp(appName, userEmail, username string) error {

	UserName := os.Getenv("SENDGRID_USER_NAME")
	sendgridAPIkey := os.Getenv("SENDGRID_API_KEY")
	sendgridTemplateId := os.Getenv("REQUEST_PICONETS_APP")

	message := username + ` is requesting for ` + appName + `.`

	fmt.Println(message)

	// Authentication.
	request := sendgrid.GetRequest(sendgridAPIkey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"

	request.Body = []byte(` {
		"from": {
			"email": "` + UserName + `"
		},    
		"personalizations": [
		  {
			"to": [
				{
					"email": "` + UserName + `"
				}
			],
			"dynamic_template_data":{ 
				"subject": "Request for picoNETs - Mail Notification",
				"message": "` + message + `" ,
				"username":"` + username + `",
				"useremail":"` + userEmail + `",
				"requesting":"` + appName + `",

			}
			
		  }
		],
		"template_id":"` + sendgridTemplateId + `" 
	  }`)

	response, err := sendgrid.API(request)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Println(response)
	return nil

}
