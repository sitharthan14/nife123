package appDeployments

import "time"

type AppDeployments struct {
	Id            string    `json:"id"`
	AppId         string    `json:"appId"`
	Region_code   string    `json:"region_code"`
	Status        string    `json:"status"`
	Deployment_id string    `json:"deployment_id"`
	Release_id    string    `json:"release_id"`
	Port          string    `json:"port"`
	App_Url       string    `json:"app_url"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	// Added New Field
	ELBRecordName string `json:"elb_record_name"`
	ELBRecordId   string `json:"elb_record_id"`
	ContainerID   string  `json:"container_id"`
}
