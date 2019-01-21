#!/bin/bash

set -e

# Launch a fleet of 2 instances and wait for it
ec2tools launch --replace --size=2 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet'
ec2tools wait

output=$(mktemp --suffix='.txt', 'test-file.XXXXXXXXXX')

# Make each instance print its own name
ec2tools ssh --output-mode=all-prefix --format echo '%n' | sort > "$output"

# Check results are correct
for name in $(ec2tools get name) ; do
    grep '^\['"$name"'\] '"$name"'$' < "$output"
    test $(grep '^\['"$name"'\] '"$name"'$' < "$output" | wc -l) -eq 1
done

rm "$output"

ec2tools stop

exit 0
