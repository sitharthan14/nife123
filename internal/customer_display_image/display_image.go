package customerdisplayimage

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"

	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"cloud.google.com/go/storage"
	"github.com/nifetency/nife.io/internal/users"
	"google.golang.org/api/option"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/nifetency/nife.io/helper"
)

type UserDetails struct {
	UserId    int    `json:"userId"`
	ImageType string `json:"type"`
}

func UploadProfileImage(w http.ResponseWriter, r *http.Request) {
	storageSelector := os.Getenv("STORAGE_SELECTOR")
	var imageLink string

	imageType := r.FormValue("type")

	keys, ok := r.URL.Query()["userId"]

	if !ok || len(keys[0]) < 1 {
		log.Println("Url Param 'key' is missing")
		return
	}
	userId := keys[0]

	userDet, err := users.GetEmailById(userId)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "User does not exist"})
		return
	}
	if userDet.Email == "" {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "User does not exist"})
		return
	}

	file, header, err := r.FormFile("photo")

	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	filename := header.Filename

	if storageSelector == "AWS" {

		accessKey := os.Getenv("AWS_ACCESS_KEY")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY_ID")
		s3Region := os.Getenv("AWS_REGION")
		s3BucketName := os.Getenv("S3_BUCKET_NAME_DISPLAYIMAGE")

		credentials, err := session.NewSession(&aws.Config{
			Region: aws.String(s3Region),
			Credentials: credentials.NewStaticCredentials(
				accessKey,
				secretKey,
				""),
		})

		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		uploader := s3manager.NewUploader(credentials)

		s3link, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(s3BucketName),
			Key:    aws.String(filename),
			Body:   file,
			ACL:    aws.String("public-read"),
		})

		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		imageLink = s3link.Location
	} else if storageSelector == "GCP" {

		cloudStorageBucketName := os.Getenv("GCP_BUCKET_NAME_DISPLAYIMAGE")
		serviceAccountKey := os.Getenv("GCP_SERVICE_ACCOUNT_KEY")

		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAccountKey))
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		bucket := client.Bucket(cloudStorageBucketName)
		object := bucket.Object(filename)

		wc := object.NewWriter(context.Background())
		if _, err := io.Copy(wc, file); err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to write file to bucket"})
			return
		}
		if err := wc.Close(); err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		attrs, err := object.Attrs(context.Background())
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		imageLink = fmt.Sprintf("https://storage.googleapis.com/%s/%s", attrs.Bucket, attrs.Name)

	}

	userid, _ := strconv.Atoi(userId)

	if imageType != "companylogo" {
		err = users.UpdateProfileImage(imageLink, userid)
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
	}

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"s3Link": imageLink,
	})
}

func UploadDockerBuildLogs(fileName string, body []byte) string {

	storageSelector := os.Getenv("STORAGE_SELECTOR")
	storageLink := ""
	if storageSelector == "GCP" {
		cloudStorageBucketName := os.Getenv("GCP_BUCKET_NAME_BUILD_LOGS")
		serviceAccountKey := os.Getenv("GCP_SERVICE_ACCOUNT_KEY")

		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAccountKey))
		if err != nil {
			return ""
		}

		bucket := client.Bucket(cloudStorageBucketName)
		object := bucket.Object(fileName)

		file := ioutil.NopCloser(bytes.NewReader(body))

		wc := object.NewWriter(context.Background())
		if _, err := io.Copy(wc, file); err != nil {
			return ""
		}
		if err := wc.Close(); err != nil {
			return ""
		}

		attrs, err := object.Attrs(context.Background())
		if err != nil {
			return ""
		}

		storageLink = fmt.Sprintf("https://storage.googleapis.com/%s/%s", attrs.Bucket, attrs.Name)
	} else if storageSelector == "AWS" {
		readFile := bytes.NewReader(body)

		accessKey := os.Getenv("AWS_ACCESS_KEY")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY_ID")
		s3Region := os.Getenv("AWS_REGION")
		s3BucketName := os.Getenv("S3_BUCKET_NAME")

		credentials, err := session.NewSession(&aws.Config{
			Region: aws.String(s3Region),
			Credentials: credentials.NewStaticCredentials(
				accessKey,
				secretKey,
				""),
		})
		if err != nil {
			return ""
		}

		uploader := s3manager.NewUploader(credentials)

		s3link, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(s3BucketName),
			Key:    aws.String(fileName),
			Body:   readFile,
			ACL:    aws.String("public-read"),
		})
		if err != nil {
			return ""
		}
		storageLink = s3link.Location
	}
	return storageLink
}

type BatchDelete struct {
	Client    s3iface.S3API
	BatchSize int
}

func DeleteProfileImage(w http.ResponseWriter, r *http.Request) {

	var dataBody UserDetails

	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	userId := strconv.Itoa(dataBody.UserId)

	UserDetails, err := users.GetEmailById(userId)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "User does not exist"})
		return
	}
	if UserDetails.Email == "" {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "User does not exist"})
		return
	}

	keys, ok := r.URL.Query()["imageName"]

	if !ok || len(keys[0]) < 1 {
		log.Println("Url Param 'key' is missing")
		return
	}
	ImageName := keys[0]

	s3BucketName := os.Getenv("S3_BUCKET_NAME_DISPLAYIMAGE")

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	err = DeleteItem(sess, &s3BucketName, &ImageName)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	if dataBody.ImageType == "" {
		err = users.UpdateProfileImage("", dataBody.UserId)
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
	}

	helper.RespondwithJSON(w, http.StatusOK, map[string]string{"message": "Deleted Successfully"})
}

func DeleteItem(sess *session.Session, bucket *string, item *string) error {
	svc := s3.New(sess)

	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: bucket,
		Key:    item,
	})
	if err != nil {
		return err
	}

	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: bucket,
		Key:    item,
	})
	if err != nil {
		return err
	}

	return nil
}
