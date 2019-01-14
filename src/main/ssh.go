package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"syscall"
	"time"
)

type sshInstanceContext struct {
	fleet      string
	ip         string
	user       string
	fleetIndex int
	totalIndex int
}

type sshContext struct {
	instances map[string]*sshInstanceContext
}

var DEFAULT_ERRMODE string = "all-prefix"
var DEFAULT_EXTMODE string = "eager-greatest"
var DEFAULT_OUTMODE string = "merge-parallel"
var DEFAULT_TIMEOUT int64  = -1
var DEFAULT_VERBOSE bool   = false

var optionErrmode *string
var optionExtmode *string
var optionOutmode *string
var optionTimeout *int64
var optionVerbose *bool

func PrintSshUsage() {
	fmt.Printf(`Usage: %s ssh [options] [ <instance-ids...> '--' ] <cmd> [ <args...> ]

Open an ssh connection with one or many instances and launch commands on them.
If no instance is specified, then launch the command on every instances.
If some instances are specified, every commands are sent to each of the
instances.
In each case, the instances output are aggregated.
The aggregation behavior is controlled by the options (see Modes).

Options:
  --context <path>            path of the context file (default: '%s')
  --error-mode <stream-mode>  stream-mode of the stderr (default: '%s')
  --exit-mode <exit-mode>     exit-mode used (default: '%s')
  --output-mode <stream-mode> stream-mode of the stdout (default: '%s')
  --timeout <sec>             cancel commands after <sec> seconds (default: %d)
  --user <user-name>          user to ssh connect to instances (default: contextual)
  --verbose                   print ssh debug output

Modes:
  The aggregation mode controls how the output stream, error stream and exit
  code of several instances are aggregated.
  The default stream-mode for stdout is '%s', for stderr is '%s'
  and the default exit-mode is '%s'.
  This is the complete list of possible modes.

  Stream-modes:
    all-prefix                Print the output of every instances. Each line
                              is prefixed by the id of the instance.

    merge-parallel            Print all outputs in parallel, grouping the same
                              output lines together.

  Exit-modes:
    eager-greatest            Execute the command on every instances and take
                              the greatest exit code.
`,
		PROGNAME, DEFAULT_CONTEXT, DEFAULT_ERRMODE, DEFAULT_EXTMODE,
		DEFAULT_OUTMODE, DEFAULT_TIMEOUT,
		DEFAULT_OUTMODE, DEFAULT_ERRMODE, DEFAULT_EXTMODE);
}

func buildCommand(instance *Ec2Instance, command []string) *exec.Cmd {
	var cmd *exec.Cmd
	var user, dest string
	var cctx context.Context
	var sshcmd []string = []string {
		"-o", "StrictHostKeyChecking=no", "-o", "LogLevel=Quiet",
		"-o", "UserKnownHostsFile=/dev/null",
	}

	if *optionTimeout >= 0 {
		sshcmd = append(sshcmd, "-o",
			fmt.Sprintf("ConnectTimeout=%d", *optionTimeout))
	}

	if *optionVerbose {
		sshcmd = append(sshcmd, "-vvv")
	}

	if *optionUser != "" {
		user = *optionUser
	} else {
		user = instance.Fleet.User
	}

	dest = user + "@" + instance.PublicIp

	sshcmd = append(sshcmd, dest)
	sshcmd = append(sshcmd, command...)

	if *optionTimeout < 0 {
		cmd = exec.Command("ssh", sshcmd...)
	} else {
		cctx, _ = context.WithTimeout(context.Background(),
			time.Duration(*optionTimeout) * time.Second)
		cmd = exec.CommandContext(cctx, "ssh", sshcmd...)
	}

	return cmd
}

func taskTransmitPrefix(instanceId string, from *io.ReadCloser, to *os.File) {
	var reader *bufio.Reader = bufio.NewReader(*from)
	var bufline string
	var line []byte
	var err error

	for {
		line, err = reader.ReadBytes('\n')
		if err != nil {
			break
		}

		bufline = fmt.Sprintf("[%s] %s", instanceId, string(line))
		_, err = to.WriteString(bufline)
		if err != nil {
			break
		}
	}
}

