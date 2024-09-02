package dockerimage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal/decode"
	"github.com/nifetency/nife.io/internal/docker"
	secretregistry "github.com/nifetency/nife.io/internal/secret_registry"
)

type Docker struct {
	Image         string `json:"image"`
	SecRegistryId string `json:"secRegistryId"`
}

func FindDockerImage(w http.ResponseWriter, r *http.Request) {
	var dataBody Docker
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	getCredentials, err := secretregistry.GetSecretDetails(dataBody.SecRegistryId, "")
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	cli, err := docker.DockerClient()
	if err != nil {
		log.Println(err)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	var UserName string
	var Password string
	if getCredentials.PassWord != nil {
		Password = decode.DePwdCode(*getCredentials.PassWord)
	}
	if getCredentials.UserName != nil {
		UserName = *getCredentials.UserName
	}
	authConfig := types.AuthConfig{
		Username: UserName,
		Password: Password,
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		log.Println(err)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	details, err := cli.DistributionInspect(context.Background(), dataBody.Image, authStr)
	if err != nil {
		log.Println(err)
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": "The Docker Image cannot be found in the selected Docker Hub Registry"})
		return
	}

	fmt.Println(details.Descriptor.Digest.String())

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"registry_image_id": details.Descriptor.Digest.String(),
	})

}
