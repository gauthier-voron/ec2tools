#!/bin/bash

set -e
set -x

# Launch a fleet (2 instances)
ec2tools launch --replace --size=2 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet'

# Wait for everyone to be ready
ec2tools wait

# Generate file names
file=$(mktemp --suffix='.txt' 'test-file.XXXXXXXXX')
rm "$file"

# Create a remote file to receive
ec2tools ssh touch "${file##*/}"

# Receive it with instance base name
ec2tools scp ":${file##*/}" "%n.txt"

# Test the received files with instance names
for instance in $(ec2tools get name) ; do
    test -f "$instance.txt"
    rm "$instance.txt"
done

ec2tools stop

exit 0
