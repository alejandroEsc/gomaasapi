// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"

	"github.com/juju/utils/set"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadBootResourcesBadSchema(t *testing.T) {
	var b bootResource
	err := json.Unmarshal([]byte("wat?"), &b)
	assert.Error(t, err)
}

func TestReadBootResources(t *testing.T) {
	var bootResources []bootResource
	err = json.Unmarshal([]byte(bootResourcesResponse), &bootResources)
	assert.Nil(t, err)

	assert.Len(t, bootResources, 5)
	trusty := bootResources[0]

	subarches := set.NewStrings("generic", "hwe-p", "hwe-q", "hwe-r", "hwe-s", "hwe-t")

	assert.Equal(t, trusty.ID, 5)
	assert.Equal(t, trusty.Name, "ubuntu/trusty")
	assert.Equal(t, trusty.Type, "Synced")
	assert.Equal(t, trusty.Architecture, "amd64/hwe-t")
	assert.ObjectsAreEqual(trusty.SubArchitectures, subarches)
	assert.Equal(t, trusty.KernelFlavor, "generic")
}

const bootResourcesResponse = `
[
    {
        "Architecture": "amd64/hwe-t",
        "type": "Synced",
        "subarches": "generic,hwe-p,hwe-q,hwe-r,hwe-s,hwe-t",
        "kflavor": "generic",
        "Name": "ubuntu/trusty",
        "ID": 5,
        "resource_uri": "/MAAS/api/2.0/boot-resources/5/"
    },
    {
        "Architecture": "amd64/hwe-u",
        "type": "Synced",
        "subarches": "generic,hwe-p,hwe-q,hwe-r,hwe-s,hwe-t,hwe-u",
        "Name": "ubuntu/trusty",
        "ID": 1,
        "resource_uri": "/MAAS/api/2.0/boot-resources/1/"
    },
    {
        "Architecture": "amd64/hwe-v",
        "type": "Synced",
        "subarches": "generic,hwe-p,hwe-q,hwe-r,hwe-s,hwe-t,hwe-u,hwe-v",
        "kflavor": "generic",
        "Name": "ubuntu/trusty",
        "ID": 3,
        "resource_uri": "/MAAS/api/2.0/boot-resources/3/"
    },
    {
        "Architecture": "amd64/hwe-w",
        "type": "Synced",
        "kflavor": "generic",
        "Name": "ubuntu/trusty",
        "ID": 4,
        "resource_uri": "/MAAS/api/2.0/boot-resources/4/"
    },
    {
        "Architecture": "amd64/hwe-x",
        "type": "Synced",
        "subarches": "generic,hwe-p,hwe-q,hwe-r,hwe-s,hwe-t,hwe-u,hwe-v,hwe-w,hwe-x",
        "kflavor": "generic",
        "Name": "ubuntu/xenial",
        "ID": 2,
        "resource_uri": "/MAAS/api/2.0/boot-resources/2/"
    }
]
`
