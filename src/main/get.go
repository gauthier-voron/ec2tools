package main

import (
	"flag"
	"fmt"
	"sort"
	"strconv"
)

var DEFAULT_DEFINED bool = false
var DEFAULT_SORT bool = false
var DEFAULT_SORT_BY string = ""
var DEFAULT_UNIQUE_INSTANCES bool = false
var DEFAULT_UNIQUE_RESULTS bool = false
var DEFAULT_UPDATE bool = false

var optionDefined *bool
var optionSort *bool
var optionSortBy *string
var optionUniqueInstances *bool
var optionUniqueResults *bool
var optionUpdate *bool

func PrintGetUsage() {
	fmt.Printf(`Usage: %s get [options] fleets
       %s get [options] [<instances-specification...> --] <properties...>

Print information about fleets and instances.
The first form print a list of the fleet names launched and not yet stopped in
the current context.
The second form print the values of the properties for a list of instances. The
instances may be specified by their name or by the name of their fleet, either
with a string or a regular expression.
If no specification is given, print properties for all instances.

Options:
  --context <path>            path of the context file (default: '%s')

  --defined                   only print defined properties

  --format                    interpret properties as printf like format (see
                              Format section)

  --sort                      sort instances by their uiid before to print
                              their properties (shortcut for '--sort-by uiid')

  --sort-by <property>        sort instances by the indicated property before
                              to print their properties

  --unique-instances          remove duplicate instances before to print their
                              properties, preserving order (duplicate results
                              may be displayed if several instances have the
                              same properties)

  --unique-results            remove duplicate results, preserving order (no
                              duplicate can be displayed)

  --update                    update context before to print

Properties:
  fleet             name of the fleet of the instances
  fiid              integer that identifies the instance inside its fleet
  ip | public-ip    public IPv4 to access the instance
  name              name of an instance, as defined by AWS
  private-ip        private IPv4: how the instance sees itself
  region            region code the instance runs in (e.g. 'us-east-2')
  uiid              integer that identifies the instance inside its context
  user              username to use for an ssh connection
  <attribute>       a custom attribute defined with the 'set' subcommand

Instance specification:
  Instances can be specified either directly by their name or by the name of
  their fleet. In this last case, the specification starts with a '@':

      i-0efc03c42c4137c64     the instance identified as 'i-0ef...' by AWS

      @my-fleet               all instances belonging to the fleet 'my-fleet'

  Additionally, an instance or a fleet name can be an exact string or a Perl
  regular expression. In this last case, it is surrounded by '/':

      /^.*[0-7]$/             all instances with a name ending with 0, ..., 7

      @/fleet-(a|b)/          all instances of fleets 'fleet-a' or 'fleet-b'

  Without additional options, the instances resulting from a single
  specification are sorted by their uiid property.
  The results from different specifications are concatenated without additional
  sorting nor re;oving of duplicate results.

Format:
  If the '--fornat' option is supplied, the properties are interpreted as
  printf like format with the following formatting sequences.

      %%d            fiid
      %%D            uiid
      %%f            fleet
      %%i            public-ip
      %%I            private-ip
      %%n            name
      %%r            region
      %%u            user

      %%{<name>}     the value of a property, as defined in the Properties
                    section (empty string if it is an undefined attribute) 

      %%%%            a '%%' character

Examples:
  Print all available fleets, one per line:

      %s get fleets
      %s get --unique-results fleet

  Print all instance names for fleets 'my-fleet', 'fleet-0' and 'fleet-1':

      %s get name @my-fleet @/^fleet-[01]$/

  Print all public IP addresses for this context:

      %s get public-ip
      %s get /^.*$/ -- public-ip
      %s get // -- public-ip

  Print the maximum id of an instance inside of its fleet:

      %s get --sort-by=fiid --unique-results fiid | tail -n 1

  Print public IP addresses from 'fleet-a' and 'my-fleet' in consisent order:

      %s get --sort @my-fleet @fleet-a -- ip

  Print a string containing the public and private IP of all instances

      %s get --format 'public-ip: %%I  /  private-ip: %%{private-ip}'

`,
		PROGNAME, PROGNAME, DEFAULT_CONTEXT,
		PROGNAME, PROGNAME, PROGNAME, PROGNAME, PROGNAME, PROGNAME,
		PROGNAME, PROGNAME, PROGNAME)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// PropertList related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// A list of properties of format associated with an instance.
//
type PropertyList struct {
	Instance   *Ec2Instance // instance with the associated properties
	Properties []*Property  // properties fetched for the instance
	Values     []string     // values of properties to show
}

// Create a new empty PropertyList associated to the given instance.
//
func NewPropertyList(instance *Ec2Instance) *PropertyList {
	var this PropertyList

	this.Instance = instance
	this.Properties = make([]*Property, 0)
	this.Values = make([]string, 0)

	return &this
}

// Add a new property with the given name to the list.
//
func (this *PropertyList) GetProperty(name string) {
	var property *Property = GetProperty(this.Instance, name)

	this.Properties = append(this.Properties, property)
	this.Values = append(this.Values, property.Value)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// Sort the instances inplace depending on the given sortkeys.
// The lengths of instances.Instances and sortkeys must be the same so for one
// instance, there is one sortkey.
// There can be duplicated values in sortkeys.
//
func sortInstances(instances *Ec2Selection, sortkeys []string) {
	var smap map[string][]*Ec2Instance
	var instance *Ec2Instance
	var key, prev string
	var i int

	if len(sortkeys) == 0 {
		return
	}

	smap = make(map[string][]*Ec2Instance)

	for i, key = range sortkeys {
		smap[key] = append(smap[key], instances.Instances[i])
	}

	sort.Strings(sortkeys)

	instances.Instances = make([]*Ec2Instance, 0, len(sortkeys))

	prev = sortkeys[0] + " "
	for i, key = range sortkeys {
		if key == prev {
			continue
		}

		for _, instance = range smap[key] {
			instances.Instances = append(instances.Instances,
				instance)
		}

		prev = key
	}
}

// Remove duplicates from the given instances, preserving the order of the
// first occurence of each instance.
//
func uniqueInstances(instances *Ec2Selection) {
	var umap map[int]*Ec2Instance = make(map[int]*Ec2Instance)
	var unique []*Ec2Instance = make([]*Ec2Instance, 0)
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		if umap[instance.UniqueIndex] == nil {
			umap[instance.UniqueIndex] = instance
			unique = append(unique, instance)
		}
	}

	instances.Instances = unique
}

// Create a slice of string that contains every first occurence of each string
// found in the given results slice, in the same order.
//
func uniqueResults(results []string) []string {
	var umap map[string]bool = make(map[string]bool)
	var unique []string = make([]string, 0, len(results))
	var result string
	var found bool

	for _, result = range results {
		_, found = umap[result]
		if !found {
			umap[result] = true
			unique = append(unique, result)
		}
	}

	return unique
}

// Return a slice containing all the fleet names for a given Ec2Index.
//
func GetAllFleets(idx *Ec2Index) []string {
	var results []string = make([]string, 0, len(idx.FleetsByName))
	var name string

	for name = range idx.FleetsByName {
		results = append(results, name)
	}

	return results
}

// Return a slice containing the name of each instance of the given
// Ec2Selection.
// The string values appear in the same order than the instances do.
// If their are duplicate instances in the given Ec2Selection, there are
// duplicate string values in the returned slice as well.
//
func GetNames(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.Name)
	}

	return results
}

