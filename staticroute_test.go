// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type staticRouteSuite struct{}

var _ = gc.Suite(&staticRouteSuite{})

func (*staticRouteSuite) TestReadStaticRoutesBadSchema(c *gc.C) {
	_, err := readStaticRoutes(twoDotOh, "wat?")
	c.Assert(err.Error(), gc.Equals, `static-route base schema check failed: expected list, got string("wat?")`)
}

func (*staticRouteSuite) TestReadStaticRoutes(c *gc.C) {
	staticRoutes, err := readStaticRoutes(twoDotOh, parseJSON(c, staticRoutesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(staticRoutes, gc.HasLen, 1)

	staticRoute := staticRoutes[0]
	c.Assert(staticRoute.ID(), gc.Equals, 2)
	c.Assert(staticRoute.Metric(), gc.Equals, int(0))
	c.Assert(staticRoute.GatewayIP(), gc.Equals, "192.168.0.1")
	source := staticRoute.Source()
	c.Assert(source, gc.NotNil)
	c.Assert(source.Name(), gc.Equals, "192.168.0.0/24")
	c.Assert(source.CIDR(), gc.Equals, "192.168.0.0/24")
	destination := staticRoute.Destination()
	c.Assert(destination, gc.NotNil)
	c.Assert(destination.Name(), gc.Equals, "Local-192")
	c.Assert(destination.CIDR(), gc.Equals, "192.168.0.0/16")
}

func (*staticRouteSuite) TestLowVersion(c *gc.C) {
	_, err := readStaticRoutes(version.MustParse("1.9.0"), parseJSON(c, staticRoutesResponse))
	c.Assert(err.Error(), gc.Equals, `no static-route read func for version 1.9.0`)
}

func (*staticRouteSuite) TestHighVersion(c *gc.C) {
	staticRoutes, err := readStaticRoutes(version.MustParse("2.1.9"), parseJSON(c, staticRoutesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(staticRoutes, gc.HasLen, 1)
}

var staticRoutesResponse = `
[
    {
        "Destination": {
            "active_discovery": false,
            "ID": 3,
            "resource_uri": "/MAAS/api/2.0/Subnets/3/",
            "allow_proxy": true,
            "rdns_mode": 2,
            "dns_servers": [
                "8.8.8.8"
            ],
            "Name": "Local-192",
            "cidr": "192.168.0.0/16",
            "space": "space-0",
            "VLAN": {
                "Fabric": "Fabric-1",
                "ID": 5002,
                "dhcp_on": false,
                "primary_rack": null,
                "resource_uri": "/MAAS/api/2.0/VLANs/5002/",
                "MTU": 1500,
                "fabric_id": 1,
                "secondary_rack": null,
                "Name": "untagged",
                "external_dhcp": null,
                "VID": 0
            },
            "gateway_ip": "192.168.0.1"
        },
        "Source": {
            "active_discovery": false,
            "ID": 1,
            "resource_uri": "/MAAS/api/2.0/Subnets/1/",
            "allow_proxy": true,
            "rdns_mode": 2,
            "dns_servers": [],
            "Name": "192.168.0.0/24",
            "cidr": "192.168.0.0/24",
            "space": "space-0",
            "VLAN": {
                "Fabric": "Fabric-0",
                "ID": 5001,
                "dhcp_on": false,
                "primary_rack": null,
                "resource_uri": "/MAAS/api/2.0/VLANs/5001/",
                "MTU": 1500,
                "fabric_id": 0,
                "secondary_rack": null,
                "Name": "untagged",
                "external_dhcp": "192.168.0.1",
                "VID": 0
            },
            "gateway_ip": null
        },
        "ID": 2,
        "resource_uri": "/MAAS/api/2.0/static-routes/2/",
        "Metric": 0,
        "gateway_ip": "192.168.0.1"
    }
]
`
