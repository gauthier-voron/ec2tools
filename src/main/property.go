package main

import (
	"strconv"
)

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

// Create a new trait (non attribute) property with the specified Property.Name
// and Property.Value fields.
//
func newTraitProperty(instance *Ec2Instance, name, value string) *Property {
	var property Property

	property.Defined = true
	property.Attribute = false
	property.Name = name
	property.Value = value
	property.Instance = instance

	return &property
}

// Return the fleet name property of the instance.
//
func GetFleet(instance *Ec2Instance) *Property {
	return newTraitProperty(instance, "fleet", instance.Fleet.Name)
}

// Return the fleet instance identifier (fiid) property of the instance.
//
func GetFiid(instance *Ec2Instance) *Property {
	return newTraitProperty(instance, "fiid",
		strconv.Itoa(instance.FleetIndex))
}

// Return the name property of the instance (the one given by AWS EC2).
//
func GetName(instance *Ec2Instance) *Property {
	return newTraitProperty(instance, "name", instance.Name)
}

// Return the public ip property of the instance.
//
func GetPublicIp(instance *Ec2Instance) *Property {
	return newTraitProperty(instance, "public-ip", instance.PublicIp)
}

// Return the private ip property of the instance.
//
func GetPrivateIp(instance *Ec2Instance) *Property {
	return newTraitProperty(instance, "private-ip", instance.PrivateIp)
}

// Return the region code property of the instance.
//
func GetRegion(instance *Ec2Instance) *Property {
	return newTraitProperty(instance, "region", instance.Fleet.Region)
}

// Return the region code property of the instance.
//
func GetUiid(instance *Ec2Instance) *Property {
	return newTraitProperty(instance, "uiid",
		strconv.Itoa(instance.UniqueIndex))
}

// Return the ssh username property of the instance.
//
func GetUser(instance *Ec2Instance) *Property {
	return newTraitProperty(instance, "user", instance.Fleet.User)
}

// Return the attribute of the instance with the given name.
// If the instance has no attribute with this name, return a Property with a
// Value field set to the empty string and a Defined field set to false.
//
func GetAttribute(instance *Ec2Instance, name string) *Property {
	var property Property

	property.Attribute = true
	property.Name = name
	property.Instance = instance
	property.Value, property.Defined = instance.Attributes[name]

	return &property
}

// Return the property of the instance with the given name.
// If the name correspond to a trait name, return the trait property.
// Otherwise, return the attribute property.
// If the instance has no trait nor attribute with this name, return am
// attribute Property with a Value field set to the empty string and a Defined
// field set to false.
//
func GetProperty(instance *Ec2Instance, name string) *Property {
	switch name {
	case "fleet":
		return GetFleet(instance)
	case "fiid":
		return GetFiid(instance)
	case "ip":
		return GetPublicIp(instance)
	case "name":
		return GetName(instance)
	case "public-ip":
		return GetPublicIp(instance)
	case "private-ip":
		return GetPrivateIp(instance)
	case "region":
		return GetRegion(instance)
	case "uiid":
		return GetUiid(instance)
	case "user":
		return GetUser(instance)
	default:
		return GetAttribute(instance, name)
	}
}

// Return the trait name corresponding to a given one letter shortcut.
// The second return value is a boolean indicating if the shortcut exists.
//
func getShortcutName(c rune) (string, bool) {
	switch c {
	case 'd':
		return "fiid", true
	case 'D':
		return "uiid", true
	case 'f':
		return "fleet", true
	case 'I':
		return "public-ip", true
	case 'i':
		return "private-ip", true
	case 'n':
		return "name", true
	case 'r':
		return "region", true
	case 'u':
		return "user", true
	default:
		return "", false
	}
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
	var percent bool = false
	var property bool = false
	var propStart, pos int
	var ret string = ""
	var name string
	var valid bool
	var c rune

	for pos, c = range pattern {
		if property {
			if c == '}' {
				name = pattern[propStart:pos]
				ret += GetProperty(instance, name).Value
				property = false
			}
			continue
		}

		if percent {
			name, valid = getShortcutName(c)

			if valid {
				ret += GetProperty(instance, name).Value
			} else {
				if c == '{' {
					property = true
					propStart = pos + 1
				} else {
					ret += string(c)
				}
			}

			percent = false
			continue
		}

		if c == '%' {
			percent = true
		} else {
			ret += string(c)
		}
	}

	return ret
}
