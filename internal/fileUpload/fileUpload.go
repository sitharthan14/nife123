package fileupload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gorilla/mux"
	"github.com/nifetency/nife.io/helper"
	"google.golang.org/api/option"
)

// const (
// 	maxPartSize = int64(5 * 1024 * 1024)
// 	maxRetries  = 3
// )


func FileUpload(w http.ResponseWriter, r *http.Request) {
	storageSelector := os.Getenv("STORAGE_SELECTOR")

	var FileName string
	keys, ok := r.URL.Query()["fileExtension"]

	if !ok || len(keys[0]) < 1 {
		log.Println("Url Param 'key' is missing")
		return
	}
	key := keys[0]

	if key == "zip" || key == "tar" || key == "tar.gz" {
		appName := mux.Vars(r)
		fileName := appName["name"] + "." + key
		FileName = fileName

	} else {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Invalid file extension"})
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Print(err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Can't find the " + key + " file to upload"})
		return
	}
	if storageSelector == "AWS" {
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
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		uploader := s3manager.NewUploader(credentials)

		s3link, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(s3BucketName),
			Key:    aws.String(FileName),
			Body:   readFile,
			ACL:    aws.String("public-read"),
		})
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
			"s3Link": s3link.Location,
		})

		
		//----------------------------------------------------------

		// accessKey := os.Getenv("AWS_ACCESS_KEY")
		// secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY_ID")
		// s3Region := os.Getenv("AWS_REGION")
		// s3BucketName := os.Getenv("S3_BUCKET_NAME")
		// creds := credentials.NewStaticCredentials(accessKey, secretKey, "")
		// _, err := creds.Get()
		// if err != nil {
		// 	fmt.Printf("bad credentials: %s", err)
		// }
		// cfg := aws.NewConfig().WithRegion(s3Region).WithCredentials(creds)
		// svc := s3.New(session.New(), cfg)

		// defer r.Body.Close()
		// size := int64(len(body))
		// fileType := http.DetectContentType(body)

		// // file, err := os.Open("react-app-public-m1.zip")
		// // if err != nil {
		// // 	fmt.Printf("err opening file: %s", err)
		// // 	return
		// // }
		// // defer file.Close()
		// // fileInfo, _ := file.Stat()
		// // size := fileInfo.Size()
		// buffer := make([]byte, size)
		// // fileType := http.DetectContentType(buffer)
		// // file.Read(buffer)

		// path := FileName
		// input := &s3.CreateMultipartUploadInput{
		// 	Bucket:      aws.String(s3BucketName),
		// 	Key:         aws.String(path),
		// 	ContentType: aws.String(fileType),
		// }

		// resp, err := svc.CreateMultipartUpload(input)
		// if err != nil {
		// 	fmt.Println(err.Error())
		// 	return
		// }
		// fmt.Println("Created multipart upload request")

		// var curr, partLength int64
		// var remaining = size
		// var completedParts []*s3.CompletedPart
		// partNumber := 1
		// for curr = 0; remaining != 0; curr += partLength {
		// 	if remaining < maxPartSize {
		// 		partLength = remaining
		// 	} else {
		// 		partLength = maxPartSize
		// 	}
		// 	completedPart, err := uploadPart(svc, resp, buffer[curr:curr+partLength], partNumber)
		// 	if err != nil {
		// 		fmt.Println(err.Error())
		// 		err := abortMultipartUpload(svc, resp)
		// 		if err != nil {
		// 			fmt.Println(err.Error())
		// 		}
		// 		return
		// 	}
		// 	remaining -= partLength
		// 	partNumber++
		// 	completedParts = append(completedParts, completedPart)
		// }

		// completeResponse, err := completeMultipartUpload(svc, resp, completedParts)
		// if err != nil {
		// 	fmt.Println(err.Error())
		// 	return
		// }
		// fmt.Printf("Successfully uploaded file: %s\n", completeResponse.String())
		// helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		// 	"s3Link": *completeResponse.Location,
		// })

	} else if storageSelector == "GCP" {
		cloudStorageBucketName := os.Getenv("GCP_BUCKET_NAME")
		serviceAccountKey := os.Getenv("GCP_SERVICE_ACCOUNT_KEY")

		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAccountKey))
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		bucket := client.Bucket(cloudStorageBucketName)
		object := bucket.Object(FileName)

		file := ioutil.NopCloser(bytes.NewReader(body))

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

		url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", attrs.Bucket, attrs.Name)

		helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
			"s3Link": url,
		})
	}
}

func InstanceKeyUpload(w http.ResponseWriter, r *http.Request) {
	storageSelector := os.Getenv("STORAGE_SELECTOR")
	var storageLink string

	appName := mux.Vars(r)
	FileName := appName["name"] + ".json"

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Print(err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Can't find the Json file to upload"})
		return
	}

	if storageSelector == "AWS" {

		readFile := bytes.NewReader(body)

		accessKey := os.Getenv("AWS_ACCESS_KEY")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY_ID")
		s3Region := os.Getenv("AWS_REGION")
		s3BucketName := os.Getenv("S3_BUCKET_NAME_INSTANCE")

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
			Key:    aws.String(FileName),
			Body:   readFile,
			// ACL:    aws.String("public-read"),
			ACL: aws.String("public-read"),
		})
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		storageLink = s3link.Location
	} else if storageSelector == "GCP" {
		cloudStorageBucketName := os.Getenv("GCP_BUCKET_NAME_INSTANCE")
		serviceAccountKey := os.Getenv("GCP_SERVICE_ACCOUNT_KEY")

		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAccountKey))
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		bucket := client.Bucket(cloudStorageBucketName)
		object := bucket.Object(FileName)

		file := ioutil.NopCloser(bytes.NewReader(body))

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

		storageLink = fmt.Sprintf("https://storage.googleapis.com/%s/%s", attrs.Bucket, attrs.Name)
	}

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"s3Link": storageLink,
	})

}

