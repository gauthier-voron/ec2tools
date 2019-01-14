package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestNewEc2Index(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()

	if len(idx.FleetsByName) != 0 {
		t.Fail()
	}

	if len(idx.InstancesByName) != 0 {
		t.Fail()
	}
}

func TestAddEc2Fleet(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet *Ec2Fleet
	var err error

	fleet, err = idx.AddEc2Fleet("a", "name", "user", "region")

	if err != nil {
		t.FailNow()
	} else if fleet.Id != "a" {
		t.Fail()
	} else if fleet.Name != "name" {
		t.Fail()
	} else if fleet.User != "user" {
		t.Fail()
	} else if fleet.Region != "region" {
		t.Fail()
	} else if len(fleet.Instances) != 0 {
		t.Fail()
	} else if fleet.Index != idx {
		t.Fail()
	} else if idx.FleetsByName["name"] != fleet {
		t.Fail()
	}
}

func TestAddEc2FleetAlreadyUsed(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet0, fleet1 *Ec2Fleet
	var err error

	fleet0, _ = idx.AddEc2Fleet("a", "name", "user0", "region0")
	fleet1, err = idx.AddEc2Fleet("b", "name", "user1", "region1")

	if err == nil {
		t.Fail()
	} else if fleet1 != nil {
		t.Fail()
	} else if fleet0.Id != "a" {
		t.Fail()
	} else if fleet0.Name != "name" {
		t.Fail()
	} else if fleet0.User != "user0" {
		t.Fail()
	} else if idx.FleetsByName["name"] != fleet0 {
		t.Fail()
	}
}

func TestStoreEc2Index(t *testing.T) {
	var path string = "context_test_TestStoreEc2Index.json"
	var expectedJson string = "{\"Fleets\":[{\"Id\":\"0\",\"Name\":\"fleet0\",\"User\":\"u\",\"Region\":\"r\",\"Instances\":[{\"Name\":\"i0\",\"PublicIp\":\"0.0.0.0\",\"PrivateIp\":\"1.0.0.0\",\"UniqueIndex\":0,\"Attributes\":{}},{\"Name\":\"i1\",\"PublicIp\":\"0.0.0.1\",\"PrivateIp\":\"1.0.0.1\",\"UniqueIndex\":1,\"Attributes\":{}}]},{\"Id\":\"1\",\"Name\":\"fleet1\",\"User\":\"u\",\"Region\":\"r\",\"Instances\":[{\"Name\":\"i2\",\"PublicIp\":\"0.0.0.2\",\"PrivateIp\":\"1.0.0.2\",\"UniqueIndex\":2,\"Attributes\":{}}]}],\"UniqueCounter\":3}"
	var idx *Ec2Index = NewEc2Index()
	var fleet0, fleet1 *Ec2Fleet
	var jsonString string
	var raw []byte
	var err error

	fleet0, _ = idx.AddEc2Fleet("0", "fleet0", "u", "r")
	fleet0.AddEc2Instance("i0", "0.0.0.0", "1.0.0.0")
	fleet0.AddEc2Instance("i1", "0.0.0.1", "1.0.0.1")

	fleet1, _ = idx.AddEc2Fleet("1", "fleet1", "u", "r")
	fleet1.AddEc2Instance("i2", "0.0.0.2", "1.0.0.2")

	err = StoreEc2Index(path, idx)
	if err != nil {
		t.FailNow()
	}

	raw, err = ioutil.ReadFile(path)
	if err != nil {
		os.Remove(path)
		t.FailNow()
	}

	jsonString = string(raw)

	if jsonString != expectedJson {
		t.Fail()
	}

	os.Remove(path)
}

