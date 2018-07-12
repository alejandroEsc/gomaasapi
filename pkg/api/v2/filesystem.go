// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

type filesystem struct {
	// There is no need for ControllerInterface based parsing of filesystems until we need it.
	// Currently the filesystem reading is only called by the Partition parsing.
	Type       string `json:"Type,omitempty"`
	MountPoint string `json:"mount_point,omitempty"`
	Label      string `json:"Label,omitempty"`
	UUID       string `json:"UUID,omitempty"`
	// no idea what the mount_options are as a value type, so ignoring for now.
}
