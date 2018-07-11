// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

// VLAN represents an instance of a Virtual LAN. VLANs are a common way to
// create logically separate networks using the same physical infrastructure.
//
// Managed switches can assign VLANs to each port in either a “tagged” or an
// “untagged” manner. A VLAN is said to be “untagged” on a particular port when
// it is the default VLAN for that port, and requires no special configuration
// in order to access.
//
// “Tagged” VLANs (traditionally used by network administrators in order to
// aggregate multiple networks over inter-switch “trunk” lines) can also be used
// with nodes in MAAS. That is, if a switch port is configured such that
// “tagged” VLAN frames can be sent and received by a MAAS node, that MAAS node
// can be configured to automatically bring up VLAN interfaces, so that the
// deployed node can make use of them.
//
// A “Default VLAN” is created for every Fabric, to which every new VLAN-aware
// object in the Fabric will be associated to by default (unless otherwise
// specified).
type vlan struct {
	// Add the Controller in when we need to do things with the VLAN.
	// Controller Controller
	ResourceURI string `json:"resource_uri,omitempty"`
	ID          int    `json:"ID,omitempty"`
	Name        string `json:"Name,omitempty"`
	Fabric      string `json:"Fabric,omitempty"`
	// VID is the VLAN ID. eth0.10 -> VID = 10.
	VID int `json:"VID,omitempty"`
	// MTU (maximum transmission unit) is the largest Size packet or frame,
	// specified in octets (eight-bit bytes), that can be sent.
	MTU           int    `json:"MTU,omitempty"`
	DHCP          bool   `json:"dhcp_on,omitempty"`
	PrimaryRack   string `json:"primary_rack,omitempty"`
	SecondaryRack string `json:"secondary_rack,omitempty"`
}
