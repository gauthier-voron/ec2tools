package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strings"
	"time"
)

const (
	IMAGE_STATE_PENDING   string = "pending"
	IMAGE_STATE_AVAILABLE string = "available"
)

// Test if an image specification is an image id.
// A string starting with "ami-" is an image id. Otherwise, it's not.
//
func IsImageId(spec string) bool {
	return ((len(spec) >= 4) && (spec[0:4] == "ami-"))
}

// Test if an image specification is an image name.
// A non empty string which is not an image id is an image name.
//
func IsImageName(spec string) bool {
	return ((len(spec) >= 1) && !IsImageId(spec))
}

// An instance image in the AWS EC2 sense.
// Represents a base image that can be used to launch instances from.
// Each image has a unique identifier generated by AWS, a user specified name
// and textual description, a region where it is stored and a state defining
// if the image can be used or not.
//
type Image struct {
	Id          string // unique id for the image (even across regions)
	Name        string // human readable name
	Description string // human readable description
	State       string // either "", "pending" or "available"
	Region      string // where the image can be used
}

// Create a new image from the specified instance.
// Use the image of the specified instance as a template to create a new image.
// The instance should not be used during this operation.
// Create the image in the region of the instance.
// User supplies human readable name and description.
// Provided name must be different from any other instance in the same region
// (this constraint only exists for image creation).
// Return the new image and a possible error.
//
func CreateImage(instance *Ec2Instance, name, description string) (*Image, error) {
	var errtxt string = "InvalidAMIName.Duplicate: AMI name"
	var region string = instance.Fleet.Region
	var req ec2.CreateImageInput
	var rep *ec2.CreateImageOutput
	var sess *session.Session
	var client *ec2.EC2
	var this Image
	var err error

	sess = session.New()
	client = ec2.New(sess, &aws.Config{
		Region: aws.String(region),
	})

	req.InstanceId = aws.String(instance.Name)
	req.Name = aws.String(name)
	req.Description = aws.String(description)

	rep, err = client.CreateImage(&req)
	if err != nil {
		if strings.Index(err.Error(), errtxt) == 0 {
			return nil, NewImageDuplicateError()
		} else {
			return nil, err
		}
	}

	this.Id = *rep.ImageId
	this.Name = name
	this.Description = description
	this.State = ""
	this.Region = region

	return &this, nil
}

// Create a new image with a given id on the given region.
// Useful to create a local object without any network operation, then refresh
// it later if needed.
//
func NewImage(region, id string) *Image {
	var this Image

	this.Id = id
	this.Name = ""
	this.Description = ""
	this.State = ""
	this.Region = region

	return &this
}

// Update the fields of the Image by asking to AWS EC2 servers.
// Fetch the most up-to-date Name, Description and State from the AWS EC2
// servers.
// This operation never modify the Id or the Region.
// Return nil if the update succeed and if the image still exists./
// If the update succeed but the image does not exist anymore, return am
// ImageUnknownError.
//
func (this *Image) Refresh() error {
	var rep *ec2.DescribeImagesOutput
	var req ec2.DescribeImagesInput
	var sess *session.Session
	var image *ec2.Image
	var client *ec2.EC2
	var err error

	sess = session.New()
	client = ec2.New(sess, &aws.Config{Region: aws.String(this.Region)})

	req.ImageIds = []*string{aws.String(this.Id)}

	rep, err = client.DescribeImages(&req)
	if err != nil {
		return err
	} else if len(rep.Images) < 1 {
		return NewImageUnknownError()
	} else if len(rep.Images) > 1 {
		panic("should not happen")
	} else if *rep.Images[0].ImageId != this.Id {
		panic("should not happen")
	}

	image = rep.Images[0]

	if image.Name != nil {
		this.Name = *image.Name
	} else {
		this.Name = ""
	}

	if image.Description != nil {
		this.Description = *image.Description
	} else {
		this.Description = ""
	}

	this.State = *image.State

	return nil
}

