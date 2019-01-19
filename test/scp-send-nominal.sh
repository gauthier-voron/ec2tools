#!/bin/bash

set -e

# Launch a fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet'

# Wait for everyone to be ready
ec2tools wait

# Create a file to send
file=$(mktemp --suffix='.txt' 'test-file.XXXXXXXXX')
cat > "$file" <<EOF
This is a text file !
EOF

# Send it with same base name to everyone
ec2tools scp "$file"

# Send it with specified destimation name to everyone
ec2tools scp "$file" ':my-file.txt'

# Remove local file
rm "$file"

# Test that every one received it with same base name
ec2tools ssh stat "${file##*/}"

# Test that every one received it with specified base name
ec2tools ssh stat 'my-file.txt'

ec2tools stop

exit 0