// Return a slice containing the public IPv4 address of each instance of the
// given Ec2Selection.
// The string values appear in the same order than the instances do.
// If their are duplicate instances in the given Ec2Selection, there are
// duplicate string values in the returned slice as well.
//
func GetPublicIps(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.PublicIp)
	}

	return results
}

// Return a slice containing the private IPv4 address of each instance of the
// given Ec2Selection.
// The string values appear in the same order than the instances do.
// If their are duplicate instances in the given Ec2Selection, there are
// duplicate string values in the returned slice as well.
//
func GetPrivateIps(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.PrivateIp)
	}

	return results
}

// Return a slice containing the region code of each instance of the given
// Ec2Selection.
// The string values appear in the same order than the instances do.
// If their are duplicate instances in the given Ec2Selection, there are
// duplicate string values in the returned slice as well.
//
func GetRegions(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.Fleet.Region)
	}

	return results
}

// Return a slice containing the connection username of each instance of the
// given Ec2Selection.
// The string values appear in the same order than the instances do.
// If their are duplicate instances in the given Ec2Selection, there are
// duplicate string values in the returned slice as well.
//
func GetUsers(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.Fleet.User)
	}

	return results
}

// Return a slice containing the fleet index of each instance of the given
// Ec2Selection (as a string).
// The string values appear in the same order than the instances do.
// If their are duplicate instances in the given Ec2Selection, there are
// duplicate string values in the returned slice as well.
//
func GetFiids(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, strconv.Itoa(instance.FleetIndex))
	}

	return results
}

// Return a slice containing the fleet name of each instance of the given
// Ec2Selection.
// The string values appear in the same order than the instances do.
// If their are duplicate instances in the given Ec2Selection, there are
// duplicate string values in the returned slice as well.
//
func GetFleets(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.Fleet.Name)
	}

	return results
}

// Return a slice containing the unique index of each instance of the given
// Ec2Selection (as a string).
// The string values appear in the same order than the instances do.
// If their are duplicate instances in the given Ec2Selection, there are
// duplicate string values in the returned slice as well.
//
func GetUiids(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, strconv.Itoa(instance.UniqueIndex))
	}

	return results
}

