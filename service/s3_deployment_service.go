package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

type EnvVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type EnvVariables struct {
	Variables []EnvVariable `json:"envVariables"`
}

func SetEnvironmentVariables(envVars []*model.S3EnvVariables) {
	for _, v := range envVars {
		os.Setenv(*v.Name, *v.Value)
	}
}
func ExecuteCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type ResponseData struct {
	ImageName []byte   `json:"imageName"`
	BuildLogs []string `json:"buildLogs"`
}

func CreateS3Bucket(svc *s3.S3, bucketName string) error {
	_, err := svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return err
	}
	err = DisableBlockPublicAccess(svc, bucketName)
	if err != nil {
		fmt.Println("Error disabling Block Public Access settings:", err)
		return err
	}
	// Set bucket policy to allow public read access
	bucketPolicy := `{
        "Version": "2008-10-17",
        "Statement": [
            {
                "Sid": "Stmt1380877761162",
                "Effect": "Allow",
                "Principal": {
                    "AWS": "*"
                },
                "Action": "s3:GetObject",
                "Resource": "arn:aws:s3:::` + bucketName + `/*"
            }
        ]
    }`

	_, err = svc.PutBucketPolicy(&s3.PutBucketPolicyInput{
		Bucket: aws.String(bucketName),
		Policy: aws.String(bucketPolicy),
	})
	if err != nil {
		return err
	}
	return nil
}

func DownloadFile(url string, filepath string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	return err
}

func Unzip(src, dest string) error {
	cmd := exec.Command("unzip", "-o", src, "-d", dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func FindPackageJSON(rootDir string) (string, error) {
	var packageDir string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			packageJSONPath := filepath.Join(path, "package.json")
			packageLockJSONPath := filepath.Join(path, "package-lock.json")
			if _, err := os.Stat(packageJSONPath); err == nil {
				if _, err := os.Stat(packageLockJSONPath); err == nil {
					packageDir = path
					return fmt.Errorf("found both package.json and package-lock.json at %s", packageDir)
				}
			}
		}
		return nil
	})

	if err != nil {
		if packageDir != "" {
			return packageDir, nil
		}
		return "", err
	}

	return "", fmt.Errorf("no directory with both package.json and package-lock.json found under %s", rootDir)
}

func UploadFilesToS3(svc *s3.S3, bucketName, directory string) error {

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}

		relativePath = strings.ReplaceAll(relativePath, string(os.PathSeparator), "/")

		fileContent, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		contentType := GetContentType(relativePath)

		_, err = svc.PutObject(&s3.PutObjectInput{
			Bucket:      aws.String(bucketName),
			Key:         aws.String(relativePath),
			Body:        bytes.NewReader(fileContent),
			ContentType: aws.String(contentType),
		})
		if err != nil {
			return err
		}

		fmt.Printf("Uploaded file %s\n", relativePath)

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func EnableStaticWebsiteHosting(svc *s3.S3, bucketName string) (string, error) {

	s3Region := os.Getenv("AWS_REGION")

	_, err := svc.PutBucketWebsite(&s3.PutBucketWebsiteInput{
		Bucket: aws.String(bucketName),
		WebsiteConfiguration: &s3.WebsiteConfiguration{
			IndexDocument: &s3.IndexDocument{
				Suffix: aws.String("index.html"),
			},
		},
	})
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("http://%s.s3-website.%s.amazonaws.com", bucketName, s3Region)

	return url, nil
}

func DisableBlockPublicAccess(svc *s3.S3, bucketName string) error {
	_, err := svc.PutPublicAccessBlock(&s3.PutPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
		PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(false),
			IgnorePublicAcls:      aws.Bool(false),
			BlockPublicPolicy:     aws.Bool(false),
			RestrictPublicBuckets: aws.Bool(false),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func DeployOnS3(bucketName, buildFileName, accessKey, secretKey, s3Region, oldDir string) (string, error) {

	creds := credentials.NewStaticCredentials(accessKey, secretKey, "")
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s3Region),
		Credentials: creds,
	})
	if err != nil {
		err = os.Chdir(oldDir)
		if err != nil {
			return "", err
		}
		return "", err
	}

	svc := s3.New(sess)

	err = CreateS3Bucket(svc, bucketName)
	if err != nil {
		err = os.Chdir(oldDir)
		if err != nil {
			return "", err
		}
		return "", err
	}

	wd, err := os.Getwd()
	if err != nil {
		err = os.Chdir(oldDir)
		if err != nil {
			return "", err
		}
		return "", err
	}

	directory := filepath.Join(wd, buildFileName)

	err = UploadFilesToS3(svc, bucketName, directory)
	if err != nil {
		err = os.Chdir(oldDir)
		if err != nil {
			return "", err
		}
		return "", err
	}

	staticwebsite, err := EnableStaticWebsiteHosting(svc, bucketName)
	if err != nil {
		err = os.Chdir(oldDir)
		if err != nil {
			return "", err
		}
		return "", err
	}
	return staticwebsite, nil
}

