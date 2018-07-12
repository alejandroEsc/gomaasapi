// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"

	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/set"
	gc "gopkg.in/check.v1"
	"github.com/juju/gomaasapi/pkg/api/util"
)

type bootResourceSuite struct{}

var _ = gc.Suite(&bootResourceSuite{})

func (*bootResourceSuite) TestReadBootResourcesBadSchema(c *gc.C) {
	var b bootResource

	twoDotOh := []byte("wat?")
	err := json.Unmarshal(twoDotOh, &b)

	c.Check(err, jc.Satisfies, util.IsDeserializationError)
	c.Assert(err.Error(), gc.Equals, `boot resource base schema check failed: expected list, got string("wat?")`)
}

func (*bootResourceSuite) TestReadBootResources(c *gc.C) {
	var bootResources []bootResource
	err = json.Unmarshal([]byte(blockdevicesWithNullsResponse), &bootResources)

	c.Assert(err, jc.ErrorIsNil)
	c.Assert(bootResources, gc.HasLen, 5)
	trusty := bootResources[0]

	subarches := set.NewStrings("generic", "hwe-p", "hwe-q", "hwe-r", "hwe-s", "hwe-t")
	c.Assert(trusty.ID, gc.Equals, 5)
	c.Assert(trusty.Name, gc.Equals, "ubuntu/trusty")
	c.Assert(trusty.Type, gc.Equals, "Synced")
	c.Assert(trusty.Architecture, gc.Equals, "amd64/hwe-t")
	c.Assert(trusty.SubArchitectures, jc.DeepEquals, subarches)
	c.Assert(trusty.KernelFlavor, gc.Equals, "generic")
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
