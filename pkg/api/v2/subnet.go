// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

type Subnet struct {
	// Add the ControllerInterface in when we need to do things with the Subnet.
	// ControllerInterface ControllerInterface
	ResourceURI string `json:"resource_uri,omitempty"`
	ID          int    `json:"ID,omitempty"`
	Name        string `json:"Name,omitempty"`
	Space       string `json:"Space,omitempty"`
	VLAN        *VLAN  `json:"VLAN,omitempty"`
	Gateway     string `json:"gateway_ip,omitempty"`
	CIDR        string `json:"cidr,omitempty"`
	// DNSServers is a list of ip addresses of the DNS servers for the Subnet.
	// This list may be empty.
	DNSServers []string `json:"dns_servers,omitempty"`
}