func TestLoadEc2Index(t *testing.T) {
	var path string = "context_test_TestLoadEc2Index.json"
	var loadedJson string = "{\"Fleets\":[{\"Id\":\"0\",\"Name\":\"fleet0\",\"User\":\"u\",\"Region\":\"r\",\"Instances\":[{\"Name\":\"i0\",\"PublicIp\":\"0.0.0.0\",\"PrivateIp\":\"1.0.0.0\",\"UniqueIndex\":0,\"Attributes\":{}},{\"Name\":\"i1\",\"PublicIp\":\"0.0.0.1\",\"PrivateIp\":\"1.0.0.1\",\"UniqueIndex\":1,\"Attributes\":{}}]},{\"Id\":\"1\",\"Name\":\"fleet1\",\"User\":\"u\",\"Region\":\"r\",\"Instances\":[{\"Name\":\"i2\",\"PublicIp\":\"0.0.0.2\",\"PrivateIp\":\"1.0.0.2\",\"UniqueIndex\":2,\"Attributes\":{}}]}],\"UniqueCounter\":3}"
	var idx *Ec2Index
	var fleet *Ec2Fleet
	var found bool
	var err error

	err = ioutil.WriteFile(path, []byte(loadedJson), 0644)
	if err != nil {
		t.FailNow()
	}

	idx, err = LoadEc2Index(path)
	os.Remove(path)

	if err != nil {
		t.Fail()
	} else if idx == nil {
		t.FailNow()
	}

	if len(idx.FleetsByName) != 2 {
		t.Fail()
	} else if len(idx.InstancesByName) != 3 {
		t.Fail()
	}

	fleet, found = idx.FleetsByName["fleet0"]
	if !found {
		t.FailNow()
	}

	if fleet.Id != "0" {
		t.Fail()
	} else if fleet.Name != "fleet0" {
		t.Fail()
	} else if fleet.User != "u" {
		t.Fail()
	} else if fleet.Region != "r" {
		t.Fail()
	} else if len(fleet.Instances) != 2 {
		t.FailNow()
	} else if fleet.Instances[0].Name != "i0" {
		t.Fail()
	} else if fleet.Instances[0].PublicIp != "0.0.0.0" {
		t.Fail()
	} else if fleet.Instances[0].PrivateIp != "1.0.0.0" {
		t.Fail()
	} else if fleet.Instances[0].Fleet != fleet {
		t.Fail()
	} else if fleet.Instances[0].FleetIndex != 0 {
		t.Fail()
	} else if fleet.Instances[0].UniqueIndex != 0 {
		t.Fail()
	} else if fleet.Instances[1].Name != "i1" {
		t.Fail()
	} else if fleet.Instances[1].PublicIp != "0.0.0.1" {
		t.Fail()
	} else if fleet.Instances[1].PrivateIp != "1.0.0.1" {
		t.Fail()
	} else if fleet.Instances[1].Fleet != fleet {
		t.Fail()
	} else if fleet.Instances[1].FleetIndex != 1 {
		t.Fail()
	} else if fleet.Instances[1].UniqueIndex != 1 {
		t.Fail()
	} else if fleet.Index != idx {
		t.Fail()
	}

	fleet, found = idx.FleetsByName["fleet1"]
	if !found {
		t.FailNow()
	}

	if fleet.Id != "1" {
		t.Fail()
	} else if fleet.Name != "fleet1" {
		t.Fail()
	} else if fleet.User != "u" {
		t.Fail()
	} else if fleet.Region != "r" {
		t.Fail()
	} else if len(fleet.Instances) != 1 {
		t.FailNow()
	} else if fleet.Instances[0].Name != "i2" {
		t.Fail()
	} else if fleet.Instances[0].PublicIp != "0.0.0.2" {
		t.Fail()
	} else if fleet.Instances[0].PrivateIp != "1.0.0.2" {
		t.Fail()
	} else if fleet.Instances[0].Fleet != fleet {
		t.Fail()
	} else if fleet.Instances[0].FleetIndex != 0 {
		t.Fail()
	} else if fleet.Instances[0].UniqueIndex != 2 {
		t.Fail()
	} else if fleet.Index != idx {
		t.Fail()
	}
}

func TestFilterInstanceEc2Selection(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet0, fleet1 *Ec2Fleet
	var sel *Ec2Selection
	var err error

	fleet0, _ = idx.AddEc2Fleet("0", "fleet0", "u", "r")
	fleet0.AddEc2Instance("i0", "0.0.0.0", "1.0.0.0")
	fleet0.AddEc2Instance("i1", "0.0.0.1", "1.0.0.1")

	fleet1, _ = idx.AddEc2Fleet("1", "fleet1", "u", "r")
	fleet1.AddEc2Instance("i2", "0.0.0.2", "1.0.0.2")

	sel, err = idx.Select([]string{"i2", "i0"})

	if sel == nil {
		t.FailNow()
	} else if err != nil {
		t.FailNow()
	}

	if len(sel.Instances) != 2 {
		t.FailNow()
	} else if sel.Instances[0].Name != "i2" {
		t.FailNow()
	} else if sel.Instances[1].Name != "i0" {
		t.FailNow()
	}
}

