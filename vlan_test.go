// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type vlanSuite struct{}

var _ = gc.Suite(&vlanSuite{})

func (*vlanSuite) TestReadVLANsBadSchema(c *gc.C) {
	_, err := readVLANs(twoDotOh, "wat?")
	c.Assert(err.Error(), gc.Equals, `VLAN base schema check failed: expected list, got string("wat?")`)
}

func (s *vlanSuite) TestReadVLANsWithName(c *gc.C) {
	vlans, err := readVLANs(twoDotOh, parseJSON(c, vlanResponseWithName))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(vlans, gc.HasLen, 1)
	readVLAN := vlans[0]
	s.assertVLAN(c, readVLAN, &vlan{
		ID:            1,
		Name:          "untagged",
		Fabric:        "Fabric-0",
		VID:           2,
		MTU:           1500,
		DHCP:          true,
		PrimaryRack:   "a-rack",
		SecondaryRack: "",
	})
}

func (*vlanSuite) assertVLAN(c *gc.C, givenVLAN, expectedVLAN *vlan) {
	c.Check(givenVLAN.ID(), gc.Equals, expectedVLAN.ID)
	c.Check(givenVLAN.Name(), gc.Equals, expectedVLAN.Name)
	c.Check(givenVLAN.Fabric(), gc.Equals, expectedVLAN.Fabric)
	c.Check(givenVLAN.VID(), gc.Equals, expectedVLAN.VID)
	c.Check(givenVLAN.MTU(), gc.Equals, expectedVLAN.MTU)
	c.Check(givenVLAN.DHCP(), gc.Equals, expectedVLAN.DHCP)
	c.Check(givenVLAN.PrimaryRack(), gc.Equals, expectedVLAN.PrimaryRack)
	c.Check(givenVLAN.SecondaryRack(), gc.Equals, expectedVLAN.SecondaryRack)
}

func (s *vlanSuite) TestReadVLANsWithoutName(c *gc.C) {
	vlans, err := readVLANs(twoDotOh, parseJSON(c, vlanResponseWithoutName))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(vlans, gc.HasLen, 1)
	readVLAN := vlans[0]
	s.assertVLAN(c, readVLAN, &vlan{
		ID:            5006,
		Name:          "",
		Fabric:        "maas-management",
		VID:           30,
		MTU:           1500,
		DHCP:          true,
		PrimaryRack:   "4y3h7n",
		SecondaryRack: "",
	})
}

func (*vlanSuite) TestLowVersion(c *gc.C) {
	_, err := readVLANs(version.MustParse("1.9.0"), parseJSON(c, vlanResponseWithName))
	c.Assert(err.Error(), gc.Equals, `no VLAN read func for version 1.9.0`)
}

func (*vlanSuite) TestHighVersion(c *gc.C) {
	vlans, err := readVLANs(version.MustParse("2.1.9"), parseJSON(c, vlanResponseWithoutName))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(vlans, gc.HasLen, 1)
}

const (
	vlanResponseWithName = `
[
    {
        "Name": "untagged",
        "VID": 2,
        "primary_rack": "a-rack",
        "resource_uri": "/MAAS/api/2.0/VLANs/1/",
        "ID": 1,
        "secondary_rack": null,
        "Fabric": "Fabric-0",
        "MTU": 1500,
        "dhcp_on": true
    }
]
`
	vlanResponseWithoutName = `
[
    {
        "dhcp_on": true,
        "ID": 5006,
        "MTU": 1500,
        "Fabric": "maas-management",
        "VID": 30,
        "primary_rack": "4y3h7n",
        "Name": null,
        "external_dhcp": null,
        "resource_uri": "/MAAS/api/2.0/VLANs/5006/",
        "secondary_rack": null
    }
]
`
)
