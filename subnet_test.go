// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type subnetSuite struct{}

var _ = gc.Suite(&subnetSuite{})

func (*subnetSuite) TestNilVLAN(c *gc.C) {
	var empty subnet
	c.Check(empty.VLAN == nil, jc.IsTrue)
}

func (*subnetSuite) TestReadSubnetsBadSchema(c *gc.C) {
	_, err := readSubnets(twoDotOh, "wat?")
	c.Assert(err.Error(), gc.Equals, `Subnet base schema check failed: expected list, got string("wat?")`)
}

func (*subnetSuite) TestReadSubnets(c *gc.C) {
	subnets, err := readSubnets(twoDotOh, parseJSON(c, subnetResponse))
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

func (*subnetSuite) TestLowVersion(c *gc.C) {
	_, err := readSubnets(version.MustParse("1.9.0"), parseJSON(c, subnetResponse))
	c.Assert(err.Error(), gc.Equals, `no Subnet read func for version 1.9.0`)
}

func (*subnetSuite) TestHighVersion(c *gc.C) {
	subnets, err := readSubnets(version.MustParse("2.1.9"), parseJSON(c, subnetResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(subnets, gc.HasLen, 2)
}

var subnetResponse = `
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
