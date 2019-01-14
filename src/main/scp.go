package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var DEFAULT_FORCE_SEND bool = false

var optionForceSend *bool

func PrintScpUsage() {
	fmt.Printf(`Usage: %s scp [options] <source-file> [ <instance-ids...> ]
       %s scp [options] <dest-pattern> <source-file> [ <instance-ids...> ]

Copy files from and to remote instances through a secure connection. If the
first non optional argument contains a format pattern (e.g. '%%i'), then it is
a destination pattern, otherwise it is a source file.
If you need to transfer a source file with a '%%' character in its name, use
the '--force-send' option.
If no instance id is specified, then copy from or to every instances in the
context.

Options:
  --context <path>            path of the context file (default: '%s')
  --force-send                consider the first argument as a source file
  --user <user-name>          user to ssh connect to instances (default: contextual)
  --verbose                   print scp debug output

Format:
  If the first argument is a destination pattern, then it contains at least one
  format pattern. A format pattern is a '%%' character followed by a letter
  just like printf format. The available format patterns are:

    %%d                        the index of the sending instance inside its
                              fleet

    %%D                        the index of the sending instance among every
                              fleets

    %%f                        the fleet name of the sending instance

    %%i                        the id of the sending instance

    %%I                        the public IP address of the sending instance

    %%%%                        a non formatted '%%' character

  Each instance transfers a file to its corresponding formatted destination
  file. If several instances have the same destination file, the result is
  unspecified.
`,
		PROGNAME, PROGNAME,
		DEFAULT_CONTEXT)
}

func buildScpCommand(instance *Ec2Instance, local, source string) *exec.Cmd {
	var user, remote string
	var scpcmd []string = []string {
		"-o", "StrictHostKeyChecking=no", "-o", "LogLevel=Quiet",
		"-o", "UserKnownHostsFile=/dev/null", "-r",
	}

	if *optionVerbose {
		scpcmd = append(scpcmd, "-vvv")
	}

	if *optionUser != "" {
		user = *optionUser
	} else {
		user = instance.Fleet.User
	}

	remote = user + "@" + instance.PublicIp

	if source == "" {
		scpcmd = append(scpcmd, local, remote + ":")
	} else {
		scpcmd = append(scpcmd, remote + ":" + source, local)
	}

	return exec.Command("scp", scpcmd...)
}

func buildDestPath(instance *Ec2Instance, pattern string) string {
	var percent bool = false
	var ret string = ""
	var pos int = 0
	var c rune

	for _, c = range pattern {
		pos += 1

		if percent {
			switch c {
			case 'd':
				ret += fmt.Sprintf("%d", instance.FleetIndex)
			case 'D':
				ret += fmt.Sprintf("%d", instance.UniqueIndex)
			case 'f':
				ret += instance.Fleet.Name
			case 'i':
				ret += instance.Name
			case 'I':
				ret += instance.PublicIp
			case '%':
				ret += "%"
			default:
				Error("invalid format pattern: '%%%c' " +
					"(character %d)", c, pos)
			}

			percent = false
			continue
		}

		if c == '%' {
			percent = true
		} else {
			ret += string(c)
		}
	}

	return ret
}

func taskReceive(instance *Ec2Instance, local, source string, notif chan bool){
	var dest string = buildDestPath(instance, local)
	var cmd *exec.Cmd
	var err error

	cmd = buildScpCommand(instance, dest, source)
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
	}

	notif <- (err == nil)
}

func doReceive(instances *Ec2Selection, local, source string) {
	var waiter chan bool = make(chan bool)
	var instance *Ec2Instance
	var success bool

	for _, instance = range instances.Instances {
		go taskReceive(instance, local, source, waiter)
	}

	success = true
	for _, instance = range instances.Instances {
		success = success && <-waiter
	}

	close(waiter)

	if success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func taskSend(instance *Ec2Instance, local string, notifier chan bool) {
	var cmd *exec.Cmd
	var err error

	cmd = buildScpCommand(instance, local, "")
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
	}

	notifier <- (err == nil)
}

func doSend(instances *Ec2Selection, local string) {
	var waiter chan bool = make(chan bool)
	var instance *Ec2Instance
	var success bool

	for _, instance = range instances.Instances {
		go taskSend(instance, local, waiter)
	}

	success = true
	for _, instance = range instances.Instances {
		success = success && <-waiter
	}

	close(waiter)

	if success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func Scp(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var instances *Ec2Selection
	var specs []string
	var local, source string
	var ctx *Ec2Index
	var err error

	optionContext = flags.String("context", DEFAULT_CONTEXT, "")
	optionForceSend = flags.Bool("force-send", DEFAULT_FORCE_SEND, "")
	optionUser = flags.String("user", "", "")
	optionVerbose = flags.Bool("verbose", DEFAULT_VERBOSE, "")

	flags.Parse(args[1:])
	args = flags.Args()

	if len(args) < 1 {
		Error("missing source-file operand");
	}

	local = args[0]

	if strings.Contains(local, "%") && !*optionForceSend {
		if len(args) < 2 {
			Error("missing source-file operand");
		}
		source = args[1]
		specs = args[2:]
	} else {
		source = ""
		specs = args[1:]
	}

	ctx, err = LoadEc2Index(*optionContext)
	if err != nil {
		Error("no context: %s", *optionContext)
	}

	if len(specs) >= 1 {
		instances, err = ctx.Select(specs)
		if err != nil {
			Error("invalid specification: %s", err.Error())
		}
	} else {
		instances, _ = ctx.Select([]string{"//"})
	}

	if source != "" {
		doReceive(instances, local, source)
	} else {
		doSend(instances, local)
	}
}
