package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func PrintUpdateUsage() {
	fmt.Printf(`Usage: %s update [options]

Update the information about launched fleets and instances.
Modify the context accordingly.

Options:
  --context <path>            path of the context file (default: '%s')
`,
		PROGNAME, DEFAULT_CONTEXT)
}

func probeInstances(client *ec2.EC2, instances []*ec2.ActiveInstance) *map[string]*ContextInstance {
	var instanceIds []*string = make([]*string, len(instances))
	var ret map[string]*ContextInstance = make(map[string]*ContextInstance)
	var result *ec2.DescribeInstancesOutput
	var params ec2.DescribeInstancesInput
	var ainstance *ec2.ActiveInstance
	var reservation *ec2.Reservation
	var instance *ec2.Instance
	var err error
	var idx int

	if len(instances) == 0 {
		return &ret
	}

	for idx, ainstance = range instances {
		instanceIds[idx] = ainstance.InstanceId
	}

	params.InstanceIds = instanceIds
	result, err = client.DescribeInstances(&params)
	if err != nil {
		return nil
	}

	for _, reservation = range result.Reservations {
		for _, instance = range reservation.Instances {
			if instance.PublicIpAddress == nil {
				continue
			}

			ret[*instance.InstanceId] = &ContextInstance {
				PublicIp: *instance.PublicIpAddress,
			}
		}
	}

	return &ret
}

func probeFleet(fleet *ContextFleet) *ContextFleet {
	var params ec2.DescribeSpotFleetInstancesInput
	var result *ec2.DescribeSpotFleetInstancesOutput
	var instances *map[string]*ContextInstance
	var sess *session.Session
	var ret ContextFleet
	var client *ec2.EC2
	var err error

	sess = session.New()
	client = ec2.New(sess, &aws.Config { Region: &fleet.Region })

	params.SpotFleetRequestId = &fleet.Id
	result, err = client.DescribeSpotFleetInstances(&params)
	if err != nil {
		return nil
	}

	instances = probeInstances(client, result.ActiveInstances)
	if instances == nil {
		return nil
	}

	ret.Id = fleet.Id
	ret.User = fleet.User
	ret.Region = fleet.Region
	ret.Instances = *instances

	return &ret
}

func taskProbeFleet(fleet *ContextFleet, retchan chan *ContextFleet) {
	var payload *ContextFleet = probeFleet(fleet)
	retchan <-payload
}

func UpdateContext(ctx *Context) {
	var fleetChans map[string]chan *ContextFleet
	var fleet *ContextFleet
	var fleetName string

	fleetChans = make(map[string]chan *ContextFleet, len(ctx.Fleets))

	for fleetName = range ctx.Fleets {
		fleetChans[fleetName] = make(chan *ContextFleet)
		go taskProbeFleet(ctx.Fleets[fleetName], fleetChans[fleetName])
	}

	for fleetName = range ctx.Fleets {
		fleet = <- fleetChans[fleetName]

		if fleet != nil {
			ctx.Fleets[fleetName] = fleet
		} else {
			Warning("cannot update fleet '%s'", fleetName)
		}

		close(fleetChans[fleetName])
	}
}

func Update(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var ctx *Context

	optionContext = flags.String("context", DEFAULT_CONTEXT, "")

	flags.Parse(args[1:])

	if (len(flags.Args()) > 0) {
		Error("unexpected operand: %s", flags.Args()[0])
	}

	ctx = LoadContext(*optionContext)

	UpdateContext(ctx)

	StoreContext(*optionContext, ctx)
}
