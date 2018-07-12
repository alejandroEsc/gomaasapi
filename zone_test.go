// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	"encoding/json"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

type zoneSuite struct{}

var _ = gc.Suite(&zoneSuite{})

func (*zoneSuite) TestReadZonesBadSchema(c *gc.C) {
	var z zone
	err = json.Unmarshal([]byte("wat?"), &z)
	c.Assert(err.Error(), gc.Equals, `Zone base schema check failed: expected list, got string("wat?")`)
}

func (*zoneSuite) TestReadZones(c *gc.C) {
	var zones []zone
	err = json.Unmarshal([]byte(zoneResponse), &zones)

	c.Assert(err, jc.ErrorIsNil)
	c.Assert(zones, gc.HasLen, 2)
	c.Assert(zones[0].Name, gc.Equals, "default")
	c.Assert(zones[0].Description, gc.Equals, "default Description")
	c.Assert(zones[1].Name, gc.Equals, "special")
	c.Assert(zones[1].Description, gc.Equals, "special Description")
}

const zoneResponse = `
[
    {
        "Description": "default Description",
        "resource_uri": "/MAAS/api/2.0/zones/default/",
        "Name": "default"
    }, {
        "Description": "special Description",
        "resource_uri": "/MAAS/api/2.0/zones/special/",
        "Name": "special"
    }
]
`
