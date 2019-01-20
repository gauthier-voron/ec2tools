package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
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

type ReaderTransmitter interface {
	Transmit(to *os.File)
}

type ReaderTransmitterAllPrefix struct {
	Instances []*Ec2Instance
	Readers   []io.Reader
}

type ReaderTransmitterMergeParallel struct {
	Instances []*Ec2Instance
	Readers   []io.Reader
}

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
		DEFAULT_OUTMODE, DEFAULT_ERRMODE, DEFAULT_EXTMODE)
}

func buildSshCmdline(instance *Ec2Instance, cmdline []string, timeout *int,
	verbose bool, user *string) []string {

	var sshuser, dest string
	var sshcmd []string = []string{"ssh",
		"-o", "StrictHostKeyChecking=no", "-o", "LogLevel=Quiet",
		"-o", "UserKnownHostsFile=/dev/null",
	}

	if timeout != nil {
		sshcmd = append(sshcmd, "-o",
			fmt.Sprintf("ConnectTimeout=%d", *timeout))
	}

	if verbose {
		sshcmd = append(sshcmd, "-vvv")
	}

	if user != nil {
		sshuser = *user
	} else {
		sshuser = instance.Fleet.User
	}

	dest = sshuser + "@" + instance.PublicIp

	sshcmd = append(sshcmd, dest)
	sshcmd = append(sshcmd, cmdline...)

	return sshcmd
}

type SshProcessBuilder struct {
	instance *Ec2Instance // remote instance to execute on
	cmdline  []string     // command to execute on remote instance
	timeout  *int         // optional timeout (in seconds)
	user     *string      // optional ssh user
	verbose  bool         // enable verbose mode
}

func BuildSshProcess(instance *Ec2Instance, cmdline []string) *SshProcessBuilder {
	var this SshProcessBuilder

	this.instance = instance
	this.cmdline = cmdline
	this.timeout = nil
	this.user = nil
	this.verbose = false

	return &this
}

func (this *SshProcessBuilder) Timeout(timeout int) *SshProcessBuilder {
	this.timeout = &timeout
	return this
}

func (this *SshProcessBuilder) User(user string) *SshProcessBuilder {
	this.user = &user
	return this
}

func (this *SshProcessBuilder) Verbose() *SshProcessBuilder {
	this.verbose = true
	return this
}

func (this *SshProcessBuilder) Build() *Process {
	var cmdline []string

	cmdline = buildSshCmdline(this.instance, this.cmdline, this.timeout,
		this.verbose, this.user)

	return NewProcess(cmdline)
}

