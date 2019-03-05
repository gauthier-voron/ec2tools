package main

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

