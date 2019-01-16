#!/bin/bash
#
#  Run test scripts located in 'test' or specified on the command line. Each
#  script is a validation test for ec2tools. A script with an exit code of 0
#  is successful, otherwise it is a failure.
#


# Use colored output:
#   0 -> yes
#   1 -> no
#
COLOR=

# In what file to write the configuration.
#
CONFIG_PATH=


__NTEST_TOTAL=0
__NTEST_FAIL=0
__LTEST_FAIL=""


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

# Set the CONFIG_PATH variable according to a --config option value.
#
set_config() {
    if [ "x$1" = 'x' ] ; then
	error "invalid value for --config: ''"
    elif [ -e "$1" -a ! -f "$1" ] ; then
	error "invalid value for --config: '$1' (not a file)"
    fi

    CONFIG_PATH="$1"
}

# Indicate 0 if a printer should use color, or 1 if it should not.
#
use_color() {
    return $COLOR
}

# Print the test status: running with an optional progress string.
#
print_test_running() {
    local script="$1" ; shift
    local progress="$1" ; shift

    if use_color ; then
	printf "\r\033[34;1m==>\033[0;1m %s: %s \033[0m" "$script" "$progress"
    else
	printf "\r==> %s: %s " "$script" "$progress"
    fi
}

# Print the test is successful.
#
print_test_success() {
    local script="$1" ; shift

    if use_color ; then
	printf "\r\033[32;1m==>\033[0;1m %s: \033[32;1msuccess\033[0m                  \n" "$script"
    else
	printf "\r==> %s: success                            \n" "$script"
    fi
}

