package main

import (
	"flag"
	"fmt"
	"os"
)

var PROGNAME string = "ec2tools"
var VERSION string = "2.1.0"
var AUTHOR string = "Gauthier Voron"
var MAILTO string = "gauthier.voron@sydney.edu.au"

func Error(format string, a ...interface{}) {
	Warning(format, a...)
	fmt.Fprintf(os.Stderr, "Please type '%s --help' for more "+
		"information\n", PROGNAME)
	os.Exit(1)
}

func Warning(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s: ", PROGNAME)
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprintf(os.Stderr, "\n")
}

func PrintUsage() {
	fmt.Printf(`Usage: %s <command> [<args...>]

Launch, stop, manage and run programs on AWS EC2 instances.
Provide a convenient way to use Amazon EC2 spot instances over several regions
from simple bash scripts.

Commands:
  describe     describe a saved base image
  drop         deregister a saved base image
  get          obtain information on fleets or instances
  help         display help on a specific command
  launch       launch a new fleet of instances
  save         save an instance as a base image
  stop         stop one, several or all instances
  scp          copy files from and to instances
  set          add information on fleets or instances
  ssh          launch arbitrary commands on instances
  update       update the state of the launched instances
  wait         wait for some instances to be ready
`, PROGNAME)
}

func printVersion() {
	fmt.Println(PROGNAME, VERSION)
	fmt.Println(AUTHOR)
	fmt.Println(MAILTO)
}

func main() {
	var help *bool = flag.Bool("help", false, "")
	var version *bool = flag.Bool("version", false, "")
	var command string

	flag.Parse()

	if *help {
		PrintUsage()
		return
	}

	if *version {
		printVersion()
		return
	}

	if len(flag.Args()) == 0 {
		Error("missing command operand")
	}

	command = flag.Args()[0]

	if command == "describe" {
		Describe(flag.Args())
	} else if command == "drop" {
		Drop(flag.Args())
	} else if command == "get" {
		Get(flag.Args())
	} else if command == "help" {
		Help(flag.Args())
	} else if command == "launch" {
		Launch(flag.Args())
	} else if command == "save" {
		Save(flag.Args())
	} else if command == "scp" {
		Scp(flag.Args())
	} else if command == "set" {
		Set(flag.Args())
	} else if command == "ssh" {
		Ssh(flag.Args())
	} else if command == "stop" {
		Stop(flag.Args())
	} else if command == "update" {
		Update(flag.Args())
	} else if command == "wait" {
		Wait(flag.Args())
	} else {
		Error("invalid command operand: %s", command)
	}
}
