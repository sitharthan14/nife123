package apprelease

import (
	"time"
)

type AppRelease struct {
	Id                 string    `json:"id"`
	AppId              string    `json:"app_id"`
	Version            string    `json:"version"`
	Description        string    `json:"description"`
	Reason             string    `json:"reason"`
	DeploymentStrategy string    `json:"deployment_strategy"`
	Status             string    `json:"status"`
	UserId             string    `json:"user_id"`
	ImageName          string    `json:"image_name"`
	Port               int       `json:"port"`
	CreatedAt          time.Time `json:"createdAt"`
	ArchiveUrl         string    `json:"archive_url"`
	BuilderType        string    `json:"builder_type"`
	RoutingPolicy      string    `json:"routing_policy"`
}
