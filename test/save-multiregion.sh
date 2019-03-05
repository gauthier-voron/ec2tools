#!/bin/bash

set -e
set -x

iname="${0##*/}"

# Launch a template fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 --region='ap-southeast-2' --secgroup='sg-0e9b9bbee1dfc700a' \
	 'template-fleet'

# Wait for the fleet to be ready
ec2tools wait 'template-fleet'

# Modify the instance image
ec2tools ssh touch "ec2tools-test"

# Save the image
ec2tools save --replace --description="test image for $0" \
	 --region='ap-southeast-2,us-east-2' '@template-fleet' \
	 -- "${TEST_IMAGE}/$iname"

# Kill the template fleet
ec2tools stop 'template-fleet'

# Launch a fleet from the save image in the same region
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 --image="${TEST_IMAGE}/$iname" --region='ap-southeast-2' \
	 --secgroup='sg-0e9b9bbee1dfc700a' 'test-fleet-ap'

# Launch a fleet from the save image in another region
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 --image="${TEST_IMAGE}/$iname" --region='us-east-2' \
	 --secgroup='sg-98338af0' 'test-fleet-us'

# Wait for the fleets to be ready
ec2tools wait 'test-fleet-ap' 'test-fleet-us'

# Test the modifications are here in source region
ec2tools ssh '@test-fleet-ap' -- test -f "ec2tools-test"

# Test the modifications are here in remote region
ec2tools ssh '@test-fleet-us' -- test -f "ec2tools-test"

ec2tools stop
ec2tools drop "${TEST_IMAGE}/$iname"

exit 0
