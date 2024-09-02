package service

import (
	"fmt"

	// "github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/api/model"
	env "github.com/nifetency/nife.io/helper"
	clusterDetails "github.com/nifetency/nife.io/internal/cluster_info"
	"github.com/nifetency/nife.io/internal/duplo"
	"github.com/nifetency/nife.io/pkg/helper"
)

func AddOrRemoveDuplo(region []*string, appName string, userId string) error {

	if len(region) != 0 {
		for _, regCode := range region {
			deploymentType, err := clusterDetails.GetClusterDetails(*regCode, userId)
			if err != nil {
				return err
			}
			if *deploymentType.InterfaceType != "REST" {
				return nil
			}

			appDetails, err := GetApp(appName, userId)
			if err != nil {
				return err
			}

			internalPort, err := helper.GetInternalPort(appDetails.Config.Definition)
			if err != nil {
				return err
			}
			externalPort, err := helper.GetExternalPort(appDetails.Config.Definition)
			if err != nil {
				return err
			}

			var environmentArgument string

			if appDetails.EnvArgs != nil {
				environmentArgument, err = env.EnvironmentArgument(*appDetails.EnvArgs, *deploymentType.InterfaceType, "", model.Requirement{})
				if err != nil {
					return err
				}
			}

			fmt.Println(environmentArgument)

			for _, i := range appDetails.Regions {
				if i.Code == regCode && appDetails.Status != "Terminated" {

					reDeploy := UpdateDuplo{
						AppName:             appName,
						Image:               *appDetails.Config.Build.Image,
						Status:              appDetails.Status,
						UserId:              userId,
						InternalPort:        int(internalPort),
						ExternalPort:        int(externalPort),
						AgentPlatForm:       *deploymentType.ExternalAgentPlatForm,
						ExternalBaseAddress: *deploymentType.ExternalBaseAddress,
						TenantId:            *deploymentType.TenantID,
					}

					err = reDeploy.UpdateDuploApp()
					if err != nil {
						return err
					}
					return nil
				}

			}

			duplo := DuploDetails{
				AppName:             appName,
				Image:               *appDetails.Config.Build.Image,
				UserId:              userId,
				AllocationTag:       *deploymentType.AllocationTag,
				EnvArgs:             environmentArgument,
				InternalPort:        int(internalPort),
				ExternalPort:        int(externalPort),
				AgentPlatForm:       *deploymentType.ExternalAgentPlatForm,
				ExternalBaseAddress: *deploymentType.ExternalBaseAddress,
				TenantId:            *deploymentType.TenantID,
			}

			err = duplo.CreateDuploService()

			if err != nil {
				fmt.Println(err)
				return fmt.Errorf(err.Error())
			}

			err = DeployType(2, appName)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

type SecretElement struct {
	RegistryName        string
	TenantId            string
	ExternalBaseAddress string
}

func (secret *SecretElement) CheckandDeleteSecret()error {

	getSecret := duplo.Getsecret{
		Name:                secret.RegistryName,
		TenantId:            secret.TenantId,
		ExternalBaseAddress: secret.ExternalBaseAddress,
	}

	value, err := getSecret.GetDuploSecret()
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	if value {
		delSecret := duplo.DeleteSecret{
			SecretName:          secret.RegistryName,
			ExternalBaseAddress: secret.ExternalBaseAddress,
			TenantId:            secret.TenantId,
		}

		err = delSecret.DeleteDuploSecret()

		if err != nil {
			return fmt.Errorf(err.Error())
		}
	}
	return nil
}
