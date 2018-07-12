// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"testing"
)

type subnetSuite struct{}

var _ = gc.Suite(&subnetSuite{})

func TestNilVLAN(t *testing.T) {
	var empty subnet
	c.Check(empty.VLAN == nil, jc.IsTrue)
}

func TestReadSubnetsBadSchema(t *testing.T) {
	var s subnet
	err = json.Unmarshal([]byte("wat?"), &s)

	c.Assert(err.Error(), gc.Equals, `Subnet base schema check failed: expected list, got string("wat?")`)
}

func TestReadSubnets(t *testing.T) {
	var subnets []subnet
	err = json.Unmarshal([]byte(subnetResponse), &subnets)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(subnets, gc.HasLen, 2)

	subnet := subnets[0]
	c.Assert(subnet.ID, gc.Equals, 1)
	c.Assert(subnet.Name, gc.Equals, "192.168.100.0/24")
	c.Assert(subnet.Space, gc.Equals, "space-0")
	c.Assert(subnet.Gateway, gc.Equals, "192.168.100.1")
	c.Assert(subnet.CIDR, gc.Equals, "192.168.100.0/24")
	vlan := subnet.VLAN
	c.Assert(vlan, gc.NotNil)
	c.Assert(vlan.Name, gc.Equals, "untagged")
	c.Assert(subnet.DNSServers, jc.DeepEquals, []string{"8.8.8.8", "8.8.4.4"})
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
