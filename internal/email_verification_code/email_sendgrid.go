package emailverificationcode

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"github.com/sendgrid/sendgrid-go"

)

type UserEmail struct {
	Email string `json:"email"`
}

func OtpforRegister(email string, code int)(error) {
	
	UserName := os.Getenv("SENDGRID_USER_NAME")
	sendgridAPIkey := os.Getenv("SENDGRID_API_KEY")
	sendgridTemplateId := os.Getenv("OTP_VERTIFICATION_TEMPLATE_ID")

	
  Code := strconv.Itoa(code)

  // Authentication.
  request := sendgrid.GetRequest(sendgridAPIkey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
  
	request.Body = []byte(` {
		"from": {
			"email": "`+UserName+`"
		},    
		"personalizations": [
		  {
			"to": [
				{
					"email": "`+email+`"
				}
			],
			"dynamic_template_data":{ 
				"code": "`+Code+`" 
			}
			
		  }
		],
		"template_id":"`+sendgridTemplateId+`" 
	  }`)

	_, err := sendgrid.API(request)
	if err != nil {
		log.Println(err)
	} 
  fmt.Println("Email Sent!")
  return err
	
}
