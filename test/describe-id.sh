#!/bin/bash

set -e
set -x

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

# Kill the template fleet
ec2tools stop 'template-fleet'

# Describe the image by name
ec2tools describe "${TEST_IMAGE}/$iname" > 'output-name'

# Get the id of the image
id=$(perl -wnle '/(ami-\S+)/ and print $1' 'output-name')

# Describe the image by name
ec2tools describe "$id" > 'output-id'

# Compare outputs
diff 'output-id' 'output-name'

ec2tools drop "${TEST_IMAGE}/$iname"

exit 0