func KubeConfigFileUpload(w http.ResponseWriter, r *http.Request) {

	storageSelector := os.Getenv("STORAGE_SELECTOR")
	var storageLink string

	s3Name := mux.Vars(r)
	FileName := s3Name["name"] + ".yaml"

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Print(err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Can't find the Yaml file to upload"})
		return
	}
	if storageSelector == "AWS" {
		readFile := bytes.NewReader(body)

		accessKey := os.Getenv("AWS_ACCESS_KEY")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY_ID")
		s3Region := os.Getenv("AWS_REGION")
		s3BucketName := os.Getenv("S3_BUCKET_NAME_INSTANCE")

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
			Key:    aws.String(FileName),
			Body:   readFile,
			// ACL:    aws.String("public-read"),
			ACL: aws.String("public-read"),
		})
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}
		storageLink = s3link.Location
	} else if storageSelector == "GCP" {
		cloudStorageBucketName := os.Getenv("GCP_BUCKET_NAME_INSTANCE")
		serviceAccountKey := os.Getenv("GCP_SERVICE_ACCOUNT_KEY")

		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAccountKey))
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
		}

		bucket := client.Bucket(cloudStorageBucketName)
		object := bucket.Object(FileName)

		file := ioutil.NopCloser(bytes.NewReader(body))

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

		storageLink = fmt.Sprintf("https://storage.googleapis.com/%s/%s", attrs.Bucket, attrs.Name)
	}

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"s3Link": storageLink,
	})
}

func FinopsGCPKeyUpload(w http.ResponseWriter, r *http.Request) {
	var storageLink string

	appName := mux.Vars(r)
	FileName := appName["name"] + ".json"

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Print(err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Can't find the Json file to upload"})
		return
	}

	cloudStorageBucketName := os.Getenv("GCP_BUCKET_NAME_INSTANCE")
	serviceAccountKey := os.Getenv("GCP_SERVICE_ACCOUNT_KEY")

	client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAccountKey))
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	bucket := client.Bucket(cloudStorageBucketName)
	object := bucket.Object(FileName)

	file := ioutil.NopCloser(bytes.NewReader(body))

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

	storageLink = fmt.Sprintf("https://storage.googleapis.com/%s/%s", attrs.Bucket, attrs.Name)

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"s3Link": storageLink,
	})

}

// func completeMultipartUpload(svc *s3.S3, resp *s3.CreateMultipartUploadOutput, completedParts []*s3.CompletedPart) (*s3.CompleteMultipartUploadOutput, error) {
// 	completeInput := &s3.CompleteMultipartUploadInput{
// 		Bucket:   resp.Bucket,
// 		Key:      resp.Key,
// 		UploadId: resp.UploadId,
// 		MultipartUpload: &s3.CompletedMultipartUpload{
// 			Parts: completedParts,
// 		},
// 	}
// 	return svc.CompleteMultipartUpload(completeInput)
// }

// func uploadPart(svc *s3.S3, resp *s3.CreateMultipartUploadOutput, fileBytes []byte, partNumber int) (*s3.CompletedPart, error) {
// 	tryNum := 1
// 	partInput := &s3.UploadPartInput{
// 		Body:          bytes.NewReader(fileBytes),
// 		Bucket:        resp.Bucket,
// 		Key:           resp.Key,
// 		PartNumber:    aws.Int64(int64(partNumber)),
// 		UploadId:      resp.UploadId,
// 		ContentLength: aws.Int64(int64(len(fileBytes))),
// 	}

// 	for tryNum <= maxRetries {
// 		uploadResult, err := svc.UploadPart(partInput)
// 		if err != nil {
// 			if tryNum == maxRetries {
// 				if aerr, ok := err.(awserr.Error); ok {
// 					return nil, aerr
// 				}
// 				return nil, err
// 			}
// 			fmt.Printf("Retrying to upload part #%v\n", partNumber)
// 			tryNum++
// 		} else {
// 			fmt.Printf("Uploaded part #%v\n", partNumber)
// 			return &s3.CompletedPart{
// 				ETag:       uploadResult.ETag,
// 				PartNumber: aws.Int64(int64(partNumber)),
// 			}, nil
// 		}
// 	}
// 	return nil, nil
// }

// func abortMultipartUpload(svc *s3.S3, resp *s3.CreateMultipartUploadOutput) error {
// 	fmt.Println("Aborting multipart upload for UploadId#" + *resp.UploadId)
// 	abortInput := &s3.AbortMultipartUploadInput{
// 		Bucket:   resp.Bucket,
// 		Key:      resp.Key,
// 		UploadId: resp.UploadId,
// 	}
// 	_, err := svc.AbortMultipartUpload(abortInput)
// 	return err
// }
