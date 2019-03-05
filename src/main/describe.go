package main

import (
	"flag"
	"fmt"
)

type describeParameters struct {
}

var describeParams describeParameters

func PrintDescribeUsage() {
	fmt.Printf(`Usage: %s describe <id | name>

Describe a base image stored on AWS EC2. The image is specified either by its
unique id or by its name. Search for the corresponding image in every AWS EC2
datacenters and print information about it.

`,
		PROGNAME)
}

func printDescription(ilist *ImageList) {
	var lregion, lname, lid, lstate, ldescription int
	var format string
	var image *Image
	var i int

	lregion = len("region")
	lname = len("name")
	lid = len("id")
	lstate = len("state")
	ldescription = len("description")

	for _, image = range ilist.Images {
		if len(image.Region) > lregion {
			lregion = len(image.Region)
		}
		if len(image.Name) > lname {
			lname = len(image.Name)
		}
		if len(image.Id) > lid {
			lid = len(image.Id)
		}
		if len(image.State) > lstate {
			lstate = len(image.State)
		}
		if len(image.Description) > ldescription {
			ldescription = len(image.Description)
		}
	}

	format = fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%-%ds %%-%ds\n",
		lregion, lname, lid, lstate, ldescription)

	fmt.Printf(format, "region", "name", "id", "state", "description")

	for i = 0; i < lregion; i++ {
		fmt.Printf("-")
	}
	fmt.Printf(" ")
	for i = 0; i < lname; i++ {
		fmt.Printf("-")
	}
	fmt.Printf(" ")
	for i = 0; i < lid; i++ {
		fmt.Printf("-")
	}
	fmt.Printf(" ")
	for i = 0; i < lstate; i++ {
		fmt.Printf("-")
	}
	fmt.Printf(" ")
	for i = 0; i < ldescription; i++ {
		fmt.Printf("-")
	}
	fmt.Printf("\n")

	for _, image = range ilist.Images {
		fmt.Printf(format, image.Region, image.Name, image.Id,
			image.State, image.Description)
	}
}

func Describe(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var ilist *ImageList
	var spec string
	var err error

	flags.Parse(args[1:])

	if len(flags.Args()) == 0 {
		Error("missing image-name operand")
	}

	ilist = NewImageList()

	for _, spec = range flags.Args() {
		if IsImageId(spec) {
			err = ilist.Find(spec)
		} else {
			err = ilist.Fetch(spec)
		}
	}

	if err != nil {
		Error("cannot fetch image '%s': %s", spec, err.Error())
	}

	printDescription(ilist)
}
