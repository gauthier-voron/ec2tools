#!/bin/bash

set -e

# Launch a fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet-0'

# Launch a fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet-1'

# Wait for everyone to be ready
ec2tools wait

# Create a file to send
file=$(mktemp --suffix='.txt' 'test-file.XXXXXXXXX')
cat > "$file" <<EOF
This is a text file !
EOF

# Send it with same base name to first fleet
ec2tools scp '@test-fleet-0' -- "$file"

# Send it with specified destimation name to second fleet
ec2tools scp '@test-fleet-1' -- "$file" ':my-file.txt'

# Remove local file
rm "$file"

# Test that first fleet received it with same base name
ec2tools ssh '@test-fleet-0' -- stat "${file##*/}"

# Test that second fleet received it with specified base name
ec2tools ssh '@test-fleet-1' -- stat 'my-file.txt'

ec2tools stop

exit 0
