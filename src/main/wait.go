package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

const (
	PROCESSED_OPTION_COUNT_TYPE_NUMBER  = iota
	PROCESSED_OPTION_COUNT_TYPE_PERCENT = iota
)

type waitParameters struct {
	OptionContext *string
	OptionCount   *string
	OptionTimeout *string
	OptionWaitFor *string
}

var DEFAULT_WAIT_CONTEXT string = DEFAULT_CONTEXT
var DEFAULT_WAIT_COUNT string = "100%"
var DEFAULT_WAIT_TIMEOUT string = ""
var DEFAULT_WAIT_WAIT_FOR string = "ip"

var waitParams waitParameters

type processedOptionCount struct {
	Type   int // NUMBER or PERCENT
	Number int // integer base value
}

var waitProcOptionCount processedOptionCount

var waitProcOptionTimeout *Timeout

func PrintWaitUsage() {
	fmt.Printf(`Usage: %s wait [options] [<fleet-spec...>]

Wait one, many, or all fleets to have all or some of their instances ready.
By default, "ready" means the instance has a public IPv4 address. This
definition can be modified by options.
The fleet specifications can be either exact fleet names or regular
expressions. In this last case, it starts and ends with a '/' character.
If no fleet specification is supplied, wait for all fleets.

Options:

  --context <path>            path of the context file (default: '%s')

  --count <count|proportion>  the minimum count of instances per fleet
                              specification (or the minimum proportion if
                              argument ends with a '%%') to wait

  --timeout <timespec>        maximum time to wait the instances specified in
                              format like '30' (seconds), '1m20' or even
                              '1h 40m 30s'

  --wait-for <wait-type>      when to consider an instance is ready: 'ip' when
                              it has a public IPv4 address. 'ssh' when it is
                              reachable via ssh.
`,
		PROGNAME, DEFAULT_CONTEXT)
}

func computeRequiredCount(maximumCount int) int {
	if waitProcOptionCount.Type == PROCESSED_OPTION_COUNT_TYPE_NUMBER {
		return waitProcOptionCount.Number
	} else {
		return (maximumCount * waitProcOptionCount.Number) / 100
	}
}

