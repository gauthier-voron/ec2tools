#!/bin/bash

set -e

# Launch a fleet of 1 instance and wait for it
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet'
ec2tools wait

# Invoke remote command on all instances with custom full debug ssh
ec2tools ssh --command='ssh -vvv' true 2> 'error'

# Test that the verbose mode produced some output
grep -q 'debug3' 'error'

ec2tools stop

exit 0
