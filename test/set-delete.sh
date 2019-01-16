#!/bin/bash

set -e

# Create a fake context as we wokr locally
cp 'test/context-10instances-sydney.json' '.ec2tools'

# Set a non empty 'test-property' for two instances
ec2tools set '/^i-01/' -- 'test-property' 'on'

# Only two have a defined value
test $(ec2tools get --defined 'test-property' | wc -l) -eq 2

# Set one of them to empty
ec2tools set '/^i-016' -- 'test-property' ''

# Still two have a defined value (one of them is empty)
test $(ec2tools get --defined 'test-property' | wc -l) -eq 2

# Delete one of them
ec2tools set --delete '/^i-016/' -- 'test-property'

# Now only one has a defined value
test $(ec2tools get --defined 'test-property' | wc -l) -eq 1

# Remove the fake context so nobody complains
rm '.ec2tools'

exit 0
