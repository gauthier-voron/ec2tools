#!/bin/bash

set -e

# Create a fake context as we wokr locally
cp 'test/context-10instances-sydney.json' '.ec2tools'

# Create a new property 'test-property' different for each instance
# Test that the property has been set correctly
ec2tools get name | while read name ; do
    prop="${name:2:3}${name:7:3}"
    ec2tools set "$name" -- 'test-property' "$prop"

    gprop=$(ec2tools get "$name" -- 'test-property')
    test "x$gprop" = "x$prop"
done

# Remove the fake context so nobody complains
rm '.ec2tools'

exit 0
