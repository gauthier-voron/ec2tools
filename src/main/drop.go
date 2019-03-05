package main

import (
	"flag"
	"fmt"
	"strings"
)

type dropParameters struct {
	OptionRegion *string
}

var DEFAULT_DROP_REGION string = "*"

var dropParams dropParameters

var dropProcOptionRegion []string

func PrintDropUsage() {
	fmt.Printf(`Usage: %s drop [options] <id | name>

Deregister a base image specified by either its unique id or its name. A
deregistered image cannot be used to launch new fleets. However, fleets already
launched with a deregistered image continue to work normally.
By default, deregister the image from every AWS EC2 datacenters. This behavior
can be modified with options.

Options:

  --region <region-name>      deregister from the specified region instead of
                              every regions, accept multiple region names
                              separated by commas or '*'

`,
		PROGNAME)
}

func processDropOptionRegion() {
	if *dropParams.OptionRegion == "*" {
		dropProcOptionRegion = ListRegions()
	} else {
		dropProcOptionRegion = strings.Split(*dropParams.OptionRegion,
			",")
	}
}

func doDrop(spec string) {
	var ilist *ImageList
	var err error

	ilist = NewImageList()

	if IsImageId(spec) {
		err = ilist.Find(spec, dropProcOptionRegion...)
	} else {
		err = ilist.Fetch(spec, dropProcOptionRegion...)
	}

	if err != nil {
		Error("cannot fetch images: %s", err.Error())
	}

	err = ilist.Deregister()
	if err != nil {
		Error("cannot deregister images: %s", err.Error())
	}
}

func Drop(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var spec string

	dropParams.OptionRegion = flags.String("region", DEFAULT_DROP_REGION, "")

	flags.Parse(args[1:])

	processDropOptionRegion()

	if len(flags.Args()) == 0 {
		Error("missing image-name operand")
	} else if len(flags.Args()) > 1 {
		Error("unexpected operand '%s'", flags.Args()[1])
	} else {
		spec = flags.Args()[0]
	}

	doDrop(spec)
}
