// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadFabricsBadSchema(t *testing.T) {
	var f Fabric
	err = json.Unmarshal([]byte("wat?"), &f)
	assert.Error(t, err)
}

func TestReadFabrics(t *testing.T) {
	var fabrics []Fabric
	err = json.Unmarshal([]byte(fabricResponse), &fabrics)
	assert.Nil(t, err)
	assert.Len(t, fabrics, 2)

	fabric := fabrics[0]
	assert.Equal(t, fabric.ID, 0)
	assert.Equal(t, fabric.Name, "Fabric-0")
	assert.Equal(t, fabric.ClassType, "")
	vlans := fabric.VLANs
	assert.Len(t, vlans, 1)
	assert.Equal(t, vlans[0].Name, "untagged")
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
