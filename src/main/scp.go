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
	fmt.Printf(`
Usage: %s scp [options] [ <instance-specifications...> -- ]
              <local-paths...> [ :<remote-path> ]
       %s scp [options] [ <instance-specifications...> -- ]
              :<remote-paths...> <local-pattern>

Copy files from and to remote instances through a secure connection.
If the first path operand starts with a ':' character, operate in receive mode,
otherwise, operate in send mode.

In send mode, copy one or more local files or directories to the specified
instances.
If no path operand starts with a ':', they are all local paths. Send them to
the home directory on the remote instances.
If the last path starts with a ':', this is a remote path. Send all the local
paths to this remote path. If there are more than one local path, the remote
path must be an existing remote directory.

In receive mode, copy one or more remote files or directories to the paths
specified by the local pattern.
If there is more than one remote path, they must all start with a ':'
character.
The local pattern is a printf like pattern that get substitued for each
specified instance (see '%s help format' for more information about patterns).
The pattern must produce different strings for each instance.

If no instance is specified, apply to all instances.

Return zero if all copies success. Otherwise, return a non zero exit status and
print failing instances errors.

Options:
  --context <path>            path of the context file (default: '%s')
  --user <user-name>          user to ssh connect to instances (default: contextual)
  --verbose                   print scp debug output in case of failure
`,
		PROGNAME, PROGNAME, PROGNAME,
		DEFAULT_CONTEXT)
}

