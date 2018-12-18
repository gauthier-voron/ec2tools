EC2 Tools
=========

Easy manipulation of AWS EC2 instances from shell.
This repository provides the `ec2tools` command that can be used to launch
several AWS EC2 spot instance fleets among several regions in the world and
easily interact with them from a shell script.

Build instructions
------------------

Be sure to have the [Go tools suite](https://golang.org/dl/) installed, then
build the tools by typing `make all`.

Usage instructions
------------------

The `ec2tools` command contains several sub-commands. The most useful is the
`help` subcommand which describe how other subcommands work. For instance,
type `./ec2tools help launch` to get help on the `launch` subcommand. You can
also invoke the help command with no argument to get a summary of the available
subcommands and of what they do.

The file `example.sh` is an example of how to use the `ec2tools` command to
achieve multi-region tasks in small shell scripts.
