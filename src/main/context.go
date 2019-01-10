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

// ----------------------------------------------------------------------------
// Error related code
// ----------------------------------------------------------------------------

// An error related to an Ec2Index.
// Implements the error interface.
//
type Ec2IndexError struct {
	message string
}

// The implementation of error.Error() method for Ec2IndexError.
//
func (this *Ec2IndexError) Error() string {
	return this.message
}

// ----------------------------------------------------------------------------
// Index, fleets and instances manipulation related code
// ----------------------------------------------------------------------------

// The entry access point for every fleets / instances managed by ec2tools.
//
type Ec2Index struct {
	FleetsByName    map[string]*Ec2Fleet    // every fleets listed by Name
	InstancesByName map[string]*Ec2Instance // every instances by Name
	uniqueCounter   int                     // unique id of next instance
}

// The representation of an EC2 fleet inside ec2tools.
// The properties shared by all instances of a single fleet are listed in this
// structure too.
//
type Ec2Fleet struct {
	Name      string         // name of the fleet given by user
	User      string         // name to use to ssh instances of the fleet
	Region    string         // ec2 region code for this fleet
	Instances []*Ec2Instance // instances of this fleet
	Index     *Ec2Index      // pointer to the index
}

// The representation of an EC2 instance inside ec2tools.
// Has a back pointer to its parent fleet.
//
type Ec2Instance struct {
	Name        string            // ec2 id of this instance
	PublicIp    string            // public IPv4 (seen from outside ec2)
	PrivateIp   string            // private IPv4 (seen from the instance)
	Fleet       *Ec2Fleet         // pointer to the parent fleet
	FleetIndex  int               // id inside Fleet.Instances
	UniqueIndex int               // unique id among all fleets
	Attributes  map[string]string // user defined attributes
}

// Create a new empty index.
//
func NewEc2Index() *Ec2Index {
	var idx Ec2Index

	idx.FleetsByName = make(map[string]*Ec2Fleet)
	idx.InstancesByName = make(map[string]*Ec2Instance)
	idx.uniqueCounter = 0

	return &idx
}

// Create an empty Ec2Fleet associated to this Ec2Index.
// The fleet is defined by the name of the fleet, as defined by the software
// user, the user name to ssh the instances of the fleet and the EC2 region
// code.
// Return an error if the fleet name is already used.
//
func (this *Ec2Index) AddEc2Fleet(name, user, region string) (*Ec2Fleet, error) {
	var fleetNameDup bool
	var err Ec2IndexError
	var fleet Ec2Fleet

	_, fleetNameDup = this.FleetsByName[name]
	if fleetNameDup {
		err.message = "Fleet name already used"
		return nil, &err
	}

	fleet.Name = name
	fleet.User = user
	fleet.Region = region
	fleet.Instances = make([]*Ec2Instance, 0)
	fleet.Index = this

	this.FleetsByName[name] = &fleet

	return &fleet, nil
}

// Remove an Ec2Fleet from this Ec2Index as well as all the associated
// Ec2Instance objects.
// After removal, the fleet should not been used.
// Return an error if the fleet is not part of the Ec2Index.
//
func (this *Ec2Index) RemoveEc2Fleet(fleet *Ec2Fleet) error {
	var instance *Ec2Instance
	var err Ec2IndexError
	var fleetFound bool

	_, fleetFound = this.FleetsByName[fleet.Name]
	if !fleetFound {
		err.message = "Fleet not indexed"
		return &err
	}

	delete(this.FleetsByName, fleet.Name)

	for _, instance = range fleet.Instances {
		delete(this.InstancesByName, instance.Name)
	}

	fleet.Index = nil

	return nil
}

// Add an Ec2Instance to this Ec2Fleet.
// A new instance is defined by the EC2 instance name, its public IPv4 address
// and its private IPv4 address.
// Return an error if the instance name is already used in the index associated
// to this fleet (this however should not happen as AWS generate unique names
// for instances).
//
func (this *Ec2Fleet) AddEc2Instance(name, publicIp, privateIp string) (*Ec2Instance, error) {
	var instance Ec2Instance
	var instanceNameDup bool
	var err Ec2IndexError

	if this.Index == nil {
		err.message = "Fleet not linked to an index"
		return nil, &err
	}

	_, instanceNameDup = this.Index.InstancesByName[name]
	if instanceNameDup {
		err.message = "Instance name already used"
		return nil, &err
	}

	instance.Name = name
	instance.PublicIp = publicIp
	instance.PrivateIp = privateIp
	instance.Fleet = this
	instance.FleetIndex = len(this.Instances)
	instance.UniqueIndex = this.Index.uniqueCounter
	instance.Attributes = make(map[string]string)

	this.Instances = append(this.Instances, &instance)

	this.Index.uniqueCounter += 1
	this.Index.InstancesByName[instance.Name] = &instance

	return &instance, nil
}

// ----------------------------------------------------------------------------
// Load and store related code.
// ----------------------------------------------------------------------------

// Storage type for Ec2Index.
// Contains the same information but with no redundancy or pointers, more
// suitable for marshaling.
//
type ec2index struct {
	Fleets []*ec2fleet // storage for Ec2Index.FleetsByName
	// InstancesByName: computable from ec2index.fleets
	UniqueCounter int // storage for Ec2Index.uniqueCounter
}

// Storage type for Ec2Fleet.
// See type ec2index for more information.
//
type ec2fleet struct {
	Name      string         // storage for Ec2Fleet.Name
	User      string         // storage for Ec2Fleet.User
	Region    string         // storage for Ec2Fleet.Region
	Instances []*ec2instance // storage for Ec2Fleet.Instances
}

