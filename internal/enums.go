package internal

type AppStatus string

const (
	AppStatusNew        AppStatus = "New"
	AppStatusActive     AppStatus = "Active"
	AppStatusSuspended  AppStatus = "Suspended"
	AppStatusTerminated AppStatus = "Terminated"
)
