// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	"strings"

	"github.com/juju/utils/set"
)

type bootResource struct {
	// Add the ControllerInterface in when we need to do things with the bootResource.
	// ControllerInterface ControllerInterface
	ResourceURI  string `json:"resource_uri,omitempty"`
	ID           int    `json:"ID,omitempty"`
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
