package regions

import (
	"fmt"
	"log"
	"os"

	"github.com/sendgrid/sendgrid-go"
)

func SentRegRequest(senderUserName, userEmail,regions string) error {

	apiKey := os.Getenv("SENDGRID_API_KEY")	

	request := sendgrid.GetRequest(apiKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"

	userName := os.Getenv("SENDGRID_USER_NAME")
	templateId := os.Getenv("REGION_REQUEST_TEMPLATE_ID")

	request.Body = []byte(` {
		"from": {
			"email": "` + userName + `"
		},
		"personalizations": [
		  {
			"to": [
				{
					"email": "` + userName + `"
				}
			],
			"dynamic_template_data":{
				"subject": "New Region Request"
				"username": "` + senderUserName + `"
				"email": "(` + userEmail + `)"
				"regions":"` + regions + `"
			}

		  }
		],
		"template_id":"` + templateId + `"
	  }`)

	respnse, err := sendgrid.API(request)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Println(respnse)

	return nil

}
