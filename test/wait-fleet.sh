#!/bin/bash

set -e

# Should success
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet'

# Should not launch (price too low) and hang for a long time
ec2tools launch --replace --size=3 --price=0.001 --key="$TEST_KEY" \
	 'never-fleet'

# Wait only the successful fleet
ec2tools wait --wait-for='ip' 'test-fleet'

# Check the waited fleet has an ip (1 instance)
test $(ec2tools get --update ip | wc -l) -eq 1
ret=$?

ec2tools stop

exit $ret
