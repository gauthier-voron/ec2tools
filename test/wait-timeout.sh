#!/bin/bash

set -e

# Should not launch (price too low) and hang for a long time
ec2tools launch --replace --size=1 --price=0.001 --key="$TEST_KEY" 'test-fleet'

# Should stop after 2 seconds
ec2tools wait --timeout=2 'test-fleet' &
pid=$!

sleep 3

# If it did not stop, kill it.
if ps $pid > /dev/null 2>&1 ; then
    kill -TERM $pid
    sleep 3
    if ps $pid > /dev/null 2>&1 ; then
	kill -KILL $pid
    fi

    ec2tools stop
    exit 1
fi

# On timeout, ec2tools should exit with a failure code.
! wait $pid

ec2tools stop

exit 0
