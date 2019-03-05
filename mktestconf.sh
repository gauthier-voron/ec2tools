#!/bin/bash
#
#   Ask user for AWS EC2 credentials and configuration options required to
#   launch validation tests of ec2tools.
#

# Use colored output:
#   0 -> yes
#   1 -> no
#
COLOR=

# In what file to write the configuration.
#
CONFIG_PATH=


# Parse the command line options.
#
parseopts() {
    local shortopts="$1" ; shift
    local longopts=""
    local separator=''

    while [ $# -gt 0 ] ; do
	if [ "x$1" = 'x--' ] ; then
	    break
	fi
	longopts="${longopts}${separator}$1"
	separator=','
	shift
    done

    getopt -l "$longopts" -o "$shortopts" "$@"
}

# Print an error message on stderr and exit.
#
error() {
    printf "%s: %s\n" "$0" "$1" >&2
    printf "Please type '%s --help' for more information\n" "$0" >&2
    exit 1
}

# Set the COLOR variable according to a --color option value.
#
set_color() {
    case "$1" in
	yes|true|1)  COLOR=0 ;;
	no|false|0)  COLOR=1 ;;
	auto)        test -t 1; COLOR=$? ;;
	*)           error "invalid value for --color: '$1'" ;;
    esac
}

# Indicate 0 if a printer should use color, or 1 if it should not.
#
use_color() {
    return $COLOR
}

# Print the header message on stdout.
#
print_header() {
    if use_color ; then
	printf "\033[1mAutomated tests need some user specific configuration.\033[0m\n"
	echo
	printf "Please answer these questions to configure how to launch AWS instances with\n"
	printf "your EC2 credentials. The test launch instances for quick tests only which\n"
	printf "should not be expensive.\n"
	echo
    else
	printf "Automated tests need some user specific configuration.\n"
	echo
	printf "Please answer these questions to configure how to launch AWS instances with\n"
	printf "your EC2 credentials. The test launch instances for quick tests only which\n"
	printf "should not be expensive.\n"
	echo
    fi
}

# Print the footer message on stdout.
#
print_footer() {
    if use_color ; then
	echo
	printf "Your configuration has been saved in \033[36;1m'%s'\033[0m.\n" "${CONFIG_PATH}"
	printf "This file is a regular shell script containing variable definitions. Feel free\n"
	printf "to edit it at anytime. This configuration file must \033[31;1m*not*\033[0m be versioned.\n"
    else
	echo
	printf "Your configuration has been saved in '%s'.\n" "${CONFIG_PATH}"
	printf "This file is a regular shell script containing variable definitions. Feel free\n"
	printf "to edit it at anytime. This configuration file must *not* be versioned.\n"
    fi
}

# Ask the question given as first argument, then read the user answer and
# store it inside the variable given as second argument.
#
ask() {
    # $1 -> message to print
    # $2 -> destination variable
    if use_color ; then
	printf "\033[34;1m*\033[0;1m %s:\033[0m " "$1"
    else
	printf "* %s: " "$1"
    fi
    eval "read $2"
}

# Dump a list of variables given in arguments in a shell script.
# In this script, each variable is exported with its value.
# Dump the generated script on stdout.
#
dump_config() {
    local var

    printf "#!/bin/bash\n"
    printf "#\n"
    printf "#   Configuration file for runtest.sh\n"
    printf "#   This file is sourced from runtest.sh to get the AWS EC2 credential and\n"
    printf "#   configuration options required to launch test instances.\n"
    printf "#   You can edit this file manually or run %s to generate a new\n"  "$0"
    printf "#   configuration.\n"
    printf "#\n"
    echo

    for var in "$@" ; do
	printf 'export %s="%s"\n' $var "$(eval "echo \$$var")"
    done
}

# Print a usage message for this script on stdout.
#
usage() {
    printf "Usage: %s [options] [<config-path>]\n" "$0"
    echo
    printf "Ask user for AWS EC2 credentials and configuration options required to launch\n"
    printf "validation tests of ec2tools.\n"
    echo
    printf "Options:\n"
    printf "  -c, --color <yes|no|auto>             use colored output\n"
    printf "  -h, --help                            print this message and exit\n"
    printf "  -V, --version                         print version information and exit\n"
}

# Print version information for this script on stdout.
#
version() {
    printf "%s %s\n" 'mktestconf.sh' '1.0.0'
    printf "%s\n" 'Gauthier Voron'
    printf "%s\n" '<gauthier.voron@sydney.edu.au>'
}

# - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
#                           Main script starts here
# - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

# Step 1: Parsing of options and arguments
#

OPT_SHORT='c:hV'
OPT_LONG=('color:' 'help' 'version')

eval "set -- `parseopts "$OPT_SHORT" "${OPT_LONG[@]}" -- "$@"`"

while true ; do
    case "$1" in
	-c|--color)    shift; set_color "$1" ;;
	-h|--help)     usage; exit 0 ;;
	-V|--version)  version; exit 0 ;;
	--)            shift; break ;;
    esac
    shift
done

if [ "x$1" = 'x' ] ; then
    CONFIG_PATH='.runtest.conf'
elif [ -e "$1" -a ! -f "$1" ] ; then
    error "invalid operand config-path: not a file"
else
    CONFIG_PATH="$1"
fi

if [ "x$COLOR" = 'x' ] ; then
    set_color 'auto'
fi


# Step 2: Configuration gathering
# Ask the user to answer questions.
#

print_header

ask "Ssh key to use for AWS instances" TEST_KEY
ask "Price to use for AWS instances (USD)" TEST_PRICE
ask "Name to use for created AWS images" TEST_IMAGE


# Step 3: Write the configuration on disk
# Dump all the assigned variables to the config file.
#

dump_config TEST_KEY TEST_PRICE TEST_IMAGE > "${CONFIG_PATH}"
print_footer

exit 0