// Storage type for Ec2Instance.
// See type ec2index for more information.
//
type ec2instance struct {
	Name      string // storage for Ec2Instance.Name
	PublicIp  string // storage for Ec2Instance.PublicIp
	PrivateIp string // storage for Ec2Instance.PrivateIp
	// Fleet: no backpointer
	// FleetIndex: computable from ec2fleet.instances
	UniqueIndex int // storage for Ec2Instance.UniqueIndex
	Attributes  map[string]string
}

// Convert an Ec2Index to an ec2index.
// Transform a data structure suitable for in-memory navigation to a data
// structure efficient for storage.
//
func packEc2Index(idx *Ec2Index) *ec2index {
	var sortedFleetsName []string
	var pidx ec2index
	var fleet *Ec2Fleet
	var name string

	pidx.Fleets = make([]*ec2fleet, 0, len(idx.FleetsByName))
	pidx.UniqueCounter = idx.uniqueCounter

	sortedFleetsName = make([]string, 0, len(idx.FleetsByName))

	for name = range idx.FleetsByName {
		sortedFleetsName = append(sortedFleetsName, name)
	}

	sort.Strings(sortedFleetsName)

	for _, name = range sortedFleetsName {
		fleet = idx.FleetsByName[name]
		pidx.Fleets = append(pidx.Fleets, packEc2Fleet(fleet))
	}

	return &pidx
}

// Convert an Ec2Fleet to an ec2fleet.
// Transform a data structure suitable for in-memory navigation to a data
// structure efficient for storage.
//
func packEc2Fleet(fleet *Ec2Fleet) *ec2fleet {
	var pfleet ec2fleet
	var instance *Ec2Instance

	pfleet.Name = fleet.Name
	pfleet.User = fleet.User
	pfleet.Region = fleet.Region
	pfleet.Instances = make([]*ec2instance, 0, len(fleet.Instances))

	for _, instance = range fleet.Instances {
		pfleet.Instances = append(pfleet.Instances,
			packEc2Instance(instance))
	}

	return &pfleet
}

// Convert an Ec2Instance to an ec2instance.
// Transform a data structure suitable for in-memory navigation to a data
// structure efficient for storage.
//
func packEc2Instance(instance *Ec2Instance) *ec2instance {
	var pinstance ec2instance

	pinstance.Name = instance.Name
	pinstance.PublicIp = instance.PublicIp
	pinstance.PrivateIp = instance.PrivateIp
	pinstance.UniqueIndex = instance.UniqueIndex
	pinstance.Attributes = instance.Attributes

	return &pinstance
}

// Convert an ec2index to an Ec2Index.
// Transform a data structure efficient for storage to a data structure
// suitable for in-memory navigation.
//
func unpackEc2Index(pidx *ec2index) *Ec2Index {
	var idx Ec2Index
	var fleet *Ec2Fleet
	var instance *Ec2Instance
	var pfleet *ec2fleet

	idx.FleetsByName = make(map[string]*Ec2Fleet)

	for _, pfleet = range pidx.Fleets {
		fleet = unpackEc2Fleet(&idx, pfleet)
		idx.FleetsByName[fleet.Name] = fleet
	}

	idx.InstancesByName = make(map[string]*Ec2Instance)

	for _, fleet = range idx.FleetsByName {
		for _, instance = range fleet.Instances {
			idx.InstancesByName[instance.Name] = instance
		}
	}

	idx.uniqueCounter = pidx.UniqueCounter

	return &idx
}

// Convert an ec2fleet to an Ec2Fleet.
// Transform a data structure efficient for storage to a data structure
// suitable for in-memory navigation.
//
func unpackEc2Fleet(idx *Ec2Index, pfleet *ec2fleet) *Ec2Fleet {
	var fleet Ec2Fleet
	var pinstance *ec2instance
	var index int

	fleet.Name = pfleet.Name
	fleet.User = pfleet.User
	fleet.Region = pfleet.Region
	fleet.Instances = make([]*Ec2Instance, 0, len(pfleet.Instances))
	fleet.Index = idx

	for index, pinstance = range pfleet.Instances {
		fleet.Instances = append(fleet.Instances,
			unpackEc2Instance(pinstance, &fleet, index))
	}

	return &fleet
}

// Convert an ec2instance to an Ec2Instance.
// Transform a data structure efficient for storage to a data structure
// suitable for in-memory navigation.
//
func unpackEc2Instance(pinstance *ec2instance, fleet *Ec2Fleet, index int) *Ec2Instance {
	var instance Ec2Instance

	instance.Name = pinstance.Name
	instance.PublicIp = pinstance.PublicIp
	instance.PrivateIp = pinstance.PrivateIp
	instance.Fleet = fleet
	instance.FleetIndex = index
	instance.UniqueIndex = pinstance.UniqueIndex
	instance.Attributes = pinstance.Attributes

	return &instance
}

// Load an index from a json file.
// Unmarshal a compact data structure then build an Ec2Index from this compact
// structure adding fast referencing and backpointers.
//
func LoadEc2Index(path string) (*Ec2Index, error) {
	var pidx ec2index
	var raw []byte
	var err error

	raw, err = ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(raw, &pidx)
	if err != nil {
		return nil, err
	}

	return unpackEc2Index(&pidx), nil
}

// Store an index into a json file.
// Start by converting the index in a smaller, more compact data structure
// without pointer loop, then marshal this data structure in json.
//
func StoreEc2Index(path string, idx *Ec2Index) error {
	var raw []byte
	var err error

	if len(idx.FleetsByName) == 0 {
		os.Remove(path)
		return nil
	}

	raw, err = json.Marshal(packEc2Index(idx))
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, raw, 0644)
}

// The encapsulation for a set of instances.
// This is more convenient to carry across function calls than a plain slice.
//
type Ec2Selection struct {
	Instances []*Ec2Instances
}
