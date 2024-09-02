package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/alecthomas/log4go"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	"github.com/nifetency/nife.io/internal/auth"
	"github.com/nifetency/nife.io/internal/links"
	"github.com/nifetency/nife.io/service"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

func (r *mutationResolver) NodeAction(ctx context.Context, input *model.StartAndStopVM) (*model.VMInstanceMessage, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	ctx = context.Background()
	getHostDet, err := service.GetHostDetails(user.ID, "")
	if err != nil {
		fmt.Println(err)
		log4go.Error("Module: NodeAction, MethodName: GetHostDetails, Message: %s user:%s", err.Error(), user.ID)
		return &model.VMInstanceMessage{}, err
	}
	log4go.Info("Module: NodeAction, MethodName: GetHostDetails, Message: Fetching host details based on user Id is successfully completed, user: %s", user.ID)

	var vmInst model.Host
	for _, i := range getHostDet {
		if *input.InstanceName == *i.InstanceName {
			vmInst.InstanceName = i.InstanceName
			vmInst.ServiceAccountURL = i.ServiceAccountURL
			vmInst.Zone = i.Zone
			vmInst.ID = i.ID
		}
	}

	CheckHost := model.Host{}

	if CheckHost == vmInst {
		return nil, fmt.Errorf("Cannot find the Instance, Try To create a New")
	}

	_, err = links.GetFileFromPrivateS3(*vmInst.ServiceAccountURL)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: NodeAction, MethodName: GetFileFromS3, Message: %s user:%s", err.Error(), user.ID)
		return nil, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: NodeAction, MethodName: GetFileFromS3, Message:successfully reached, user: %s", user.ID)

	// keyFile, err := helper.WriteFileToTemp("temp.json")
	// if err != nil {
	// 	log.Println(err)
	// 	return nil, fmt.Errorf(err.Error())
	// }

	// keyFile.Write(reader)

	computeService, err := compute.NewService(ctx, option.WithCredentialsFile("temp.json"))
	if err != nil {
		return nil, err
	}

	credentials, err := helper.ReadFile()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if *input.Action == "start" {

		checkStatus, err := service.GetInstanceStatus(credentials.ProjectID, *vmInst.Zone, *input.InstanceName)
		if err != nil {
			fmt.Println(err)
		}
		if checkStatus == "RUNNING" {
			service.UpdateHostStatus(*vmInst.ID, "running")
			return nil, fmt.Errorf("This Instance is already RUNNING")
		}

		cic := computeService.Instances.Start(credentials.ProjectID, *vmInst.Zone, *input.InstanceName)
		_, err = cic.Do()
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		time.Sleep(time.Second * 3)

		for {
			status, err := service.GetInstanceStatus(credentials.ProjectID, *vmInst.Zone, *input.InstanceName)
			if err != nil {
				fmt.Println(err)
			}
			err = service.UpdateHostStatus(*vmInst.ID, "running")
			if err != nil {
				fmt.Println(err)
			}
			if "RUNNING" == status {

				userDetAct, err := service.GetById(user.ID)
				if err != nil {
					log4go.Error("Module: NodeAction, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: NodeAction, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

				AddOperation := service.Activity{
					Type:       "INSTANCE",
					UserId:     user.ID,
					Activities: status,
					Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Started the Instance " + *input.InstanceName,
					RefId:      strconv.Itoa(*vmInst.ID),
				}

				_, err = service.InsertActivity(AddOperation)
				if err != nil {
					fmt.Println(err)
					return nil, err
				}
				err = service.SendSlackNotification(user.ID, AddOperation.Message)
				if err != nil {
					log4go.Error("Module: NodeAction, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
				}
				break
			}
		}

	} else if *input.Action == "stop" {

		checkStatus, err := service.GetInstanceStatus(credentials.ProjectID, *vmInst.Zone, *input.InstanceName)
		if err != nil {
			fmt.Println(err)
		}

		if checkStatus == "TERMINATED" || checkStatus == "STOPPING" {
			service.UpdateHostStatus(*vmInst.ID, "stopped")
			return nil, fmt.Errorf("This Instance is already TERMINATED")
		}

		cic := computeService.Instances.Stop(credentials.ProjectID, *vmInst.Zone, *input.InstanceName)
		_, err = cic.Do()
		if err != nil {
			fmt.Println(err)
			return nil, err
		}

		time.Sleep(time.Second * 5)

		for {
			status, err := service.GetInstanceStatus(credentials.ProjectID, *vmInst.Zone, *input.InstanceName)
			if err != nil {
				fmt.Println(err)
			}
			err = service.UpdateHostStatus(*vmInst.ID, "stopped")
			if err != nil {
				fmt.Println(err)
			}
			if "TERMINATED" == status {

				userDetAct, err := service.GetById(user.ID)
				if err != nil {
					log4go.Error("Module: NodeAction, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
					return nil, err
				}
				log4go.Info("Module: NodeAction, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

				AddOperation := service.Activity{
					Type:       "INSTANCE",
					UserId:     user.ID,
					Activities: status,
					Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Stopped the Instance " + *input.InstanceName,
					RefId:      strconv.Itoa(*vmInst.ID),
				}

				_, err = service.InsertActivity(AddOperation)
				if err != nil {
					fmt.Println(err)
					return nil, err
				}
				err = service.SendSlackNotification(user.ID, AddOperation.Message)
				if err != nil {
					log4go.Error("Module: NodeAction, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
				}

				break
			}
		}

	}

	// keyFile.Close()

	err = helper.DeletedSourceFile("temp.json")

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf(err.Error())
	}

	message := "Action Completed"

	return &model.VMInstanceMessage{Message: &message}, nil
}

func (r *mutationResolver) CreateHost(ctx context.Context, input *model.Host) (*model.VMInstanceMessage, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.VMInstanceMessage{}, fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	if input.InstanceName == nil || input.OrgID == nil {
		return &model.VMInstanceMessage{}, fmt.Errorf("Instance Name and Organization Cannot be Empty")
	}
	// if *input.InstanceName != "" || *input.Zone != "" {
	// 	err := service.CheckStringAlphabet(*input.Zone)
	// 	if err != nil {
	// 		return "", fmt.Errorf("The  Zone field Should Not Contains Empty Space or Special Character")
	// 	}
	// 	err = service.CheckStringAlphabet(*input.InstanceName)
	// 	if err != nil {
	// 		return "", fmt.Errorf("The Instance Name field Should Not Contains Empty Space or Special Character")
	// 	}

	// }
	var instanceIds int

	checkHost, err := service.GetHostDetailsByNameAndOrgId(*input.InstanceName, *input.Type, user.ID, *input.OrgID)
	if err != nil {
		log.Println(err)
		log4go.Error("Module: CreateHost, MethodName: GetHostDetailsByNameAndOrgId, Message: %s user:%s", err.Error(), user.ID)
		return &model.VMInstanceMessage{}, fmt.Errorf(err.Error())
	}
	log4go.Info("Module: CreateHost, MethodName: GetHostDetailsByNameAndOrgId, Message: Checking the instance is already available for the organization , user: %s", user.ID)

	if checkHost.ID != nil {
		return &model.VMInstanceMessage{}, fmt.Errorf("The Instance you have created is already present in the Organization")
	}

	//--------------------Creating GCP instance-----------
	if *input.Type == "GCP" {
		if input.Zone == nil {
			return &model.VMInstanceMessage{}, fmt.Errorf("Zone Cannot be Empty")
		}
		_, err := links.GetFileFromPrivateS3(*input.ServiceAccountURL)
		if err != nil {
			log.Println(err)
			log4go.Error("Module: CreateHost, MethodName: GetFileFromS3, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, fmt.Errorf(err.Error())
		}
		log4go.Info("Module: CreateHost, MethodName: GetFileFromS3, Message:successfully reached, user: %s", user.ID)

		// keyFile, err := helper.WriteFileToTemp("temp.json")
		// if err != nil {
		// 	log.Println(err)
		// 	return "", fmt.Errorf(err.Error())
		// }

		// keyFile.Write(reader)

		credentials, err := helper.ReadFile()
		if err != nil {
			fmt.Println(err)
			return &model.VMInstanceMessage{}, err
		}
		checkStatus, err := service.GetInstanceStatus(credentials.ProjectID, *input.Zone, *input.InstanceName)
		if err != nil {
			fmt.Println(err)
		}

		// keyFile.Close()

		err = helper.DeletedSourceFile("temp.json")

		if err != nil {
			log.Println(err)
			return nil, fmt.Errorf(err.Error())
		}

		var status1 string
		if checkStatus == "TERMINATED" {
			status1 = "stopped"
		} else {
			status1 = "running"
		}

		_, instanceIds, err = service.CreateHostService(user.ID, status1, *input)
		if err != nil {
			fmt.Println(err)
			log4go.Error("Module: CreateHost, MethodName: CreateHostService, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, err
		}
		log4go.Info("Module: CreateHost, MethodName: CreateHostService, Message: Host service is successfully created, user: %s", user.ID)
	}
	//--------------------Creating AWS instance-----------

	if *input.Type == "AWS" {
		if input.Zone == nil {
			return &model.VMInstanceMessage{}, fmt.Errorf("Zone Cannot be Empty")
		}

		if input.AccessKey == nil || input.SecretKey == nil || input.InstanceID == nil {
			return &model.VMInstanceMessage{}, fmt.Errorf(" AccessKey, SecretKey and Instance ID  Cannot be Empty")
		}

		sess, _ := session.NewSession(&aws.Config{
			Region: aws.String(*input.Zone),
			Credentials: credentials.NewStaticCredentials(
				*input.AccessKey,
				*input.SecretKey,
				"",
			),
		})

		result, err := service.GetInstances(sess)
		if err != nil {
			fmt.Println(err)
			return &model.VMInstanceMessage{}, fmt.Errorf(err.Error())
		}
		var status string

		for _, r := range result.Reservations {
			for _, i := range r.Instances {
				if *input.InstanceID == *i.InstanceId {
					status = *i.State.Name
					if *input.InstanceName != *i.Tags[0].Value {
						msg := "The given instance name is not match with the Instance Id"
						return &model.VMInstanceMessage{Message: &msg}, nil
					}
				}
			}
		}

		_, instanceIds, err = service.CreateHostAWSService(user.ID, status, *input)
		if err != nil {
			fmt.Println(err)
			log4go.Error("Module: CreateHost, MethodName: CreateHostAWSService, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, err
		}
		log4go.Info("Module: CreateHost, MethodName: CreateHostAWSService, Message: Host service is successfully created, user: %s", user.ID)

	}
	//--------------------Creating Azure instance-----------

	if *input.Type == "Azure" {
		if *input.SubscriptionID == "" || *input.ResourceGroupName == "" {
			return nil, fmt.Errorf("SubscriptionID and Resource Group Name Cannot be Empty")
		}

		statusAzure, err := service.AzureInstanceStatus(*input.SubscriptionID, *input.ResourceGroupName, *input.InstanceName, *input.ClientID, *input.ClientSecret, *input.TenantID)
		if err != nil {
			log4go.Error("Module: CreateHost, MethodName: AzureInstanceStatus, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, fmt.Errorf("unable to locate the virtual machine. Verify that the information provided is valid")
		}
		log4go.Info("Module: CreateHost, MethodName: AzureInstanceStatus, Message: Fetching the status of the given VM instance("+*input.InstanceName+"): "+statusAzure+", user: %s", user.ID)

		_, instanceIds, err = service.CreateHostAzureService(user.ID, statusAzure, *input)
		if err != nil {
			fmt.Println(err)
			log4go.Error("Module: CreateHost, MethodName: CreateHostAzureService, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, err
		}
		log4go.Info("Module: CreateHost, MethodName: CreateHostAzureService, Message: Host service is successfully created, user: %s", user.ID)

	}

	// getInst, err := service.GetHostDetailsByName(*input.InstanceName)
	// if err != nil {
	// 	return &model.VMInstanceMessage{}, err
	// }

	userDetAct, err := service.GetById(user.ID)
	if err != nil {
		log4go.Error("Module: CreateHost, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
		return &model.VMInstanceMessage{}, err
	}
	log4go.Info("Module: CreateHost, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

	AddOperation := service.Activity{
		Type:       "INSTANCE",
		UserId:     user.ID,
		Activities: "CREATED",
		Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Created a Instance " + *input.InstanceName + " in " + *input.Type,
		RefId:      *input.OrgID,
	}

	_, err = service.InsertActivity(AddOperation)
	if err != nil {
		fmt.Println(err)
		return &model.VMInstanceMessage{}, err
	}
	err = service.SendSlackNotification(user.ID, AddOperation.Message)
	if err != nil {
		log4go.Error("Module: CreateHost, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
	}

	msgSuccess := "Inserted Successfully"
	return &model.VMInstanceMessage{ID: &instanceIds, Message: &msgSuccess}, err
}

func (r *mutationResolver) DeleteHost(ctx context.Context, id *int) (string, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return "", fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return "", err
	}

	checkStatus, err := service.GetHostDetailsById(*id)
	if err != nil {
		fmt.Println(err)
		log4go.Error("Module: DeleteHost, MethodName: GetHostDetailsById, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	if *checkStatus.Status == "running" {
		return "The " + *checkStatus.InstanceName + " instance is in Running state, Terminate the instance and try to delete.", err
	}

	_, err = service.DeleteHostActivity(*id)
	if err != nil {
		fmt.Println(err)
		log4go.Error("Module: DeleteHost, MethodName: DeleteHostActivity, Message: %s user:%s", err.Error(), user.ID)
		return "", err
	}
	log4go.Info("Module: DeleteHost, MethodName: DeleteHostActivity, Message: Host activity is successfully deleted, user: %s", user.ID)

	return "Deleted Successfully", nil
}

func (r *mutationResolver) NodeActionAws(ctx context.Context, input *model.StartAndStopVM) (*model.VMInstanceMessage, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	getHostDet, err := service.GetHostDetails(user.ID, "")
	if err != nil {
		fmt.Println(err)
		log4go.Error("Module: NodeActionAws, MethodName: GetHostDetails, Message: %s user:%s", err.Error(), user.ID)
		return &model.VMInstanceMessage{}, err
	}
	log4go.Info("Module: NodeActionAws, MethodName: GetHostDetails, Message: Fetching host details based on user Id is successfully completed, user: %s", user.ID)

	var vmInst model.Host
	for _, i := range getHostDet {
		if *input.InstanceName == *i.InstanceName {
			vmInst.InstanceName = i.InstanceName
			vmInst.InstanceID = i.InstanceID
			vmInst.AccessKey = i.AccessKey
			vmInst.SecretKey = i.SecretKey
			vmInst.Zone = i.Zone
			vmInst.ID = i.ID
		}
	}

	CheckHost := model.Host{}

	if CheckHost == vmInst {
		return nil, fmt.Errorf("Cannot find the Instance, Try To create a New")
	}

	if (*input.Action != "start" && *input.Action != "stop") || *vmInst.InstanceID == "" {
		fmt.Println("You must supply a START or STOP state and an instance ID")
		return &model.VMInstanceMessage{}, err
	}

	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(*vmInst.Zone),
		Credentials: credentials.NewStaticCredentials(
			*vmInst.AccessKey,
			*vmInst.SecretKey,
			"",
		),
	})

	svc := ec2.New(sess)
	var status string

	if *input.Action == "start" {

		result, err := service.GetInstances(sess)
		if err != nil {
			fmt.Println("Got an error retrieving information about your Amazon EC2 instances:")
			fmt.Println(err)
			log4go.Error("Module: NodeActionAws, MethodName: GetInstances, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, err
		}
		log4go.Info("Module: NodeActionAws, MethodName: GetInstances, Message: Getting the status of the instance - %s , user: %s", vmInst.InstanceName, user.ID)

		for _, r := range result.Reservations {
			for _, i := range r.Instances {
				if *vmInst.InstanceID == *i.InstanceId {
					status = *i.State.Name
				}
			}
		}
		if status == "running" {
			service.UpdateHostStatus(*vmInst.ID, "running")
			return nil, fmt.Errorf("This Instance is already RUNNING")
		}

		err = service.StartInstance(svc, vmInst.InstanceID)
		if err != nil {
			fmt.Println("Got an error starting instance")
			fmt.Println(err)
			log4go.Error("Module: NodeActionAws, MethodName: StartInstance, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, err
		}
		log4go.Info("Module: NodeActionAws, MethodName: StartInstance, Message: %s instance is successfully started, user: %s", vmInst.InstanceName, user.ID)

		err = service.UpdateHostStatus(*vmInst.ID, "running")
		if err != nil {
			fmt.Println(err)
			log4go.Error("Module: NodeActionAws, MethodName: UpdateHostStatus, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: NodeActionAws, MethodName: UpdateHostStatus, Message: Updating the status of the instance to DB , user: %s", vmInst.InstanceName, user.ID)
		for {
			result, err := service.GetInstances(sess)
			if err != nil {
				fmt.Println("Got an error retrieving information about your Amazon EC2 instances:")
				fmt.Println(err)
				log4go.Error("Module: NodeActionAws, MethodName: GetInstances, Message: %s user:%s", err.Error(), user.ID)
				return &model.VMInstanceMessage{}, err
			}
			log4go.Info("Module: NodeActionAws, MethodName: GetInstances, Message: Getting the status of the instance - %s , user: %s", vmInst.InstanceName, user.ID)

			for _, r := range result.Reservations {
				for _, i := range r.Instances {
					if *vmInst.InstanceID == *i.InstanceId {
						status = *i.State.Name
					}
				}
			}
			if status == "running" {
				break
			}
		}
		userDetAct, err := service.GetById(user.ID)
		if err != nil {
			log4go.Error("Module: NodeActionAws, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: NodeActionAws, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

		AddOperation := service.Activity{
			Type:       "INSTANCE",
			UserId:     user.ID,
			Activities: "RUNNING",
			Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Started the Instance " + *input.InstanceName,
			RefId:      strconv.Itoa(*vmInst.ID),
		}

		_, err = service.InsertActivity(AddOperation)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		err = service.SendSlackNotification(user.ID, AddOperation.Message)
		if err != nil {
			log4go.Error("Module: NodeActionAws, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
		}

	} else if *input.Action == "stop" {

		result, err := service.GetInstances(sess)
		if err != nil {
			fmt.Println("Got an error retrieving information about your Amazon EC2 instances:")
			fmt.Println(err)
			log4go.Error("Module: NodeActionAws, MethodName: GetInstances, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, err
		}
		log4go.Info("Module: NodeActionAws, MethodName: GetInstances, Message: Getting the status of the instance - %s , user: %s", vmInst.InstanceName, user.ID)

		for _, r := range result.Reservations {
			for _, i := range r.Instances {
				if *vmInst.InstanceID == *i.InstanceId {
					status = *i.State.Name
				}
			}
		}
		if status == "stopped" {
			service.UpdateHostStatus(*vmInst.ID, "stopped")
			return nil, fmt.Errorf("This Instance is already TERMINATED")
		}

		err = service.StopInstance(svc, vmInst.InstanceID)
		if err != nil {
			fmt.Println("Got an error stopping the instance")
			fmt.Println(err)
			log4go.Error("Module: NodeActionAws, MethodName: StopInstance, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, err
		}
		log4go.Info("Module: NodeActionAws, MethodName: StopInstance, Message: %s instance is successfully stopped , user: %s", vmInst.InstanceName, user.ID)

		err = service.UpdateHostStatus(*vmInst.ID, "stopped")
		if err != nil {
			fmt.Println(err)
			log4go.Error("Module: NodeActionAws, MethodName: UpdateHostStatus, Message: %s user:%s", err.Error(), user.ID)
		}
		log4go.Info("Module: NodeActionAws, MethodName: UpdateHostStatus, Message: Updating the status of the instance to DB , user: %s", vmInst.InstanceName, user.ID)
		for {
			result, err := service.GetInstances(sess)
			if err != nil {
				fmt.Println("Got an error retrieving information about your Amazon EC2 instances:")
				fmt.Println(err)
				log4go.Error("Module: NodeActionAws, MethodName: GetInstances, Message: %s user:%s", err.Error(), user.ID)
				return &model.VMInstanceMessage{}, err
			}
			log4go.Info("Module: NodeActionAws, MethodName: GetInstances, Message: Getting the status of the instance - %s , user: %s", vmInst.InstanceName, user.ID)

			var status string

			for _, r := range result.Reservations {
				for _, i := range r.Instances {
					if *vmInst.InstanceID == *i.InstanceId {
						status = *i.State.Name
					}
				}
			}
			if status == "stopped" {
				break
			}
		}
		userDetAct, err := service.GetById(user.ID)
		if err != nil {
			log4go.Error("Module: NodeActionAws, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: NodeActionAws, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

		AddOperation := service.Activity{
			Type:       "INSTANCE",
			UserId:     user.ID,
			Activities: "TERMINATED",
			Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Stopped the Instance " + *input.InstanceName,
			RefId:      strconv.Itoa(*vmInst.ID),
		}

		_, err = service.InsertActivity(AddOperation)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		err = service.SendSlackNotification(user.ID, AddOperation.Message)
		if err != nil {
			log4go.Error("Module: NodeActionAws, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
		}

	}

	message := "Action Completed"

	return &model.VMInstanceMessage{Message: &message}, nil
}

func (r *mutationResolver) NodeActionAzure(ctx context.Context, input *model.StartAndStopVM) (*model.VMInstanceMessage, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}
	_, err := service.CheckUserRole(user.ID)
	if err != nil {
		return nil, err
	}

	if input.Action == nil {
		return nil, fmt.Errorf("You must supply a START or STOP state")
	}
	getHostDet, err := service.GetHostDetails(user.ID, "")
	if err != nil {
		fmt.Println(err)
		log4go.Error("Module: NodeActionAws, MethodName: GetHostDetails, Message: %s user:%s", err.Error(), user.ID)
		return &model.VMInstanceMessage{}, err
	}
	log4go.Info("Module: NodeActionAws, MethodName: GetHostDetails, Message: Fetching host details based on user Id is successfully completed, user: %s", user.ID)

	var vmInst model.Host
	for _, i := range getHostDet {
		if *input.InstanceName == *i.InstanceName {
			vmInst.ID = i.ID
			vmInst.InstanceName = i.InstanceName
			vmInst.SubscriptionID = i.SubscriptionID
			vmInst.ResourceGroupName = i.ResourceGroupName
			vmInst.ClientID = i.ClientID
			vmInst.ClientSecret = i.ClientSecret
			vmInst.TenantID = i.TenantID
		}
	}

	CheckHost := model.Host{}

	if CheckHost == vmInst {
		return nil, fmt.Errorf("Cannot find the Instance, Try To create a New")
	}

	if *input.Action != "start" && *input.Action != "stop" {
		fmt.Println("You must supply a START or STOP state")
		return &model.VMInstanceMessage{}, err
	}

	if *input.Action == "start" {
		vmStatus, err := service.AzureStartInstance(*vmInst.SubscriptionID, *vmInst.ResourceGroupName, *vmInst.InstanceName, *vmInst.ClientID, *vmInst.ClientSecret, *vmInst.TenantID, user.ID, *vmInst.ID)
		if err != nil {
			log4go.Error("Module: NodeActionAws, MethodName: AzureStartInstance, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, err
		}
		log4go.Info("Module: NodeActionAws, MethodName: AzureStartInstance, Message: Successfully started the VM instance: "+*vmInst.InstanceName+", user: %s", user.ID)

		err = service.UpdateHostStatus(*vmInst.ID, vmStatus)
		if err != nil {
			log4go.Error("Module: NodeActionAws, MethodName: UpdateHostStatus, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, err
		}
		log4go.Info("Module: NodeActionAws, MethodName: UpdateHostStatus, Message: Updating the status of the instance to DB , user: %s", vmInst.InstanceName, user.ID)

		userDetAct, err := service.GetById(user.ID)
		if err != nil {
			log4go.Error("Module: NodeAction, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: NodeAction, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

		AddOperation := service.Activity{
			Type:       "INSTANCE",
			UserId:     user.ID,
			Activities: "RUNNING",
			Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Started the Instance " + *input.InstanceName,
			RefId:      strconv.Itoa(*vmInst.ID),
		}

		_, err = service.InsertActivity(AddOperation)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		err = service.SendSlackNotification(user.ID, AddOperation.Message)
		if err != nil {
			log4go.Error("Module: NodeAction, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
		}

	} else if *input.Action == "stop" {
		vmStatus, err := service.AzureStopInstance(*vmInst.SubscriptionID, *vmInst.ResourceGroupName, *vmInst.InstanceName, *vmInst.ClientID, *vmInst.ClientSecret, *vmInst.TenantID, user.ID, *vmInst.ID)
		if err != nil {
			log4go.Error("Module: NodeActionAws, MethodName: AzureStopInstance, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, err
		}
		log4go.Info("Module: NodeActionAws, MethodName: AzureStopInstance, Message: Successfully stopped the VM instance: "+*vmInst.InstanceName+", user: %s", user.ID)

		err = service.UpdateHostStatus(*vmInst.ID, vmStatus)
		if err != nil {
			log4go.Error("Module: NodeActionAws, MethodName: UpdateHostStatus, Message: %s user:%s", err.Error(), user.ID)
			return &model.VMInstanceMessage{}, err
		}
		log4go.Info("Module: NodeActionAws, MethodName: UpdateHostStatus, Message: Updating the status of the instance to DB , user: %s", vmInst.InstanceName, user.ID)

		userDetAct, err := service.GetById(user.ID)
		if err != nil {
			log4go.Error("Module: NodeAction, MethodName: GetById, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: NodeAction, MethodName: GetById, Message: Get user details for activity table by user: %s", user.ID)

		AddOperation := service.Activity{
			Type:       "INSTANCE",
			UserId:     user.ID,
			Activities: "TERMINATED",
			Message:    *userDetAct.FirstName + " " + *userDetAct.LastName + " has Stopped the Instance " + *input.InstanceName,
			RefId:      strconv.Itoa(*vmInst.ID),
		}

		_, err = service.InsertActivity(AddOperation)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		err = service.SendSlackNotification(user.ID, AddOperation.Message)
		if err != nil {
			log4go.Error("Module: NodeAction, MethodName: SendSlackNotification, Message: %s user:%s", err.Error(), user.ID)
		}
	}
	message := "Action Completed"
	return &model.VMInstanceMessage{Message: &message}, nil
}

func (r *queryResolver) GetHost(ctx context.Context, orgID *string) ([]*model.HostPayload, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	getHostDet, err := service.GetHostDetails(user.ID, *orgID)

	if err != nil {
		fmt.Println(err)
		log4go.Error("Module: GetHost, MethodName: GetHostDetails, Message: %s user:%s", err.Error(), user.ID)
		return nil, err
	}
	log4go.Info("Module: GetHost, MethodName: GetHostDetails, Message: Fetching host details is successfully completed, user: %s", user.ID)

	if len(getHostDet) == 0 {
		return nil, fmt.Errorf("There is No Instance for this User")
	}

	var hostDetails []*model.HostPayload

	for _, i := range getHostDet {

		hostActivityDet, err := service.GetHostActivity(*i.ID)
		if err != nil {
			fmt.Println(err)
			log4go.Error("Module: GetHost, MethodName: GetHostActivity, Message: %s user:%s", err.Error(), user.ID)
			return nil, err
		}
		log4go.Info("Module: GetHost, MethodName: GetHostActivity, Message: Fetching host activity is successfully completed, user: %s", user.ID)

		orgName, err := service.GetOrgNameById(*i.OrgID)

		result := model.HostPayload{
			ID:                i.ID,
			OrgID:             i.OrgID,
			OrgName:           &orgName,
			Type:              i.Type,
			ServiceAccountURL: i.ServiceAccountURL,
			Status:            i.Status,
			Zone:              i.Zone,
			InstanceName:      i.InstanceName,
			InstanceID:        i.InstanceID,
			AccessKey:         i.AccessKey,
			SecretKey:         i.SecretKey,
			SubscriptionID:    i.SubscriptionID,
			ResourceGroupName: i.ResourceGroupName,
			ClientID:          i.ClientID,
			ClientSecret:      i.ClientSecret,
			TenantID:          i.TenantID,
			CreatedAt:         i.CreatedAt,
			InstanceActivity:  hostActivityDet,
		}

		hostDetails = append(hostDetails, &result)

	}

	return hostDetails, err
}

func (r *queryResolver) GetHostByName(ctx context.Context, instanceName *string) (*model.HostDetails, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, fmt.Errorf("Access Denied")
	}

	instanceDet, err := service.GetHostDetailsByName(*instanceName)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &model.HostDetails{ID: instanceDet.ID, OrgID: instanceDet.OrgID, Type: instanceDet.Type, ServiceAccountURL: instanceDet.ServiceAccountURL, Status: instanceDet.Status, Zone: instanceDet.Zone, InstanceName: instanceDet.InstanceName, InstanceID: instanceDet.InstanceID, AccessKey: instanceDet.AccessKey, SecretKey: instanceDet.SecretKey, CreatedBy: instanceDet.CreatedBy, CreatedAt: instanceDet.CreatedAt, IsActive: instanceDet.IsActive}, nil
}