# Print the test has failed and show its logfile content to help diagnosis.
#
print_test_failure() {
    local script="$1" ; shift
    local logfile="$1" ; shift

    if use_color ; then
	printf "\r\033[31;1m==>\033[0;1m %s: \033[31;1mfailure\033[0m                  \n" "$script"
	cat "$logfile" | perl -wple '
	    s/^/  \033[35m>\033[0m /;
        '
    else
	printf "\r==> %s: failure                            \n" "$script"
	cat "$logfile" | sed -r 's/^/  > /'
    fi
}

# Print the summary of failed tests.
# If no failed tests, print that everything is fine and return 0.
# Otherwise, print the list of failed tests and return 1.
#
print_summary() {
    local script

    echo

    if [ $__NTEST_FAIL -eq 0 ] ; then
	if use_color ; then
	    printf "\033[32;1m::\033[0;1m All tests successful\033[0m\n"
	else
	    printf ":: All tests successful\n"
	fi
	return 0
    else
	if use_color ; then
	    printf "\033[31;1m::\033[0;1m %d/%d tests failed\033[0m\n" \
		   $__NTEST_FAIL $__NTEST_TOTAL
	else
	    printf ":: %d/%d tests failed\n" \
		   $__NTEST_FAIL $__NTEST_TOTAL
	fi
	for script in $__LTEST_FAIL ; do
	    if use_color ; then
		printf "  \033[31;1m->\033[0m %s\n" "$script"
	    else
		printf "  -> %s\n" "$script"
	    fi
	done
	return 1
    fi
}

# Get the configuration content.
# If no configuration exists, create it with the mktestconfig.sh script.
# If so, let the user to read the complete footer of this script before to go
# on the tests as it might be important.
#
acquire_config() {
    local subcolor ret

    if [ ! -e "${CONFIG_PATH}" ] ; then
	if use_color ; then
	    subcolor='yes'
	else
	    subcolor='no'
	fi

	./mktestconf.sh --color="$subcolor" "${CONFIG_PATH}"
	ret=$?

	echo
	printf "Type ENTER to continue "
	read subcolor
	echo

	if [ $ret -ne 0 ] ; then
	    return 1
	fi
    fi

    source "${CONFIG_PATH}"
}

# Launch a script in background, redirecting all its output in the given
# logfile and writing its pid in the given pidfile.
#
run_script() {
    local script="$1" ; shift
    local logfile="$1" ; shift
    local pidfile="$1" ; shift
    local pid

    (
	PATH=.:"$PATH"

	./"$script" > "$logfile" 2>&1
    ) &
    pid=$!

    echo $pid > "$pidfile"
}

# Wait for a script with the given PID for the given number of seconds.
# At each second, update a progress status so the user does not think the test
# process is frozen.
# If the script reaches the timeout, then kill it and kill every invocation of
# ec2tools, just to be sure...
# Return the exit status of the process with the given PID.
#
wait_script() {
    local pid="$1" ; shift
    local secs="$1" ; shift
    local script="$1" ; shift

    count=$secs
    while ps $pid > /dev/null 2>&1 ; do
	sleep 1
	count=$(( count - 1 ))
	print_test_running "$script" "${count}s / ${secs}s"
	if [ $count -eq 0 ] ; then
	    kill -KILL $pid
	    killall -KILL ec2tools
	fi
    done

    wait $pid
}

# Account of a test success or failure.
# Also print if the test has failed or not.
# If it has failed, print the logfile content.
#
account_ret() {
    local script="$script" ; shift
    local ret="$ret" ; shift
    local logfile="$1" ; shift

    if [ $ret -eq 0 ] ; then
	print_test_success "$script"
    else
	print_test_failure "$script" "$logfile"
	__NTEST_FAIL=$(( __NTEST_FAIL + 1 ))
	__LTEST_FAIL="${__LTEST_FAIL} $script"
    fi

    __NTEST_TOTAL=$(( __NTEST_TOTAL + 1 ))
}

# Run completely a test and account for its result.
# Redirect every test script outputs in a log file. If the test is successful,
# do not print garbage.
# Set a timeout for each script to not hang indefinitely.
# Be sure that every ec2 instance has been stopped before to go to the next
# test.
#
run_test() {
    local script="$1" ; shift
    local logfile pidfile pid ret

    logfile=$(mktemp --suffix='.log' 'test-log.XXXXXXXXXX')
    pidfile=$(mktemp --suffix='.pid' 'test-pid.XXXXXXXXXX')

    print_test_running "$script" ''

    run_script "$script" "$logfile" "$pidfile"
    pid=$(cat "$pidfile")
    rm "$pidfile"

    wait_script "$pid" 30 "$script"
    ret=$?

    ec2tools stop 2> /dev/null

    account_ret "$script" $ret "$logfile"

    rm "$logfile"
}


# Print a usage message for this script on stdout.
#
usage() {
    printf "Usage: %s [options] [<test-scripts...>]\n" "$0"
    echo
    printf "Run test scripts located in 'test' or specified on the command line. Each\n"
    printf "script is a validation test for ec2tools. A script with an exit code of 0 is\n"
    printf "successful, otherwise it is a failure.\n"
    echo
    printf "Options:\n"
    printf "  -c, --color <yes|no|auto>             use colored output\n"
    printf "  -C, --config <path>                   use a custom configuration file\n"
    printf "  -h, --help                            print this message and exit\n"
    printf "  -V, --version                         print version information and exit\n"
}

# Print version information for this script on stdout.
#
version() {
    printf "%s %s\n" 'runtest.sh' '1.0.0'
    printf "%s\n" 'Gauthier Voron'
    printf "%s\n" '<gauthier.voron@sydney.edu.au>'
}

# - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
#                           Main script starts here
# - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

# Step 1: Parsing of options and arguments
#

OPT_SHORT='c:C:hV'
OPT_LONG=('color:' 'config:' 'help' 'version')

eval "set -- `parseopts "$OPT_SHORT" "${OPT_LONG[@]}" -- "$@"`"

while true ; do
    case "$1" in
	-c|--color)    shift; set_color "$1" ;;
	-C|--config)   shift; set_config "$1" ;;
	-h|--help)     usage; exit 0 ;;
	-V|--version)  version; exit 0 ;;
	--)            shift; break ;;
    esac
    shift
done

if [ "x${CONFIG_PATH}" = 'x' ] ; then
    set_config '.runtest.conf'
fi
if [ "x$COLOR" = 'x' ] ; then
    set_color 'auto'
fi


# Step 2: Get the AWS EC2 credentials and configuration options
# If a configuration file exists, use it, otherwise launch the script to create
# it with the user's help.
# If we fail, we cannot continue
#

if ! acquire_config ; then
    error "cannot configure test scripts: aborting"
fi


# Step 3: Run the tests
# If there are some tests specified then run them, otherwise run the executable
# files inside the 'test/' directory.
#

if [ $# -eq 0 ] ; then
    for script in test/* ; do
	if [ ! -x "$script" ] ; then
	    continue
	fi
	run_test "$script"
    done
else
    for script in "$@" ; do
	run_test "$script"
    done
fi


# Step 4: Print the test summary
# Useful to indicate clearly what has failed.
#

print_summary
