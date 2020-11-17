package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"time"
)

var IAM_FLEET_ROLE string = "arn:aws:iam::965630252549:role/aws-ec2-spot-fleet-tagging-role"

var DEFAULT_AVAILABILITY_ZONE string = ""
var DEFAULT_IMAGE string = "ubuntu/images/hvm-ssd/ubuntu-xenial-16.04-amd64-server-20181114"
var DEFAULT_KEY string = "default"
var DEFAULT_PLACEMENT_GROUP string = ""
var DEFAULT_PRICE float64 = 1
var DEFAULT_REGION string = "ap-southeast-2"
var DEFAULT_REPLACE bool = false
var DEFAULT_SECGROUP string = "sg-0e9b9bbee1dfc700a"
var DEFAULT_SIZE int64 = 1
var DEFAULT_TIME string = "1h"
var DEFAULT_TYPE string = "c5.large"
var DEFAULT_USER string = "ubuntu"

var optionAvailabilityZone *string
var optionImage *string
var optionKey *string
var optionPlacementGroup *string
var optionPrice *float64
var optionRegion *string
var optionReplace *bool
var optionSecgroup *string
var optionSize *int64
var optionTime *string
var optionType *string
var optionUser *string

var launchProcOptionTime *Timeout

func PrintLaunchUsage() {
	fmt.Printf(`Usage: %s launch [options] <fleet-name>

Launch a new fleet of spot instances on AWS EC2.
The fleet receives the given name and can be referred with this name in further
commands.

Options:

  --availability-zone <zone>  name of the availability zone to use (default: '%s')

  --context <path>            path of the context file (default: '%s')

  --image <id | name>         name of the instance image or id if it starts by
                              'ami-' (default: '%s')

  --key <key-name>            name of the ssh key to use (default: '%s')

  --placement-group <group>   name of the placement group to use (default: '%s')

  --price <float>             maximum price per unit hour (default: %f)

  --region <region-name>      region where to launch instances (default: '%s')

  --replace                   replace the fleet with the same name if any

  --secgroup <id>             id of the security group to use (default: '%s')

  --size <int>                number of instances in the fleet (default: %d)

  --time <timespec>           maximum life duration of the fleet (default: '%s')

  --type <instance-type>      type of instance (default: '%s')

  --user <user-name>          user to ssh connect to instances (default: '%s')

`,
		PROGNAME, DEFAULT_AVAILABILITY_ZONE, DEFAULT_CONTEXT,
		DEFAULT_IMAGE, DEFAULT_KEY, DEFAULT_PLACEMENT_GROUP,
		DEFAULT_PRICE, DEFAULT_REGION, DEFAULT_SECGROUP, DEFAULT_SIZE,
		DEFAULT_TIME, DEFAULT_TYPE, DEFAULT_USER)
}

