package aws

import (
	"context"
	"fmt"
	"log"

	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"

	// "github.com/aws/aws-sdk-go/service/elb"

	appDeployments "github.com/nifetency/nife.io/internal/app_deployments"
)

func CreateOrDeleteRecordSetRoute53(name, elbURL, countryCode, cloudType string, isDelete bool, routingPolicy string) (string, string, string, error) {

	if cloudType == "gcp" {

		return createDNSRecordForGCP(routingPolicy, name, elbURL, countryCode, cloudType, isDelete)
	}
	prefixURL := "dualstack."
	hostedId := os.Getenv("HOSTED_ZONE_ID")

	ELBNames := make([]string, 0)
	ELBName, region, err := getDNSNameWithRegion(elbURL)
	DNSName := prefixURL + elbURL
	if err != nil {
		return "", "", "", err
	}
	ELBNames = append(ELBNames, ELBName)
	identifier := strings.Split(name, ".")[0] + "-" + countryCode

	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", "", "", err
	}

	elbClient := elasticloadbalancing.NewFromConfig(cfg, func(o *elasticloadbalancing.Options) {
		o.Region = region
	})

	elbresult, err := elbClient.DescribeLoadBalancers(context.TODO(), &elasticloadbalancing.DescribeLoadBalancersInput{LoadBalancerNames: ELBNames})

	if err != nil {
		return "", "", "", err
	}

	LBnamewithDNS := elbresult.LoadBalancerDescriptions[0].DNSName
	LBname := strings.Split(*LBnamewithDNS, "-")
	fmt.Println("LBName", LBname[0])

	TimeInSeconds := os.Getenv("AWS_ELB_TIMEOUT_SECONDS")

	idealTime, err := strconv.Atoi(TimeInSeconds)

	if err != nil {
		return "", "", "", err
	}	

	svc := elb.New(session.New(&aws.Config{Region: aws.String(region)}))
	input := &elb.ModifyLoadBalancerAttributesInput{
		LoadBalancerAttributes: &elb.LoadBalancerAttributes{
			ConnectionSettings: &elb.ConnectionSettings{
				IdleTimeout: aws.Int64(int64(idealTime)),
				
			},
		},
		LoadBalancerName: aws.String(LBname[0]),
	}

	_, err = svc.ModifyLoadBalancerAttributes(input)

	if err != nil {
		log.Println(err)
		return "", "", "", err
	}
	// Create an Amazon route53 service client
	client := route53.NewFromConfig(cfg)

	// CREATE RECORD SETS
	actionType := types.ChangeActionCreate
	if isDelete {
		actionType = types.ChangeActionDelete
	}
	resourceRecordSet, err := FormResourceRecordSet(routingPolicy, DNSName, name, elbresult, region, identifier, countryCode)
	if err != nil {
		return "", "", "", err
	}

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action:            actionType,
					ResourceRecordSet: &resourceRecordSet,
				},
			},
		},
		HostedZoneId: &hostedId, // Required
	}

	fmt.Println(params)
	_, err = client.ChangeResourceRecordSets(context.TODO(), params)
	if err != nil {
		return "", "", "", err
	}

	return name, DNSName, *elbresult.LoadBalancerDescriptions[0].CanonicalHostedZoneNameID, err
}

func DeleteDNSRecordBatch(ads *[]appDeployments.AppDeployments) error {

	var changes []types.Change
	hostedId := os.Getenv("HOSTED_ZONE_ID")
	prefixURL := "dualstack."
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}
	// Create an Amazon route53 service client
	client := route53.NewFromConfig(cfg)
	for _, ad := range *ads {

		deployment := ad
		regionCode := ""

		dnsName := prefixURL + deployment.App_Url
		_, region, _ := getDNSNameWithRegion(deployment.App_Url)
		if deployment.Region_code == "IND" {
			regionCode = "AS"
		} else if deployment.Region_code == "EUR" || deployment.Region_code == "EUR-3" {
			regionCode = "EU"
		} else {
			regionCode = "NA"
		}
		identifier := strings.Split(deployment.ELBRecordName, ".")[0] + "-" + regionCode

		copyIdentifier := identifier

		fmt.Println(copyIdentifier)
		change := types.Change{
			Action: types.ChangeActionDelete,
			ResourceRecordSet: &types.ResourceRecordSet{
				Name: &deployment.ELBRecordName, // Required
				Type: types.RRTypeA,
				AliasTarget: &types.AliasTarget{
					DNSName:              &dnsName,
					HostedZoneId:         &deployment.ELBRecordId,
					EvaluateTargetHealth: true,
				},
				Region:        types.ResourceRecordSetRegion(region),
				SetIdentifier: &copyIdentifier,
			},
		}
		changes = append(changes, change)
	}
	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &types.ChangeBatch{
			Changes: changes,
		},
		HostedZoneId: &hostedId, // Required
	}

	_, err = client.ChangeResourceRecordSets(context.TODO(), params)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil

}

