#!/bin/bash

set -e

# Create a fake context as we wokr locally
cp 'test/context-10instances-sydney.json' '.ec2tools'

# Set a non empty 'test-property' for one instance
ec2tools set '/^i-0c/' -- 'test-property' 'on'

# Set an empty 'test-property' for one instance
ec2tools set '/^i-0e/' -- 'test-property' ''

# Every instance has a value (empty if never set)
test $(ec2tools get 'test-property' | wc -l) -eq 10

# Only one has a non empty value
test $(ec2tools get 'test-property' | grep -v '^$' | wc -l) -eq 1

# Only two have a defined value
test $(ec2tools get --defined 'test-property' | wc -l) -eq 2

# Remove the fake context so nobody complains
rm '.ec2tools'

exit 0