func buildScpCommand(instance *Ec2Instance, local, source string) *exec.Cmd {
	var user, remote string
	var scpcmd []string = []string{
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
		scpcmd = append(scpcmd, local, remote+":")
	} else {
		scpcmd = append(scpcmd, remote+":"+source, local)
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
				Error("invalid format pattern: '%%%c' "+
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

func taskReceive(instance *Ec2Instance, local, src string, notif chan bool) {
	var dest string = buildDestPath(instance, local)
	var cmd *exec.Cmd
	var err error

	cmd = buildScpCommand(instance, dest, src)
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

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// Generalistic scp code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// Return the scp command line as a string slice with the specified source and
// target operands (see man scp).
//
func buildScpCmdline(operands []string) []string {
	var cmdline []string = []string{"scp",
		"-o", "StrictHostKeyChecking=no", "-o", "LogLevel=Quiet",
		"-o", "UserKnownHostsFile=/dev/null", "-r",
	}

	if *optionVerbose {
		cmdline = append(cmdline, "-vvv")
	}

	cmdline = append(cmdline, operands...)

	return cmdline
}

// Run a set of Process objects in parallel.
// If all processes exit with success, return 0.
// Otherwise, print the stderr of each failed process, prefixed with the
// corresponding *Ec2Instance (the key of the specified map) and return 1.
//
func runProcesses(processes map[*Ec2Instance]*Process) int {
	var instance *Ec2Instance
	var exitcode, pcode int
	var process *Process
	var line string
	var found bool

	for _, process = range processes {
		process.Start()
	}

	exitcode = 0
	for instance, process = range processes {
		process.WaitFinished()
		pcode, _ = process.ExitCode()

		if pcode != 0 {
			exitcode = 1

			fmt.Fprintf(os.Stderr, "instance %s failed:\n",
				instance.Name)

			line, found = process.ReadStderr()
			for found {
				fmt.Fprintf(os.Stderr, "  %s", line)
				line, found = process.ReadStderr()
			}
		}
	}

	return exitcode
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// Scp receive mode related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// Return a Process object for the given instance to receive the given source
// paths to the specified local target.
// The target is a plain string (and not a format).
//
func buildScpReceive(instance *Ec2Instance, sources []string, target string) *Process {
	var user, remote, source string
	var operands, cmdline []string

	if *optionUser != "" {
		user = *optionUser
	} else {
		user = instance.Fleet.User
	}

	remote = user + "@" + instance.PublicIp
	for _, source = range sources {
		operands = append(operands, remote + ":" + source)
	}
	operands = append(operands, target)
	cmdline = buildScpCmdline(operands)

	return NewProcess(cmdline)
}

// Perform the scp receive for the specified instances selection with the
// given source remote paths and the given target local pattern.
// This function never returns.
//
func scpDoReceive(instances *Ec2Selection, sources []string, target string) {
	var processes map[*Ec2Instance]*Process
	var targetPaths map[string]*Ec2Instance
	var instance, other *Ec2Instance
	var targetPath string
	var found bool

	processes = make(map[*Ec2Instance]*Process)
	targetPaths = make(map[string]*Ec2Instance)

	for _, instance = range instances.Instances {
		_, found = processes[instance]
		if found {
			continue
		}

		targetPath = Format(target, instance)
		other, found = targetPaths[targetPath]
		if found {
			Error("conflicting target path for instances %s "+
				"and %s: '%s'", other.Name, instance.Name,
				targetPath)
		}

		processes[instance] = buildScpReceive(instance, sources,
			targetPath)
		targetPaths[targetPath] = instance
	}

	os.Exit(runProcesses(processes))
}

// Main for scp in receive mode with the specified instances selection and
// paths arguments.
// Parse the paths arguments to check if they are valid paths and how to do
// the receive, then process to the receive.
// Assume len(paths) to be at least 1.
//
func scpReceive(instances *Ec2Selection, paths []string) {
	var target, source string
	var sources []string
	var lastpos, pos int

	lastpos	= len(paths) - 1
	target = paths[lastpos]

	if target[0] == ':' {
		Error("no local path operand")
	} else {
		sources = paths[0:lastpos]
	}

	if len(sources) == 0 {
		Error("no remote path operand")
	}

	for pos, source = range sources {
		if source[0] != ':' {
			Error("misplaced local path operand: '%s'", source)
		} else {
			sources[pos] = source[1:]
			if len(sources[pos]) == 0 {
				Error("invalid remote path operand: '%s'",
					source)
			}
		}
	}

	scpDoReceive(instances, sources, target)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// Scp send mode related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// Return a Process object for the given instance to send the given source
// paths to the specified remote target.
// The target is a string that may be empty.
//
func buildScpSend(instance *Ec2Instance, sources []string, target string) *Process {
	var user, remote string
	var operands, cmdline []string

	if *optionUser != "" {
		user = *optionUser
	} else {
		user = instance.Fleet.User
	}

	remote = user + "@" + instance.PublicIp
	operands = append(sources, remote + ":" + target)
	cmdline = buildScpCmdline(operands)

	return NewProcess(cmdline)
}

// Perform the scp send for the specified instances selection with the given
// source local paths and the given target remote path (that may be empty).
// This function never returns.
//
func scpDoSend(instances *Ec2Selection, sources []string, target string) {
	var processes map[*Ec2Instance]*Process
	var instance *Ec2Instance
	var found bool

	processes = make(map[*Ec2Instance]*Process)

	for _, instance = range instances.Instances {
		_, found = processes[instance]
		if found {
			continue
		}

		processes[instance] = buildScpSend(instance, sources, target)
	}

	os.Exit(runProcesses(processes))
}

// Main for scp in send mode with the specified instances selection and paths
// arguments.
// Parse the paths arguments to check if they are valid paths and how to do the
// send, then process to the send.
// Assume len(paths) to be at least 1.
//
func scpSend(instances *Ec2Selection, paths []string) {
	var target, source string
	var sources []string
	var lastpos int

	lastpos	= len(paths) - 1
	target = paths[lastpos]

	if target[0] == ':' {
		sources = paths[0:lastpos]
		target = target[1:]
	} else {
		sources = paths
		target = ""
	}

	if len(sources) == 0 {
		Error("no local path operand")
	}

	for _, source = range sources {
		if source[0] == ':' {
			Error("misplaced remote path operand: '%s'", source)
		}
	}

	scpDoSend(instances, sources, target)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

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
		Error("missing source-file operand")
	}

	local = args[0]

	if strings.Contains(local, "%") && !*optionForceSend {
		if len(args) < 2 {
			Error("missing source-file operand")
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
