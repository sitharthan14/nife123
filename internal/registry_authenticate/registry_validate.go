package registryauthenticate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nifetency/nife.io/internal/docker"
	"io"
	"log"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/form3tech-oss/jwt-go"
	"github.com/nifetency/nife.io/internal/decode"
)

type ParseToken struct {
	Token string
}

func SecretAuthentication(registryType, userName, passWord string) error {
	Password := decode.DePwdCode(passWord)

	if registryType == "docker_hub_registry" {
		_, err := DockerHubAuthenticate(userName, Password)
		if err != nil {
			fmt.Println(err)
			return fmt.Errorf(err.Error())
		}
		return nil
	}

	if registryType == "github_registry" {

		err := GitAuthenticate(userName, Password, "ghcr.io")
		if err != nil {
			fmt.Println(err)
			return fmt.Errorf(err.Error())
		}
		return nil
		
	}

	if registryType == "gitlab_registry" {

		err := GitAuthenticate(userName, Password, "registry.gitlab.com")
		if err != nil {
			fmt.Println(err)
			return fmt.Errorf(err.Error())
		}

		return nil

	}

	return fmt.Errorf("Something wrong in registry type %s", registryType)
}

func DockerHubAuthenticate(userName, passWord string) (string, error) {

	postBody, _ := json.Marshal(map[string]string{
		"username": userName,
		"password": passWord,
	})

	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post("https://hub.docker.com/v2/users/login/", "application/json", responseBody)
	if err != nil {
		log.Println(err)
		return "", err
	}

	if resp.StatusCode == 200 {

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return "", err
		}

		var token ParseToken
		json.Unmarshal(body, &token)

		claims := jwt.MapClaims{}
		_, _ = jwt.ParseWithClaims(token.Token, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("gkljEBRMNiobgfWMPX8743_fWACX"), nil
		})

		for key, val := range claims {
			if key == "username" {
				if val == userName {
					return token.Token, nil
				}
			}
		}
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Invalid Username and Password")
	}
	return "", err
}

func GitAuthenticate(userName, passWord, address string) error {

	authConfig := types.AuthConfig{
		Username:      userName,
		Password:      passWord,
		ServerAddress: address,
	}

	cli, err := docker.DockerClient()
	if err != nil {
		return err
	}

	authenticatedOK, err := cli.RegistryLogin(context.TODO(), authConfig)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("Invalid Username and Password")
	}

	if authenticatedOK.Status == "Login Succeeded" {
		return nil
	}
	return nil
}
