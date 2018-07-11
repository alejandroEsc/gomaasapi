// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type linkSuite struct{}

var _ = gc.Suite(&linkSuite{})

func (*linkSuite) TestNilSubnet(c *gc.C) {
	var empty link
	c.Check(empty.Subnet() == nil, jc.IsTrue)
}

func (*linkSuite) TestReadLinksBadSchema(c *gc.C) {
	_, err := readLinks(twoDotOh, "wat?")
	c.Check(err, jc.Satisfies, IsDeserializationError)
	c.Assert(err.Error(), gc.Equals, `link base schema check failed: expected list, got string("wat?")`)
}

func (*linkSuite) TestReadLinks(c *gc.C) {
	links, err := readLinks(twoDotOh, parseJSON(c, linksResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(links, gc.HasLen, 2)
	link := links[0]
	c.Assert(link.ID(), gc.Equals, 69)
	c.Assert(link.Mode(), gc.Equals, "auto")
	c.Assert(link.IPAddress(), gc.Equals, "192.168.100.5")
	subnet := link.Subnet()
	c.Assert(subnet, gc.NotNil)
	c.Assert(subnet.Name(), gc.Equals, "192.168.100.0/24")
	// Second link has missing ip_address
	c.Assert(links[1].IPAddress(), gc.Equals, "")
}

func (*linkSuite) TestLowVersion(c *gc.C) {
	_, err := readLinks(version.MustParse("1.9.0"), parseJSON(c, linksResponse))
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
}

func (*linkSuite) TestHighVersion(c *gc.C) {
	links, err := readLinks(version.MustParse("2.1.9"), parseJSON(c, linksResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(links, gc.HasLen, 2)
}

var linksResponse = `
[
    {
        "ID": 69,
        "Mode": "auto",
        "ip_address": "192.168.100.5",
        "Subnet": {
            "resource_uri": "/MAAS/api/2.0/Subnets/1/",
            "ID": 1,
            "rdns_mode": 2,
            "VLAN": {
                "resource_uri": "/MAAS/api/2.0/VLANs/1/",
                "ID": 1,
                "secondary_rack": null,
                "MTU": 1500,
                "primary_rack": "4y3h7n",
                "Name": "untagged",
                "Fabric": "Fabric-0",
                "dhcp_on": true,
                "VID": 0
            },
            "dns_servers": [],
            "space": "space-0",
            "Name": "192.168.100.0/24",
            "gateway_ip": "192.168.100.1",
            "cidr": "192.168.100.0/24"
        }
    },
	{
        "ID": 70,
        "Mode": "auto",
        "Subnet": {
            "resource_uri": "/MAAS/api/2.0/Subnets/1/",
            "ID": 1,
            "rdns_mode": 2,
            "VLAN": {
                "resource_uri": "/MAAS/api/2.0/VLANs/1/",
                "ID": 1,
                "secondary_rack": null,
                "MTU": 1500,
                "primary_rack": "4y3h7n",
                "Name": "untagged",
                "Fabric": "Fabric-0",
                "dhcp_on": true,
                "VID": 0
            },
            "dns_servers": [],
            "space": "space-0",
            "Name": "192.168.100.0/24",
            "gateway_ip": "192.168.100.1",
            "cidr": "192.168.100.0/24"
        }
    }
]
`
