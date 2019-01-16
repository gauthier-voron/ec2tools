package main

import (
	"flag"
	"fmt"
)

type setParameters struct {
	OptionContext *string
	OptionDelete  *bool
}

var DEFAULT_SET_CONTEXT string = DEFAULT_CONTEXT
var DEFAULT_SET_DELETE bool = false

var setParams setParameters

func PrintSetUsage() {
	fmt.Printf(`Usage: %s set [options] [<instances-specs...> --] <property> <value>
       %s set [options] --delete [<instances-specs...> --] <property>

Set an abritrary property for one or many instances.
If no instance is specified, set the property for all instances.
The property is specified by an arbitrary name. The only restrictions are that
it is different from the builtin properties listed in the 'get' subcommand
help message and it cannot be empty.
The property value is an arbitrary string that may be empty.
The first syntax set a property value, the second syntax delete a property.
There is a difference between an defined but empty property and an undefined
property.

Options:

  --context <path>            path of the context file (default: '%s')
`,
		PROGNAME, PROGNAME, DEFAULT_CONTEXT)
}

func DoDelete(instances *Ec2Selection, attribute string) {
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		delete(instance.Attributes, attribute)
	}
}

func DoSet(instances *Ec2Selection, attribute, value string) {
	var instance *Ec2Instance

	for _, instance = range instances.Instances {
		instance.Attributes[attribute] = value
	}
}

func Set(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var specs, properties []string
	var instances *Ec2Selection
	var hasSpecs bool
	var ctx *Ec2Index
	var arg string
	var err error

	setParams.OptionContext = flags.String("context", DEFAULT_SET_CONTEXT, "")
	setParams.OptionDelete = flags.Bool("delete", DEFAULT_SET_DELETE, "")

	flags.Parse(args[1:])
	args = flags.Args()

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

	if *setParams.OptionDelete {
		if len(properties) < 1 {
			Error("missing property operand")
		} else if len(properties) > 1 {
			Error("unexpected operand: '%s'", properties[1])
		} else if len(specs) < 1 {
			Error("missing instance-spec operand")
		}
	} else {
		if len(properties) < 1 {
			Error("missing property operand")
		} else if len(properties) < 2 {
			Error("missing value operand")
		} else if len(properties) > 2 {
			Error("unexpected operand: '%s'", properties[2])
		} else if len(specs) < 1 {
			Error("missing instance-spec operand")
		}
	}

	if len(properties[0]) == 0 {
		Error("invalid empty property name")
	} else if (properties[0] == "name") || (properties[0] == "ip") ||
		(properties[0] == "public-ip") ||
		(properties[0] == "private-ip") ||
		(properties[0] == "region") || (properties[0] == "user") ||
		(properties[0] == "fiid") || (properties[0] == "fleet") ||
		(properties[0] == "uiid") {
		Error("conflicting property name")
	}

	ctx, err = LoadEc2Index(*setParams.OptionContext)
	if err != nil {
		Error("no context: %s", *optionContext)
	}

	instances, err = ctx.Select(specs)
	if err != nil {
		Error("invalid specification: %s", err.Error())
	}

	if *setParams.OptionDelete {
		DoDelete(instances, properties[0])
	} else {
		DoSet(instances, properties[0], properties[1])
	}

	StoreEc2Index(*setParams.OptionContext, ctx)
}
