// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

// Zone represents a physical Zone that a Machine is in. The meaning of a
// physical Zone is up to you: it could identify e.g. a server rack, a network,
// or a data centre. Users can then allocate nodes from specific physical zones,
// to suit their redundancy or performance requirements.
type zone struct {
	// Add the Controller in when we need to do things with the Zone.
	// Controller Controller
	ResourceURI string `json:"resource_uri,omitempty"`
	Name        string `json:"Name,omitempty"`
	Description string `json:"Description,omitempty"`
}
