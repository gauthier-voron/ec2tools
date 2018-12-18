#!/bin/bash

# Launch several fleets on different regions:
#   - the leader fleet in Sydney with one c4.large Amazon AMI
#   - the sydney fleet in Sydney with three c5.large Ubuntu 16
#   - the ohio fleet in Ohio with three c5.large Ubuntu 16
#

./ec2tools launch --price=0.033 --size=1 --region=ap-southeast-2 \
	   --key=gauthier --image='ami-942dd1f6' --user='ec2-user'   \
	   --type='c4.large' --secgroup='sg-0e9b9bbee1dfc700a' leader

./ec2tools launch --price=0.035 --size=3 --region=ap-southeast-2 \
	   --key=gauthier --image='ami-33ab5251' --user='ubuntu' \
	   --type='c5.large' --secgroup='sg-0e9b9bbee1dfc700a' sydney

./ec2tools launch --price=0.03 --size=3 --region=us-east-2 \
	   --key=gauthier --image='ami-965e6bf3' --user='ubuntu' \
	   --type='c5.large' --secgroup='sg-98338af0' ohio


# Wait that each fleet is fully provisioned before to continue
#

printf "Waiting for instances to boot"
while [ $(./ec2tools get --update instances leader | wc -l) -ne 1 ] ; do
    sleep 1
    printf "."
done
while [ $(./ec2tools get --update instances sydney | wc -l) -ne 3 ] ; do
    sleep 1
    printf "."
done
while [ $(./ec2tools get --update instances ohio | wc -l) -ne 3 ] ; do
    sleep 1
    printf "."
done
echo


# Wait that each instance respond to ssh commands
#

printf "Waiting for instances connection"
while [ $(./ec2tools ssh --output-mode=all-prefix --timeout=10 uname | wc -l) \
	    -lt 7 ] ; do
    sleep 1
    printf "."
done
echo


# Write a script to execute on each instance and send it through scp
#

script=$(mktemp --suffix='.sh' 'script.XXXXXXXXXX')
{
    echo "#!/bin/sh"
    echo "uname -a"
    echo "ifconfig > network-stats.txt"
} > "$script"
chmod 755 "$script"
./ec2tools scp "$script"


# Execute the script on each instance and get the result back
#

./ec2tools ssh "./$script"
./ec2tools scp 'network-stats-%f-%d.txt' 'network-stats.txt'


# Stop all fleets and remove temporary files
#

./ec2tools stop
rm "$script" 'network-stat-'*'.txt'
