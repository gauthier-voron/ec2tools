#!/bin/bash

set -e
set -x

# Launch a fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet-0'

# Launch a fleet (1 instance)
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet-1'

# Wait for everyone to be ready
ec2tools wait

# Generate file names
file=$(mktemp --suffix='.txt' 'test-file.XXXXXXXXX')
rm "$file"

# Create a remote file to receive
ec2tools ssh touch "${file##*/}"

# Receive it with intance name from second fleet
ec2tools scp '@test-fleet-0' -- ":${file##*/}" "%n.txt"

# Test the received files with instance names
for instance in $(ec2tools get '@test-fleet-0' -- name) ; do
    test -f "$instance.txt"
    rm "$instance.txt"
done

ec2tools stop

exit 0
