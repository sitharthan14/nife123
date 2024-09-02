package cloudwatchlogs

type QueryResult struct {
	Timestamp string `json:"timeStamp"`
	Message   string `json:"message"`
	ResolverIp string `json:"resolverIp"`
	QueryName string `json:"queryName"`
	ResponseCode string `json:"responseCode"`
	QueryType string `json:"queryType"`
}