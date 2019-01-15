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

var waitProcOptionTimeout int

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

func validInstance(instance *Ec2Instance) bool {
	if *waitParams.OptionWaitFor == "ip" {
		return instance.PublicIp != ""
	} else if *waitParams.OptionWaitFor == "ssh" {
		Error("not yet implemented value for option --wait-for: 'ssh'")
		return false
	} else {
		Error("invalid value for option --wait-for: '%s'",
			*waitParams.OptionWaitFor)
		return false
	}
}

func validSelection(selection *Ec2Selection) bool {
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
		if validInstance(instance) {
			validCount += 1
		}
	}

	return (validCount >= requiredCount)
}

func validContext(ctx *Ec2Index, fleetSpecs []string) bool {
	var selection *Ec2Selection
	var fleetSpec string
	var err error

	for _, fleetSpec = range fleetSpecs {
		selection, err = ctx.Select([]string{fleetSpec})
		if err != nil {
			Error("invalid specification: %s", err.Error())
		}

		if !validSelection(selection) {
			return false
		}
	}

	return true
}

func WaitFleets(ctx *Ec2Index, fleetSpecs []string) bool {
	var elapsedSecs int = 0

	for !validContext(ctx, fleetSpecs) {
		if waitProcOptionTimeout > 0 {
			if elapsedSecs >= waitProcOptionTimeout {
				return false
			}
		}

		time.Sleep(1000 * time.Millisecond)
		elapsedSecs += 1

		UpdateContext(ctx)
	}

	return true
}

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
	var secs int64

	if *waitParams.OptionTimeout == "" {
		waitProcOptionTimeout = 0
		return
	}

	// Thanks to launch.go
	secs = timespecToSec(*waitParams.OptionTimeout)

	if secs < 0 {
		Error("invalid value for option --timeout: '%s'",
			*waitParams.OptionTimeout)
	}

	waitProcOptionTimeout = int(secs)
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

	success = WaitFleets(ctx, fleetSpecs)

	StoreEc2Index(*waitParams.OptionContext, ctx)

	if !success {
		os.Exit(1)
	}
}
