EC2 Tools
=========

Easy manipulation of AWS EC2 instances from shell.
This repository provides the `ec2tools` command that can be used to launch
several AWS EC2 spot instance fleets among several regions in the world and
easily interact with them from a shell script.

Build instructions
------------------

Be sure to have the [Go tools suite](https://golang.org/dl/) installed, then
build the tools by typing `make all`.

Usage instructions
------------------

The `ec2tools` command contains several sub-commands. The most useful is the
`help` subcommand which describe how other subcommands work. For instance,
type `./ec2tools help launch` to get help on the `launch` subcommand. You can
also invoke the help command with no argument to get a summary of the available
subcommands and of what they do.

#### Launch a new fleet and use it:
```
# Launch a new fleet of 3 c5.large instances in Ohio
ec2tools launch --key=my-aws-key --region=us-east-2 --image='ami-965e6bf3' \
                --user='ubuntu' --type='c5.large' --price=0.03 --size=3    \
                --secgroup='sg-98338af0' 'my-fleet-ohio'

# Wait for every instances to be ready to receive ssh commands
ec2tools wait

# Say hello
ec2tools ssh uname -a

# Stop every instances
ec2tools stop
```

#### Launch two fleets and control them separately:
```
# Launch a new fleet of 2 c5.large instances in Ohio
ec2tools launch --key=my-aws-key --region=us-east-2 --image='ami-965e6bf3' \
                --user='ubuntu' --type='c5.large' --price=0.03 --size=2    \
                --secgroup='sg-98338af0' 'my-fleet-ohio'

# Launch a new fleet of 4 c4.large instances in Sydney
ec2tools launch --key=my-aws-key --region=ap-southeast-2 --user='ec2-user' \
                --image='ami-942dd1f6' --type='c4.large' --price=0.033     \
                --size=4 --secgroup='sg-0e9b9bbee1dfc700a' 'my-fleet-sydney'

# Wait for every instances to be ready to receive ssh commands
ec2tools wait

# Say hello on Sydney
ec2tools ssh '@my-fleet-sydney' -- uname -a

# Say hello on Ohio
ec2tools ssh '@my-fleet-ohio' -- uname -a

# Make every instance print its name and its region
ec2tools ssh --format echo 'name: %n  //  region: %r'

# Stop every instances of the sydney fleet
ec2tools stop 'my-fleet-sydney'

# Stop every instances of the ohio fleet
ec2tools stop 'my-fleet-ohio'
```

#### Create a template image and use it to launch fleets:
```
# Launch a new fleet of 1 c5.large instance in Ohio
ec2tools launch --key=my-aws-key --region=us-east-2 --image='ami-965e6bf3' \
                --user='ubuntu' --type='c5.large' --price=1 --size=1    \
                --secgroup='sg-98338af0' 'my-template-fleet'

# Wait the fleet instances to be ready to receive ssh commands
ec2tools wait

# Modify the image
ec2tools scp my-binaries my-libraries

# Save the image of the fleet instance in both Ohio and Sydney
ec2tools save --region='us-east-2,ap-southeast-2' 'my-ec2tools-image'

# Stop the template fleet
ec2tool stop

# Launch a new fleet of 2 c5.large instances in Ohio basing on the new image
ec2tools launch --key=my-aws-key --region=us-east-2 --price=1 --size=2 \
                --user='ubuntu' --type='c5.large' --secgroup='sg-98338af0' \
                --image='my-ec2tools-image' 'my-fleet-ohio'

# Launch a new fleet of 7 c4.large instances in Sydney
ec2tools launch --key=my-aws-key --region=ap-southeast-2 --price=1 \
                --size=7 --user='ubuntu' --type='c4.large' \
                --secgroup='sg-0e9b9bbee1dfc700a' \
                --image='my-ec2tools-image' 'my-fleet-sydney'

# Wait the fleet instances to be ready
ec2tools wait

# Delete the new image from the Amazon servers
ec2tools drop 'my-ec2tools-image'
```

#### Send and receive files:
```
# Launch instances and wait for them
ec2tools launch ...
ec2tools wait

# Send several local files in a specific directory
ec2tools scp 'local-file-0' 'local-file-1' ':remote-directory'

# Receive a remote file with a name depending on the sending instance
ec2tools scp ':remote-file.txt' 'local-pattern-%n.txt'

# Create one directory per instance, tagged with the fleet name and the
# instance ID
ec2tools get fleet fiid | while read fleet fiid ; do
    mkdir "local-directory-$fleet-$fiid"
done

# Receive several remote files in an instance specific directory
ec2tools scp ':remote-file-0' ':remote-file-1' 'local-directory-%f-%d'

# Stop all instances
ec2tools stop
```

#### Tag instances with custom properties:
```
# Launch instances and wait for them
ec2tools launch ...
ec2tools wait

# Tag every instances in the fleet 'my-fleet-sydney'
ec2tools set '@my-fleet-sydney' -- 'my-tag' 18

# Tag every instances having a name starting with 'i-03'
ec2tools set '/^i-03/' -- 'my-tag' 37

# List every instance public IP tagged with property 'my-tag'
ec2tools get --defined 'my-tag' ip

# Use tags in ssh command
ec2tools ssh --format '@my-fleet-sydney' -- echo 'my tag is %{my-tag}'

# Stop all instances
ec2tools stop
```
