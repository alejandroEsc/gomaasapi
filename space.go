// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

type space struct {
	// Add the Controller in when we need to do things with the space.
	// Controller Controller
	ResourceURI string    `json:"resource_uri,omitempty"`
	ID          int       `json:"ID,omitempty"`
	Name        string    `json:"Name,omitempty"`
	Subnets     []*subnet `json:"Subnets,omitempty"`
}
