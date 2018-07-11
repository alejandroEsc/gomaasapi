// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

type link struct {
	ID        int     `json:"ID,omitempty"`
	Mode      string  `json:"Mode,omitempty"`
	Subnet    *subnet `json:"Subnet,omitempty"`
	IPAddress string  `json:"ip_address,omitempty"`
}