func buildCommand(instance *Ec2Instance, command []string) *exec.Cmd {
	var cmd *exec.Cmd
	var cctx context.Context
	var sshcmd []string
	var verbose bool = *optionVerbose
	var timeout *int = nil
	var user *string = nil
	var itimeout int

	if *optionTimeout >= 0 {
		itimeout = int(*optionTimeout)
		timeout = &itimeout
	}
	if *optionUser != "" {
		user = optionUser
	}

	sshcmd = buildSshCmdline(instance, command, timeout, verbose, user)

	if *optionTimeout < 0 {
		cmd = exec.Command(sshcmd[0], sshcmd[1:]...)
	} else {
		cctx, _ = context.WithTimeout(context.Background(),
			time.Duration(*optionTimeout)*time.Second)
		cmd = exec.CommandContext(cctx, sshcmd[0], sshcmd[1:]...)
	}

	return cmd
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// Transmitters related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// A transmitted for several ssh Process launched in parallel.
// Transmit the lines from each Process prefixed with the corresponding
// instance name.
//
type ReaderTransmitterAllPrefixz struct {
	Mode      bool           // true = stdout | false = stderr
	Instances []*Ec2Instance // instances corresponging to each ssh Process
	Processes []*Process     // processes to transmit the lines
}

// Create a ReaderTransmitterAllPrefix with specified parameters.
//
func newReaderTransmitterAllPrefix(instances *Ec2Selection,
	processes []*Process, mode bool) *ReaderTransmitterAllPrefixz {
	var ret ReaderTransmitterAllPrefixz

	ret.Mode = mode
	ret.Instances = instances.Instances
	ret.Processes = processes

	return &ret
}

// Create a ReaderTransmitterAllPrefix for the specified instances and
// processes for the stdout streams.
//
func NewReaderTransmitterAllPrefixStdout(instances *Ec2Selection,
	processes []*Process) *ReaderTransmitterAllPrefixz {
	return newReaderTransmitterAllPrefix(instances, processes, true)
}

// Create a ReaderTransmitterAllPrefix for the specified instances and
// processes for the stderr streams.
//
func NewReaderTransmitterAllPrefixStderr(instances *Ec2Selection,
	processes []*Process) *ReaderTransmitterAllPrefixz {
	return newReaderTransmitterAllPrefix(instances, processes, false)
}

// Transmit all lines comming from the process (and the related instance) with
// the specified index.
// A call is blocking until the transmitted stream is closed.
//
func (this *ReaderTransmitterAllPrefixz) transmitInstance(id int, to *os.File) {
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
func (this *ReaderTransmitterAllPrefixz) Transmit(to *os.File) {
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
type ReaderTransmitterMergeParallelz struct {
	Mode      bool       // true = stdout | false = stderr
	Processes []*Process // processes to transmit the lines
}

// Create a ReaderTransmitterMergeParallel with specified parameters.
//
func newReaderTransmitterMergeParallel(processes []*Process, mode bool) *ReaderTransmitterMergeParallelz {
	var ret ReaderTransmitterMergeParallelz

	ret.Mode = mode
	ret.Processes = processes

	return &ret
}

// Create a ReaderTransmitterMergeParallel for the specified processes for the
// stdout streams.
//
func NewReaderTransmitterMergeParallelStdout(processes []*Process) *ReaderTransmitterMergeParallelz {
	return newReaderTransmitterMergeParallel(processes, true)
}

// Create a ReaderTransmitterMergeParallel for the specified processes for the
// stderr streams.
//
func NewReaderTransmitterMergeParallelStderr(processes []*Process) *ReaderTransmitterMergeParallelz {
	return newReaderTransmitterMergeParallel(processes, false)
}

// Compute the format to use to print lines.
// The prefix indicating the number of emitting processes must have a fixed
// size for the whole execution.
//
func (this *ReaderTransmitterMergeParallelz) computeFormat() string {
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
func (this *ReaderTransmitterMergeParallelz) transmitFormatted(lines []string,
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
func (this *ReaderTransmitterMergeParallelz) Transmit(to *os.File) {
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

func NewReaderTransmitterAllPrefix(instances *Ec2Selection,
	readers []io.Reader) *ReaderTransmitterAllPrefix {
	var ret ReaderTransmitterAllPrefix

	ret.Instances = instances.Instances
	ret.Readers = readers

	return &ret
}

func (this *ReaderTransmitterAllPrefix) transmitInstance(id int, to *os.File) {
	var reader *bufio.Reader = bufio.NewReader(this.Readers[id])
	var instance *Ec2Instance = this.Instances[id]
	var bufline string
	var line []byte
	var err error

	for {
		line, err = reader.ReadBytes('\n')
		if err != nil {
			break
		}

		bufline = fmt.Sprintf("[%s] %s", instance.Name, string(line))
		_, err = to.WriteString(bufline)
		if err != nil {
			break
		}
	}
}

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

func NewReaderTransmitterMergeParallel(readers []io.Reader) *ReaderTransmitterMergeParallel {
	var ret ReaderTransmitterMergeParallel

	ret.Readers = readers

	return &ret
}

func (this *ReaderTransmitterMergeParallel) computeFormat() string {
	var width, buffer int
	var format string

	width = 1
	buffer = len(this.Readers)

	for buffer >= 10 {
		width += 1
		buffer /= 10
	}

	format = fmt.Sprintf("%%s[%%%dd/%%%dd] %%s", width, width)
	return format
}

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

func (this *ReaderTransmitterMergeParallel) Transmit(to *os.File) {
	var bufreader *bufio.Reader
	var reader io.Reader
	var lines []string
	var line []byte
	var err error

	for {
		lines = make([]string, 0)

		for _, reader = range this.Readers {
			bufreader = bufio.NewReader(reader)
			line, err = bufreader.ReadBytes('\n')
			if err == nil {
				lines = append(lines, string(line))
			}
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
func taskTransmitStdinz(processes []*Process) {
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
func collectExitEagerGreatestz(processes []*Process) int {
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

	go taskTransmitStdinz(processes)

	<-done
	<-done
}

// Execute the given command line on the instances of the given selection
// through ssh.
// This function never return but instead exit with the maximum exit code
// among the launched ssh processes.
//
func doSshz(instances *Ec2Selection, cmdline []string) {
	var processes []*Process = make([]*Process, len(instances.Instances))
	var builder *SshProcessBuilder
	var instance *Ec2Instance
	var i int

	for i, instance = range instances.Instances {
		builder = BuildSshProcess(instance, cmdline)

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

	os.Exit(collectExitEagerGreatestz(processes))
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

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
	var stdouts []io.Reader = make([]io.Reader, length)
	var stderrs []io.Reader = make([]io.Reader, length)
	var stdins []io.WriteCloser = make([]io.WriteCloser, length)
	var cmds []*exec.Cmd = make([]*exec.Cmd, length)
	var outTransmit, errTransmit ReaderTransmitter
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
		outTransmit = NewReaderTransmitterAllPrefix(instances, stdouts)
	} else if *optionOutmode == "merge-parallel" {
		outTransmit = NewReaderTransmitterMergeParallel(stdouts)
	} else {
		Error("unknown output mode: '%s'", *optionOutmode)
	}

	if *optionErrmode == "all-prefix" {
		errTransmit = NewReaderTransmitterAllPrefix(instances, stderrs)
	} else if *optionErrmode == "merge-parallel" {
		errTransmit = NewReaderTransmitterMergeParallel(stderrs)
	} else {
		Error("unknown errput mode: '%s'", *optionErrmode)
	}

	go outTransmit.Transmit(os.Stdout)
	go errTransmit.Transmit(os.Stderr)
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
