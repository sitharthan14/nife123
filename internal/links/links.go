package links

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sts"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	"github.com/nifetency/nife.io/internal/users"
)

// #1
type Link struct {
	ID      string
	Title   string
	Address string
	User    *users.User
}

// #2
func (link Link) Save() int64 {
	//#3
	statement, err := database.Db.Prepare("INSERT INTO Links(Title,Address, UserID) VALUES(?,?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	//#4
	res, err := statement.Exec(link.Title, link.Address, link.User.ID)
	if err != nil {
		log.Fatal(err)
	}
	//#5
	id, err := res.LastInsertId()
	if err != nil {
		log.Fatal("Error:", err.Error())
	}
	log.Print("Row inserted!")
	return id
}

func GetAll() []Link {
	stmt, err := database.Db.Prepare("select L.id, L.title, L.address, L.UserID, U.Email from Links L inner join Users U on L.UserID = U.ID")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var links []Link
	var email string
	var id string
	for rows.Next() {
		var link Link
		err := rows.Scan(&link.ID, &link.Title, &link.Address, &id, &email)
		if err != nil {
			log.Fatal(err)
		}
		link.User = &users.User{
			ID:    id,
			Email: email,
		}
		links = append(links, link)
	}
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return links
}

func GetFileFromS3(SourceURL string) ([]byte, error) {
	url := SourceURL
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	reader, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

func GetFileFromPrivateS3(s3Url string) ([]byte, error) {
	storageSelector := os.Getenv("STORAGE_SELECTOR")
	var err error
	if storageSelector == "AWS" {
		s3Region := os.Getenv("AWS_REGION")
		instanceBucketName := os.Getenv("S3_BUCKET_NAME_INSTANCE")

		fileName, err := getVMFileName(s3Url)
		if err != nil {
			return nil, err
		}

		roleSess, err := session.NewSession(&aws.Config{
			Region: aws.String(s3Region),
		})

		if err != nil {
			return nil, err
		}

		// Create a STS client
		svc := sts.New(roleSess)

		result, err := svc.GetSessionToken(&sts.GetSessionTokenInput{})
		if err != nil {
			return nil, err
		}

		file, err := os.Create("temp.json")
		if err != nil {
			return nil, err
		}

		defer file.Close()

		s3sess, err := session.NewSession(&aws.Config{

			Region: aws.String(s3Region),
			Credentials: credentials.NewStaticCredentials(
				*result.Credentials.AccessKeyId,
				*result.Credentials.SecretAccessKey,
				*result.Credentials.SessionToken,
			),
		})

		if err != nil {
			return nil, err
		}

		downloader := s3manager.NewDownloader(s3sess, func(d *s3manager.Downloader) {
			d.PartSize = 5 * 1024 * 1024
			d.Concurrency = 100
		})

		_, err = downloader.Download(file,
			&s3.GetObjectInput{
				Bucket: aws.String(instanceBucketName),
				Key:    aws.String(fileName),
			})

		if err != nil {
			return nil, err
		}
	} else if storageSelector == "GCP" {

		response, err := http.Get(s3Url)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()

		file, err := os.Create("temp.json")
		if err != nil {
			return nil, err
		}
		defer file.Close()

		_, err = io.Copy(file, response.Body)
		if err != nil {
			return nil, err
		}

	}

	return nil, err

}

func getVMFileName(s3ObjectUrl string) (string, error) {

	u, err := url.Parse(s3ObjectUrl)
	if err != nil {
		log.Fatal(err)
	}

	path := strings.SplitN(u.Path, "/", 3)
	s3FileName := path[1]

	return s3FileName, nil
}

func GetFileFromPrivateS3kubeconfig(s3Url, fileName string) ([]byte, error) {
	storageSelector := os.Getenv("STORAGE_SELECTOR")
	var err error
	if storageSelector == "AWS" {

		s3Region := os.Getenv("AWS_REGION")
		instanceBucketName := os.Getenv("S3_BUCKET_NAME_INSTANCE")

		fileName, err := getVMFileName(s3Url)
		if err != nil {
			return nil, err
		}

		roleSess, err := session.NewSession(&aws.Config{
			Region: aws.String(s3Region),
		})

		if err != nil {
			return nil, err
		}

		// Create a STS client
		svc := sts.New(roleSess)

		result, err := svc.GetSessionToken(&sts.GetSessionTokenInput{})
		if err != nil {
			return nil, err
		}

		file, err := os.Create(fileName)
		if err != nil {
			return nil, err
		}

		defer file.Close()

		s3sess, err := session.NewSession(&aws.Config{

			Region: aws.String(s3Region),
			Credentials: credentials.NewStaticCredentials(
				*result.Credentials.AccessKeyId,
				*result.Credentials.SecretAccessKey,
				*result.Credentials.SessionToken,
			),
		})

		if err != nil {
			return nil, err
		}

		downloader := s3manager.NewDownloader(s3sess, func(d *s3manager.Downloader) {
			d.PartSize = 5 * 1024 * 1024
			d.Concurrency = 100
		})

		_, err = downloader.Download(file,
			&s3.GetObjectInput{
				Bucket: aws.String(instanceBucketName),
				Key:    aws.String(fileName),
			})

		if err != nil {
			return nil, err
		}
	} else if storageSelector == "GCP" {
		response, err := http.Get(s3Url)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()

		file, err := os.Create(fileName)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		_, err = io.Copy(file, response.Body)
		if err != nil {
			return nil, err
		}
	}

	return nil, err

}
