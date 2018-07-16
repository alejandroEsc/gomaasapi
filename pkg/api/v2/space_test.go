// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadSpacesBadSchema(t *testing.T) {
	var r Space
	err = json.Unmarshal([]byte("wat?"), &r)
	assert.Error(t, err)
}

func TestReadSpaces(t *testing.T) {
	var spaces []Space
	err = json.Unmarshal([]byte(spacesResponse), &spaces)
	assert.Nil(t, err)
	assert.Len(t, spaces, 1)

	space := spaces[0]
	assert.Equal(t, space.ID, 0)
	assert.Equal(t, space.Name, "Space-0")
	subnets := space.Subnets
	assert.Len(t, subnets, 2)
	assert.Equal(t, subnets[0].ID, 34)
}

const spacesResponse = `
[
    {
        "Subnets": [
            {
                "gateway_ip": null,
                "Name": "192.168.122.0/24",
                "VLAN": {
                    "Fabric": "Fabric-1",
                    "resource_uri": "/MAAS/api/2.0/VLANs/5001/",
                    "Name": "untagged",
                    "secondary_rack": null,
                    "primary_rack": null,
                    "VID": 0,
                    "dhcp_on": false,
                    "ID": 5001,
                    "MTU": 1500
                },
                "Space": "Space-0",
                "ID": 34,
                "resource_uri": "/MAAS/api/2.0/Subnets/34/",
                "dns_servers": [],
                "cidr": "192.168.122.0/24",
                "rdns_mode": 2
            },
            {
                "gateway_ip": "192.168.100.1",
                "Name": "192.168.100.0/24",
                "VLAN": {
                    "Fabric": "Fabric-0",
                    "resource_uri": "/MAAS/api/2.0/VLANs/1/",
                    "Name": "untagged",
                    "secondary_rack": null,
                    "primary_rack": "4y3h7n",
                    "VID": 0,
                    "dhcp_on": true,
                    "ID": 1,
                    "MTU": 1500
                },
                "Space": "Space-0",
                "ID": 1,
                "resource_uri": "/MAAS/api/2.0/Subnets/1/",
                "dns_servers": [],
                "cidr": "192.168.100.0/24",
                "rdns_mode": 2
            }
        ],
        "ID": 0,
        "Name": "Space-0",
        "resource_uri": "/MAAS/api/2.0/spaces/0/"
    }
]
`
