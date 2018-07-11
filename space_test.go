// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type spaceSuite struct{}

var _ = gc.Suite(&spaceSuite{})

func (*spaceSuite) TestReadSpacesBadSchema(c *gc.C) {
	_, err := readSpaces(twoDotOh, "wat?")
	c.Assert(err.Error(), gc.Equals, `space base schema check failed: expected list, got string("wat?")`)
}

func (*spaceSuite) TestReadSpaces(c *gc.C) {
	spaces, err := readSpaces(twoDotOh, parseJSON(c, spacesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(spaces, gc.HasLen, 1)

	space := spaces[0]
	c.Assert(space.ID(), gc.Equals, 0)
	c.Assert(space.Name(), gc.Equals, "space-0")
	subnets := space.Subnets()
	c.Assert(subnets, gc.HasLen, 2)
	c.Assert(subnets[0].ID(), gc.Equals, 34)
}

func (*spaceSuite) TestLowVersion(c *gc.C) {
	_, err := readSpaces(version.MustParse("1.9.0"), parseJSON(c, spacesResponse))
	c.Assert(err.Error(), gc.Equals, `no space read func for version 1.9.0`)
}

func (*spaceSuite) TestHighVersion(c *gc.C) {
	spaces, err := readSpaces(version.MustParse("2.1.9"), parseJSON(c, spacesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(spaces, gc.HasLen, 1)
}

var spacesResponse = `
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
                "space": "space-0",
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
                "space": "space-0",
                "ID": 1,
                "resource_uri": "/MAAS/api/2.0/Subnets/1/",
                "dns_servers": [],
                "cidr": "192.168.100.0/24",
                "rdns_mode": 2
            }
        ],
        "ID": 0,
        "Name": "space-0",
        "resource_uri": "/MAAS/api/2.0/spaces/0/"
    }
]
`
