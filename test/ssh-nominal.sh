#!/bin/bash

set -e

# Launch a fleet of 1 instance and wait for it
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet'
ec2tools wait

# Invoke remote command on all instances
ec2tools ssh true

ec2tools stop

exit 0
