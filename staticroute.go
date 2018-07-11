// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

// StaticRoute defines an explicit route that users have requested to be added
// for a given Subnet.
type staticRoute struct {
	ResourceURI string `json:"resource_uri,omitempty"`
	ID          int    `json:"id,omitempty"`
	// Source is the Subnet that should have the route configured. (Machines
	// inside Source should use GatewayIP to reach Destination addresses.)
	Source *subnet `json:"source,omitempty"`
	// Destination is the Subnet that a MachineInterface wants to send packets to. We
	// want to configure a route to that Subnet via GatewayIP.
	Destination *subnet `json:"destination,omitempty"`
	// GatewayIP is the IPAddress to direct traffic to.
	GatewayIP string `json:"gateway_ip,omitempty"`
	// Metric is the routing Metric that determines whether this route will
	// take precedence over similar routes (there may be a route for 10/8, but
	// also a more concrete route for 10.0/16 that should take precedence if it
	// applies.) Metric should be a non-negative integer.
	Metric int `json:"metric,omitempty"`
}
