// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import "github.com/juju/schema"

type filesystem struct {
	Type       string
	MountPoint string
	Label      string
	UUID       string
	// no idea what the mount_options are as a value type, so ignoring for now.
}

// There is no need for Controller based parsing of filesystems until we need it.
// Currently the filesystem reading is only called by the Partition parsing.

func filesystem2_0(source map[string]interface{}) (*filesystem, error) {
	fields := schema.Fields{
		"Type":      schema.String(),
		"mount_point": schema.OneOf(schema.Nil(""), schema.String()),
		"Label":       schema.OneOf(schema.Nil(""), schema.String()),
		"UUID":        schema.String(),
		// TODO: mount_options when we know the type (note it can be
		// nil).
	}
	defaults := schema.Defaults{
		"mount_point": "",
		"Label":       "",
	}
	checker := schema.FieldMap(fields, defaults)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "filesystem 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.
	mount_point, _ := valid["mount_point"].(string)
	label, _ := valid["Label"].(string)
	result := &filesystem{
		Type:       valid["Type"].(string),
		MountPoint: mount_point,
		Label:      label,
		UUID:       valid["UUID"].(string),
	}
	return result, nil
}
