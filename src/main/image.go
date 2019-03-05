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

