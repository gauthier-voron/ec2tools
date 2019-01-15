#!/bin/bash

set -e

# Launch a fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet'

# Wait for the fleet to be ready
ec2tools wait --wait-for='ssh' 'test-fleet'

# Perform an ssh command
ec2tools ssh uname

ec2tools stop

exit 0
