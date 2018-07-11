// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type subnet struct {
	// Add the Controller in when we need to do things with the Subnet.
	// Controller Controller

	ResourceURI string

	ID    int
	Name  string
	Space string
	VLAN  *vlan

	Gateway string
	CIDR    string
	// DNSServers is a list of ip addresses of the DNS servers for the Subnet.
	// This list may be empty.
	DNSServers []string
}

func readSubnets(controllerVersion version.Number, source interface{}) ([]*subnet, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "Subnet base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range subnetDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no Subnet read func for version %s", controllerVersion)
	}
	readFunc := subnetDeserializationFuncs[deserialisationVersion]
	return readSubnetList(valid, readFunc)
}

// readSubnetList expects the Values of the sourceList to be string maps.
func readSubnetList(sourceList []interface{}, readFunc subnetDeserializationFunc) ([]*subnet, error) {
	result := make([]*subnet, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("unexpected value for Subnet %d, %T", i, value)
		}
		subnet, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "Subnet %d", i)
		}
		result = append(result, subnet)
	}
	return result, nil
}

type subnetDeserializationFunc func(map[string]interface{}) (*subnet, error)

var subnetDeserializationFuncs = map[version.Number]subnetDeserializationFunc{
	twoDotOh: subnet_2_0,
}

func subnet_2_0(source map[string]interface{}) (*subnet, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),
		"ID":           schema.ForceInt(),
		"Name":         schema.String(),
		"space":        schema.String(),
		"gateway_ip":   schema.OneOf(schema.Nil(""), schema.String()),
		"cidr":         schema.String(),
		"VLAN":         schema.StringMap(schema.Any()),
		"dns_servers":  schema.OneOf(schema.Nil(""), schema.List(schema.String())),
	}
	checker := schema.FieldMap(fields, nil) // no defaults
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "Subnet 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	vlan, err := vlan_2_0(valid["VLAN"].(map[string]interface{}))
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Since the gateway_ip is optional, we use the two part cast assignment. If
	// the cast fails, then we get the default value we care about, which is the
	// empty string.
	gateway, _ := valid["gateway_ip"].(string)

	result := &subnet{
		ResourceURI: valid["resource_uri"].(string),
		ID:          valid["ID"].(int),
		Name:        valid["Name"].(string),
		Space:       valid["space"].(string),
		VLAN:        vlan,
		Gateway:     gateway,
		CIDR:        valid["cidr"].(string),
		DNSServers:  convertToStringSlice(valid["dns_servers"]),
	}
	return result, nil
}
