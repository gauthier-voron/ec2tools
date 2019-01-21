package main

import (
	"flag"
	"fmt"
	"sort"
)

var DEFAULT_DEFINED bool = false
var DEFAULT_FORMAT bool = false
var DEFAULT_SORT bool = false
var DEFAULT_SORT_BY string = ""
var DEFAULT_UNIQUE_INSTANCES bool = false
var DEFAULT_UNIQUE_RESULTS bool = false
var DEFAULT_UPDATE bool = false

var optionDefined *bool
var optionFormat *bool
var optionSort *bool
var optionSortBy *string
var optionUniqueInstances *bool
var optionUniqueResults *bool
var optionUpdate *bool

func PrintGetUsage() {
	fmt.Printf(`Usage: %s get [options] fleets
       %s get [options] [<instances-specification...> --] <properties...>

Print information about fleets and instances.
The first form print a list of the fleet names launched and not yet stopped in
the current context.
The second form print the values of the properties for a list of instances. The
instances may be specified by their name or by the name of their fleet, either
with a string or a regular expression.
If no specification is given, print properties for all instances.

Options:
  --context <path>            path of the context file (default: '%s')

  --defined                   only print defined properties

  --format                    interpret properties as printf like format (see
                              Format section)

  --sort                      sort instances by their uiid before to print
                              their properties (shortcut for '--sort-by uiid')

  --sort-by <property>        sort instances by the indicated property before
                              to print their properties

  --unique-instances          remove duplicate instances before to print their
                              properties, preserving order (duplicate results
                              may be displayed if several instances have the
                              same properties)

  --unique-results            remove duplicate results, preserving order (no
                              duplicate can be displayed)

  --update                    update context before to print

Properties:
  fleet             name of the fleet of the instances
  fiid              integer that identifies the instance inside its fleet
  ip | public-ip    public IPv4 to access the instance
  name              name of an instance, as defined by AWS
  private-ip        private IPv4: how the instance sees itself
  region            region code the instance runs in (e.g. 'us-east-2')
  uiid              integer that identifies the instance inside its context
  user              username to use for an ssh connection
  <attribute>       a custom attribute defined with the 'set' subcommand

Instance specification:
  Instances can be specified either directly by their name or by the name of
  their fleet. In this last case, the specification starts with a '@':

      i-0efc03c42c4137c64     the instance identified as 'i-0ef...' by AWS

      @my-fleet               all instances belonging to the fleet 'my-fleet'

  Additionally, an instance or a fleet name can be an exact string or a Perl
  regular expression. In this last case, it is surrounded by '/':

      /^.*[0-7]$/             all instances with a name ending with 0, ..., 7

      @/fleet-(a|b)/          all instances of fleets 'fleet-a' or 'fleet-b'

  Without additional options, the instances resulting from a single
  specification are sorted by their uiid property.
  The results from different specifications are concatenated without additional
  sorting nor re;oving of duplicate results.

Format:
  If the '--fornat' option is supplied, the properties are interpreted as
  printf like format with the following formatting sequences.

      %%d            fiid
      %%D            uiid
      %%f            fleet
      %%i            public-ip
      %%I            private-ip
      %%n            name
      %%r            region
      %%u            user

      %%{<name>}     the value of a property, as defined in the Properties
                    section (empty string if it is an undefined attribute) 

      %%%%            a '%%' character

Examples:
  Print all available fleets, one per line:

      %s get fleets
      %s get --unique-results fleet

  Print all instance names for fleets 'my-fleet', 'fleet-0' and 'fleet-1':

      %s get name @my-fleet @/^fleet-[01]$/

  Print all public IP addresses for this context:

      %s get public-ip
      %s get /^.*$/ -- public-ip
      %s get // -- public-ip

  Print the maximum id of an instance inside of its fleet:

      %s get --sort-by=fiid --unique-results fiid | tail -n 1

  Print public IP addresses from 'fleet-a' and 'my-fleet' in consisent order:

      %s get --sort @my-fleet @fleet-a -- ip

  Print a string containing the public and private IP of all instances

      %s get --format 'public-ip: %%I  /  private-ip: %%{private-ip}'

`,
		PROGNAME, PROGNAME, DEFAULT_CONTEXT,
		PROGNAME, PROGNAME, PROGNAME, PROGNAME, PROGNAME, PROGNAME,
		PROGNAME, PROGNAME, PROGNAME)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// PropertList related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// A list of properties of format associated with an instance.
//
type PropertyList struct {
	Instance   *Ec2Instance // instance with the associated properties
	Properties []*Property  // properties fetched for the instance
	Values     []string     // values of properties to show
}

// Create a new empty PropertyList associated to the given instance.
//
func NewPropertyList(instance *Ec2Instance) *PropertyList {
	var this PropertyList

	this.Instance = instance
	this.Properties = make([]*Property, 0)
	this.Values = make([]string, 0)

	return &this
}

// Add a new property with the given name to the list.
//
func (this *PropertyList) GetProperty(name string) {
	var property *Property = GetProperty(this.Instance, name)

	this.Properties = append(this.Properties, property)
	this.Values = append(this.Values, property.Value)
}

// Add a new formatted string to the list.
//
func (this *PropertyList) AddFormat(pattern string) {
	var property *Property = GetProperty(this.Instance, "uiid")
	var value string = Format(pattern, this.Instance)

	this.Properties = append(this.Properties, property)
	this.Values = append(this.Values, value)
}

// Indicate if every properties of the list are defined.
//
func (this *PropertyList) IsFullyDefined() bool {
	var property *Property

	for _, property = range this.Properties {
		if !property.Defined {
			return false
		}
	}

	return true
}

// Concatenate all the properties and formatted string in a single string,
// separated by the given string.
//
func (this *PropertyList) ToString(separator string) string {
	var first bool = true
	var value, ret string

	ret = ""

	for _, value = range this.Values {
		if first {
			first = false
		} else {
			ret += separator
		}

		ret += value
	}

	return ret
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// PropertyList sort related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// Sort the given PropertyList according to a related key, located in the
// given strs slice at the ame index.
// Return a new slice of sorted PropertyList.
//
func sortPropertyListByStrings(lists []*PropertyList, strs []string) []*PropertyList {
	var listsByValue map[string][]*PropertyList
	var sorted []*PropertyList = make([]*PropertyList, 0)
	var values []string = make([]string, 0)
	var list *PropertyList
	var value string
	var found bool
	var i int

	listsByValue = make(map[string][]*PropertyList)

	for i, list = range lists {
		value = strs[i]

		_, found = listsByValue[value]

		if !found {
			values = append(values, value)
		}

		listsByValue[value] = append(listsByValue[value], list)
	}

	sort.Strings(values)

	for _, value = range values {
		for _, list = range listsByValue[value] {
			sorted = append(sorted, list)
		}
	}

	return sorted
}

// Sort the given PropertyList depending on the value of their associated
// Property which the name is specified.
//
func SortPropertyListByProperty(lists []*PropertyList, name string) []*PropertyList {
	var strs []string = make([]string, len(lists))
	var list *PropertyList
	var i int

	for i, list = range lists {
		strs[i] = GetProperty(list.Instance, name).Value
	}

	return sortPropertyListByStrings(lists, strs)
}

// Sort the given PropertyList depending on the value of their concatenated
// property values or formatted strings.
//
func SortPropertyListByString(lists []*PropertyList) []*PropertyList {
	var strs []string = make([]string, len(lists))
	var list *PropertyList
	var i int

	for i, list = range lists {
		strs[i] = list.ToString(" ")
	}

	return sortPropertyListByStrings(lists, strs)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// PropertyList uniq related code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

// Return a slice of PropertyList containing the first occurence of
// PropertyList from the given lists slice having a given value in strs at the
// same index.
//
func uniquePropertyListsByStrings(lists []*PropertyList, strs []string) []*PropertyList {
	var firstListByValue map[string]*PropertyList
	var uniq []*PropertyList = make([]*PropertyList, 0)
	var values []string = make([]string, 0)
	var list *PropertyList
	var value string
	var found bool
	var i int

	firstListByValue = make(map[string]*PropertyList)

	for i, list = range lists {
		value = strs[i]

		_, found = firstListByValue[value]

		if !found {
			firstListByValue[value] = list
			values = append(values, value)
		}
	}

	for _, value = range values {
			uniq = append(uniq, firstListByValue[value])
	}

	return uniq
}

// Return a slice of PropertyList containing the first occurence from the lists
// slice with the same instance.
//
func UniquePropertyListsByInstance(lists []*PropertyList) []*PropertyList {
	var strs []string = make([]string, len(lists))
	var list *PropertyList
	var i int

	for i, list = range lists {
		strs[i] = list.Instance.Name
	}

	return uniquePropertyListsByStrings(lists, strs)
}

// Return a slice of PropertyList containing the first occurence from the lists
// slice with the same result string.
//
func UniquePropertyListsByString(lists []*PropertyList) []*PropertyList {
	var strs []string = make([]string, len(lists))
	var list *PropertyList
	var i int

	for i, list = range lists {
		strs[i] = list.ToString(" ")
	}

	return uniquePropertyListsByStrings(lists, strs)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// Main routines code
// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -

func doGetFleets(idx *Ec2Index) {
	var name string

	for name = range idx.FleetsByName {
		fmt.Println(name)
	}
}

func doGetProperties(selection *Ec2Selection, propstrs []string) {
	var lists []*PropertyList = make([]*PropertyList, 0)
	var instance *Ec2Instance
	var list *PropertyList
	var str string

	for _, instance = range selection.Instances {
		lists = append(lists, NewPropertyList(instance))
	}

	if *optionUniqueInstances {
		lists = UniquePropertyListsByInstance(lists)
	}

	for _, str = range propstrs {
		for _, list = range lists {
			if *optionFormat {
				list.AddFormat(str)
			} else {
				list.GetProperty(str)
			}
		}
	}

	if *optionUniqueResults {
		lists = UniquePropertyListsByString(lists)
	}

	if *optionSortBy != "" {
		lists = SortPropertyListByProperty(lists, *optionSortBy)
	}

	for _, list = range lists {
		if !*optionDefined || list.IsFullyDefined() {
			fmt.Println(list.ToString(" "))
		}
	}
}

func Get(args []string) {
	var flags *flag.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	var specs, propstrs []string
	var instances *Ec2Selection
	var ctx *Ec2Index
	var hasSpecs bool
	var arg string
	var err error

	optionContext = flags.String("context", DEFAULT_CONTEXT, "")
	optionDefined = flags.Bool("defined", DEFAULT_DEFINED, "")
	optionFormat = flags.Bool("format", DEFAULT_FORMAT, "")
	optionSort = flags.Bool("sort", DEFAULT_SORT, "")
	optionSortBy = flags.String("sort-by", DEFAULT_SORT_BY, "")
	optionUniqueInstances = flags.Bool("unique-instances", DEFAULT_UNIQUE_INSTANCES, "")
	optionUniqueResults = flags.Bool("unique-results", DEFAULT_UNIQUE_RESULTS, "")
	optionUpdate = flags.Bool("update", DEFAULT_UPDATE, "")

	flags.Parse(args[1:])
	args = flags.Args()

	if len(args) < 1 {
		Error("missing property operand")
	}

	hasSpecs = false
	for _, arg = range args {
		if (arg == "--") && !hasSpecs {
			hasSpecs = true
			specs = propstrs
			propstrs = make([]string, 0)
			continue
		}

		propstrs = append(propstrs, arg)
	}

	if len(propstrs) < 1 {
		Error("missing property operand")
	}

	ctx, err = LoadEc2Index(*optionContext)
	if err != nil {
		Error("no context: %s", *optionContext)
	}

	if *optionSort {
		*optionSortBy = "uiid"
	}

	if propstrs[0] == "fleets" {
		if len(propstrs) > 1 {
			Error("unexpected operand: '%s'", propstrs[1])
		} else if hasSpecs {
			Error("unexpected instance specification: '%s'",
				specs[0])
		}

		doGetFleets(ctx)
	} else {
		if *optionUpdate {
			UpdateContext(ctx)
			StoreEc2Index(*optionContext, ctx)
		}

		if !hasSpecs {
			instances, _ = ctx.Select([]string{"//"})
		} else {
			instances, err = ctx.Select(specs)
			if err != nil {
				Error("invalid specification: %s", err.Error())
			}
		}

		doGetProperties(instances, propstrs)
	}
}
