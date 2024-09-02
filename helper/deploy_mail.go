package helper

import (
	"fmt"
	"log"
	"os"

	"github.com/sendgrid/sendgrid-go"
)

func DeployMail(user, toEmail, username, appName, region, subject, logoURL, message string) error {

	apiKey := os.Getenv("SENDGRID_API_KEY")

	request := sendgrid.GetRequest(apiKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"

	userName := os.Getenv("SENDGRID_USER_NAME")
	templateId := os.Getenv("DEPLOY_MAIL_NOTIFICATION")
	Message := username + ` has ` + message + ` the app ` + appName + ` to the region ` + region

	request.Body = []byte(` {
		"from": {
			"email": "` + userName + `"
		},    
		"personalizations": [
		  {
			"to": [
				{
					"email": "` + toEmail + `"
				}
			],
			"dynamic_template_data":{ 
				"subject": "` + subject + ` Mail Notification"
				"s3url": "` + logoURL + `"
				"user": "Admin"
				"message":"` + Message + `" , 
			}
			
		  }
		],
		"template_id":"` + templateId + `"
	  }`)

	response, err := sendgrid.API(request)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Println(response)
	return nil

}

func RequestingBYOH(toEmail, useremail, byohName, password, ipAddress, region, fullName string) error {

	apiKey := os.Getenv("SENDGRID_API_KEY")

	request := sendgrid.GetRequest(apiKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"

	userName := os.Getenv("SENDGRID_USER_NAME")
	templateId := os.Getenv("DEPLOY_MAIL_NOTIFICATION")
	message := "" + fullName + " (" + useremail + ") is requesting for a BYOH region (" + region + ") for this nife account " + useremail + ""

	message1 := message
	request.Body = []byte(` {
		"from": {
			"email": "` + userName + `"
		},    
		"personalizations": [
		  {
			"to": [
				{
					"email": "` + toEmail + `"
				}
			],
			"dynamic_template_data":{ 
				"subject": "Requesting BYOH"
				"s3url": ""
				"user": "Admin"
				"message":"` + message1 + `"
			}
			
		  }
		],
		"template_id":"` + templateId + `"
	  }`)

	response, err := sendgrid.API(request)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Println(response)
	return nil

}
