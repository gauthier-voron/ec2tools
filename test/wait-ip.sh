#!/bin/bash

set -e

# Launch a first fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet-0'

# Launch a second fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet-1'

# Wait for the fleet to exist
ec2tools wait --wait-for='ip'

# Check everyone has an ip (2 instances)
test $(ec2tools get --update ip | wc -l) -eq 2
ret=$?

ec2tools stop

exit $ret
