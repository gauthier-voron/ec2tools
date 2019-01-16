package main

import (
	"fmt"
)

func PrintSetUsage() {
	fmt.Printf(`Usage: %s set [options] [<instances-specs...> --] <property> <value>
       %s set [options] --delete [<instances-specs...> --] <property>

Set an abritrary property for one or many instances.
If no instance is specified, set the property for all instances.
The property is specified by an arbitrary name. The only restrictions are that
it is different from the builtin properties listed in the 'get' subcommand
help message and it cannot be empty.
The property value is an arbitrary string that may be empty.
The first syntax set a property value, the second syntax delete a property.
There is a difference between an defined but empty property and an undefined
property.

Options:

  --context <path>            path of the context file (default: '%s')
`,
		PROGNAME, PROGNAME, DEFAULT_CONTEXT)
}

func Set(args []string) {
}
