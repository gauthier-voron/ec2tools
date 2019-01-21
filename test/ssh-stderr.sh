#!/bin/bash

set -e

# Launch a fleet of 1 instance and wait for it
ec2tools launch --replace --size=1 --price="$TEST_PRICE" --key="$TEST_KEY" \
	 'test-fleet'
ec2tools wait

# Create a script that print interleaved output and error before to exit
script=$(mktemp --suffix='.sh' 'test-script.XXXXXXXXXX')
cat > "$script" <<EOF
#!/bin/bash

echo "stderr 0" >&2
sleep 1
echo "stdout 0" >&1
sleep 1
echo "stderr 1" >&2
sleep 1
echo "stdout 1" >&1

exit 0
EOF
chmod 755 "$script"

# Upload this script and remove it from local site
ec2tools scp "$script"
rm "$script"

echo 'Upload successful'

# Create pipes to receive output and error from the remote instance
# Launch the background processes to read and register each stream input dates
testdir=$(mktemp -d --suffix='.d' 'test-dir.XXXXXXXXXX')
mkfifo "$testdir/stdout.fifo"
mkfifo "$testdir/stderr.fifo"

while read line ; do
    date '+%s'
done < "$testdir/stdout.fifo" > "$testdir/stdout.log" &
opid=$!

while read line ; do
    date '+%s'
done < "$testdir/stderr.fifo" > "$testdir/stderr.log" &
epid=$!

# Launch the remote script and wit for completion
ec2tools ssh "./$script" 1> "$testdir/stdout.fifo" 2> "$testdir/stderr.fifo"

# Wait a second, the time for the background processes to terminate
sleep 1

# If they are not terminated now, kill them
if ps $opid > /dev/null ; then
    kill $opid
fi
if ps $epid > /dev/null ; then
    kill $epid
fi

echo 'Execution successful'

# Check each stream received exactly two lines
test $(cat "$testdir/stdout.log" | wc -l) -eq 2
test $(cat "$testdir/stderr.log" | wc -l) -eq 2

echo 'Count of lines correct'

# Check that the two stream are interleaved
out0=$(tail -n +1 "$testdir/stdout.log" | head -n 1)
out1=$(tail -n +2 "$testdir/stdout.log" | head -n 1)
err0=$(tail -n +1 "$testdir/stderr.log" | head -n 1)
err1=$(tail -n +2 "$testdir/stderr.log" | head -n 1)

echo 'Sequence of lines:'
printf 'err0: %d\n' "$err0"
printf 'out0: %d\n' "$out0"
printf 'err1: %d\n' "$err1"
printf 'out1: %d\n' "$out1"

test $err0 -le $out0
test $out0 -le $err1
test $err1 -le $out1

echo 'Order of lines correct'

# Remove the pipes and log files
rm -rf "$testdir"

exit 0
