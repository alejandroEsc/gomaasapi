// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

type fabricSuite struct{}

var _ = gc.Suite(&fabricSuite{})

func (*fabricSuite) TestReadFabricsBadSchema(c *gc.C) {
	var f fabric
	err = json.Unmarshal([]byte("wat?"), &f)
	c.Assert(err.Error(), gc.Equals, `Fabric base schema check failed: expected list, got string("wat?")`)
}

func (*fabricSuite) TestReadFabrics(c *gc.C) {
	var fabrics []fabric
	err = json.Unmarshal([]byte(fabricResponse), &fabrics)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(fabrics, gc.HasLen, 2)

	fabric := fabrics[0]
	c.Assert(fabric.ID, gc.Equals, 0)
	c.Assert(fabric.Name, gc.Equals, "Fabric-0")
	c.Assert(fabric.ClassType, gc.Equals, "")
	vlans := fabric.VLANs
	c.Assert(vlans, gc.HasLen, 1)
	c.Assert(vlans[0].Name, gc.Equals, "untagged")
}

const fabricResponse = `
[
    {
        "Name": "Fabric-0",
        "ID": 0,
        "class_type": null,
        "VLANs": [
            {
                "Name": "untagged",
                "VID": 0,
                "primary_rack": "4y3h7n",
                "resource_uri": "/MAAS/api/2.0/VLANs/1/",
                "ID": 1,
                "secondary_rack": null,
                "Fabric": "Fabric-0",
                "MTU": 1500,
                "dhcp_on": true
            }
        ],
        "resource_uri": "/MAAS/api/2.0/fabrics/0/"
    },
    {
        "Name": "Fabric-1",
        "ID": 1,
        "class_type": null,
        "VLANs": [
            {
                "Name": "untagged",
                "VID": 0,
                "primary_rack": null,
                "resource_uri": "/MAAS/api/2.0/VLANs/5001/",
                "ID": 5001,
                "secondary_rack": null,
                "Fabric": "Fabric-1",
                "MTU": 1500,
                "dhcp_on": false
            }
        ],
        "resource_uri": "/MAAS/api/2.0/fabrics/1/"
    }
]
`
