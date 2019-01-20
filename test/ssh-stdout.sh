#!/bin/bash

set -e

# Launch a fleet of 2 instances and wait for it
ec2tools launch --replace --size=2 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet'
ec2tools wait

# Create one directory and two files inside
ec2tools ssh mkdir 'test-dir'
ec2tools ssh touch 'test-dir/foo'
ec2tools ssh touch 'test-dir/bar'

# Check that ls -la on the test directory on remote gives 5 lines
test $(ec2tools ssh ls -la 'test-dir' | wc -l) -eq 5

ec2tools stop

exit 0