func transmitAllPrefix(instances *Ec2Selection, froms []io.ReadCloser, to *os.File) {
	var instance *Ec2Instance
	var idx int

	for idx, instance = range instances.Instances {
		go taskTransmitPrefix(instance.Name, &froms[idx], to)
	}
}

func taskChannelLines(from *io.ReadCloser, chn chan string) {
	var reader *bufio.Reader = bufio.NewReader(*from)
	var line []byte
	var err error

	for {
		line, err = reader.ReadBytes('\n')
		if err != nil {
			break
		}

		chn <- string(line)
	}

	close(chn)
}

func transmitMergeParallel(froms []io.ReadCloser, to *os.File) {
	var chns []chan string = make([]chan string, len(froms))
	var bufline, line, totalFormat, partialFormat string
	var mergedLines map[string]int
	var count, idx, width, tmp int
	var alive bool

	width = 1
	tmp = len(froms)
	for tmp >= 10 {
		width += 1
		tmp /= 10
	}

	totalFormat = fmt.Sprintf("*[%%%dd/%%d] %%s", width)
	partialFormat = fmt.Sprintf(" [%%%dd/%%d] %%s", width)

	for idx, _ = range froms {
		chns[idx] = make(chan string)
		go taskChannelLines(&froms[idx], chns[idx])
	}

	for {
		mergedLines = make(map[string]int)
		count = 0

		for idx, _ = range froms {
			line, alive = <-chns[idx]

			if alive {
				mergedLines[line] += 1
				count += 1
			}
		}

		if count == 0 {
			break
		}

		for line = range mergedLines {
			if mergedLines[line] == count {
				bufline = fmt.Sprintf(totalFormat,
					mergedLines[line], count, line)
			} else {
				bufline = fmt.Sprintf(partialFormat,
					mergedLines[line], count, line)
			}
			to.WriteString(bufline)
		}
	}
}

func taskTransmitStdin(stdins []io.WriteCloser) {
	var reader *bufio.Reader = bufio.NewReader(os.Stdin)
	var stdin io.WriteCloser
	var line []byte
	var err error

	for {
		line, err = reader.ReadBytes('\n')
		if err != nil {
			break
		}

		for _, stdin = range stdins {
			stdin.Write(line)
		}
	}

	for _, stdin = range stdins {
		stdin.Close()
	}
}

func collectExitEagerGreatest(cmds []*exec.Cmd) int {
	var exerr *exec.ExitError
	var cmd *exec.Cmd
	var tmp, max int
	var err error

	max = 0

	for _, cmd = range cmds {
		err = cmd.Wait()
		if err == nil {
			continue
		}

		switch err.(type) {
		case *exec.ExitError:
			exerr = err.(*exec.ExitError)
			tmp = exerr.Sys().(syscall.WaitStatus).ExitStatus()
		default:
			tmp = 256
		}

		if tmp > max {
			max = tmp
		}
	}

	return max
}

func doSsh(instances *Ec2Selection, command []string) {
	var length int = len(instances.Instances)
	var stdouts []io.ReadCloser = make([]io.ReadCloser, length)
	var stderrs []io.ReadCloser = make([]io.ReadCloser, length)
	var stdins []io.WriteCloser = make([]io.WriteCloser, length)
	var cmds []*exec.Cmd = make([]*exec.Cmd, length)
	var instance *Ec2Instance
	var err error
	var idx int

	for idx, instance = range instances.Instances {
		cmds[idx] = buildCommand(instance, command)

		stdins[idx], err = cmds[idx].StdinPipe()
		if err != nil {
			Error("cannot prepare command '%s'\n", command)
		}

		stdouts[idx], err = cmds[idx].StdoutPipe()
		if err != nil {
			Error("cannot prepare command '%s'\n", command)
		}

		stderrs[idx], err = cmds[idx].StderrPipe()
		if err != nil {
			Error("cannot prepare command '%s'\n", command)
		}
	}

	for idx, _ = range instances.Instances {
		err = cmds[idx].Start()
		if err != nil {
			Error("cannot launch command '%s'\n", command)
		}
	}

	if *optionOutmode == "all-prefix" {
		transmitAllPrefix(instances, stdouts, os.Stdout)
	} else if *optionOutmode == "merge-parallel" {
		transmitMergeParallel(stdouts, os.Stdout)
	} else {
		Error("unknown output mode: '%s'", *optionOutmode)
	}

	if *optionErrmode == "all-prefix" {
		transmitAllPrefix(instances, stderrs, os.Stderr)
	} else if *optionErrmode == "merge-parallel" {
		transmitMergeParallel(stderrs, os.Stderr)
	} else {
		Error("unknown errput mode: '%s'", *optionErrmode)
	}

	go taskTransmitStdin(stdins)

	os.Exit(collectExitEagerGreatest(cmds))
}

