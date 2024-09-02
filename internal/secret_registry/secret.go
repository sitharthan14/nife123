package secretregistry

import (
	"fmt"

	"os"

	"github.com/nifetency/nife.io/api/model"

	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

type SecretResponse struct {
	UserName, Password, Url, KeyFileContent, RegistryName, SecretType string
}

func SecretRegData(data map[string]interface{}, registryType, databaseName string) (SecretResponse, error) {

	secretType := os.Getenv("DUPLO_SECRET_TYPE")
	switch {
	case registryType == "":
		payLoad := SerializeRegistryType(data, "", secretType)
		return payLoad, nil

	case registryType == "PAT":
		payLoad := SerializeRegistryType(data, "PAT", secretType)
		return payLoad, nil

	case registryType == "docker_hub_registry":
		payLoad := SerializeRegistryType(data, "https://index.docker.io/v1/", secretType)
		return payLoad, nil

	case registryType == "private_registry":
		payLoad := SerializeRegistryType(data, data["url"].(string), "")
		return payLoad, nil

	case registryType == "digital_ocean_registry":
		payLoad := SerializeRegistryType(data, "", "")
		return payLoad, nil

	case registryType == "github_registry":
		payLoad := SerializeRegistryType(data, "ghcr.io", "")
		return payLoad, nil

	case registryType == "gitlab_registry":
		payLoad := SerializeRegistryType(data, "registry.gitlab.com", "")
		return payLoad, nil

	case registryType == "mysql":
		payLoad := SerializeRegistryType(data, databaseName, "")
		return payLoad, nil

	case registryType == "postgres":
		payLoad := SerializeRegistryType(data, databaseName, "")
		return payLoad, nil

	case registryType == "gcp_container_registry":
		keyFilecontent := data["keyFilecontent"].(string)
		url := data["url"].(string)
		payLoad := SecretResponse{
			KeyFileContent: keyFilecontent,
			Url:            url,
		}
		return payLoad, nil

	case registryType == "env":
		payLoad := SerializeRegistryType(data, "", secretType)
		return payLoad, nil

	case registryType == "azure_container_registry":
		registryName := data["registryName"].(string)
		userName := data["userName"].(string)
		passWord := data["passWord"].(string)
		payLoad := SecretResponse{
			UserName:     userName,
			Password:     passWord,
			RegistryName: registryName,
		}
		return payLoad, nil

	default:
		return SecretResponse{}, fmt.Errorf("Doesn't supported %s this registry type", registryType)
	}
}

func CheckRegistryType(name string) (string, string, error) {

	query := `select registry_type, createdBy from organization_secrets where name = ?`

	selDB, err := database.Db.Query(query, name)
	if err != nil {
		return "", "", err
	}

	var registryType, createdBy string

	for selDB.Next() {

		err = selDB.Scan(&registryType, &createdBy)
	}
	if err != nil {
		return "", "", err
	}

	return registryType, createdBy, nil
}

func SerializeRegistryType(data map[string]interface{}, url, secretdata string) SecretResponse {
	userName := data["userName"].(string)
	passWord := data["passWord"].(string)
	payLoad := SecretResponse{
		UserName:   userName,
		Password:   passWord,
		Url:        url,
		SecretType: secretdata,
	}
	return payLoad

}

func GetSecretDetails(id, name string) (model.GetUserSecret, error) {
	var Query string
	if id != "" {
		Query = "SELECT name, organization_id, registry_type, username, password, url, key_file_content,registry_name,secret_type FROM organization_secrets WHERE id=? and is_active = 1"
	} else {
		Query = "SELECT name, organization_id, registry_type, username, password, url, key_file_content,registry_name,secret_type FROM organization_secrets WHERE name=? and is_active = 1"
		id = name
	}
	selDB, err := database.Db.Query(Query, id)

	if err != nil {
		return model.GetUserSecret{}, err
	}

	defer selDB.Close()
	var userSecret model.GetUserSecret
	for selDB.Next() {
		err = selDB.Scan(&userSecret.Name, &userSecret.OrganizationID, &userSecret.RegistryType, &userSecret.UserName, &userSecret.PassWord,
			&userSecret.URL, &userSecret.KeyFileContent, &userSecret.RegistryName, &userSecret.SecretType)
		if err != nil {
			return model.GetUserSecret{}, err
		}
	}
	return userSecret, nil
}
