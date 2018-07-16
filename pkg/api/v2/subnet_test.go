// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadSubnetsBadSchema(t *testing.T) {
	var s subnet
	err = json.Unmarshal([]byte("wat?"), &s)
	assert.Error(t, err)
}

func TestReadSubnets(t *testing.T) {
	var subnets []subnet
	err = json.Unmarshal([]byte(subnetResponse), &subnets)
	assert.Nil(t, err)
	assert.Len(t, subnets, 2)

	subnet := subnets[0]
	assert.Equal(t, subnet.ID, 1)
	assert.Equal(t, subnet.Name, "192.168.100.0/24")
	assert.Equal(t, subnet.Space, "space-0")
	assert.Equal(t, subnet.Gateway, "192.168.100.1")
	assert.Equal(t, subnet.CIDR, "192.168.100.0/24")
	vlan := subnet.VLAN
	assert.NotNil(t, vlan)
	assert.Equal(t, vlan.Name, "untagged")
	assert.EqualValues(t, subnet.DNSServers, []string{"8.8.8.8", "8.8.4.4"})
}

const subnetResponse = `
[
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
        "space": "space-0",
        "ID": 1,
        "resource_uri": "/MAAS/api/2.0/Subnets/1/",
        "dns_servers": ["8.8.8.8", "8.8.4.4"],
        "cidr": "192.168.100.0/24",
        "rdns_mode": 2
    },
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
        "space": "space-0",
        "ID": 34,
        "resource_uri": "/MAAS/api/2.0/Subnets/34/",
        "dns_servers": null,
        "cidr": "192.168.122.0/24",
        "rdns_mode": 2
    }
]
`
