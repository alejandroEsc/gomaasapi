// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestReadZonesBadSchema(t *testing.T) {
	var z zone
	err = json.Unmarshal([]byte("wat?"), &z)
	assert.Error(t, err)
}

func TestReadZones(t *testing.T) {
	var zones []zone
	err = json.Unmarshal([]byte(zoneResponse), &zones)
	assert.Nil(t, err)

	assert.Len(t, zones, 2)
	assert.Equal(t, zones[0].Name,  "default")
	assert.Equal(t, zones[0].Description,  "default Description")
	assert.Equal(t, zones[1].Name,  "special")
	assert.Equal(t, zones[1].Description, "special Description")
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