func TestFilterDuplicateInstanceEc2Selection(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet *Ec2Fleet
	var sel *Ec2Selection
	var err error

	fleet, _ = idx.AddEc2Fleet("0", "fleet0", "u", "r")
	fleet.AddEc2Instance("i0", "0.0.0.0", "1.0.0.0")

	sel, err = idx.Select([]string{"i0", "i0"})

	if sel == nil {
		t.FailNow()
	} else if err != nil {
		t.FailNow()
	}

	if len(sel.Instances) != 2 {
		t.FailNow()
	} else if sel.Instances[0].Name != "i0" {
		t.FailNow()
	} else if sel.Instances[1].Name != "i0" {
		t.FailNow()
	}
}

func TestFilterFleetEc2Selection(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet0, fleet1 *Ec2Fleet
	var sel *Ec2Selection
	var err error

	fleet0, _ = idx.AddEc2Fleet("0", "fleet0", "u", "r")
	fleet0.AddEc2Instance("i0", "0.0.0.0", "1.0.0.0")
	fleet0.AddEc2Instance("i1", "0.0.0.1", "1.0.0.1")

	fleet1, _ = idx.AddEc2Fleet("1", "fleet1", "u", "r")
	fleet1.AddEc2Instance("i2", "0.0.0.2", "1.0.0.2")

	sel, err = idx.Select([]string{"@fleet0"})

	if sel == nil {
		t.FailNow()
	} else if err != nil {
		t.FailNow()
	}

	if len(sel.Instances) != 2 {
		t.FailNow()
	} else if sel.Instances[0].Name != "i0" {
		t.FailNow()
	} else if sel.Instances[1].Name != "i1" {
		t.FailNow()
	}
}

func TestMatchFleetEc2Selection(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet0, fleet1 *Ec2Fleet
	var sel *Ec2Selection
	var err error

	fleet0, _ = idx.AddEc2Fleet("0", "fleet0", "u", "r")
	fleet0.AddEc2Instance("i0", "0.0.0.0", "1.0.0.0")
	fleet0.AddEc2Instance("i1", "0.0.0.1", "1.0.0.1")

	fleet1, _ = idx.AddEc2Fleet("1", "fleet1", "u", "r")
	fleet1.AddEc2Instance("i2", "0.0.0.2", "1.0.0.2")

	sel, err = idx.Select([]string{"@/^fle.*$/"})

	if sel == nil {
		t.FailNow()
	} else if err != nil {
		t.FailNow()
	}

	if len(sel.Instances) != 3 {
		t.FailNow()
	} else if sel.Instances[0].Name != "i0" {
		t.FailNow()
	} else if sel.Instances[1].Name != "i1" {
		t.FailNow()
	} else if sel.Instances[2].Name != "i2" {
		t.FailNow()
	}
}

func TestMatchFleetOrderingEc2Selection(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var fleet0, fleet1 *Ec2Fleet
	var sel *Ec2Selection
	var err error

	fleet0, _ = idx.AddEc2Fleet("0", "fleet0", "u", "r")
	fleet1, _ = idx.AddEc2Fleet("1", "fleet1", "u", "r")

	fleet0.AddEc2Instance("i0", "0.0.0.0", "1.0.0.0")
	fleet1.AddEc2Instance("i2", "0.0.0.2", "1.0.0.2")
	fleet0.AddEc2Instance("i1", "0.0.0.1", "1.0.0.1")

	sel, err = idx.Select([]string{"@/fleet\\d+/"})

	if sel == nil {
		t.FailNow()
	} else if err != nil {
		t.FailNow()
	}

	if len(sel.Instances) != 3 {
		t.FailNow()
	} else if sel.Instances[0].Name != "i0" {
		t.FailNow()
	} else if sel.Instances[1].Name != "i2" {
		t.FailNow()
	} else if sel.Instances[2].Name != "i1" {
		t.FailNow()
	}
}

func TestMatchIllRegexpEc2Selection(t *testing.T) {
	var idx *Ec2Index = NewEc2Index()
	var sel *Ec2Selection
	var err error

	sel, err = idx.Select([]string{"/*/"})

	if sel != nil {
		t.FailNow()
	} else if err == nil {
		t.FailNow()
	}
}
