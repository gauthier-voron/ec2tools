package main

import (
	"fmt"
)

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

func Wait(args []string) {
}