func DeleteZipFile(folderPath, fileName string) error {
	// Construct the full file path for the ZIP file
	zipFilePath := filepath.Join(folderPath, fileName)

	// Check if the ZIP file exists
	_, err := os.Stat(zipFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// ZIP file doesn't exist, return without attempting deletion
			return fmt.Errorf("ZIP file does not exist: %s", zipFilePath)
		}
		// Other error occurred
		return err
	}

	// Attempt to remove the ZIP file
	err = os.Remove(zipFilePath)
	if err != nil {
		return err
	}

	return nil
}
func DeleteS3BucketDeployment(bucketName, accessKey, secretKey, s3Region string) error {
	creds := credentials.NewStaticCredentials(accessKey, secretKey, "")
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s3Region),
		Credentials: creds,
	})
	if err != nil {
		return err
	}

	svc := s3.New(sess)

	// List all objects in the bucket
	listParams := &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	}
	resp, err := svc.ListObjects(listParams)
	if err != nil {
		return err
	}

	// Delete each object
	for _, obj := range resp.Contents {
		delParams := &s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    obj.Key,
		}
		_, err := svc.DeleteObject(delParams)
		if err != nil {
			return err
		}
		fmt.Printf("Deleted object %s from bucket %s\n", *obj.Key, bucketName)
	}

	// Delete the bucket
	delBucketParams := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}
	_, err = svc.DeleteBucket(delBucketParams)
	if err != nil {
		return err
	}
	fmt.Printf("Bucket %s deleted successfully\n", bucketName)

	return nil
}

