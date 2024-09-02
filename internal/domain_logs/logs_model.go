package domainlogs

import "time"

type DomainLogModel struct {
	Url            string    `json:"url"`
	OrganizationId string    `json:"organizationId"`
	UserId         string    `json:"userId"`
	IpAddress      string    `json:"ipAddress"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	CreatedAt      time.Time `json:"createdAt"`
	AppId          string    `json:"appId"`
}
