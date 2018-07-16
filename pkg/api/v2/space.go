// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

type Space struct {
	// Add the ControllerInterface in when we need to do things with the Space.
	// ControllerInterface ControllerInterface
	ResourceURI string    `json:"resource_uri,omitempty"`
	ID          int       `json:"ID,omitempty"`
	Name        string    `json:"Name,omitempty"`
	Subnets     []*Subnet `json:"Subnets,omitempty"`
}
