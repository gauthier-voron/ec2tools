package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sort"
)

type ContextInstance struct {
	PublicIp string
}

type ContextFleet struct {
	Id        string
	User      string
	Region    string
	Instances map[string]*ContextInstance
}

type Context struct {
	Fleets map[string]*ContextFleet
}

var DEFAULT_CONTEXT string = ".ec2tools"

var optionContext *string

func LoadContext(path string) *Context {
	var ctx *Context
	var ctxraw []byte
	var err error

	ctxraw, err = ioutil.ReadFile(path)
	if err != nil {
		ctx = &Context {
			Fleets: make(map[string]*ContextFleet),
		}
	} else {
		err = json.Unmarshal(ctxraw, &ctx)
		if err != nil {
			Error("corrupted context file '%s'", path)
		}
	}

	return ctx
}

func StoreContext(path string, ctx *Context) {
	var ctxraw []byte
	var err error

	if len(ctx.Fleets) == 0 {
		os.Remove(path)
		return
	}

	ctxraw, err = json.Marshal(ctx)
	if err != nil {
		Error("cannot marshal context")
	}

	err = ioutil.WriteFile(path, ctxraw, 0644)
	if err != nil {
		Error("cannot write file '%s'", path)
	}
}

func (this *Context) AddFleet(name, id, user, region string) (bool, *ContextFleet) {
	var fleet *ContextFleet = this.Fleets[name]

	if (fleet != nil) {
		return false, fleet
	}

	fleet = &ContextFleet {
		Id:        id,
		User:      user,
		Region:    region,
		Instances: make(map[string]*ContextInstance, 0),
	}

	this.Fleets[name] = fleet
	return true, fleet
}

func (this *ContextFleet) AddInstance(id, publicIp string) (bool, *ContextInstance) {
	var inst *ContextInstance = this.Instances[id]

	if (inst != nil) {
		return false, inst
	}

	inst = &ContextInstance {
		PublicIp: publicIp,
	}

	this.Instances[id] = inst
	return true, inst
}


type ReverseContextInstance struct {
	FleetName  string
	FleetIndex int
	PublicIp   string
	User       string
	TotalIndex int
}

type ReverseContext struct {
	SelectedInstances  []string
	InstanceProperties map[string]*ReverseContextInstance
}

func (this *Context) buildReverseProperties() *ReverseContext {
	var fleetNames, instanceIds []string
	var fleetName, instanceId string
	var totalIndex, fleetIndex int
	var instance *ContextInstance
	var rctx ReverseContext

	rctx.InstanceProperties = make(map[string]*ReverseContextInstance)

	fleetNames = make([]string, 0, len(this.Fleets))
	for fleetName = range this.Fleets {
		fleetNames = append(fleetNames, fleetName)
	}
	sort.Strings(fleetNames)

	totalIndex = 0
	for _, fleetName = range fleetNames {
		instanceIds = make([]string, 0,
			len(this.Fleets[fleetName].Instances))
		for instanceId = range this.Fleets[fleetName].Instances {
			instanceIds = append(instanceIds, instanceId)
		}
		sort.Strings(instanceIds)

		fleetIndex = 0
		for _, instanceId = range instanceIds {
			instance = this.Fleets[fleetName].Instances[instanceId]

			rctx.InstanceProperties[instanceId] =
				&ReverseContextInstance {
				FleetName: fleetName,
				PublicIp: instance.PublicIp,
				User: this.Fleets[fleetName].User,
				FleetIndex: fleetIndex,
				TotalIndex: totalIndex,
			}

			fleetIndex += 1
			totalIndex += 1
		}
	}

	return &rctx
}

func (this *Context) BuildReverse() *ReverseContext {
	var rctx *ReverseContext = this.buildReverseProperties()
	var instanceId string

	rctx.SelectedInstances = make([]string, 0,
		len(rctx.InstanceProperties))

	for instanceId = range rctx.InstanceProperties {
		rctx.SelectedInstances = append(rctx.SelectedInstances,
			instanceId)
	}

	return rctx
}

func (this *Context) BuildReverseFor(instanceIds[] string) (string, *ReverseContext) {
	var rctx *ReverseContext = this.buildReverseProperties()
	var instanceId, errstr string

	rctx.SelectedInstances = make([]string, 0, len(instanceIds))
	errstr = ""

	for _, instanceId = range instanceIds {
		if rctx.InstanceProperties[instanceId] == nil {
			errstr = instanceId
		} else {
			rctx.SelectedInstances =
				append(rctx.SelectedInstances, instanceId)
		}
	}

	return errstr, rctx
}