func buildFleetRequest() *ec2.RequestSpotFleetInput {
	var spec ec2.SpotFleetLaunchSpecification
	var conf ec2.SpotFleetRequestConfigData
	var placement ec2.SpotPlacement
	var req ec2.RequestSpotFleetInput
	var until time.Time = launchProcOptionTime.DeadlineDate()
	var ilist *ImageList
	var sgroupid *string
	var image *Image
	var err error

	if IsImageId(*optionImage) {
		spec.ImageId = aws.String(*optionImage)
	} else {
		ilist = NewImageList()

		err = ilist.Fetch(*optionImage, *optionRegion)
		if err != nil {
			Error("cannot use image '%s': %s", *optionImage,
				err.Error())
		}

		if len(ilist.Images) > 1 {
			Error("more than one image named '%s' in region %s",
				*optionImage, *optionRegion)
		}

		_, err = ilist.WaitAvailable(NewTimeoutNone())
		if err != nil {
			Error("cannot wait image '%s' to be available",
				*optionImage)
		}

		if len(ilist.Images) < 1 {
			Error("no image named '%s' in region %s",
				*optionImage, *optionRegion)
		}

		for _, image = range ilist.Images {
			spec.ImageId = aws.String(image.Id)
			break
		}
	}

	if IsSecurityGroupId(*optionSecgroup) {
		sgroupid = optionSecgroup
	} else {
		sgroupid, err = GetSecurityGroupId(*optionSecgroup,
			*optionRegion)

		if err != nil {
			Error("cannot find security group '%s' in region '%s'",
				*optionSecgroup, *optionRegion)
		}
	}

	spec.InstanceType = optionType
	spec.KeyName = optionKey
	spec.SecurityGroups = []*ec2.GroupIdentifier{
		&ec2.GroupIdentifier{
			GroupId: aws.String(*sgroupid),
		},
	}

	if *optionAvailabilityZone != "" {
		placement.AvailabilityZone = optionAvailabilityZone
		spec.Placement = &placement
	}
	if *optionPlacementGroup != "" {
		placement.GroupName = optionPlacementGroup
		spec.Placement = &placement
	}

	// If only the guys from Amazon knew how to do their fucking job...
	//
	until, _ = time.Parse(time.RFC3339, until.UTC().Format(time.RFC3339))

	conf.IamFleetRole = &IAM_FLEET_ROLE
	conf.SpotPrice = aws.String(fmt.Sprintf("%f", *optionPrice))
	conf.TargetCapacity = optionSize
	conf.TerminateInstancesWithExpiration = aws.Bool(true)
	conf.Type = aws.String("request")
	conf.ValidUntil = aws.Time(until)
	conf.LaunchSpecifications = []*ec2.SpotFleetLaunchSpecification{
		&spec,
	}

	req.DryRun = aws.Bool(false)
	req.SpotFleetRequestConfig = &conf

	return &req
}

func doLaunch(fleetName string) {
	var fleetRequest *ec2.RequestSpotFleetInput = buildFleetRequest()
	var response *ec2.RequestSpotFleetOutput
	var sess *session.Session
	var req *request.Request
	var client *ec2.EC2
	var ctx *Ec2Index
	var err error

	ctx, err = LoadEc2Index(*optionContext)
	if err != nil {
		ctx = NewEc2Index()
	}

	if ctx.FleetsByName[fleetName] != nil {
		if *optionReplace {
			DoStop(ctx, []string{fleetName})
		} else {
			Error("fleet '%s' already exists", fleetName)
		}
	}

	sess = session.New()
	client = ec2.New(sess, &aws.Config{Region: optionRegion})

	req, response = client.RequestSpotFleetRequest(fleetRequest)
	err = req.Send()
	if err != nil {
		Error("launch request failed: %s", err.Error())
	}

	ctx.AddEc2Fleet(*response.SpotFleetRequestId, fleetName, *optionUser,
		*optionRegion, int(*optionSize))

	StoreEc2Index(*optionContext, ctx)
}

func processLaunchOptionTime() {
	launchProcOptionTime = NewTimeoutFromSpec(*optionTime)

	if launchProcOptionTime == nil {
		Error("invalid value for option --time: '%s'", *optionTime)
	}
}

func Launch(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var fleetName string

	optionAvailabilityZone = flags.String("availability-zone",
		DEFAULT_AVAILABILITY_ZONE, "")
	optionContext = flags.String("context", DEFAULT_CONTEXT, "")
	optionImage = flags.String("image", DEFAULT_IMAGE, "")
	optionKey = flags.String("key", DEFAULT_KEY, "")
	optionPlacementGroup = flags.String("placement-group",
		DEFAULT_PLACEMENT_GROUP, "")
	optionPrice = flags.Float64("price", DEFAULT_PRICE, "")
	optionRegion = flags.String("region", DEFAULT_REGION, "")
	optionReplace = flags.Bool("replace", DEFAULT_REPLACE, "")
	optionSecgroup = flags.String("secgroup", DEFAULT_SECGROUP, "")
	optionSize = flags.Int64("size", DEFAULT_SIZE, "")
	optionTime = flags.String("time", DEFAULT_TIME, "")
	optionType = flags.String("type", DEFAULT_TYPE, "")
	optionUser = flags.String("user", DEFAULT_USER, "")

	flags.Parse(args[1:])

	if len(flags.Args()) < 1 {
		Error("missing fleet-name operand")
	} else if len(flags.Args()) > 1 {
		Error("unexpected operand: %s", flags.Args()[1])
	}

	fleetName = flags.Args()[0]

	processLaunchOptionTime()

	doLaunch(fleetName)
}
