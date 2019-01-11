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

// The information necessary to perform a concurrent update of an Ec2Index by
// probing AWS.
//
type updateJob struct {
	index   *Ec2Index
	mailbox chan *updateGoRequest
	ack     chan bool
}

// A concurrent update request to add a new instance to a given fleet.
// This is to be sent to a central updater goroutine.
//
type updateGoRequest struct {
	fleetName    string // name of the fleet to add an instance
	instanceName string // name of the instance to add
	publicIp     string // public IPv4 of the instance to add
	privateIp    string // private IPv4 of the instance to add
}

// Receive concurrent update requests to the context and modify the context
// sequentially.
// Receive the update through a channel.
// Return when the channel is closed.
//
func updateIndex(job *updateJob) {
	var req *updateGoRequest
	var instance *Ec2Instance
	var found bool

	for req = range job.mailbox {
		instance, found = job.index.InstancesByName[req.instanceName]

		if found {
			instance.PublicIp = req.publicIp
			instance.PrivateIp = req.privateIp
		} else {
			job.index.FleetsByName[req.fleetName].AddEc2Instance(
				req.instanceName, req.publicIp, req.privateIp)
		}
	}

	job.ack <- true
}

// Create a new update job that goroutine can use to update concurrently a
// given Ec2Index.
//
func newUpdateJob(index *Ec2Index) *updateJob {
	var job updateJob

	job.index = index
	job.mailbox = make(chan *updateGoRequest, 16)
	job.ack = make(chan bool)

	go updateIndex(&job)

	return &job
}

// Terminate an update job.
// Wait for any pending update to finish, then free associated resources.
// Ensure the related index is not modified after this call returns.
//
func (this *updateJob) terminate() {
	close(this.mailbox)
	<-this.ack
}

// Signal an instance and its properties to the central updater routine.
// If the central updater already know the instance and it has not changed
// since the last update, it ignores it silently.
//
func (this *updateJob) raise(fleetName, name, publicIp, privateIp string) {
	var req updateGoRequest

	req.fleetName = fleetName
	req.instanceName = name
	req.publicIp = publicIp
	req.privateIp = privateIp

	this.mailbox <- &req
}

// A subpart of an update job.
// Typically, the part of an update job specific to a given fleet.
type updateSubjob struct {
	Parent *updateJob // the main job of this subjob
	Fleet  *Ec2Fleet  // the fleet specific for this subjob
	Client *ec2.EC2   // client to use to communicate with AWS
}

// Raise a new instance to update as specified by AWS.
// Only raise instances which already have a public IP.
//
func raiseInstance(instance *ec2.Instance, subjob *updateSubjob) {
	if instance.PublicIpAddress == nil {
		return
	}

	subjob.Parent.raise(subjob.Fleet.Name, *instance.InstanceId,
		*instance.PublicIpAddress, *instance.PrivateIpAddress)
}

// Raise a new list of instances to update as specified by AWS.
// Only raise instances which already have a public IP.
//
func raiseInstances(list *ec2.DescribeInstancesOutput, subjob *updateSubjob) {
	var reservation *ec2.Reservation
	var instance *ec2.Instance

	for _, reservation = range list.Reservations {
		for _, instance = range reservation.Instances {
			raiseInstance(instance, subjob)
		}
	}
}

// Probe AWS to get the properties of a given list of active instances and
// update these instances with the probed properties.
// Return an AWS related error or nil if everything goes well.
//
func probeInstancesz(list []*ec2.ActiveInstance, subjob *updateSubjob) error {
	var output *ec2.DescribeInstancesOutput
	var input ec2.DescribeInstancesInput
	var instance *ec2.ActiveInstance
	var err error
	var idx int

	input.InstanceIds = make([]*string, len(list))

	for idx, instance = range list {
		input.InstanceIds[idx] = instance.InstanceId
	}

	output, err = subjob.Client.DescribeInstances(&input)
	if err != nil {
		return err
	}

	raiseInstances(output, subjob)
	return nil
}

// Probe AWS to get the list of instances related to a given fleet and the
// properties of these instances, then update the index.
// Return an AWS related error or nil if everything goes well.
//
func probeFleetInstances(subjob *updateSubjob) error {
	var input ec2.DescribeSpotFleetInstancesInput
	var output *ec2.DescribeSpotFleetInstancesOutput
	var err error

	input.SpotFleetRequestId = &subjob.Fleet.Id
	output, err = subjob.Client.DescribeSpotFleetInstances(&input)
	if err != nil {
		return err
	}

	return probeInstancesz(output.ActiveInstances, subjob)
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
