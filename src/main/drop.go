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

// func dropIdFrom(id, region string) bool {
// 	var req ec2.DeregisterImageInput
// 	var sess *session.Session
// 	var client *ec2.EC2
// 	var err error

// 	sess = session.New()
// 	client = ec2.New(sess, &aws.Config{Region: aws.String(region)})

// 	req.ImageId = aws.String(id)
// 	_, err = client.DeregisterImage(&req)

// 	return (err == nil)
// }

// func dropId(id string) {
// 	var rets chan bool = make(chan bool, len(dropProcOptionRegion))
// 	var i, launched int
// 	var region string
// 	var selected bool
// 	var ret bool

// 	launched = 0
// 	for region, selected = range dropProcOptionRegion {
// 		if !selected {
// 			continue
// 		}

// 		go func(i, r string) {
// 			rets <- dropIdFrom(i, r)
// 		}(id, region)

// 		launched++
// 	}

// 	ret = false
// 	for i = 0; i < launched; i++ {
// 		ret = ret || <-rets
// 	}

// 	if !ret {
// 		Error("unknown image id '%s' for specified regions", id)
// 	}
// }

// func dropName(name string) {
// 	var rets chan bool = make(chan bool, len(dropProcOptionRegion))
// 	var region, id string
// 	var i, launched int
// 	var image *Image
// 	var err error
// 	var ret bool

// 	image, err = FetchImage(name)
// 	if err != nil {
// 		Error("cannot use image '%s': %s", name, err.Error())
// 	}

// 	launched = 0
// 	for region, id = range image.IdPerRegion {
// 		if dropProcOptionRegion[region] {
// 			go func(i, r string) {
// 				rets <- dropIdFrom(i, r)
// 			}(id, region)

// 			launched++
// 		}
// 	}

// 	ret = false
// 	for i = 0; i < launched; i++ {
// 		ret = ret || <-rets
// 	}

// 	if !ret {
// 		Error("unknown image name '%s' for specified regions", name)
// 	}
// }

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
