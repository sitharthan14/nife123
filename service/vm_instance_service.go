package service

import (
	"context"
	"fmt"
	"log"
	"time"

	computeAzure "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-07-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/alecthomas/log4go"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

func CreateHostService(userId, status string, input model.Host) (string, int, error) {
	statement, err := database.Db.Prepare("INSERT INTO host (org_id, type, service_account_url, status, zone, instance_name, created_by, created_at) VALUES (?,?,?,?,?,?,?,?)")
	if err != nil {
		return "", 0, err
	}
	defer statement.Close()
	res, err := statement.Exec(input.OrgID, input.Type, input.ServiceAccountURL, status, input.Zone, input.InstanceName, userId, time.Now())
	if err != nil {
		return "", 0, err
	}
	idInstance, err := res.LastInsertId()
	return "", int(idInstance), err
}

func DeleteHostActivity(id int) (string, error) {
	statement, err := database.Db.Prepare(`UPDATE host SET is_active = 0  WHERE id = ?`)
	if err != nil {
		return "", err
	}
	defer statement.Close()
	_, err = statement.Exec(&id)
	if err != nil {
		return "", err
	}
	return "", err
}

func GetInstanceStatus(Project, zone, instanceName string) (string, error) {

	ctx := context.Background()
	computeService, err := compute.NewService(ctx, option.WithCredentialsFile("temp.json"))

	if err != nil {
		panic(err)
	}

	cic := computeService.Instances.List(Project, zone).Filter("name=" + instanceName)
	computeList, err := cic.Do()
	if err != nil {
		return "", err
	}

	return computeList.Items[0].Status, err
}

func CheckStringAlphabet(str string) error {
	for _, charVariable := range str {
		if (charVariable < 'a' || charVariable > 'z') && (charVariable < 'A' || charVariable > 'Z') {
			return fmt.Errorf("The Instance Name and Zone field Should Not Contains Empty Space or Special Character")
		}
	}
	return nil
}

func GetHostDetails(userId, orgId string) ([]*model.Host, error) {
	var query string
	if orgId != "" {
		query = "select id, org_id, type, service_account_url, status, zone, instance_name, instance_id, access_key, secret_key, subscription_id, resource_group_name, client_id, client_secret, tenant_id, created_at from host where  is_active = 1 and org_id = " + `"` + orgId + `"`
	} else {
		query = "select id, org_id, type, service_account_url, status, zone, instance_name, instance_id, access_key, secret_key, subscription_id, resource_group_name, client_id, client_secret, tenant_id, created_at from host where is_active = 1 and created_by =" + userId
	}

	selDB, err := database.Db.Query(query)
	if err != nil {
		log.Println(err)
		return []*model.Host{}, err
	}
	result := []*model.Host{}
	defer selDB.Close()
	for selDB.Next() {

		var host model.Host
		err = selDB.Scan(&host.ID, &host.OrgID, &host.Type, &host.ServiceAccountURL, &host.Status, &host.Zone, &host.InstanceName, &host.InstanceID, &host.AccessKey, &host.SecretKey, &host.SubscriptionID, &host.ResourceGroupName, &host.ClientID, &host.ClientSecret, &host.TenantID, &host.CreatedAt)
		if err != nil {
			return []*model.Host{}, err
		}
		result = append(result, &host)
	}
	return result, nil
}

func GetHostDetailsByName(name string) (*model.HostDetails, error) {

	query := "select id, org_id, type, service_account_url, status, zone, instance_name, instance_id, access_key, secret_key, created_at, is_active from host where is_active = 1 and instance_name = ?"

	selDB, err := database.Db.Query(query, name)
	if err != nil {
		log.Println(err)
		return &model.HostDetails{}, err
	}
	var host model.HostDetails
	defer selDB.Close()
	for selDB.Next() {

		err = selDB.Scan(&host.ID, &host.OrgID, &host.Type, &host.ServiceAccountURL, &host.Status, &host.Zone, &host.InstanceName, &host.InstanceID, &host.AccessKey, &host.SecretKey, &host.CreatedAt, &host.IsActive)
		if err != nil {
			return &model.HostDetails{}, err
		}
	}
	return &host, nil
}

func GetHostDetailsById(id int) (*model.HostDetails, error) {

	query := "select id, org_id, type, service_account_url, status, zone, instance_name, instance_id, access_key, secret_key, created_at, is_active from host where status = ? and id = ?"

	selDB, err := database.Db.Query(query, "running", id)
	if err != nil {
		log.Println(err)
		return &model.HostDetails{}, err
	}
	var host model.HostDetails
	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&host.ID, &host.OrgID, &host.Type, &host.ServiceAccountURL, &host.Status, &host.Zone, &host.InstanceName, &host.InstanceID, &host.AccessKey, &host.SecretKey, &host.CreatedAt, &host.IsActive)
	if err != nil {
		return &model.HostDetails{}, err
	}

	return &host, nil
}

