package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func PrintStopUsage() {
	fmt.Printf(`Usage: %s stop [options] [<fleet-name...>]

Stop one or more fleets on AWS EC2.
Stopping a fleet also stops all of the associated instances.
If no fleet is specified, stop every fleets.

Options:
  --context <path>            path of the context file (default: '%s')
`,
		PROGNAME, DEFAULT_CONTEXT)
}

func requestStop(region string, ids []*string) bool {
	var params ec2.CancelSpotFleetRequestsInput
	var sess *session.Session
	var client *ec2.EC2
	var err error

	sess = session.New()
	client = ec2.New(sess, &aws.Config { Region: &region })

	params.SpotFleetRequestIds = ids
	params.TerminateInstances = aws.Bool(true)
	_, err = client.CancelSpotFleetRequests(&params)
	if err != nil {
		return false
	}

	return true
}

func taskRequestStop(region string, ids []*string, retchan chan bool) {
	var payload bool = requestStop(region, ids)
	retchan <-payload
}

func doRegionStops(ctx *Context, regionFleets map[string][]*string) {
	var regionChans map[string]chan bool
	var fleetName, region string
	var id *string
	var ret bool

	regionChans = make(map[string]chan bool)

	for region = range regionFleets {
		regionChans[region] = make(chan bool)
		go taskRequestStop(region, regionFleets[region],
			regionChans[region])
	}

	for region = range regionFleets {
		ret = <- regionChans[region]

		if ret {
			for _, id = range regionFleets[region] {
				for fleetName = range ctx.Fleets {
					if ctx.Fleets[fleetName].Id != *id {
						continue
					}

					delete(ctx.Fleets, fleetName)
					break
				}
			}
		} else {
			Warning("cannot cancel fleets for region '%s'", region)
		}

		close(regionChans[region])
	}
}

func DoStop(ctx *Context, fleetNames []string) {
	var regionFleets map[string][]*string
	var fleet *ContextFleet
	var fleetName string

	regionFleets = make(map[string][]*string)

	if len(fleetNames) == 0 {
		for fleetName = range ctx.Fleets {
			fleet = ctx.Fleets[fleetName]
			regionFleets[fleet.Region] =
				append(regionFleets[fleet.Region], &fleet.Id)
		}
	} else {
		for _, fleetName = range fleetNames {
			if ctx.Fleets[fleetName] == nil {
				Error("unknown fleet-name: '%s'", fleetName)
			}
		}
		for _, fleetName = range fleetNames {
			fleet = ctx.Fleets[fleetName]
			regionFleets[fleet.Region] =
				append(regionFleets[fleet.Region], &fleet.Id)
		}
	}

	doRegionStops(ctx, regionFleets)
}

func Stop(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var ctx *Context

	optionContext = flags.String("context", DEFAULT_CONTEXT, "")

	flags.Parse(args[1:])

	ctx = LoadContext(*optionContext)

	DoStop(ctx, flags.Args())

	StoreContext(*optionContext, ctx)
}
