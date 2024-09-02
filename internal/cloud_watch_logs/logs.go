package cloudwatchlogs

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	log "github.com/sirupsen/logrus"
)

func Cloudwatchlogs() (string, error) {

	var START_TIME int64

	getSyncTime, err := GetLastSyncLogs()

	if err != nil {
		return "", err
	}
	if getSyncTime.LastSyncTime == "" {
		logStartTime := os.Getenv("INITIAL_LOGGING_FROM_NOW_MINS")
		customTime, _ := strconv.Atoi(logStartTime)
		beforeMinutes := time.Now().Add(-time.Minute * time.Duration(customTime)).Unix()
		START_TIME = beforeMinutes

	}

	logStartTime := os.Getenv("INITIAL_LOGGING_FROM_NOW_MINS")
	initalTimeToLog, _ := strconv.Atoi(logStartTime)

	if initalTimeToLog <= 0 {
		ts, err := time.Parse(time.RFC3339, getSyncTime.LastSyncTime)
		if err != nil {
			return "", fmt.Errorf("something went wrong while parsing")
		}
		startTime := ts.UTC().Unix()
		START_TIME = startTime

	}

	if getSyncTime.LastSyncTime != "" {
		ts, err := time.Parse(time.RFC3339, getSyncTime.LastSyncTime)
		if err != nil {
			return "", fmt.Errorf("something went wrong while parsing")
		}
		startTime := ts.UTC().Unix()
		START_TIME = startTime
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	if err != nil {
		log.WithField("err", err).Fatal("unable to auth against aws")
	}
	limit := os.Getenv("CLOUD_WATCH_QUERY_LOG_LIMIT")
	queryLogLimit, _ := strconv.Atoi(limit)
	svc := cloudwatchlogs.New(sess)

	sqi := &cloudwatchlogs.StartQueryInput{
		//StartTime: aws.Int64(time.Now().Add(-(time.Hour * 24 * 22)).Unix()),
		StartTime:    aws.Int64(START_TIME),
		EndTime:      aws.Int64(time.Now().UTC().Unix()),
		Limit:        aws.Int64(int64(queryLogLimit)),
		LogGroupName: aws.String("/aws/route53/apps.nifetency.com"),
		QueryString:  aws.String("fields @timestamp, @message,resolverIp, responseCode, queryName, queryType | sort @timestamp desc "),
	}
	sqo, err := svc.StartQuery(sqi)
	if err != nil {
		fmt.Println("unable to start insights query", err)

	}

	gqri := &cloudwatchlogs.GetQueryResultsInput{QueryId: sqo.QueryId}
	req, resp := svc.GetQueryResultsRequest(gqri)
	time.Sleep(time.Second * 2)
	for {
		if err := req.Send(); err == nil {
			break
		} else {

			log.Warn("query not completed, retying")
			break

		}
	}
	if err != nil {
		fmt.Println("error fetching insights query result", err)

		// log.WithField("err", err).Fatal("error fetching insights query result")

	}

	for _, item := range resp.Results {

		var result QueryResult
		for _, itemResults := range item {
			switch field := *itemResults.Field; field {
			case "@timestamp":
				result.Timestamp = *itemResults.Value
			case "@message":
				result.Message = *itemResults.Value
			case "resolverIp":
				result.ResolverIp = *itemResults.Value
			case "responseCode":
				result.ResponseCode = *itemResults.Value
			case "queryName":
				result.QueryName = *itemResults.Value
			case "queryType":
				result.QueryType = *itemResults.Value
			default:
			}
		}

		_, _ = InsertLog(result)
	}
	return "", nil
}
