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

var DEFAULT_IMAGE string    = "ami-33ab5251"
var DEFAULT_KEY string      = "default"
var DEFAULT_PRICE float64   = 0.01
var DEFAULT_REGION string   = "ap-southeast-2"
var DEFAULT_REPLACE bool    = false
var DEFAULT_SECGROUP string = "sg-0e9b9bbee1dfc700a"
var DEFAULT_SIZE int64      = 1
var DEFAULT_TIME string     = "1h"
var DEFAULT_TYPE string     = "c5.large"
var DEFAULT_USER string     = "ubuntu"

var optionImage *string
var optionKey *string
var optionPrice *float64
var optionRegion *string
var optionReplace *bool
var optionSecgroup *string
var optionSize *int64
var optionTime *string
var optionType *string
var optionUser *string

func PrintLaunchUsage() {
	fmt.Printf(`Usage: %s launch [options] <fleet-name>

Launch a new fleet of spot instances on AWS EC2.
The fleet receives the given name and can be referred with this name in further
commands.

Options:
  --context <path>            path of the context file (default: '%s')
  --image <id>                id of the instance image (default: '%s')
  --key <key-name>            name of the ssh key to use (default: '%s')
  --price <float>             maximum price per unit hour (default: %f)
  --region <region-name>      region where to launch instances (default: '%s')
  --replace                   replace the fleet with the same name if any
  --secgroup <id>             id of the security group to use (default: '%s')
  --size <int>                number of instances in the fleet (default: %d)
  --time <timespec>           maximum life duration of the fleet (default: '%s')
  --type <instance-type>      type of instance (default: '%s')
  --user <user-name>          user to ssh connect to instances (default: '%s')
`,
		PROGNAME, DEFAULT_CONTEXT, DEFAULT_IMAGE, DEFAULT_KEY,
		DEFAULT_PRICE, DEFAULT_REGION, DEFAULT_SECGROUP, DEFAULT_SIZE,
		DEFAULT_TIME, DEFAULT_TYPE, DEFAULT_USER)
}

func timespecToSec(timespec string) int64 {
	var mode int = 0  // 0 = num/sp, 1 = num, 2 = unit, 3 = sp
	var defaultMult int64 = 1
	var acc int64 = 0
	var ret int64 = 0
	var c rune

	// mode variation along string walk
	//
	// '  12d13  h14m  17   s   '
	//  001121133211200113332000
	//
	// '  h'
	//  00X
	//
	// '13 11'
	//  113X
	//
	// '11hm'
	//  112X
	//
	// '11h  m'
	//  11200X

	for _, c = range timespec {
		if (c >= '0') && (c <= '9') {
			if mode == 3 {
				return -1
			} else {
				mode = 1
			}

			acc *= 10
			acc += int64(c) - '0'
		} else if c == ' ' {
			if mode == 1 {
				mode = 3
			} else if mode == 2 {
				mode = 0
			}
		} else {
			if (mode == 0) || (mode == 2) {
				return -1
			} else {
				mode = 2
			}

			switch c {
			case 'd':
				ret += acc * 86400
				defaultMult = 3600
			case 'h':
				ret += acc * 3600
				defaultMult = 60
			case 'm':
				ret += acc * 60
				defaultMult = 1
			case 's':
				ret += acc
				defaultMult = -1
			default:
				return -1
			}

			acc = 0
		}
	}

	if acc != 0 {
		if defaultMult == -1 {
			return -1
		} else {
			ret += acc * defaultMult
		}
	}

	return ret
}

func buildFleetRequest() *ec2.RequestSpotFleetInput {
	var spec ec2.SpotFleetLaunchSpecification
	var conf ec2.SpotFleetRequestConfigData
	var req ec2.RequestSpotFleetInput
	var until time.Time = time.Now()
	var tm int64 = timespecToSec(*optionTime)

	if tm < 0 {
		Error("invalid time specification: '%s'", *optionTime)
	} else {
		until = until.Add(time.Duration(tm * 1000000000))
	}

	spec.ImageId = optionImage
	spec.InstanceType = optionType
	spec.KeyName = optionKey
	spec.SecurityGroups = []*ec2.GroupIdentifier {
		&ec2.GroupIdentifier {
			GroupId: aws.String(*optionSecgroup),
		},
	}

	conf.IamFleetRole = &IAM_FLEET_ROLE
	conf.SpotPrice = aws.String(fmt.Sprintf("%f", *optionPrice))
	conf.TargetCapacity = optionSize
	conf.TerminateInstancesWithExpiration = aws.Bool(true)
	conf.Type = aws.String("request")
	conf.ValidUntil = &until
	conf.LaunchSpecifications = []*ec2.SpotFleetLaunchSpecification {
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
	client = ec2.New(sess, &aws.Config { Region: optionRegion })

	req, response = client.RequestSpotFleetRequest(fleetRequest)
	err = req.Send()
	if err != nil {
		Error("launch request failed: %s", err.Error())
	}

	ctx.AddEc2Fleet(*response.SpotFleetRequestId, fleetName, *optionUser,
		*optionRegion)

	StoreEc2Index(*optionContext, ctx)
}

func Launch(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var fleetName string

	optionContext = flags.String("context", DEFAULT_CONTEXT, "")
	optionImage = flags.String("image", DEFAULT_IMAGE, "")
	optionKey = flags.String("key", DEFAULT_KEY, "")
	optionPrice = flags.Float64("price", DEFAULT_PRICE, "")
	optionRegion = flags.String("region", DEFAULT_REGION, "")
	optionReplace = flags.Bool("replace", DEFAULT_REPLACE, "")
	optionSecgroup = flags.String("secgroup", DEFAULT_SECGROUP, "")
	optionSize = flags.Int64("size", DEFAULT_SIZE, "")
	optionTime = flags.String("time", DEFAULT_TIME, "")
	optionType = flags.String("type", DEFAULT_TYPE, "")
	optionUser = flags.String("user", DEFAULT_USER, "")

	flags.Parse(args[1:])

	if (len(flags.Args()) < 1) {
		Error("missing fleet-name operand")
	} else if (len(flags.Args()) > 1) {
		Error("unexpected operand: %s", flags.Args()[1])
	}

	fleetName = flags.Args()[0]

	doLaunch(fleetName)
}
