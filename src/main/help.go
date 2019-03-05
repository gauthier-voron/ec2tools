package main

import (
	"fmt"
)

func PrintHelpUsage() {
	fmt.Printf(`Usage: %s help <command>

Display a help message for the specified command.
`,
		PROGNAME)
}

func Help(args []string) {
	var command string

	if len(args) > 2 {
		Error("unexpected operand: %s", args[2])
	}

	if len(args) < 2 {
		PrintUsage()
		return
	}

	command = args[1]

	if command == "get" {
		PrintGetUsage()
	} else if command == "help" {
		PrintHelpUsage()
	} else if command == "launch" {
		PrintLaunchUsage()
	} else if command == "save" {
		PrintSaveUsage()
	} else if command == "scp" {
		PrintScpUsage()
	} else if command == "set" {
		PrintSetUsage()
	} else if command == "ssh" {
		PrintSshUsage()
	} else if command == "stop" {
		PrintStopUsage()
	} else if command == "update" {
		PrintUpdateUsage()
	} else if command == "wait" {
		PrintWaitUsage()
	} else {
		Error("invalid command operand: %s", command)
	}
}
