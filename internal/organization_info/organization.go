package organizationInfo

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/google/uuid"
	"github.com/nifetency/nife.io/helper"
	ci "github.com/nifetency/nife.io/internal/cluster_info"
	oragnizationUsers "github.com/nifetency/nife.io/internal/organizaiton_users"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateOrgainzation(name string, userId, defaultType string) (string, error) {

	id := uuid.New().String()
	slug := name
	name = strings.ToUpper(name)
	statement, err := database.Db.Prepare("INSERT INTO organization(id,name,slug,type,is_deleted) VALUES(?,?,?,?,?)")
	if err != nil {
		return "", err
	}

	re, err := regexp.Compile(`[^\w]`)
	if err != nil {
		fmt.Println(err)
	}

	slug = re.ReplaceAllString(slug, "")
	slug = strings.ToLower(slug)

	_, err = statement.Exec(id, name, slug, defaultType, 0)
	if err != nil {
		return "", err
	}

	err = MapOrganization(slug, id, userId, "Admin")
	if err != nil {
		log.Println(err)
		return "", err
	}

	return id, nil
}

func CreateSubOrgainzation(name string, userId, defaultType, parentOrgId string) (string, error) {

	id := uuid.New().String()
	slug := name
	name = strings.ToUpper(name)
	statement, err := database.Db.Prepare("INSERT INTO organization(id,parent_orgid,name,slug,type,is_deleted) VALUES(?,?,?,?,?,?)")
	if err != nil {
		return "", err
	}

	re, err := regexp.Compile(`[^\w]`)
	if err != nil {
		fmt.Println(err)
	}

	slug = re.ReplaceAllString(slug, "")
	slug = strings.ToLower(slug)

	_, err = statement.Exec(id, parentOrgId, name, slug, defaultType, 0)
	if err != nil {
		return "", err
	}
	err = MapOrganization(slug, id, userId, "Admin")
	if err != nil {
		log.Println(err)
		return "", err
	}

	return id, nil
}

func MapOrganization(orgName, orgId, userId, userRole string) error {

	clusterDetails, err := ci.GetActiveClusterDetails(userId)
	if err != nil {
		return err
	}

	statement, err := database.Db.Prepare("INSERT INTO organization_regions (id,organization_id,region_code,is_default) VALUES(?,?,?,?)")
	if err != nil {
		return err
	}

	for _, c := range *clusterDetails {
		k8sPath := "k8s_config/" + c.Region_code
		var fileSavePath string
		if c.Cluster_config_path == "" {
			err := os.Mkdir(k8sPath, 0755)
			if err != nil {
				return err
			}

			fileSavePath = k8sPath + "/config"

			_, err = GetFileFromPrivateS3kubeconfigs(*c.ClusterConfigURL, fileSavePath)
			if err != nil {
				return err
			}
			c.Cluster_config_path = "./k8s_config/" + c.Region_code + "/config"

		}

		isdefault := 0
		id := uuid.New()

		re, err := regexp.Compile(`[^\w]`)
		if err != nil {
			helper.DeletedSourceFile("k8s_config/" + c.Region_code)
			fmt.Println(err)
		}

		orgName = re.ReplaceAllString(orgName, "")
		orgName = strings.ToLower(orgName)

		if c.Cluster_config_path != "" {

			err = CreateNamespaceInCluster(orgName, c.Cluster_config_path)
			if err != nil {
				helper.DeletedSourceFile("k8s_config/" + c.Region_code)
				log.Println(err)
				return err
			}
			helper.DeletedSourceFile("k8s_config/" + c.Region_code)
		}

		if c.Region_code == "IND" {
			isdefault = 1
		}
		// NEED TO SET DEFAULT TO IND
		_, err = statement.Exec(id, orgId, c.Region_code, isdefault)
		if err != nil {
			return err
		}

	}
	userid, _ := strconv.Atoi(userId)
	roleId, err := oragnizationUsers.GetRoleIdByUserId(userid)
	if err != nil {
		return err
	}

	err = oragnizationUsers.AddUserToOrg(orgId, userId, roleId)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func CreateNamespaceInCluster(orgName, configPath string) error {
	clientset, err := helper.LoadK8SConfig(configPath)
	if err != nil {
		log.Println(err)
	}
	nsName := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: orgName,
		},
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.Background(), nsName, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func fourDigits() int64 {
	max := big.NewInt(9999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		log.Println(err)
	}
	return n.Int64()
}

func RandomNumber4Digit() int64 {
	var randNo int64
	for i := 0; i < 5; i++ {
		s := fourDigits()
		randNo = s
	}
	return randNo
}

func CheckOrgExistByUser(organizationId, userId string) (bool, error) {

	query := `SELECT EXISTS (select u.firstName, u.lastName, ou.user_id,
		ou.joined_at, u.email, ou.id, ou.role_id
		from organization_users ou join user u on ou.user_id = u.id where ou.organization_id = ? and u.id = ?)`
	selDB, err := database.Db.Query(query, organizationId, userId)
	if err != nil {
		return false, err
	}
	var orgCheck bool
	defer selDB.Close()

	for selDB.Next() {
		err = selDB.Scan(&orgCheck)
		if err != nil {
			log.Println(err)
			return false, err
		}
	}

	return orgCheck, nil

}

func getVMFileNames(s3ObjectUrl string) (string, error) {

	u, err := url.Parse(s3ObjectUrl)
	if err != nil {
		log.Fatal(err)
	}

	path := strings.SplitN(u.Path, "/", 3)
	s3FileName := path[1]

	return s3FileName, nil
}

func GetFileFromPrivateS3kubeconfigs(s3Url, fileNamePath string) ([]byte, error) {
	storageSelector := os.Getenv("STORAGE_SELECTOR")
	var err error
	if storageSelector == "AWS" {
		s3Region := os.Getenv("AWS_REGION")
		instanceBucketName := os.Getenv("S3_BUCKET_NAME_INSTANCE")

		fileName, err := getVMFileNames(s3Url)
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

		file, err := os.Create(fileNamePath)
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

		file, err := os.Create(fileNamePath)
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