func checkStreamMode(mode string) bool {
	if mode == "all-prefix" {
		return true
	} else if mode == "merge-parallel" {
		return true
	} else {
		return false
	}
}

func checkExitMode(mode string) bool {
	if mode == "eager-greatest" {
		return true
	} else {
		return false
	}
}

func BuildAllSshContext(ctx *Context) *sshContext {
	var all []string = make([]string, 0)
	var fleetName, instanceId string

	for fleetName = range ctx.Fleets {
		for instanceId = range ctx.Fleets[fleetName].Instances {
			all = append(all, instanceId)
		}
	}

	return BuildSshContext(ctx, all)
}

func BuildSshContext(ctx *Context, instanceIds []string) *sshContext {
	var sctx map[string]*sshInstanceContext
	var totalIndex, fleetIndex int
	var names, ids []string
	var name, id string

	sctx = make(map[string]*sshInstanceContext)

	names = make([]string, 0, len(ctx.Fleets))
	for name = range ctx.Fleets {
		names = append(names, name)
	}
	sort.Strings(names)

	totalIndex = 0
	for _, name = range names {
		ids = make([]string, 0, len(ctx.Fleets[name].Instances))
		for id = range ctx.Fleets[name].Instances {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		fleetIndex = 0
		for _, id = range ids {
			sctx[id] = &sshInstanceContext {
				fleet: name,
				ip: ctx.Fleets[name].Instances[id].PublicIp,
				user: ctx.Fleets[name].User,
				fleetIndex: fleetIndex,
				totalIndex: totalIndex,
			}

			fleetIndex += 1
			totalIndex += 1
		}
	}

	for _, id = range instanceIds {
		if sctx[id] == nil {
			Error("unknown instance-id: '%s'", id)
		}
	}

	return &sshContext { sctx }
}

func Ssh(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var instances *Ec2Selection
	var command []string
	var specs []string
	var hasSpecs bool
	var ctx *Ec2Index
	var arg string
	var err error

	optionContext = flags.String("context", DEFAULT_CONTEXT, "")
	optionErrmode = flags.String("error-mode", DEFAULT_ERRMODE, "")
	optionExtmode = flags.String("exit-mode", DEFAULT_EXTMODE, "")
	optionOutmode = flags.String("output-mode", DEFAULT_OUTMODE, "")
	optionTimeout = flags.Int64("timeout", DEFAULT_TIMEOUT, "")
	optionUser = flags.String("user", "", "")
	optionVerbose = flags.Bool("verbose", DEFAULT_VERBOSE, "")

	flags.Parse(args[1:])
	args = flags.Args()

	if (len(args) < 1) {
		Error("missing instance-id operand")
	}

	hasSpecs = false
	for _, arg = range args {
		if (arg == "--") && !hasSpecs {
			hasSpecs = true
			specs = command
			command = make([]string, 0)
			continue
		}

		command = append(command, arg)
	}

	if !checkStreamMode(*optionErrmode) {
		Error("invalid stream-mode for stderr: '%s'", *optionErrmode)
	} else if !checkExitMode(*optionExtmode) {
		Error("invalid exit-mode: '%s'", *optionExtmode)
	} else if !checkStreamMode(*optionOutmode) {
		Error("invalid stream-mode for stdout: '%s'", *optionOutmode)
	}

	ctx, err = LoadEc2Index(*optionContext)
	if err != nil {
		Error("no context: %s", *optionContext)
	}

	if !hasSpecs {
		instances, _ = ctx.Select([]string{"//"})
	} else {
		instances, err = ctx.Select(specs)
		if err != nil {
			Error("invalid specification: %s", err.Error())
		}
	}

	doSsh(instances, command)
}
