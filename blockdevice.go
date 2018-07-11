// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type blockdevice struct {
	ResourceURI string

	ID      int
	Name    string
	Model   string
	IDPath  string
	Path    string
	UsedFor string
	Tags    []string

	BlockSize uint64
	UsedSize  uint64
	Size      uint64

	Partitions []*partition
}


func readBlockDevices(controllerVersion version.Number, source interface{}) ([]*blockdevice, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "blockdevice base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range blockdeviceDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no blockdevice read func for version %s", controllerVersion)
	}
	readFunc := blockdeviceDeserializationFuncs[deserialisationVersion]
	return readBlockDeviceList(valid, readFunc)
}

// readBlockDeviceList expects the Values of the sourceList to be string maps.
func readBlockDeviceList(sourceList []interface{}, readFunc blockdeviceDeserializationFunc) ([]*blockdevice, error) {
	result := make([]*blockdevice, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for blockdevice %d, %T", i, value)
		}
		blockdevice, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "blockdevice %d", i)
		}
		result = append(result, blockdevice)
	}
	return result, nil
}

type blockdeviceDeserializationFunc func(map[string]interface{}) (*blockdevice, error)

var blockdeviceDeserializationFuncs = map[version.Number]blockdeviceDeserializationFunc{
	twoDotOh: blockdevice_2_0,
}

func blockdevice_2_0(source map[string]interface{}) (*blockdevice, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),

		"ID":       schema.ForceInt(),
		"Name":     schema.String(),
		"Model":    schema.OneOf(schema.Nil(""), schema.String()),
		"id_path":  schema.OneOf(schema.Nil(""), schema.String()),
		"Path":     schema.String(),
		"used_for": schema.String(),
		"Tags":     schema.List(schema.String()),

		"block_size": schema.ForceUint(),
		"used_size":  schema.ForceUint(),
		"Size":       schema.ForceUint(),

		"Partitions": schema.List(schema.StringMap(schema.Any())),
	}
	checker := schema.FieldMap(fields, nil)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "blockdevice 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	partitions, err := readPartitionList(valid["Partitions"].([]interface{}), partition_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	model, _ := valid["Model"].(string)
	idPath, _ := valid["id_path"].(string)
	result := &blockdevice{
		ResourceURI: valid["resource_uri"].(string),

		ID:      valid["ID"].(int),
		Name:    valid["Name"].(string),
		Model:   model,
		IDPath:  idPath,
		Path:    valid["Path"].(string),
		UsedFor: valid["used_for"].(string),
		Tags:    convertToStringSlice(valid["Tags"]),

		BlockSize: valid["block_size"].(uint64),
		UsedSize:  valid["used_size"].(uint64),
		Size:      valid["Size"].(uint64),

		Partitions: partitions,
	}
	return result, nil
}
