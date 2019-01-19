package main

import (
	"testing"
)

func TestGetTraits(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet *Ec2Fleet
	var instance *Ec2Instance
	var property *Property
	var name string

	fleet, _ = idx.AddEc2Fleet("a", "fleet", "user", "region", 1)
	instance, _ = fleet.AddEc2Instance("name", "public-ip", "private-ip")

	for _, name = range []string{"fleet", "fiid", "name", "public-ip",
		"private-ip", "region", "uiid", "user"} {
		property = GetProperty(instance, name)

		if property == nil {
			t.FailNow()
		} else if !property.Defined {
			t.FailNow()
		} else if property.Attribute {
			t.FailNow()
		} else if property.Name != name {
			t.FailNow()
		}
	}

	for _, name = range []string{"fleet", "name", "public-ip",
		"private-ip", "region", "user"} {
		property = GetProperty(instance, name)

		if property.Value != name {
			t.FailNow()
		}
	}
	for _, name = range []string{"fiid", "uiid"} {
		property = GetProperty(instance, name)

		if property.Value != "0" {
			t.FailNow()
		}
	}
}

func TestGetAttributes(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet *Ec2Fleet
	var instance *Ec2Instance
	var property *Property

	fleet, _ = idx.AddEc2Fleet("a", "fleet", "user", "region", 1)
	instance, _ = fleet.AddEc2Instance("name", "public-ip", "private-ip")

	instance.Attributes["toto"] = "aaa"

	property = GetProperty(instance, "toto")
	if property == nil {
		t.FailNow()
	} else if !property.Defined {
		t.FailNow()
	} else if !property.Attribute {
		t.FailNow()
	} else if property.Name != "toto" {
		t.FailNow()
	} else if property.Value != "aaa" {
		t.FailNow()
	}
}

func TestGetConflictingAttribute(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet *Ec2Fleet
	var instance *Ec2Instance
	var property *Property

	fleet, _ = idx.AddEc2Fleet("a", "fleet", "user", "region", 1)
	instance, _ = fleet.AddEc2Instance("name", "public-ip", "private-ip")

	instance.Attributes["name"] = "aaa"

	property = GetProperty(instance, "name")

	if property == nil {
		t.FailNow()
	} else if !property.Defined {
		t.FailNow()
	} else if property.Attribute {
		t.FailNow()
	} else if property.Name != "name" {
		t.FailNow()
	} else if property.Value != "name" {
		t.FailNow()
	}

	property = GetAttribute(instance, "name")

	if property == nil {
		t.FailNow()
	} else if !property.Defined {
		t.FailNow()
	} else if !property.Attribute {
		t.FailNow()
	} else if property.Name != "name" {
		t.FailNow()
	} else if property.Value != "aaa" {
		t.FailNow()
	}
}

func TestGetUndefinedAttributes(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet *Ec2Fleet
	var instance *Ec2Instance
	var property *Property

	fleet, _ = idx.AddEc2Fleet("a", "fleet", "user", "region", 1)
	instance, _ = fleet.AddEc2Instance("name", "public-ip", "private-ip")

	property = GetProperty(instance, "toto")
	if property == nil {
		t.FailNow()
	} else if property.Defined {
		t.FailNow()
	} else if !property.Attribute {
		t.FailNow()
	} else if property.Name != "toto" {
		t.FailNow()
	} else if property.Value != "" {
		t.FailNow()
	}
}

func TestFormatShort(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet *Ec2Fleet
	var instance *Ec2Instance
	var pattern, obtained, expected string

	fleet, _ = idx.AddEc2Fleet("a", "fleet", "user", "region", 1)
	instance, _ = fleet.AddEc2Instance("name", "public-ip", "private-ip")

	instance.Attributes["toto"] = "aaa"

	pattern = "%d %D %f %n %I %i %r %u %% %? %{} %{toto} %{titi}"
	expected = "0 0 fleet name public-ip private-ip region user % ?  aaa "

	obtained = Format(pattern, instance)

	if obtained != expected {
		t.FailNow()
	}
}

func TestFormatLong(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet *Ec2Fleet
	var instance *Ec2Instance
	var pattern, obtained, expected string

	fleet, _ = idx.AddEc2Fleet("a", "fleet", "user", "region", 1)
	instance, _ = fleet.AddEc2Instance("name", "public-ip", "private-ip")

	instance.Attributes["toto"] = "aaa"

	pattern = "%{fiid} %{uiid} %{fleet} %{name} %{public-ip} %{private-ip} %{region} %{user} %{} %{toto} %{titi}"
	expected = "0 0 fleet name public-ip private-ip region user  aaa "

	obtained = Format(pattern, instance)

	if obtained != expected {
		t.FailNow()
	}
}
