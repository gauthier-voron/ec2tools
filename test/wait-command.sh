#!/bin/bash

set -x
set -e

# Launch a fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet'

# Wait for every fleet to be ready with custom full debug ssh
ec2tools wait --command="ssh -vvv" 'test-fleet' 2> 'error'

# Test that the verbose mode produced some output
grep -q 'debug3' 'error'

# Perform an ssh command
ec2tools ssh uname

ec2tools stop

exit 0
