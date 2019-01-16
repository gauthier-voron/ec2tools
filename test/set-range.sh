#!/bin/bash

set -e

# Create a fake context as we wokr locally
cp 'test/context-10instances-sydney.json' '.ec2tools'

# Set a 'test-property' for a range of instances
ec2tools set '/^i-0[14]/' '/^i-0c/' -- 'test-property' 'on'

# Every instance has a value (empty if never set)
test $(ec2tools get 'test-property' | wc -l) -eq 10

# Only previously selected instances has a non empty property
test $(ec2tools get 'test-property' | grep -v '^$' | wc -l) -eq 5

# Remove the fake context so nobody complains
rm '.ec2tools'

exit 0
