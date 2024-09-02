package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"

	"github.com/99designs/gqlgen/graphql"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/nifetency/nife.io/api/model"
)

func (r *mutationResolver) SingleUpload(ctx context.Context, file graphql.Upload) (*model.File, error) {
	content, err := ioutil.ReadAll(file.File)
	if err != nil {
		return &model.File{}, nil
	}

	readFile := bytes.NewReader(content)

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
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
		return &model.File{}, err
	}

	uploader := s3manager.NewUploader(credentials)

	S3link, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3BucketName),
		Key:    aws.String(file.Filename),
		Body:   readFile,
		ACL:    aws.String("public-read"),
	})
	if err != nil {
		return &model.File{}, err
	}
	return &model.File{
		Link: S3link.Location,
	}, err
}