// Copy this image to the specified region.
// The new copy receive the specified name and description.
// The region may be equals to the region of this Image.
// The name and description can be equals to the ones of another image.
// If the copy succeed, return a new Image with a "" State.
// Otherwise, return no Image and an error.
//
func (this *Image) Copy(region, name, description string) (*Image, error) {
	var rep *ec2.CopyImageOutput
	var req ec2.CopyImageInput
	var sess *session.Session
	var client *ec2.EC2
	var copy Image
	var err error

	req.SourceImageId = aws.String(this.Id)
	req.Description = aws.String(description)
	req.Name = aws.String(name)
	req.SourceRegion = aws.String(this.Region)

	sess = session.New()
	client = ec2.New(sess, &aws.Config{Region: aws.String(region)})

	rep, err = client.CopyImage(&req)
	if err != nil {
		return nil, err
	}

	copy.Id = *rep.ImageId
	copy.Name = name
	copy.Description = description
	copy.State = ""
	copy.Region = region

	return &copy, nil
}

// Remove this Image from the AWS EC2 servers.
// This only invokes the EC2 deregister function and report any error that
// occured.
//
func (this *Image) Deregister() error {
	var req ec2.DeregisterImageInput
	var sess *session.Session
	var client *ec2.EC2
	var err error

	sess = session.New()
	client = ec2.New(sess, &aws.Config{Region: aws.String(this.Region)})

	req.ImageId = aws.String(this.Id)

	_, err = client.DeregisterImage(&req)
	return err

}

// Wait for this image to be either "pending" or "available".
// User can specify a Timeout for how long to wait (possibly NewTimeoutNone()).
// Return a boolean to indicate if the image is in the desired state at the
// return of the function and an error to indicate if something gone wrong.
//
func (this *Image) WaitPending(t *Timeout) (bool, error) {
	return this.waitState(t, IMAGE_STATE_PENDING, IMAGE_STATE_AVAILABLE)
}

// Wait for this image to be "available".
// User can specify a Timeout for how long to wait (possibly NewTimeoutNone()).
// Return a boolean to indicate if the image is in the desired state at the
// return of the function and an error to indicate if something gone wrong.
//
func (this *Image) WaitAvailable(t *Timeout) (bool, error) {
	return this.waitState(t, IMAGE_STATE_AVAILABLE)
}

// Wait for this Image to be in one of the specified states.
// Inner function behind WaitPending() and WaitState()
//
func (this *Image) waitState(t *Timeout, states ...string) (bool, error) {
	var state string
	var err error

	for !t.IsOver() {
		for _, state = range states {
			if this.State == state {
				return true, nil
			}
		}

		time.Sleep(1000 * time.Millisecond)

		err = this.Refresh()
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

// A deduplicated list of images from various regions.
// The goal of this structure is to manipulate several Image objects in
// parallel. For instance, refreshing several images in parallel instead of
// sequentially.
//
type ImageList struct {
	Images map[string]*Image
}

// Create a new empty ImageList.
// This function never fails.
//
func NewImageList() *ImageList {
	var this ImageList

	this.Images = make(map[string]*Image)

	return &this
}

// An error indicating that another image with the same name already exists.
//
type ImageDuplicateError struct {
}

// Create a new ImageDuplicateError.
//
func NewImageDuplicateError() *ImageDuplicateError {
	return &ImageDuplicateError{}
}

// Make ImageDuplicateError to be an error.
//
func (this *ImageDuplicateError) Error() string {
	return "ImageDuplicateError"
}

// An error indicating that no image with this name or id exist (in a given
// region).
//
type ImageUnknownError struct {
}

// Create a new ImageUnknownError.
//
func NewImageUnknownError() *ImageUnknownError {
	return &ImageUnknownError{}
}

// Make ImageUnknownError to be an error.
//
func (this *ImageUnknownError) Error() string {
	return "ImageUnknownError"
}
