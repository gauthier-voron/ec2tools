package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

var DEFAULT_ERRMODE string = "all-prefix"
var DEFAULT_EXTMODE string = "eager-greatest"
var DEFAULT_OUTMODE string = "merge-parallel"
var DEFAULT_TIMEOUT int64 = -1
var DEFAULT_VERBOSE bool = false

var optionErrmode *string
var optionExtmode *string
var optionOutmode *string
var optionTimeout *int64
var optionVerbose *bool

func PrintSshUsage() {
	fmt.Printf(`Usage: %s ssh [options] [ <instance-specs...> '--' ] <cmd> [ <args...> ]

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
  --format                    interpret the cmd and args as printf format
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
		DEFAULT_OUTMODE, DEFAULT_ERRMODE, DEFAULT_EXTMODE)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// Ssh process related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// Builder for an ssh Process.
// Store temporary configuration for the process to build.
// When the configuration is done, use it to create the Process.
//
type SshProcessBuilder struct {
	instance *Ec2Instance // remote instance to execute on
	cmdline  []string     // command to execute on remote instance
	timeout  *int         // optional timeout (in seconds)
	user     *string      // optional ssh user
	verbose  bool         // enable verbose mode
}

// Create a new SshProcessBuilder for the specified instance and doing the
// specified command line.
// The optional values receive their default values.
//
func BuildSshProcess(instance *Ec2Instance, cmdline []string) *SshProcessBuilder {
	var this SshProcessBuilder

	this.instance = instance
	this.cmdline = cmdline
	this.timeout = nil
	this.user = nil
	this.verbose = false

	return &this
}

// Set the timeout to the specified number of seconds.
// The given number of seconds must be strictly positive.
//
func (this *SshProcessBuilder) Timeout(timeout int) *SshProcessBuilder {
	this.timeout = &timeout
	return this
}

// Set the ssh username to the specified string.
// The given username must be a non empty string.
//
func (this *SshProcessBuilder) User(user string) *SshProcessBuilder {
	this.user = &user
	return this
}

// Set the ssh process to verbose mode.
//
func (this *SshProcessBuilder) Verbose() *SshProcessBuilder {
	this.verbose = true
	return this
}

// Build an ssh Process based on this configuration.
// The returned Process is not started yet.
//
func (this *SshProcessBuilder) Build() *Process {
	var sshuser, dest string
	var sshcmd []string = []string{"ssh",
		"-o", "StrictHostKeyChecking=no", "-o", "LogLevel=Quiet",
		"-o", "UserKnownHostsFile=/dev/null",
	}

	if this.timeout != nil {
		sshcmd = append(sshcmd, "-o",
			fmt.Sprintf("ConnectTimeout=%d", *this.timeout))
	}

	if this.verbose {
		sshcmd = append(sshcmd, "-vvv")
	}

	if this.user != nil {
		sshuser = *this.user
	} else {
		sshuser = this.instance.Fleet.User
	}

	dest = sshuser + "@" + this.instance.PublicIp

	sshcmd = append(sshcmd, dest)
	sshcmd = append(sshcmd, this.cmdline...)

	return NewProcess(sshcmd)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// Transmitters related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// An object to transmit some infromation to the specified file or stream.
// Its main purpose is to transmit the streams incoming from an ssh process
// with appropriate additional information and in the appropriate format.
//
type ReaderTransmitter interface {
	// Transmit to the specified output until there is nothing more to
	// transmit.
	// A call may block if the transmitted information comes from a stream.
	//
	Transmit(to *os.File)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// A transmitted for several ssh Process launched in parallel.
// Transmit the lines from each Process prefixed with the corresponding
// instance name.
//
type ReaderTransmitterAllPrefix struct {
	Mode      bool           // true = stdout | false = stderr
	Instances []*Ec2Instance // instances corresponging to each ssh Process
	Processes []*Process     // processes to transmit the lines
}

// Create a ReaderTransmitterAllPrefix with specified parameters.
//
func newReaderTransmitterAllPrefix(instances *Ec2Selection,
	processes []*Process, mode bool) *ReaderTransmitterAllPrefix {
	var ret ReaderTransmitterAllPrefix

	ret.Mode = mode
	ret.Instances = instances.Instances
	ret.Processes = processes

	return &ret
}

// Create a ReaderTransmitterAllPrefix for the specified instances and
// processes for the stdout streams.
//
func NewReaderTransmitterAllPrefixStdout(instances *Ec2Selection,
	processes []*Process) *ReaderTransmitterAllPrefix {
	return newReaderTransmitterAllPrefix(instances, processes, true)
}

// Create a ReaderTransmitterAllPrefix for the specified instances and
// processes for the stderr streams.
//
func NewReaderTransmitterAllPrefixStderr(instances *Ec2Selection,
	processes []*Process) *ReaderTransmitterAllPrefix {
	return newReaderTransmitterAllPrefix(instances, processes, false)
}

// Transmit all lines comming from the process (and the related instance) with
// the specified index.
// A call is blocking until the transmitted stream is closed.
//
func (this *ReaderTransmitterAllPrefix) transmitInstance(id int, to *os.File) {
	var instance *Ec2Instance = this.Instances[id]
	var process *Process = this.Processes[id]
	var bufline, line string
	var err error
	var has bool

	for {
		if this.Mode {
			line, has = process.ReadStdout()
		} else {
			line, has = process.ReadStderr()
		}

		if !has {
			break
		}

		bufline = fmt.Sprintf("[%s] %s", instance.Name, line)
		_, err = to.WriteString(bufline)
		if err != nil {
			break
		}
	}
}

// Transmit all the lines of the related instances and processes with
// sequential consistency.
// Each line is prefixed by the name of the emitting instance.
//
func (this *ReaderTransmitterAllPrefix) Transmit(to *os.File) {
	var done chan bool = make(chan bool)
	var idx int

	for idx = range this.Instances {
		go func(i int) {
			this.transmitInstance(i, to)
			done <- true
		}(idx)
	}

	for _ = range this.Instances {
		<-done
	}

	close(done)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// A transmitted for several ssh Process launched in parallel.
// Merge all the similar lines emitted in parallel and prefix each line
// version with the number of processes emitting this line.
//
type ReaderTransmitterMergeParallel struct {
	Mode      bool       // true = stdout | false = stderr
	Processes []*Process // processes to transmit the lines
}

// Create a ReaderTransmitterMergeParallel with specified parameters.
//
func newReaderTransmitterMergeParallel(processes []*Process, mode bool) *ReaderTransmitterMergeParallel {
	var ret ReaderTransmitterMergeParallel

	ret.Mode = mode
	ret.Processes = processes

	return &ret
}

// Create a ReaderTransmitterMergeParallel for the specified processes for the
// stdout streams.
//
func NewReaderTransmitterMergeParallelStdout(processes []*Process) *ReaderTransmitterMergeParallel {
	return newReaderTransmitterMergeParallel(processes, true)
}

// Create a ReaderTransmitterMergeParallel for the specified processes for the
// stderr streams.
//
func NewReaderTransmitterMergeParallelStderr(processes []*Process) *ReaderTransmitterMergeParallel {
	return newReaderTransmitterMergeParallel(processes, false)
}

// Compute the format to use to print lines.
// The prefix indicating the number of emitting processes must have a fixed
// size for the whole execution.
//
func (this *ReaderTransmitterMergeParallel) computeFormat() string {
	var width, buffer int
	var format string

	width = 1
	buffer = len(this.Processes)

	for buffer >= 10 {
		width += 1
		buffer /= 10
	}

	format = fmt.Sprintf("%%s[%%%dd/%%%dd] %%s", width, width)
	return format
}

// Merge the specified lines to account how many different versions there are
// and how many occurences for each of them, then print them with the
// appropriate prefix.
//
func (this *ReaderTransmitterMergeParallel) transmitFormatted(lines []string,
	to *os.File) {
	var packedLines map[string]int = make(map[string]int)
	var line, bufline, format string
	var count, max int

	for _, line = range lines {
		packedLines[line] += 1
	}

	format = this.computeFormat()

	max = len(lines)
	for line, count = range packedLines {
		if count == max {
			bufline = fmt.Sprintf(format, "*", count, max, line)
		} else {
			bufline = fmt.Sprintf(format, " ", count, max, line)
		}

		to.WriteString(bufline)
	}
}

// Transmit all the lines of the related processes merged with occurence count
// displayed.
//
func (this *ReaderTransmitterMergeParallel) Transmit(to *os.File) {
	var process *Process
	var lines []string
	var line string
	var has bool

	for {
		lines = make([]string, 0)

		for _, process = range this.Processes {
			if this.Mode {
				line, has = process.ReadStdout()
			} else {
				line, has = process.ReadStderr()
			}

			if has {
				lines = append(lines, string(line))
			}
		}

		if len(lines) == 0 {
			break
		}

		this.transmitFormatted(lines, to)
	}
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// Ssh process execution related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// Read on this process stdin and transmit each line to each of the specified
// processes.
// Once this process stdin closes, then close all the processes stdin.
//
func taskTransmitStdin(processes []*Process) {
	var reader *bufio.Reader = bufio.NewReader(os.Stdin)
	var process *Process
	var line []byte
	var err error

	for {
		line, err = reader.ReadBytes('\n')
		if err != nil {
			break
		}

		for _, process = range processes {
			process.WriteStdin(string(line))
		}
	}

	for _, process = range processes {
		process.CloseStdin()
	}
}

// Collect the exit status of all the specified processes and return the max
// of them.
// If a process has an exit status less than 0 or greater than 255, it is
// considered as 255.
//
func collectExitEagerGreatest(processes []*Process) int {
	var process *Process
	var tmp, max int

	max = 0

	for _, process = range processes {
		process.WaitFinished()

		tmp, _ = process.ExitCode()
		if (tmp < 0) || (tmp > 255) {
			tmp = 255
		}

		if tmp > max {
			max = tmp
		}
	}

	return max
}

// Transmit the input and output streams of the given processes, related to
// the specified instances.
// The transmission occurs accoring to the '--output-mode' and '--error-mode'
// options.
// Return when there is nothing more to transmit from the processes stdout and
// stderr.
//
func transmitStreams(instances *Ec2Selection, processes []*Process) {
	var done chan bool = make(chan bool)
	var outTransmit, errTransmit ReaderTransmitter

	if *optionOutmode == "all-prefix" {
		outTransmit = NewReaderTransmitterAllPrefixStdout(instances,
			processes)
	} else if *optionOutmode == "merge-parallel" {
		outTransmit =
			NewReaderTransmitterMergeParallelStdout(processes)
	} else {
		Error("unknown output mode: '%s'", *optionOutmode)
	}

	if *optionErrmode == "all-prefix" {
		errTransmit = NewReaderTransmitterAllPrefixStderr(instances,
			processes)
	} else if *optionErrmode == "merge-parallel" {
		errTransmit =
			NewReaderTransmitterMergeParallelStderr(processes)
	} else {
		Error("unknown errput mode: '%s'", *optionErrmode)
	}

	go func() {
		outTransmit.Transmit(os.Stdout)
		done <- true
	}()

	go func() {
		errTransmit.Transmit(os.Stderr)
		done <- true
	}()

	go taskTransmitStdin(processes)

	<-done
	<-done
}

// Execute the given command line on the instances of the given selection
// through ssh.
// This function never return but instead exit with the maximum exit code
// among the launched ssh processes.
//
func doSsh(instances *Ec2Selection, cmdline []string) {
	var processes []*Process = make([]*Process, len(instances.Instances))
	var builder *SshProcessBuilder
	var instance *Ec2Instance
	var cmdargs []string
	var cmdarg string
	var i, j int

	for i, instance = range instances.Instances {
		if *optionFormat {
			cmdargs = make([]string, len(cmdline))
			for j, cmdarg = range cmdline {
				cmdargs[j] = Format(cmdarg, instance)
			}
		} else {
			cmdargs = cmdline
		}

		builder = BuildSshProcess(instance, cmdargs)

		if *optionTimeout >= 0 {
			builder.Timeout(int(*optionTimeout))
		}
		if *optionUser != "" {
			builder.User(*optionUser)
		}
		if *optionVerbose {
			builder.Verbose()
		}

		processes[i] = builder.Build()
	}

	for i, _ = range processes {
		processes[i].Start()
	}

	transmitStreams(instances, processes)

	os.Exit(collectExitEagerGreatest(processes))
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

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
	optionFormat = flags.Bool("format", DEFAULT_FORMAT, "")
	optionOutmode = flags.String("output-mode", DEFAULT_OUTMODE, "")
	optionTimeout = flags.Int64("timeout", DEFAULT_TIMEOUT, "")
	optionUser = flags.String("user", "", "")
	optionVerbose = flags.Bool("verbose", DEFAULT_VERBOSE, "")

	flags.Parse(args[1:])
	args = flags.Args()

	if len(args) < 1 {
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
