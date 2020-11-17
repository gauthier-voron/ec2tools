package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Test if an security group specification is a security group id.
// A string starting with "sg-" is an security group id. Otherwise, it's not.
//
func IsSecurityGroupId(spec string) bool {
	return ((len(spec) >= 3) && (spec[0:3] == "sg-"))
}

// Get the id of the security group with the given name inside the specified
// region.
//
func GetSecurityGroupId(name, region string) (*string, error) {
	var rep *ec2.DescribeSecurityGroupsOutput
	var req ec2.DescribeSecurityGroupsInput
	var sess *session.Session
	var filter ec2.Filter
	var client *ec2.EC2
	var err error

	sess = session.New()
	client = ec2.New(sess, &aws.Config{Region: aws.String(region)})

	filter.Name = aws.String("group-name")
	filter.Values = []*string{aws.String(name)}
	req.Filters = []*ec2.Filter{&filter}

	rep, err = client.DescribeSecurityGroups(&req)

	if err != nil {
		return nil, err
	}

	if len(rep.SecurityGroups) < 1 {
		return nil, NewSecurityGroupUnknownError()
	}

	return rep.SecurityGroups[0].GroupId, nil
}

// An error indicating that no security group with this name exist (in a given
// region).
//
type SecurityGroupUnknownError struct {
}

// Create a new SecurityGroupUnknownError.
//
func NewSecurityGroupUnknownError() *SecurityGroupUnknownError {
	return &SecurityGroupUnknownError{}
}

// Make SecurityGroupUnknownError to be an error.
//
func (this *SecurityGroupUnknownError) Error() string {
	return "SecurityGroupUnknownError"
}
