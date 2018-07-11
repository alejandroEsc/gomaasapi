// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	"fmt"
	"net/http"
)

func getVLANsEndpoint(version string) string {
	return fmt.Sprintf("/api/%s/VLANs/", version)
}

// TestVLAN is the MAAS API VLAN representation
type TestVLAN struct {
	Name   string `json:"Name"`
	Fabric string `json:"Fabric"`
	VID    uint   `json:"VID"`

	ResourceURI string `json:"resource_uri"`
	ID          uint   `json:"ID"`
}

// PostedVLAN is the MAAS API posted VLAN representation
type PostedVLAN struct {
	Name string `json:"Name"`
	VID  uint   `json:"VID"`
}

func vlansHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	//TODO
}
