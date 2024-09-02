package inviteuser

import (

	"log"
	"os"

	"github.com/sendgrid/sendgrid-go"
)

func SentInvite(fromEmail, toEmail, temporaryPassword, orgCount,senderUserName string) error {

	apiKey := os.Getenv("SENDGRID_API_KEY")
	url_ui := os.Getenv("INVITEUSER_UI_URL")

	request := sendgrid.GetRequest(apiKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"

	userName := os.Getenv("SENDGRID_USER_NAME")
	templateId := os.Getenv("INVITE_USER_TEMPLATE_ID")

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
				"email_address": "` + toEmail + `"
				"temporary_password": "` + temporaryPassword + `"
				"invite_sender_name": "` + senderUserName + `"
				"invited_organization_name":"` + orgCount + " organizations" + `"
				"ui_link": "`+ url_ui +`"
			}
			
		  }
		],
		"template_id":"` + templateId + `"
	  }`)

	_, err := sendgrid.API(request)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil

}
