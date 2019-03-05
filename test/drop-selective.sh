#!/bin/bash

set -e
set -x

iname="${0##*/}"

# Launch a template fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 --region='ap-southeast-2' 'template-fleet'

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

# Drop the image in all regions
ec2tools drop --region='us-east-2' "${TEST_IMAGE}/$iname"

# Describe the image, their should be no region us-east-2 but still a region
# ap-southeast-2
ec2tools describe "${TEST_IMAGE}/$iname" > 'output'
! grep -q 'us-east-2' < 'output'
grep -q  'ap-southeast-2' < 'output'

exit 0
