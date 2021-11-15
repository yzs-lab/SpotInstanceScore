package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"log"
)

var client *ec2.Client
var azID map[string]string

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		log.Fatal(err)
	}
	client = ec2.NewFromConfig(cfg)


	azID = make(map[string]string)
	regionNames := []string{"us-east-1", "us-east-2", "us-west-1", "us-west-2"}
	for _, region := range regionNames {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
		c := ec2.NewFromConfig(cfg)
		input := ec2.DescribeAvailabilityZonesInput{

		}
		output, err := c.DescribeAvailabilityZones(context.TODO(), &input)
		if err != nil {
			panic(err)
		}
		for _, az := range output.AvailabilityZones {
			azID[aws.ToString(az.ZoneId)] = aws.ToString(az.ZoneName)
		}
	}
	fmt.Println(len(azID))
}

type spotPlacementScoresCase struct {
	instanceTypes 			[]string
	regionNames 			[]string
	targetCapacity 			int32
	singleAvailabilityZone 	bool

	name 					string
}
var spotPlacementScoresCases []spotPlacementScoresCase

func querySpotPlacementScores(targetCapacity int32, instanceTypes []string, regionNames []string, singleAvailabilityZone bool) {
	input := ec2.GetSpotPlacementScoresInput{
		TargetCapacity:                   aws.Int32(targetCapacity),
		InstanceTypes:                    instanceTypes,
		RegionNames:                      regionNames,
		SingleAvailabilityZone:           aws.Bool(singleAvailabilityZone),
	}
	output, err := client.GetSpotPlacementScores(context.TODO(), &input)
	if err != nil {
		panic(err)
	}

	for _, result := range output.SpotPlacementScores {
		fmt.Printf("%s %s %s %d\n", aws.ToString(result.Region), aws.ToString(result.AvailabilityZoneId), azID[aws.ToString(result.AvailabilityZoneId)], aws.ToInt32(result.Score))
	}
}


func main() {
	spotPlacementScoresCases = []spotPlacementScoresCase{
		{
			instanceTypes: []string{"p3.2xlarge", "p3.8xlarge", "p3.16xlarge"},
			regionNames: []string{"us-east-1", "us-east-2", "us-west-1", "us-west-2"},
			targetCapacity: int32(1),
			singleAvailabilityZone: true,
			name: "all GPU instance",
		},
		{
			instanceTypes: []string{"p3.2xlarge"},
			regionNames: []string{"us-east-1", "us-east-2", "us-west-1", "us-west-2"},
			targetCapacity: int32(1),
			singleAvailabilityZone: true,
			name: "p3.2xlarge",
		},
	}

	for _, spsCase := range spotPlacementScoresCases {
		fmt.Println(spsCase.name)
		querySpotPlacementScores(spsCase.targetCapacity, spsCase.instanceTypes, spsCase.regionNames, spsCase.singleAvailabilityZone)
	}
}
