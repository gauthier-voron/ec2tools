#!/bin/bash

set -e
set -x

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

# Send it with same base name to everyone with custom full debug ssh
ec2tools scp --command='scp -vvv' "$file" 2> 'error'

# Remove local file
rm "$file"

# Test that the verbose mode produced some output
grep -q 'debug3' 'error'

# Test that every one received it with same base name
ec2tools ssh stat "${file##*/}"

ec2tools stop

exit 0
