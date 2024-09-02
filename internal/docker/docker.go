package docker

import (
	"fmt"
	"log"

	"github.com/docker/docker/client"
)

func DockerClient()(*client.Client,error){
	cli, err := client.NewEnvClient()
	if err != nil {
		log.Println(err)
		return &client.Client{},fmt.Errorf(err.Error())
	}

	return cli, nil

}