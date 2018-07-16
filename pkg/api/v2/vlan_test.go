// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadVLANsBadSchema(t *testing.T) {
	var v vlan
	err = json.Unmarshal([]byte("wat?"), &v)
	assert.Error(t, err)
}

func TestReadVLANsWithName(t *testing.T) {
	var vlans []vlan
	err = json.Unmarshal([]byte(vlanResponseWithName), &vlans)
	assert.Nil(t, err)
	assert.Len(t, vlans, 1)
	readVLAN := vlans[0]
	expectedVLAN := &vlan{
		ResourceURI:   "/MAAS/api/2.0/VLANs/1/",
		ID:            1,
		Name:          "untagged",
		Fabric:        "Fabric-0",
		VID:           2,
		MTU:           1500,
		DHCP:          true,
		PrimaryRack:   "a-rack",
		SecondaryRack: "",
	}
	assertVLAN(t, expectedVLAN, &readVLAN)
}

func assertVLAN(t *testing.T, givenVLAN, expectedVLAN *vlan) {
	assert.EqualValues(t, expectedVLAN, givenVLAN)
	//c.Check(givenVLAN.ID, gc.Equals, expectedVLAN.ID)
	//c.Check(givenVLAN.Name, gc.Equals, expectedVLAN.Name)
	//c.Check(givenVLAN.Fabric, gc.Equals, expectedVLAN.Fabric)
	//c.Check(givenVLAN.VID, gc.Equals, expectedVLAN.VID)
	//c.Check(givenVLAN.MTU, gc.Equals, expectedVLAN.MTU)
	//c.Check(givenVLAN.DHCP, gc.Equals, expectedVLAN.DHCP)
	//c.Check(givenVLAN.PrimaryRack, gc.Equals, expectedVLAN.PrimaryRack)
	//c.Check(givenVLAN.SecondaryRack, gc.Equals, expectedVLAN.SecondaryRack)
}

func TestReadVLANsWithoutName(t *testing.T) {
	var vlans []vlan
	err = json.Unmarshal([]byte(vlanResponseWithoutName), &vlans)
	assert.Nil(t, err)
	assert.Len(t, vlans, 1)
	readVLAN := vlans[0]
	expectedVLAN := &vlan{
		ResourceURI:   "/MAAS/api/2.0/VLANs/5006/",
		ID:            5006,
		Name:          "",
		Fabric:        "maas-management",
		VID:           30,
		MTU:           1500,
		DHCP:          true,
		PrimaryRack:   "4y3h7n",
		SecondaryRack: "",
	}
	assertVLAN(t, expectedVLAN, &readVLAN)
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
