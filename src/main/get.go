package main

import (
	"flag"
	"fmt"
)

var DEFAULT_UPDATE bool = false

var optionUpdate *bool

func PrintGetUsage() {
	fmt.Printf(`Usage: %s [options] get fleets
       %s [options] get instances [<fleet-names...>]
       %s [options] get ip [<instance-ids...>]
       %s [options] get user [<instance-ids...>]

Print information about fleets and instances.
If a fleet name or instance id is provided, then print information about these
fleets or instances, otherwise, print about every fleets or instances.

Options:
  --context <path>            path of the context file (default: '%s')
  --update                    update context before to print
`,
		PROGNAME, PROGNAME, PROGNAME, PROGNAME, DEFAULT_CONTEXT)
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
		UpdateContext(ctx)
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
