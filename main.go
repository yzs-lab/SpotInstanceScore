package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"log"
	"os"
	"strings"
	"time"
)

var client *ec2.Client
var s3client *s3.Client
var azID map[string]string
var now string
var outputFileStr string

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		log.Fatal(err)
	}
	client = ec2.NewFromConfig(cfg)
	s3client = s3.NewFromConfig(cfg)


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

	now = fmt.Sprint(time.Now().Format(time.RFC3339))
	outputFileStr = "time,region,instance,azID,azName,capacity,single,score\n"
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
		// outputFileStr = "time,region,instance,azID,azName,capacity,single,score\n"
		s := fmt.Sprintf("%s,%s,%s,%s,%s,%d,%v,%d\n",
			now,
			aws.ToString(result.Region), strings.Join(instanceTypes, "+"),
			aws.ToString(result.AvailabilityZoneId), azID[aws.ToString(result.AvailabilityZoneId)],
			targetCapacity, singleAvailabilityZone, aws.ToInt32(result.Score))
		outputFileStr += s
	}
}

// var GPUInstanceTypes = []string{"p2.xlarge", "p2.8xlarge", "p2.16xlarge", "p3.2xlarge", "p3.8xlarge", "p3.16xlarge", "p3dn.24xlarge", "p4d.24xlarge"}


var GPUInstanceTypes = []string{"p2.xlarge", "p2.8xlarge", "p3.2xlarge", "p3.8xlarge"}
var regionNames = []string{"us-east-1", "us-east-2", "us-west-1", "us-west-2"}
var singleAvailabilityZoneRange = []bool{true, false}
var targetCapacityRange = []int32{1, 2, 4}

func saveResult() {
	filename := now + ".csv"
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	_, err = file.WriteString(outputFileStr)
	if err != nil {
		panic(err)
	}


	uploader := manager.NewUploader(s3client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("bucket")),
		Key:    aws.String(filename),
		Body:   strings.NewReader(outputFileStr),
	})
	if err != nil {
		panic(err)
	}
}

func main() {
	spotPlacementScoresCases = []spotPlacementScoresCase{
		{
			instanceTypes: GPUInstanceTypes,
			regionNames: regionNames,
			targetCapacity: int32(1),
			singleAvailabilityZone: true,
			name: "all_1_true",
		},
		{
			instanceTypes: GPUInstanceTypes,
			regionNames: regionNames,
			targetCapacity: int32(2),
			singleAvailabilityZone: true,
			name: "all_2_true",
		},
		{
			instanceTypes: GPUInstanceTypes,
			regionNames: regionNames,
			targetCapacity: int32(4),
			singleAvailabilityZone: true,
			name: "all_4_true",
		},
	}

	for _, instance := range GPUInstanceTypes {
		for _, single := range singleAvailabilityZoneRange {
			for _, capacity := range targetCapacityRange {
				spotPlacementScoresCases = append(spotPlacementScoresCases, spotPlacementScoresCase{
					instanceTypes:          []string{instance},
					regionNames:            regionNames,
					targetCapacity:         capacity,
					singleAvailabilityZone: single,
					name:                   fmt.Sprintf("%s_%d_%v", instance, capacity, single),
				})
			}
		}
	}

	for _, spsCase := range spotPlacementScoresCases {
		fmt.Println(spsCase.name)
		querySpotPlacementScores(spsCase.targetCapacity, spsCase.instanceTypes, spsCase.regionNames, spsCase.singleAvailabilityZone)
	}

	saveResult()
}
