// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

// VLAN represents an instance of a Virtual LAN. VLANs are a common way to
// create logically separate networks using the same physical infrastructure.
//
// Managed switches can assign VLANs to each port in either a “tagged” or an
// “untagged” manner. A VLAN is said to be “untagged” on a particular port when
// it is the default VLAN for that port, and requires no special configuration
// in order to access.
//
// “Tagged” VLANs (traditionally used by network administrators in order to
// aggregate multiple networks over inter-switch “trunk” lines) can also be used
// with nodes in MAAS. That is, if a switch port is configured such that
// “tagged” VLAN frames can be sent and received by a MAAS node, that MAAS node
// can be configured to automatically bring up VLAN interfaces, so that the
// deployed node can make use of them.
//
// A “Default VLAN” is created for every Fabric, to which every new VLAN-aware
// object in the Fabric will be associated to by default (unless otherwise
// specified).
type vlan struct {
	// Add the Controller in when we need to do things with the VLAN.
	// Controller Controller

	ResourceURI string `json:"resource_uri,omitempty"`

	ID     int
	Name   string
	Fabric string

	// VID is the VLAN ID. eth0.10 -> VID = 10.
	VID int
	// MTU (maximum transmission unit) is the largest Size packet or frame,
	// specified in octets (eight-bit bytes), that can be sent.
	MTU  int
	DHCP bool

	PrimaryRack   string
	SecondaryRack string
}

func readVLANs(controllerVersion version.Number, source interface{}) ([]*vlan, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "VLAN base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range vlanDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no VLAN read func for version %s", controllerVersion)
	}
	readFunc := vlanDeserializationFuncs[deserialisationVersion]
	return readVLANList(valid, readFunc)
}

func readVLANList(sourceList []interface{}, readFunc vlanDeserializationFunc) ([]*vlan, error) {
	result := make([]*vlan, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("unexpected value for VLAN %d, %T", i, value)
		}
		vlan, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "VLAN %d", i)
		}
		result = append(result, vlan)
	}
	return result, nil
}

type vlanDeserializationFunc func(map[string]interface{}) (*vlan, error)

var vlanDeserializationFuncs = map[version.Number]vlanDeserializationFunc{
	twoDotOh: vlan_2_0,
}

func vlan_2_0(source map[string]interface{}) (*vlan, error) {
	fields := schema.Fields{
		"ID":           schema.ForceInt(),
		"resource_uri": schema.String(),
		"Name":         schema.OneOf(schema.Nil(""), schema.String()),
		"Fabric":       schema.String(),
		"VID":          schema.ForceInt(),
		"MTU":          schema.ForceInt(),
		"dhcp_on":      schema.Bool(),
		// racks are not always set.
		"primary_rack":   schema.OneOf(schema.Nil(""), schema.String()),
		"secondary_rack": schema.OneOf(schema.Nil(""), schema.String()),
	}
	checker := schema.FieldMap(fields, nil)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "VLAN 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	// Since the primary and secondary racks are optional, we use the two
	// part cast assignment. If the case fails, then we get the default value
	// we care about, which is the empty string.
	primary_rack, _ := valid["primary_rack"].(string)
	secondary_rack, _ := valid["secondary_rack"].(string)
	name, _ := valid["Name"].(string)

	result := &vlan{
		ResourceURI:   valid["resource_uri"].(string),
		ID:            valid["ID"].(int),
		Name:          name,
		Fabric:        valid["Fabric"].(string),
		VID:           valid["VID"].(int),
		MTU:           valid["MTU"].(int),
		DHCP:          valid["dhcp_on"].(bool),
		PrimaryRack:   primary_rack,
		SecondaryRack: secondary_rack,
	}
	return result, nil
}
