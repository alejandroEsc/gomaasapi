// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

// Zone represents a physical Zone that a MachineInterface is in. The meaning of a
// physical Zone is up to you: it could identify e.g. a server rack, a network,
// or a data centre. Users can then allocate nodes from specific physical zones,
// to suit their redundancy or performance requirements.
type Zone struct {
	// Add the ControllerInterface in when we need to do things with the Zone.
	// ControllerInterface ControllerInterface
	ResourceURI string `json:"resource_uri,omitempty"`
	Name        string `json:"Name,omitempty"`
	Description string `json:"Description,omitempty"`
}
