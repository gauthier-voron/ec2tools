package main

import (
	"flag"
	"fmt"
	"strconv"
)

var DEFAULT_UPDATE bool = false

var optionUpdate *bool

func PrintGetUsage() {
	fmt.Printf(`Usage: %s get [options] fleets
       %s get [options] <property> <instances-specification...>

Print information about fleets and instances.
The first form print a list of the fleet names launched and not yet stopped in
the current context.
The second form print the value of a property for a list of instances. The
instances may be specified by their name or by the name of their fleet, either
with a string or a regular expression.
If no specification is given, print properties for all instances.

Properties:
  fleet             name of the fleet of the instances
  fiid              integer that identifies the instance inside its fleet
  ip | public-ip    public IPv4 to access the instance
  name              name of an instance, as defined by AWS
  private-ip        private IPv4: how the instance sees itself
  region            region code the instance runs in (e.g. 'us-east-2')
  uiid              integer that identifies the instance inside its context
  user              username to use for an ssh connection

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

Options:
  --context <path>            path of the context file (default: '%s')

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

Examples:
  Print all available fleets, one per line:

      %s get fleets
      %s get --unique-results fleet

  Print all instance names for fleets 'my-fleet', 'fleet-0' and 'fleet-1':

      %s get name @my-fleet @/^fleet-[01]$/

  Print all public IP addresses for this context:

      %s get public-ip
      %s get public-ip /^.*$/
      %s get public-ip //

  Print the maximum id of an instance inside of its fleet:

      %s get --sort-by=fiid --unique-results fiid | tail -n 1

  Print public IP addresses from 'fleet-a' and 'my-fleet' in consisent order:

      %s get --sort ip @my-fleet @fleet-a
`,
		PROGNAME, PROGNAME, DEFAULT_CONTEXT,
		PROGNAME, PROGNAME, PROGNAME, PROGNAME, PROGNAME, PROGNAME,
		PROGNAME, PROGNAME)
}

func getFleets(args []string, ctx *Context) {
	var fleetName string

	if len(args) > 0 {
		Error("unexpected operand '%s'", args[0])
	}

	for fleetName = range ctx.Fleets {
		fmt.Printf("%s\n", fleetName)
	}
}

func getInstances(args []string, ctx *Context) {
	var fleetNames []string = make([]string, 0)
	var fleetName, instanceId string

	if len(args) == 0 {
		for fleetName = range ctx.Fleets {
			fleetNames = append(fleetNames, fleetName)
		}
	} else {
		for _, fleetName = range args {
			if ctx.Fleets[fleetName] == nil {
				Error("unknown fleet name: '%s'", fleetName)
			}
			fleetNames = append(fleetNames, fleetName)
		}
	}

	for _, fleetName = range fleetNames {
		for instanceId = range ctx.Fleets[fleetName].Instances {
			fmt.Printf("%s\n", instanceId)
		}
	}
}

func getIp(args []string, ctx *Context) {
	var instances map[string]*ContextInstance
	var fleetName, instId string
	var fleet *ContextFleet

	instances = make(map[string]*ContextInstance)

	for fleetName = range ctx.Fleets {
		fleet = ctx.Fleets[fleetName]
		for instId = range fleet.Instances {
			instances[instId] = fleet.Instances[instId]
		}
	}

	if len(args) == 0 {
		for instId = range instances {
			fmt.Printf("%s\n", instances[instId].PublicIp)
		}
	} else {
		for _, instId = range args {
			if instances[instId] == nil {
				Error("unknown instance id: '%s'", instId)
			}
		}
		for _, instId = range args {
			fmt.Printf("%s\n", instances[instId].PublicIp)
		}
	}
}

func getUser(args []string, ctx *Context) {
}

func GetAllFleets(idx *Ec2Index) []string {
	var results []string = make([]string, 0, len(idx.FleetsByName))
	var name string

	for name = range idx.FleetsByName {
		results = append(results, name)
	}

	return results
}

func GetNames(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.Name)
	}

	return results
}

func GetPublicIps(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.PublicIp)
	}

	return results
}

func GetPrivateIps(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.PrivateIp)
	}

	return results
}

func GetRegions(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.Fleet.Region)
	}

	return results
}

func GetUsers(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.Fleet.User)
	}

	return results
}

func GetFiids(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, strconv.Itoa(instance.FleetIndex))
	}

	return results
}

func GetFleets(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, instance.Fleet.Name)
	}

	return results
}

func GetUiids(instances *Ec2Selection) []string {
	var results []string = make([]string, 0, len(instances.Instances))
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		results = append(results, strconv.Itoa(instance.UniqueIndex))
	}

	return results
}

func Get(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var ctx *Context

	optionContext = flags.String("context", DEFAULT_CONTEXT, "")
	optionUpdate = flags.Bool("update", DEFAULT_UPDATE, "")

	flags.Parse(args[1:])
	args = flags.Args()
	
	if (len(args) < 1) {
		Error("missing type operand")
	}

	ctx = LoadContext(*optionContext)

	if *optionUpdate {
		// UpdateContext(ctx)
		StoreContext(*optionContext, ctx)
	}

	if args[0] == "fleets" {
		getFleets(args[1:], ctx)
	} else if args[0] == "instances" {
		getInstances(args[1:], ctx)
	} else if args[0] == "ip" {
		getIp(args[1:], ctx)
	} else if args[0] == "user" {
		getUser(args[1:], ctx)
	} else {
		Error("invalid type operand: '%s'", args[0])
	}
}