func CreateS3Deployment(s3Deployment model.S3DeployInput, userId, staticUrl, envArgs, buildCommands, deploymentTime, buildTime string) (string, error) {
	statement, err := database.Db.Prepare("INSERT INTO s3_app_deployment (id, name, status, app_url, envArgs, build_commands, organization_id, build_file_url, deployment_time, build_time, createdBy, createdAt) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	defer statement.Close()
	_, err = statement.Exec(id, s3Deployment.S3AppName, "Active", staticUrl, envArgs, buildCommands, s3Deployment.OrganizationID, s3Deployment.S3Url, deploymentTime, buildTime, userId, time.Now())
	if err != nil {
		return "", err
	}
	return id, nil
}
func DeleteS3Deployment(s3AppName, userId string) error {
	statement, err := database.Db.Prepare(`UPDATE s3_app_deployment SET status = ? WHERE createdBy = ? and name = ?;`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec("Terminated", userId, s3AppName)
	if err != nil {
		return err
	}
	return nil
}

func CheckS3DeploymentUser(s3AppName, userId string) (string, error) {

	query := `SELECT createdBy FROM s3_app_deployment where createdBy = ? and name = ?;`

	selDB, err := database.Db.Query(query, userId, s3AppName)
	if err != nil {
		return "", err
	}
	defer selDB.Close()
	var userIds string

	for selDB.Next() {
		err = selDB.Scan(&userIds)
		if err != nil {
			return "", err
		}
	}

	return userIds, nil
}

func CheckS3AppName(s3AppName string) (string, error) {

	query := `SELECT name FROM s3_app_deployment where name = ?;`

	selDB, err := database.Db.Query(query, s3AppName)
	if err != nil {
		return "", err
	}
	defer selDB.Close()
	var appname string

	for selDB.Next() {
		err = selDB.Scan(&appname)
		if err != nil {
			return "", err
		}
	}

	return appname, nil
}

func GetAllS3Deployments(userId string) ([]model.S3Deployments, error) {

	query := `SELECT id, name, status, app_url, envArgs, build_commands, organization_id, deployment_time, build_time, createdBy, createdAt FROM s3_app_deployment where createdBy = ? and status = ?;`

	selDB, err := database.Db.Query(query, userId, "Active")
	if err != nil {
		return []model.S3Deployments{}, err
	}
	defer selDB.Close()
	var getS3Dep model.S3Deployments
	var result []model.S3Deployments
	var envArgs string
	var buildCommand string

	for selDB.Next() {
		err = selDB.Scan(&getS3Dep.ID, &getS3Dep.S3AppName, &getS3Dep.Status, &getS3Dep.AppURL, &envArgs, &buildCommand, &getS3Dep.OrganizationID, &getS3Dep.DeploymentTime, &getS3Dep.BuildTime, &getS3Dep.CreatedBy, &getS3Dep.CreatedAt)
		if err != nil {
			return []model.S3Deployments{}, err
		}
		if envArgs != "" {
			err := json.Unmarshal([]byte(envArgs), &getS3Dep.EnvVariablesS3)
			if err != nil {
				return nil, err
			}
		}
		if buildCommand != "" {
			err = json.Unmarshal([]byte(buildCommand), &getS3Dep.BuildCommandsS3)
			if err != nil {
				return nil, err
			}
		}

		orgDet, err := GetOrganization(*getS3Dep.OrganizationID, "")
		if err != nil {
			return nil, err

		}
		getS3Dep.OrgDetails = orgDet

		userDet, err := GetById(userId)
		if err != nil {
			return nil, err
		}

		getS3Dep.UserDetails = &userDet

		result = append(result, getS3Dep)
	}

	return result, nil
}

func GetAllS3DeploymentsByOrgId(userId, orgId string) ([]model.S3Deployments, error) {

	query := `SELECT id, name, status, app_url, envArgs, build_commands, organization_id, deployment_time, build_time, createdBy, createdAt FROM s3_app_deployment where createdBy = ? and status = ? and organization_id = ?;`

	selDB, err := database.Db.Query(query, userId, "Active", orgId)
	if err != nil {
		return []model.S3Deployments{}, err
	}
	defer selDB.Close()
	var getS3Dep model.S3Deployments
	var result []model.S3Deployments
	var envArgs string
	var buildCommand string

	for selDB.Next() {
		err = selDB.Scan(&getS3Dep.ID, &getS3Dep.S3AppName, &getS3Dep.Status, &getS3Dep.AppURL, &envArgs, &buildCommand, &getS3Dep.OrganizationID, &getS3Dep.DeploymentTime, &getS3Dep.BuildTime, &getS3Dep.CreatedBy, &getS3Dep.CreatedAt)
		if err != nil {
			return []model.S3Deployments{}, err
		}

		userDet, err := GetById(userId)
		if err != nil {
			return nil, err
		}

		getS3Dep.UserDetails = &userDet

		result = append(result, getS3Dep)
	}

	return result, nil
}

func GetS3DeploymentsCountByUsreId(userId string) (int, error) {

	query := `SELECT count(name) FROM s3_app_deployment where createdBy = ? and status = ?;`

	selDB, err := database.Db.Query(query, userId, "Active")
	if err != nil {
		return 0, err
	}
	defer selDB.Close()
	var appCount int

	for selDB.Next() {
		err = selDB.Scan(&appCount)
		if err != nil {
			return 0, err
		}
	}

	return appCount, nil
}

func GetS3DeploymentsByAppName(userId, appName string) (model.S3Deployments, error) {

	query := `SELECT id, name, status, app_url, envArgs, build_commands, organization_id, deployment_time, build_time, createdBy, createdAt FROM s3_app_deployment where createdBy = ? and name = ?;`

	selDB, err := database.Db.Query(query, userId, appName)
	if err != nil {
		return model.S3Deployments{}, err
	}
	defer selDB.Close()
	var getS3Dep model.S3Deployments
	var envArgs string
	var buildCommand string

	for selDB.Next() {
		err = selDB.Scan(&getS3Dep.ID, &getS3Dep.S3AppName, &getS3Dep.Status, &getS3Dep.AppURL, &envArgs, &buildCommand, &getS3Dep.OrganizationID, &getS3Dep.DeploymentTime, &getS3Dep.BuildTime, &getS3Dep.CreatedBy, &getS3Dep.CreatedAt)
		if err != nil {
			return model.S3Deployments{}, err
		}
		if envArgs != "" {
			err := json.Unmarshal([]byte(envArgs), &getS3Dep.EnvVariablesS3)
			if err != nil {
				return model.S3Deployments{}, err
			}
		}
		if buildCommand != "" {
			err = json.Unmarshal([]byte(buildCommand), &getS3Dep.BuildCommandsS3)
			if err != nil {
				return model.S3Deployments{}, err
			}
		}

		orgDet, err := GetOrganization(*getS3Dep.OrganizationID, "")
		if err != nil {
			return model.S3Deployments{}, err

		}
		getS3Dep.OrgDetails = orgDet

		userDet, err := GetById(userId)
		if err != nil {
			return model.S3Deployments{}, err
		}

		getS3Dep.UserDetails = &userDet

	}

	return getS3Dep, nil
}

func GetContentType(filename string) string {
	extension := filepath.Ext(filename)
	switch extension {
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	// Add more cases for other file types as needed
	default:
		return "application/octet-stream" // Default to binary data if type is unknown
	}
}

func HasIndexHTMLFile(path string) bool {
    _, err := os.Stat(filepath.Join(path, "index.html"))
    if err != nil {
        if os.IsNotExist(err) {
            return false
        }
    }
    return true
}

func FindIndexHTMLFilePath(rootPath string) string {
    if HasIndexHTMLFile(rootPath) {
        return rootPath
    }

    var indexPath string
    err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() && path != rootPath {
            if HasIndexHTMLFile(path) {
                indexPath = path
                return filepath.SkipDir 
            }
        }
        return nil
    })

    if err != nil {
        fmt.Println("Error:", err)
    }

    return indexPath 
}