#!/bin/bash

set -e

iname="${0##*/}"

# Launch a template fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'template-fleet'

# Wait for the fleet to be ready
ec2tools wait 'template-fleet'

# Modify the instance image
ec2tools ssh touch "ec2tools-test"

# Save the image
ec2tools save --replace --description="test image for $0" '@template-fleet' \
	 -- "${TEST_IMAGE}/$iname"

# Modify the instance image again
ec2tools ssh touch "ec2tools-test-again"

# Replace the saved image
ec2tools save --replace --description="test image for $0" '@template-fleet' \
	 -- "${TEST_IMAGE}/$iname"

# Kill the template fleet
ec2tools stop 'template-fleet'

# Launch a fleet from the save image
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 --image="${TEST_IMAGE}/$iname" 'test-fleet'

# Wait for the fleet to be ready
ec2tools wait 'test-fleet'

# Test the modifications are here
ec2tools ssh test -f "ec2tools-test"
ec2tools ssh test -f "ec2tools-test-again"

ec2tools stop
ec2tools drop "${TEST_IMAGE}/$iname"

exit 0