func getDNSNameWithRegion(elbURL string) (string, string, error) {
	str := strings.Split(elbURL, ".")
	DNSName := strings.Split(str[0], "-")
	return DNSName[0], str[1], nil
}

func createDNSRecordForGCP(routingPolicy string, name, elbURL, countryCode, cloudType string, isDelete bool) (string, string, string, error) {
	hostedId := os.Getenv("HOSTED_ZONE_ID")
	identifier := strings.Split(name, ".")[0] + "-" + countryCode
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", "", "", err
	}
	// Create an Amazon route53 service client
	client := route53.NewFromConfig(cfg)
	// CREATE RECORD SETS
	actionType := types.ChangeActionCreate
	if isDelete {
		actionType = types.ChangeActionDelete
	}
	resourceRecordSetforGCP, err := FormResourceRecordSetForGCP(routingPolicy, elbURL, name, identifier, "NA")

	if err != nil {
		return "", "", "", err
	}
	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action:            actionType,
					ResourceRecordSet: &resourceRecordSetforGCP,
				},
			},
		},
		HostedZoneId: &hostedId, // Required
	}

	_, err = client.ChangeResourceRecordSets(context.TODO(), params)
	if err != nil {
		return "", "", "", err
	}

	return name, "", "", err
}

func FormResourceRecordSet(routingPolicy string, DNSName string, Rname string, elbresult *elasticloadbalancing.DescribeLoadBalancersOutput, region, identifier string, countryCode string) (types.ResourceRecordSet, error) {

	if routingPolicy == "Geolocation" {
		return types.ResourceRecordSet{
			Name: &Rname, // Required
			Type: types.RRTypeA,
			AliasTarget: &types.AliasTarget{
				DNSName:              &DNSName,
				HostedZoneId:         elbresult.LoadBalancerDescriptions[0].CanonicalHostedZoneNameID,
				EvaluateTargetHealth: true,
			},
			GeoLocation: &types.GeoLocation{
				ContinentCode: &countryCode,
			},
			SetIdentifier: &identifier,
		}, nil
	}

	return types.ResourceRecordSet{
		Name: &Rname, // Required
		Type: types.RRTypeA,
		AliasTarget: &types.AliasTarget{
			DNSName:              &DNSName,
			HostedZoneId:         elbresult.LoadBalancerDescriptions[0].CanonicalHostedZoneNameID,
			EvaluateTargetHealth: true,
		},
		Region:        types.ResourceRecordSetRegion(region),
		SetIdentifier: &identifier,
	}, nil

}

func FormResourceRecordSetForGCP(routingPolicy string, elbURL string, Rname string, identifier string, countryCode string) (types.ResourceRecordSet, error) {

	if routingPolicy == "Geolocation" {
		return types.ResourceRecordSet{
			Name: &Rname, // Required
			Type: types.RRTypeA,
			ResourceRecords: []types.ResourceRecord{
				{
					Value: aws.String(elbURL),
				},
			},
			TTL: aws.Int64(60),
			GeoLocation: &types.GeoLocation{
				ContinentCode: &countryCode,
			},
			SetIdentifier: &identifier,
		}, nil
	}

	return types.ResourceRecordSet{
		Name: &Rname, // Required
		Type: types.RRTypeA,
		ResourceRecords: []types.ResourceRecord{
			{
				Value: aws.String(elbURL),
			},
		},
		TTL:           aws.Int64(60),
		Region:        types.ResourceRecordSetRegion("us-east-2"),
		SetIdentifier: &identifier,
	}, nil

}
