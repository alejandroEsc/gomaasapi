// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

// Fabric represents a set of interconnected VLANs that are capable of mutual
// communication. A Fabric can be thought of as a logical grouping in which
// VLANs can be considered unique.
//
// For example, a distributed network may have a Fabric in London containing
// VLAN 100, while a separate Fabric in San Francisco may contain a VLAN 100,
// whose attached Subnets are completely different and unrelated.
type fabric struct {
	// Add the Controller in when we need to do things with the Fabric.
	// Controller Controller

	ResourceURI string `json:"resource_uri,omitempty"`

	ID        int
	Name      string
	ClassType string

	VLANs []*vlan
}

func readFabrics(controllerVersion version.Number, source interface{}) ([]*fabric, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "Fabric base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range fabricDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no Fabric read func for version %s", controllerVersion)
	}
	readFunc := fabricDeserializationFuncs[deserialisationVersion]
	return readFabricList(valid, readFunc)
}

// readFabricList expects the Values of the sourceList to be string maps.
func readFabricList(sourceList []interface{}, readFunc fabricDeserializationFunc) ([]*fabric, error) {
	result := make([]*fabric, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("unexpected value for Fabric %d, %T", i, value)
		}
		fabric, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "Fabric %d", i)
		}
		result = append(result, fabric)
	}
	return result, nil
}

type fabricDeserializationFunc func(map[string]interface{}) (*fabric, error)

var fabricDeserializationFuncs = map[version.Number]fabricDeserializationFunc{
	twoDotOh: fabric_2_0,
}

func fabric_2_0(source map[string]interface{}) (*fabric, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),
		"ID":           schema.ForceInt(),
		"Name":         schema.String(),
		"class_type":   schema.OneOf(schema.Nil(""), schema.String()),
		"VLANs":        schema.List(schema.StringMap(schema.Any())),
	}
	checker := schema.FieldMap(fields, nil) // no defaults
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "Fabric 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	vlans, err := readVLANList(valid["VLANs"].([]interface{}), vlan_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Since the class_type is optional, we use the two part cast assignment. If
	// the cast fails, then we get the default value we care about, which is the
	// empty string.
	classType, _ := valid["class_type"].(string)

	result := &fabric{
		ResourceURI: valid["resource_uri"].(string),
		ID:          valid["ID"].(int),
		Name:        valid["Name"].(string),
		ClassType:   classType,
		VLANs:       vlans,
	}
	return result, nil
}