func GetHostActivity(hostId int) ([]*model.Activity, error) {

	query := "select id, type, activities, message ,created_at from activity where ref_id = ? and type = ? order by created_at desc"

	selDB, err := database.Db.Query(query, hostId, "HOST")
	if err != nil {
		log.Println(err)
		return []*model.Activity{}, err
	}
	result := []*model.Activity{}
	defer selDB.Close()

	for selDB.Next() {
		var host model.Activity

		err = selDB.Scan(&host.ID, &host.Type, &host.Activities, &host.Message, &host.CreatedAt)
		if err != nil {
			return []*model.Activity{}, err
		}
		result = append(result, &host)
	}
	return result, nil
}

func UpdateHostStatus(id int, status string) error {
	statement, err := database.Db.Prepare("Update host set status = ?  where id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(status, id)
	if err != nil {
		return err
	}

	return nil
}

func CreateHostAWSService(userId, status string, input model.Host) (string, int, error) {
	statement, err := database.Db.Prepare("INSERT INTO host (org_id, type, status, zone, instance_name, instance_id, access_key, secret_key, created_by, created_at) VALUES (?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return "", 0, err
	}
	defer statement.Close()
	res, err := statement.Exec(input.OrgID, input.Type, status, input.Zone, input.InstanceName, input.InstanceID, input.AccessKey, input.SecretKey, userId, time.Now())
	if err != nil {
		return "", 0, err
	}
	idInstance, err := res.LastInsertId()

	return "", int(idInstance), err
}

func CreateHostAzureService(userId, status string, input model.Host) (string, int, error) {
	statement, err := database.Db.Prepare("INSERT INTO host (org_id, type, status, zone, instance_name, subscription_id, resource_group_name, client_id, client_secret, tenant_id, created_by, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return "", 0, err
	}
	defer statement.Close()
	res, err := statement.Exec(input.OrgID, input.Type, status, input.Zone, input.InstanceName, input.SubscriptionID, input.ResourceGroupName, input.ClientID, input.ClientSecret, input.TenantID, userId, time.Now())
	if err != nil {
		return "", 0, err
	}
	idInstance, err := res.LastInsertId()
	return "", int(idInstance), err
}

func GetInstances(sess *session.Session) (*ec2.DescribeInstancesOutput, error) {
	svc := ec2.New(sess)
	result, err := svc.DescribeInstances(nil)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// StartInstance starts an Amazon EC2 instance.
// Inputs:
//     svc is an Amazon EC2 service client
//     instanceID is the ID of the instance
// Output:
//     If success, nil
//     Otherwise, an error from the call to StartInstances
func StartInstance(svc ec2iface.EC2API, instanceID *string) error {
	input := &ec2.StartInstancesInput{
		InstanceIds: []*string{
			instanceID,
		},
		DryRun: aws.Bool(true),
	}
	_, err := svc.StartInstances(input)
	awsErr, ok := err.(awserr.Error)

	if ok && awsErr.Code() == "DryRunOperation" {
		// Set DryRun to be false to enable starting the instances
		input.DryRun = aws.Bool(false)
		_, err = svc.StartInstances(input)
		if err != nil {
			return err
		}

		return nil
	}

	return err
}

// StopInstance stops an Amazon EC2 instance.
// Inputs:
//     svc is an Amazon EC2 service client
//     instance ID is the ID of the instance
// Output:
//     If success, nil
//     Otherwise, an error from the call to StopInstances
func StopInstance(svc ec2iface.EC2API, instanceID *string) error {
	input := &ec2.StopInstancesInput{
		InstanceIds: []*string{
			instanceID,
		},
		DryRun: aws.Bool(true),
	}
	_, err := svc.StopInstances(input)
	awsErr, ok := err.(awserr.Error)
	if ok && awsErr.Code() == "DryRunOperation" {
		input.DryRun = aws.Bool(false)
		_, err = svc.StopInstances(input)
		if err != nil {
			return err
		}

		return nil
	}

	return err
}

func GetHostDetailsByNameAndOrgId(name, cloudType, userId, OrgId string) (*model.HostDetails, error) {

	query := `select id, org_id, type, service_account_url, status, zone, instance_name, instance_id, access_key, secret_key, created_at, is_active from host
	where (is_active = 1 and instance_name = ?) and (created_by = ? and type = ?) and org_id = ? `

	selDB, err := database.Db.Query(query, name, userId, cloudType, OrgId)
	if err != nil {
		log.Println(err)
		return &model.HostDetails{}, err
	}
	var host model.HostDetails
	defer selDB.Close()
	for selDB.Next() {

		err = selDB.Scan(&host.ID, &host.OrgID, &host.Type, &host.ServiceAccountURL, &host.Status, &host.Zone, &host.InstanceName, &host.InstanceID, &host.AccessKey, &host.SecretKey, &host.CreatedAt, &host.IsActive)
		if err != nil {
			return &model.HostDetails{}, err
		}
	}
	return &host, nil
}

func AzureInstanceStatus(subscriptionId, resourceGroupName, vmName, clientId, clientSecret, tenantId string) (string, error) {
	config := auth.NewClientCredentialsConfig(clientId, clientSecret, tenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		log.Println(err)
		return "", err
	}

	vmClient := computeAzure.NewVirtualMachinesClient(subscriptionId) // Create a new compute client.
	vmClient.Authorizer = authorizer

	vmStatus, err := vmClient.InstanceView(context.Background(), resourceGroupName, vmName) // Get the virtual machine instance status.
	if err != nil {
		fmt.Println("Failed to get virtual machine instance status:", err)
		return "", err
	}

	var status string
	for _, r := range *vmStatus.Statuses {
		status = string(*r.DisplayStatus)
		if status == "VM stopped" {
			status = "stopped"
			break
		} else if status == "VM running" {
			status = "running"
			break
		}
	}

	return status, nil
}

func AzureStartInstance(subscriptionId, resourceGroupName, vmName, clientId, clientSecret, tenantId, userId string, instanceId int) (string, error) {

	config := auth.NewClientCredentialsConfig(clientId, clientSecret, tenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		log.Println(err)
		return "", err
	}

	vmStatus, err := AzureInstanceStatus(subscriptionId, resourceGroupName, vmName, clientId, clientSecret, tenantId)
	if err != nil {
		log4go.Error("Module: AzureStartInstance, MethodName: AzureInstanceStatus, Message: %s user:%s", err.Error(), userId)
		return "", err
	}
	log4go.Info("Module: AzureStartInstance, MethodName: AzureInstanceStatus, Message: Fetching status of the instance: "+vmName+", user: %s", userId)

	if vmStatus == "running" {
		UpdateHostStatus(instanceId, "running")
		return "", fmt.Errorf("This instance is already in " + vmStatus + " state")
	}

	// Create a new compute client.
	vmClient := computeAzure.NewVirtualMachinesClient(subscriptionId)
	vmClient.Authorizer = authorizer

	// Start the virtual machine instance.
	_, err = vmClient.Start(context.Background(), resourceGroupName, vmName)
	if err != nil {
		log4go.Error("Module: AzureStartInstance, MethodName: Start, Message: Failed to start virtual machine instance - %s user:%s", err.Error(), userId)
		return "", err
	}
	log4go.Info("Module: AzureStartInstance, MethodName: Start, Message: Successfully started the instance: "+vmName+", user: %s", userId)

	var vmActionStatus string
	for {
		vmActionStatus, err = AzureInstanceStatus(subscriptionId, resourceGroupName, vmName, clientId, clientSecret, tenantId)
		if err != nil {
			log4go.Error("Module: AzureStartInstance, MethodName: AzureInstanceStatus, Message: %s user:%s", err.Error(), userId)
			return "", err
		}
		if vmActionStatus == "running" {
			break
		}
	}
	log4go.Info("Module: AzureStartInstance, MethodName: AzureInstanceStatus, Message: Fetching status of the instance: "+vmName+", user: %s", userId)

	return vmActionStatus, nil
}

func AzureStopInstance(subscriptionId, resourceGroupName, vmName, clientId, clientSecret, tenantId, userId string, instanceId int) (string, error) {

	config := auth.NewClientCredentialsConfig(clientId, clientSecret, tenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		log.Println(err)
		return "", err
	}

	vmStatus, err := AzureInstanceStatus(subscriptionId, resourceGroupName, vmName, clientId, clientSecret, tenantId)
	if err != nil {
		log4go.Error("Module: AzureStopInstance, MethodName: AzureInstanceStatus, Message: %s user:%s", err.Error(), userId)
		return "", err
	}
	log4go.Info("Module: AzureStopInstance, MethodName: AzureInstanceStatus, Message: Fetching status of the instance: "+vmName+", user: %s", userId)

	if vmStatus == "stopped" {
		UpdateHostStatus(instanceId, "stopped")
		return "", fmt.Errorf("This instance is already in " + vmStatus + " state")
	}

	// Create a new compute client.
	vmClient := computeAzure.NewVirtualMachinesClient(subscriptionId)
	vmClient.Authorizer = authorizer

	// Start the virtual machine instance.
	skipShutdown := false
	_, err = vmClient.PowerOff(context.Background(), resourceGroupName, vmName, &skipShutdown)
	if err != nil {
		log4go.Error("Module: AzureStopInstance, MethodName: PowerOff, Message: Failed to stop virtual machine instance - %s user:%s", err.Error(), userId)
		return "", err
	}
	log4go.Info("Module: AzureStopInstance, MethodName: PowerOff, Message: Successfully stopped the instance: "+vmName+", user: %s", userId)

	var vmActionStatus string
	for {
		vmActionStatus, err = AzureInstanceStatus(subscriptionId, resourceGroupName, vmName, clientId, clientSecret, tenantId)
		if err != nil {
			log4go.Error("Module: AzureStopInstance, MethodName: AzureInstanceStatus, Message: %s user:%s", err.Error(), userId)
			return "", err
		}
		if vmActionStatus == "stopped" {
			break
		}
	}
	log4go.Info("Module: AzureStopInstance, MethodName: AzureInstanceStatus, Message: Fetching status of the instance: "+vmName+", user: %s", userId)

	return vmActionStatus, nil
}



func UpdateInstanceOrganization(instanceId int, orgId string) error {
	statement, err := database.Db.Prepare("Update host set org_id = ?  where id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(orgId, instanceId)
	if err != nil {
		return err
	}

	return nil
}