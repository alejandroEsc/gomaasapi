// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"strings"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/utils/set"
	"github.com/juju/version"
)

type bootResource struct {
	// Add the Controller in when we need to do things with the bootResource.
	// Controller Controller
	ResourceURI string `json:"resource_uri,omitempty"`
	ID           int `json:"ID,omitempty"`
	Name         string `json:"Name,omitempty"`
	Type         string `json:"type,omitempty"`
	Architecture string `json:"Architecture,omitempty"`
	SubArches    string `json:"subarches,omitempty"`
	KernelFlavor string `json:"kflavor,omitempty"`
}

// SubArchitectures implements BootResource.
func (b *bootResource) SubArchitectures() set.Strings {
	return set.NewStrings(strings.Split(b.SubArches, ",")...)
}


func readBootResources(controllerVersion version.Number, source interface{}) ([]*bootResource, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "boot resource base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range bootResourceDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no boot resource read func for version %s", controllerVersion)
	}
	readFunc := bootResourceDeserializationFuncs[deserialisationVersion]
	return readBootResourceList(valid, readFunc)
}

// readBootResourceList expects the Values of the sourceList to be string maps.
func readBootResourceList(sourceList []interface{}, readFunc bootResourceDeserializationFunc) ([]*bootResource, error) {
	result := make([]*bootResource, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for boot resource %d, %T", i, value)
		}
		bootResource, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "boot resource %d", i)
		}
		result = append(result, bootResource)
	}
	return result, nil
}

type bootResourceDeserializationFunc func(map[string]interface{}) (*bootResource, error)

var bootResourceDeserializationFuncs = map[version.Number]bootResourceDeserializationFunc{
	twoDotOh: bootResource_2_0,
}

func bootResource_2_0(source map[string]interface{}) (*bootResource, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),
		"ID":           schema.ForceInt(),
		"Name":         schema.String(),
		"type":         schema.String(),
		"Architecture": schema.String(),
		"subarches":    schema.String(),
		"kflavor":      schema.String(),
	}
	defaults := schema.Defaults{
		"subarches": "",
		"kflavor":   "",
	}
	checker := schema.FieldMap(fields, defaults)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "boot resource 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	result := &bootResource{
		ResourceURI:  valid["resource_uri"].(string),
		ID:           valid["ID"].(int),
		Name:         valid["Name"].(string),
		Type:         valid["type"].(string),
		Architecture: valid["Architecture"].(string),
		SubArches:    valid["subarches"].(string),
		KernelFlavor: valid["kflavor"].(string),
	}
	return result, nil
}