func selectFleets(ctx *Ec2Index, specs []string) []*Ec2Selection {
	var selections []*Ec2Selection = make([]*Ec2Selection, len(specs))
	var spec string
	var err error
	var i int

	for i, spec = range specs {
		selections[i], err = ctx.Select([]string{spec})
		if err != nil {
			Error("invalid specification: %s", err.Error())
		}
	}

	return selections
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// ValidityMap related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// Store which instance is valid, whatever the definition of valid is.
//
type ValidityMap interface {
	// Update the validity of a given instance.
	// Possibly use a background goroutine but be sure it finishes when
	// the given timeout expires.
	//
	UpdateValidity(instance *Ec2Instance, timeout *Timeout)

	// Check if the given instance is valid.
	// An instance that has never been updated with UpdateValidity() cannot
	// be valid.
	//
	IsValid(instance *Ec2Instance) bool

	// Ensure all the background goroutines are finished.
	//
	Finalize()
}

// A validityMap defining validity has "owning a public IPv4 address".
//
type ValidityMapIp struct {
	PublicIps map[*Ec2Instance]string
}

// Create a new and empty ValidityNapIp.
//
func NewValidityMapIp() *ValidityMapIp {
	var this ValidityMapIp

	this.PublicIps = make(map[*Ec2Instance]string)

	return &this
}

// Update the validity of the specified instance.
// Ignore the timeout parameter.
//
func (this *ValidityMapIp) UpdateValidity(instance *Ec2Instance,
	timeout *Timeout) {

	if instance.PublicIp != "" {
		this.PublicIps[instance] = instance.PublicIp
	}
}

// Check if the given instance is valid: does it have a Public IPv4.
//
func (this *ValidityMapIp) IsValid(instance *Ec2Instance) bool {
	var found bool

	_, found = this.PublicIps[instance]

	return found
}

// Does nothing since their is no background task.
//
func (this *ValidityMapIp) Finalize() {
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// A validityMap defining validity has "can be reached by ssh".
// To test that, launch ssh background processes on update (unless there is
// already one running).
// The remote process is siply the `true` command.
// If the ssh process exits successfully, the instance is valid.
//
type ValidityMapSsh struct {
	Processes map[*Ec2Instance]*Process
}

// Create a new empty ValidityMapSsh.
//
func NewValidityMapSsh() *ValidityMapSsh {
	var this ValidityMapSsh

	this.Processes = make(map[*Ec2Instance]*Process)

	return &this
}

// Try to see if the specified instance is reachable.
// If the process has no associated ssh background process, launch one.
// If it has an associated ssh background process that finished with failure,
// launch a new one.
//
func (this *ValidityMapSsh) UpdateValidity(instance *Ec2Instance,
	timeout *Timeout) {

	var builder *SshProcessBuilder
	var proc *Process
	var found, exited bool
	var exitcode int

	proc, found = this.Processes[instance]

	if found {
		exitcode, exited = proc.ExitCode()
	}

	if (!found || (exited && (exitcode != 0))) {
		builder = BuildSshProcess(instance, []string{"true"})

		if !timeout.IsNone() {
			if timeout.RemainingSeconds() > 15 {
				builder.Timeout(15)
			} else {
				builder.Timeout(timeout.RemainingSeconds())
			}
		}

		this.Processes[instance] = builder.Build()
		this.Processes[instance].Start()
	}
}

// Indicate if the given instance has been reached successfully by ssh.
//
func (this *ValidityMapSsh) IsValid(instance *Ec2Instance) bool {
	var proc *Process
	var found, exited bool
	var exitcode int

	proc, found = this.Processes[instance]

	if found {
		exitcode, exited = proc.ExitCode()
	}

	return (found && exited && (exitcode == 0))
}

// Wait for all the background ssh processes to end.
// Since all the ssh processes launched by this map have a timeout, this method
// is guarantee to return (in a ten of seconds max).
//
func (this *ValidityMapSsh) Finalize() {
	var proc *Process

	for _, proc = range this.Processes {
		proc.WaitFinished()
	}
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// Main waiting loop related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// Update the validity map for all instances contained in the given
// selections.
//
func updateValidityMap(validityMap ValidityMap, selections []*Ec2Selection) {

	var selection *Ec2Selection
	var instance *Ec2Instance

	for _, selection = range selections {
		for _, instance = range selection.Instances {
			validityMap.UpdateValidity(instance,
				waitProcOptionTimeout)
		}
	}
}

// Check if the selection is valid.
// The selection is valid if sufficiently many instances are reported valid
// according to the validity map.
// The "sufficiently many" is defined by the `waitProcOptionCount` global
// variable.
//
func validSelection(selection *Ec2Selection, validityMap ValidityMap) bool {
	var validCount, requiredCount int
	var instance *Ec2Instance
	var maximumCount int = 0
	var fleet *Ec2Fleet

	for _, fleet = range selection.Fleets {
		maximumCount += fleet.Size
	}

	requiredCount = computeRequiredCount(maximumCount)
	validCount = 0

	for _, instance = range selection.Instances {
		if validityMap.IsValid(instance) {
			validCount += 1
		}
	}

	return (validCount >= requiredCount)
}

// Wait for sufficiently many instances to be valid for the given selections.
// Update the given context and update the validity state of the instances
// every seconds.
// If not enough instances are reported valid before the end of the timeout,
// return false, otherwise return true.
//
func waitFleets(ctx *Ec2Index, specs []string) bool {
	var selections []*Ec2Selection
	var selection *Ec2Selection
	var validityMap ValidityMap
	var valid bool

	if *waitParams.OptionWaitFor == "ssh" {
		validityMap = NewValidityMapSsh()
	} else if *waitParams.OptionWaitFor == "ip" {
		validityMap = NewValidityMapIp()
	} else {
		Error("invalid value for option --wait-for: '%s'",
			*waitParams.OptionWaitFor)
	}

	for !waitProcOptionTimeout.IsOver() {
		selections = selectFleets(ctx, specs)

		updateValidityMap(validityMap, selections)

		valid = true
		for _, selection = range selections {
			if !validSelection(selection, validityMap) {
				valid = false
				break
			}
		}

		if valid {
			validityMap.Finalize()
			return true
		}

		time.Sleep(1000 * time.Millisecond)

		UpdateContext(ctx)
	}

	validityMap.Finalize()
	return false
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// Argument parsing and option processing related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

func processOptionCount() {
	var mustEnd = false
	var hasStarted = false
	var number int
	var c rune

	waitProcOptionCount.Type = PROCESSED_OPTION_COUNT_TYPE_NUMBER

	for _, c = range *waitParams.OptionCount {
		if mustEnd {
			Error("invalid value for option --count: '%s'",
				*waitParams.OptionCount)
		}

		if (c >= '0') && (c <= '9') {
			number *= 10
			number += int(c) - '0'
			hasStarted = true
		} else if c == '%' {
			waitProcOptionCount.Type =
				PROCESSED_OPTION_COUNT_TYPE_PERCENT
			mustEnd = true

			if number > 100 {
				Error("invalid value for option --count: '%s'",
					*waitParams.OptionCount)
			}
		} else {
			Error("invalid value for option --count: '%s'",
				*waitParams.OptionCount)
		}
	}

	if !hasStarted {
		Error("invalid value for option --count: '%s'",
			*waitParams.OptionCount)
	}

	waitProcOptionCount.Number = number
}

func processOptionTimeout() {
	if *waitParams.OptionTimeout == "" {
		waitProcOptionTimeout = NewTimeoutNone()
	} else {
		waitProcOptionTimeout =
			NewTimeoutFromSpec(*waitParams.OptionTimeout)
		if waitProcOptionTimeout == nil {
			Error("invalid value for option --timeout: '%s'",
				*waitParams.OptionTimeout)
		}
	}
}

func Wait(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var fleetSpecs []string
	var fleetSpec string
	var ctx *Ec2Index
	var success bool
	var err error

	waitParams.OptionContext = flags.String("context", DEFAULT_WAIT_CONTEXT, "")
	waitParams.OptionCount = flags.String("count", DEFAULT_WAIT_COUNT, "")
	waitParams.OptionTimeout = flags.String("timeout", DEFAULT_WAIT_TIMEOUT, "")
	waitParams.OptionWaitFor = flags.String("wait-for", DEFAULT_WAIT_WAIT_FOR, "")

	flags.Parse(args[1:])

	processOptionCount()
	processOptionTimeout()

	if len(flags.Args()) == 0 {
		fleetSpecs = []string{"@//"}
	} else {
		fleetSpecs = make([]string, 0, len(flags.Args()))
		for _, fleetSpec = range flags.Args() {
			fleetSpecs = append(fleetSpecs, "@"+fleetSpec)
		}
	}

	ctx, err = LoadEc2Index(*waitParams.OptionContext)
	if err != nil {
		Error("no context: %s", *waitParams.OptionContext)
	}

	success = waitFleets(ctx, fleetSpecs)

	StoreEc2Index(*waitParams.OptionContext, ctx)

	if !success {
		os.Exit(1)
	}
}