func GetAttributes(instances *Ec2Selection, name string) ([]string, []bool) {
	var sresults []string = make([]string, 0, len(instances.Instances))
	var bresults []bool = make([]bool, 0, len(instances.Instances))
	var instance *Ec2Instance
	var value string
	var found bool

	for _, instance = range instances.Instances {
		value, found = instance.Attributes[name]

		if found {
			sresults = append(sresults, value)
		} else {
			sresults = append(sresults, "")
		}

		bresults = append(bresults, found)
	}

	return sresults, bresults
}

// Return a slice containing the property value of each instance of the given
// Ec2Selection, given this property name as a string.
// The string values appear in the same order than the instances do.
// If their are duplicate instances in the given Ec2Selection, there are
// duplicate string values in the returned slice as well.
//
func getInstancesProperty(instances *Ec2Selection, property string) ([]string, []bool) {
	var sresults []string
	var bresults []bool
	var idx int

	if property == "name" {
		sresults = GetNames(instances)
	} else if (property == "ip") || (property == "public-ip") {
		sresults = GetPublicIps(instances)
	} else if property == "private-ip" {
		sresults = GetPrivateIps(instances)
	} else if property == "region" {
		sresults = GetRegions(instances)
	} else if property == "user" {
		sresults = GetUsers(instances)
	} else if property == "fiid" {
		sresults = GetFiids(instances)
	} else if property == "fleet" {
		sresults = GetFleets(instances)
	} else if property == "uiid" {
		sresults = GetUiids(instances)
	} else {
		return GetAttributes(instances, property)
	}

	bresults = make([]bool, len(sresults))

	for idx, _ = range sresults {
		bresults[idx] = true
	}

	return sresults, bresults
}

func Get(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var results, sortkeys, specs, properties, defresults []string
	var instances *Ec2Selection
	var hasSpecs, defined bool
	var arg, result string
	var defineds []bool
	var idx *Ec2Index
	var err error
	var i int

	optionContext = flags.String("context", DEFAULT_CONTEXT, "")
	optionDefined = flags.Bool("defined", DEFAULT_DEFINED, "")
	optionSort = flags.Bool("sort", DEFAULT_SORT, "")
	optionSortBy = flags.String("sort-by", DEFAULT_SORT_BY, "")
	optionUniqueInstances = flags.Bool("unique-instances", DEFAULT_UNIQUE_INSTANCES, "")
	optionUniqueResults = flags.Bool("unique-results", DEFAULT_UNIQUE_RESULTS, "")
	optionUpdate = flags.Bool("update", DEFAULT_UPDATE, "")

	flags.Parse(args[1:])
	args = flags.Args()

	if len(args) < 1 {
		Error("missing property operand")
	}

	hasSpecs = false
	specs = []string{"//"}
	properties = make([]string, 0)

	for _, arg = range args {
		if (arg == "--") && !hasSpecs {
			hasSpecs = true
			specs = properties
			properties = make([]string, 0)
			continue
		}

		properties = append(properties, arg)
	}

	if len(properties) < 1 {
		Error("missing property operand")
	} else if len(properties) > 1 {
		Error("too many property operands")
	} else if len(specs) < 1 {
		Error("missing instance-spec operand")
	}

	if *optionSort {
		if *optionSortBy != DEFAULT_SORT_BY {
			Error("options 'sort' and 'sort-by' are exclusives")
		} else {
			*optionSortBy = "uiid"
		}
	}

	idx, err = LoadEc2Index(*optionContext)
	if err != nil {
		Error("no context: %s", *optionContext)
	}

	if *optionUpdate {
		UpdateContext(idx)
		StoreEc2Index(*optionContext, idx)
	}

	if args[0] == "fleets" {
		if hasSpecs {
			Error("unexpected instance-spec operand")
		}
		results = GetAllFleets(idx)
	} else {
		instances, err = idx.Select(specs)
		if err != nil {
			Error("invalid specification: %s", err.Error())
		}

		if *optionUniqueInstances {
			uniqueInstances(instances)
		}

		if *optionSortBy != "" {
			sortkeys, _ = getInstancesProperty(instances,
				*optionSortBy)

			sortInstances(instances, sortkeys)
		}

		results, defineds = getInstancesProperty(instances, properties[0])

		if *optionDefined {
			defresults = make([]string, 0)
			for i, defined = range defineds {
				if defined {
					defresults =
						append(defresults, results[i])
				}
			}
			results = defresults
		}
	}

	if *optionUniqueResults {
		results = uniqueResults(results)
	}

	for _, result = range results {
		fmt.Println(result)
	}
}
