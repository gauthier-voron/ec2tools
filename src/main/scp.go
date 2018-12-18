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

func buildScpCommand(rctx *ReverseContext, instanceId, local, source string) *exec.Cmd {
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
		user = rctx.InstanceProperties[instanceId].User
	}

	remote = user + "@" + rctx.InstanceProperties[instanceId].PublicIp

	if source == "" {
		scpcmd = append(scpcmd, local, remote + ":")
	} else {
		scpcmd = append(scpcmd, remote + ":" + source, local)
	}

	return exec.Command("scp", scpcmd...)
}

func buildDestPath(rctx *ReverseContext, pattern, instanceId string) string {
	var prop *ReverseContextInstance = rctx.InstanceProperties[instanceId]
	var percent bool = false
	var ret string = ""
	var pos int = 0
	var c rune

	for _, c = range pattern {
		pos += 1

		if percent {
			switch c {
			case 'd':
				ret += fmt.Sprintf("%d", prop.FleetIndex)
			case 'D':
				ret += fmt.Sprintf("%d", prop.TotalIndex)
			case 'f':
				ret += prop.FleetName
			case 'i':
				ret += instanceId
			case 'I':
				ret += prop.PublicIp
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

func taskReceive(rctx *ReverseContext, id, local, source string,
	         notif chan bool) {
	var dest string = buildDestPath(rctx, local, id)
	var cmd *exec.Cmd
	var err error

	cmd = buildScpCommand(rctx, id, dest, source)
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
	}

	notif <- (err == nil)
}

func doReceive(rctx *ReverseContext, local, source string) {
	var waiter chan bool = make(chan bool)
	var instanceId string
	var success bool

	for _, instanceId = range rctx.SelectedInstances {
		go taskReceive(rctx, instanceId, local, source, waiter)
	}

	success = true
	for _, instanceId = range rctx.SelectedInstances {
		success = success && <-waiter
	}

	close(waiter)

	if success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func taskSend(rctx *ReverseContext, instanceId, local string,
	      notifier chan bool) {
	var cmd *exec.Cmd
	var err error

	cmd = buildScpCommand(rctx, instanceId, local, "")
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
	}

	notifier <- (err == nil)
}

func doSend(rctx *ReverseContext, local string) {
	var waiter chan bool = make(chan bool)
	var instanceId string
	var success bool

	for _, instanceId = range rctx.SelectedInstances {
		go taskSend(rctx, instanceId, local, waiter)
	}

	success = true
	for _, instanceId = range rctx.SelectedInstances {
		success = success && <-waiter
	}

	close(waiter)

	if success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func getAllInstanceIds(ctx *Context) *sshContext {
	var ret []string = make([]string, 0)
	var fleetName, instanceId string

	for fleetName = range ctx.Fleets {
		for instanceId = range ctx.Fleets[fleetName].Instances {
			ret = append(ret, instanceId)
		}
	}

	return BuildSshContext(ctx, ret)
}

func Scp(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var instanceIds []string
	var local, source string
	var rctx *ReverseContext
	var errstr string
	var ctx *Context

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
		instanceIds = args[2:]
	} else {
		source = ""
		instanceIds = args[1:]
	}

	ctx = LoadContext(*optionContext)

	if len(instanceIds) > 1 {
		errstr, rctx = ctx.BuildReverseFor(instanceIds)
		if errstr != "" {
			Error("invalid instance-id: '%s'", errstr)
		}
	} else {
		rctx = ctx.BuildReverse()
	}

	if source != "" {
		doReceive(rctx, local, source)
	} else {
		doSend(rctx, local)
	}
}
