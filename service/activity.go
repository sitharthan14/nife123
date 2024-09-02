package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	"github.com/slack-go/slack"
)

type Activity struct {
	Id         string `json:"id"`
	Type       string `json:"type"`
	UserId     string `json:"user_id"`
	Activities string `json:"activities"`
	Message    string `json:"message"`
	RefId      string `json:"ref_id"`
}

func InsertActivity(activity Activity) (string, error) {

	statement, err := database.Db.Prepare("INSERT INTO activity (id, type, user_id, activities, message, ref_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, activity.Type, activity.UserId, activity.Activities, activity.Message, activity.RefId, time.Now())
	if err != nil {
		return "", err
	}
	return "", nil
}

func ErrorActivity(userId, appName, errMsg string) error {

	ErrorOperation := Activity{
		Type:       "APP-ERROR",
		UserId:     userId,
		Activities: "DEPLOYED",
		Message:    errMsg,
		RefId:      appName,
	}

	_, err := InsertActivity(ErrorOperation)
	if err != nil {
		return err
	}
	err = SendSlackNotification(userId, ErrorOperation.Message)

	return err
}

func SendSlackNotification(userId, message string) error {

	webhookURL, err := GetUserSlackWebhookURL(userId)
	if err != nil {
		return err
	}
	teamBoardURL := os.Getenv("TEAMBOARD_URL")

	if webhookURL != "" {
		currentTime := time.Now()
		attachment := slack.Attachment{
			Color:         "#160044",
			AuthorName:    "Oikos",
			AuthorSubname: "by Nife",
			AuthorLink:    "https://nife.io/",
			AuthorIcon:    "https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Ftse1.mm.bing.net%2Fth%3Fid%3DOIP.Get2p3wZnkxWyo2N1fhsSwHaHa%26pid%3DApi&f=1&ipt=11ea53a6a92f01e699e33c93fd240a2eb1b3a4cbf6627d6387139c8fac50134c&ipo=images",
			Actions: []slack.AttachmentAction{
				slack.AttachmentAction{
					Type:  "button",
					Text:  "Nife Dashboard",
					URL:   teamBoardURL,
					Style: "primary",
				},
			},
			Text:       fmt.Sprintf("On %s %d: *%s*", currentTime.Month(), currentTime.Day(), message),
			FooterIcon: "https://github.com/nifetency/finOps",
		}
		payload := slack.WebhookMessage{
			Attachments: []slack.Attachment{attachment},
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to send message, received status code: %d", resp.StatusCode)
		}
	}
	return nil
}
