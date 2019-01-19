package main

// The property of a given instance.
// A property can be either a trait or an attribute.
// A trait is an inherent and predefined property of the instance, like its
// public ip or its fleet name.
// An attribute is a proerty defined by user at runtime.
//
type Property struct {
	Defined   bool         // the property is defined
	Attribute bool         // the property is a user defined attribute
	Name      string       // property name
	Value     string       // property value
	Instance  *Ec2Instance // instance with this property
}

// Return the fleet name property of the instance.
//
func GetFleet(instance *Ec2Instance) *Property {
	return nil
}

// Return the fleet instance identifier (fiid) property of the instance.
//
func GetFiid(instance *Ec2Instance) *Property {
	return nil
}

// Return the name property of the instance (the one given by AWS EC2).
//
func GetName(instance *Ec2Instance) *Property {
	return nil
}

// Return the public ip property of the instance.
//
func GetPublicIp(instance *Ec2Instance) *Property {
	return nil
}

// Return the private ip property of the instance.
//
func GetPrivateIp(instance *Ec2Instance) *Property {
	return nil
}

// Return the region code property of the instance.
//
func GetRegion(instance *Ec2Instance) *Property {
	return nil
}

// Return the region code property of the instance.
//
func GetUiid(instance *Ec2Instance) *Property {
	return nil
}

// Return the ssh username property of the instance.
//
func GetUser(instance *Ec2Instance) *Property {
	return nil
}

// Return the attribute of the instance with the given name.
// If the instance has no attribute with this name, return a Property with a
// Value field set to the empty string and a Defined field set to false.
//
func GetAttribute(instance *Ec2Instance, name string) *Property {
	return nil
}

// Return the property of the instance with the given name.
// If the name correspond to a trait name, return the trait property.
// Otherwise, return the attribute property.
// If the instance has no trait nor attribute with this name, return am
// attribute Property with a Value field set to the empty string and a Defined
// field set to false.
//
func GetProperty(instance *Ec2Instance, name string) *Property {
	return nil
}

// Parse a string containing printf like formats and apply it to a specific
// instance.
// The format has the following rules:
//   - a format sequence starts after a '%' character
//   - if a format sequence starts with a shortcut character, replace it with
//     the corresponding instance property
//   - if a format sequence starts with a '{' character, read until the next
//     '}' character to find the name of the property, then replace the whole
//     sequence with the instance property
//   - if the property is not defined, consider it as an empty string
//   - otherwise, replace the sequence with the character following the '%'
// Return the replaced string.
//
func Format(pattern string, instance *Ec2Instance) string {
	return pattern
}
