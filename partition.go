// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type partition struct {
	ResourceURI string

	ID   int
	Path string
	UUID string

	UsedFor string
	Size    uint64

	FileSystem *filesystem
}

func readPartitions(controllerVersion version.Number, source interface{}) ([]*partition, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "partition base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range partitionDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no partition read func for version %s", controllerVersion)
	}
	readFunc := partitionDeserializationFuncs[deserialisationVersion]
	return readPartitionList(valid, readFunc)
}

// readPartitionList expects the Values of the sourceList to be string maps.
func readPartitionList(sourceList []interface{}, readFunc partitionDeserializationFunc) ([]*partition, error) {
	result := make([]*partition, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for partition %d, %T", i, value)
		}
		partition, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "partition %d", i)
		}
		result = append(result, partition)
	}
	return result, nil
}

type partitionDeserializationFunc func(map[string]interface{}) (*partition, error)

var partitionDeserializationFuncs = map[version.Number]partitionDeserializationFunc{
	twoDotOh: partition_2_0,
}

func partition_2_0(source map[string]interface{}) (*partition, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),

		"ID":   schema.ForceInt(),
		"Path": schema.String(),
		"UUID": schema.OneOf(schema.Nil(""), schema.String()),

		"used_for": schema.String(),
		"Size":     schema.ForceUint(),

		"filesystem": schema.OneOf(schema.Nil(""), schema.StringMap(schema.Any())),
	}
	defaults := schema.Defaults{
		"UUID": "",
	}
	checker := schema.FieldMap(fields, defaults)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "partition 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	var filesystem *filesystem
	if fsSource := valid["filesystem"]; fsSource != nil {
		filesystem, err = filesystem2_0(fsSource.(map[string]interface{}))
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	uuid, _ := valid["UUID"].(string)
	result := &partition{
		ResourceURI: valid["resource_uri"].(string),
		ID:          valid["ID"].(int),
		Path:        valid["Path"].(string),
		UUID:        uuid,
		UsedFor:     valid["used_for"].(string),
		Size:        valid["Size"].(uint64),
		FileSystem:  filesystem,
	}
	return result, nil
}
