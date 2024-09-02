package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/internal/auth"
	cloudwatchlogs "github.com/nifetency/nife.io/internal/cloud_watch_logs"
)

func (r *mutationResolver) ClientSideLog(ctx context.Context, input model.ClientSideLogs) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}

	if !cloudwatchlogs.LogFileExists("internal/ui_logs/UI-Logs") {

		_, err := os.Create("internal/ui_logs/UI-Logs")
		if err != nil {
			return "", err
		}
	}

	f, err := os.OpenFile("internal/ui_logs/UI-Logs", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}

	defer f.Close()
	
	dt := time.Now()

	data := fmt.Sprintln(dt.String() + " UserId: " + user.ID + " Level: " + input.Level + " Message: " + input.Message)

	_, err = f.Write([]byte(data))

	if err != nil {
		return "", err
	}

	return "Successfully Inserted Logs", nil
}

func (r *queryResolver) GetQueryLog(ctx context.Context, input model.GetQueryLog) ([]*model.QueryLogOutput, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	userInput := model.GetQueryLog{
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
		HostName:  input.HostName,
	}
	getLog := cloudwatchlogs.GetQueryLogs(userInput)
	return getLog, nil
}
