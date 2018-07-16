// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

type Link struct {
	ID        int     `json:"ID,omitempty"`
	Mode      string  `json:"Mode,omitempty"`
	Subnet    *Subnet `json:"Subnet,omitempty"`
	IPAddress string  `json:"ip_address,omitempty"`
}
