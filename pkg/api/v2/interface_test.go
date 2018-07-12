// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"net/http"

	"encoding/json"

	"github.com/juju/errors"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/util"
)

type interfaceSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&interfaceSuite{})

func (*interfaceSuite) TestNilVLAN(c *gc.C) {
	var empty MachineNetworkInterface
	c.Check(empty.VLAN == nil, jc.IsTrue)
}

func (*interfaceSuite) TestReadInterfacesBadSchema(c *gc.C) {
	var b MachineNetworkInterface
	err = json.Unmarshal([]byte("wat?"), &b)

	c.Check(err, jc.Satisfies, util.IsDeserializationError)
	c.Assert(err.Error(), gc.Equals, `interface base schema check failed: expected list, got string("wat?")`)
}

func (*interfaceSuite) TestReadInterfacesNulls(c *gc.C) {
	var iface MachineNetworkInterface
	err = json.Unmarshal([]byte(interfaceNullsResponse), &iface)

	c.Assert(err, jc.ErrorIsNil)

	c.Check(iface.MACAddress, gc.Equals, "")
	c.Check(iface.Tags, jc.DeepEquals, []string{})
	c.Check(iface.VLAN, gc.IsNil)
}

func (s *interfaceSuite) checkInterface(c *gc.C, iface *MachineNetworkInterface) {
	c.Check(iface.ID, gc.Equals, 40)
	c.Check(iface.Name, gc.Equals, "eth0")
	c.Check(iface.Type, gc.Equals, "physical")
	c.Check(iface.Enabled, jc.IsTrue)
	c.Check(iface.Tags, jc.DeepEquals, []string{"foo", "bar"})

	c.Check(iface.MACAddress, gc.Equals, "52:54:00:c9:6a:45")
	c.Check(iface.EffectiveMTU, gc.Equals, 1500)

	c.Check(iface.Parents, jc.DeepEquals, []string{"bond0"})
	c.Check(iface.Children, jc.DeepEquals, []string{"eth0.1", "eth0.2"})

	vlan := iface.VLAN
	c.Assert(vlan, gc.NotNil)
	c.Check(vlan.Name, gc.Equals, "untagged")

	links := iface.Links
	c.Assert(links, gc.HasLen, 1)
	c.Check(links[0].ID, gc.Equals, 69)
}

func (s *interfaceSuite) TestReadInterfaces(c *gc.C) {
	var interfaces []MachineNetworkInterface
	err = json.Unmarshal([]byte(interfacesResponse), &interfaces)

	c.Assert(err, jc.ErrorIsNil)
	c.Assert(interfaces, gc.HasLen, 1)
	s.checkInterface(c, &interfaces[0])
}

func (s *interfaceSuite) TestReadInterface(c *gc.C) {
	var iface MachineNetworkInterface
	err = json.Unmarshal([]byte(interfacesResponse), &iface)

	c.Assert(err, jc.ErrorIsNil)
	s.checkInterface(c, &iface)
}

func (s *interfaceSuite) getServerAndNewInterface(c *gc.C) (*client.SimpleTestServer, *MachineNetworkInterface) {
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/devices/", http.StatusOK, devicesResponse)
	devices, err := controller.Devices(DevicesArgs{})
	c.Assert(err, jc.ErrorIsNil)
	device := devices[0]
	server.AddPostResponse(device.interfacesURI()+"?op=create_physical", http.StatusOK, interfaceResponse)
	iface, err := device.CreateInterface(minimalCreateInterfaceArgs())
	c.Assert(err, jc.ErrorIsNil)
	return server, iface
}

func (s *interfaceSuite) TestDelete(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	// Successful delete is 204 - StatusNoContent - We hope, would be consistent
	// with device deletions.
	server.AddDeleteResponse(iface.ResourceURI, http.StatusNoContent, "")
	err := iface.Delete()
	c.Assert(err, jc.ErrorIsNil)
}

func (s *interfaceSuite) TestDelete404(c *gc.C) {
	_, iface := s.getServerAndNewInterface(c)
	// No Path, so 404
	err := iface.Delete()
	c.Assert(err, jc.Satisfies, util.IsNoMatchError)
}

func (s *interfaceSuite) TestDeleteForbidden(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddDeleteResponse(iface.ResourceURI, http.StatusForbidden, "")
	err := iface.Delete()
	c.Assert(err, jc.Satisfies, util.IsPermissionError)
}

func (s *interfaceSuite) TestDeleteUnknown(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddDeleteResponse(iface.ResourceURI, http.StatusConflict, "")
	err := iface.Delete()
	c.Assert(err, jc.Satisfies, util.IsUnexpectedError)
}

func (s *interfaceSuite) TestLinkSubnetArgs(c *gc.C) {
	for i, test := range []struct {
		args    LinkSubnetArgs
		errText string
	}{{
		errText: "missing Mode not valid",
	}, {
		args:    LinkSubnetArgs{Mode: LinkModeDHCP},
		errText: "missing Subnet not valid",
	}, {
		args:    LinkSubnetArgs{Mode: InterfaceLinkMode("foo")},
		errText: `unknown Mode value ("foo") not valid`,
	}, {
		args: LinkSubnetArgs{Mode: LinkModeDHCP, Subnet: &subnet{}},
	}, {
		args: LinkSubnetArgs{Mode: LinkModeStatic, Subnet: &subnet{}},
	}, {
		args: LinkSubnetArgs{Mode: LinkModeLinkUp, Subnet: &subnet{}},
	}, {
		args:    LinkSubnetArgs{Mode: LinkModeDHCP, Subnet: &subnet{}, IPAddress: "10.10.10.10"},
		errText: `setting IP Address when Mode is not LinkModeStatic not valid`,
	}, {
		args: LinkSubnetArgs{Mode: LinkModeStatic, Subnet: &subnet{}, IPAddress: "10.10.10.10"},
	}, {
		args:    LinkSubnetArgs{Mode: LinkModeLinkUp, Subnet: &subnet{}, IPAddress: "10.10.10.10"},
		errText: `setting IP Address when Mode is not LinkModeStatic not valid`,
	}, {
		args:    LinkSubnetArgs{Mode: LinkModeDHCP, Subnet: &subnet{}, DefaultGateway: true},
		errText: `specifying DefaultGateway for Mode "DHCP" not valid`,
	}, {
		args: LinkSubnetArgs{Mode: LinkModeStatic, Subnet: &subnet{}, DefaultGateway: true},
	}, {
		args:    LinkSubnetArgs{Mode: LinkModeLinkUp, Subnet: &subnet{}, DefaultGateway: true},
		errText: `specifying DefaultGateway for Mode "LINK_UP" not valid`,
	}} {
		c.Logf("test %d", i)
		err := test.args.Validate()
		if test.errText == "" {
			c.Check(err, jc.ErrorIsNil)
		} else {
			c.Check(err, jc.Satisfies, errors.IsNotValid)
			c.Check(err.Error(), gc.Equals, test.errText)
		}
	}
}

func (s *interfaceSuite) TestLinkSubnetValidates(c *gc.C) {
	_, iface := s.getServerAndNewInterface(c)
	err := iface.LinkSubnet(LinkSubnetArgs{})
	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err.Error(), gc.Equals, "missing Mode not valid")
}

func (s *interfaceSuite) TestLinkSubnetGood(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	// The changed information is there just for the test to show that the response
	// is parsed and the interface updated
	response := util.UpdateJSONMap(c, interfaceResponse, map[string]interface{}{
		"Name": "eth42",
	})
	server.AddPostResponse(iface.ResourceURI+"?op=link_subnet", http.StatusOK, response)
	args := LinkSubnetArgs{
		Mode:           LinkModeStatic,
		Subnet:         &subnet{ID: 42},
		IPAddress:      "10.10.10.10",
		DefaultGateway: true,
	}
	err := iface.LinkSubnet(args)
	c.Check(err, jc.ErrorIsNil)
	c.Check(iface.Name, gc.Equals, "eth42")

	request := server.LastRequest()
	form := request.PostForm
	c.Assert(form.Get("Mode"), gc.Equals, "STATIC")
	c.Assert(form.Get("Subnet"), gc.Equals, "42")
	c.Assert(form.Get("ip_address"), gc.Equals, "10.10.10.10")
	c.Assert(form.Get("default_gateway"), gc.Equals, "true")
}

func (s *interfaceSuite) TestLinkSubnetMissing(c *gc.C) {
	_, iface := s.getServerAndNewInterface(c)
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &subnet{ID: 42},
	}
	err := iface.LinkSubnet(args)
	c.Check(err, jc.Satisfies, util.IsBadRequestError)
}

func (s *interfaceSuite) TestLinkSubnetForbidden(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddPostResponse(iface.ResourceURI+"?op=link_subnet", http.StatusForbidden, "bad user")
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &subnet{ID: 42},
	}
	err := iface.LinkSubnet(args)
	c.Check(err, jc.Satisfies, util.IsPermissionError)
	c.Check(err.Error(), gc.Equals, "bad user")
}

func (s *interfaceSuite) TestLinkSubnetNoAddressesAvailable(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddPostResponse(iface.ResourceURI+"?op=link_subnet", http.StatusServiceUnavailable, "no addresses")
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &subnet{ID: 42},
	}
	err := iface.LinkSubnet(args)
	c.Check(err, jc.Satisfies, util.IsCannotCompleteError)
	c.Check(err.Error(), gc.Equals, "no addresses")
}

func (s *interfaceSuite) TestLinkSubnetUnknown(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddPostResponse(iface.ResourceURI+"?op=link_subnet", http.StatusMethodNotAllowed, "wat?")
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &subnet{ID: 42},
	}
	err := iface.LinkSubnet(args)
	c.Check(err, jc.Satisfies, util.IsUnexpectedError)
	c.Assert(err.Error(), gc.Equals, "unexpected: ServerError: 405 Method Not Allowed (wat?)")
}

func (s *interfaceSuite) TestUnlinkSubnetValidates(c *gc.C) {
	_, iface := s.getServerAndNewInterface(c)
	err := iface.UnlinkSubnet(nil)
	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err.Error(), gc.Equals, "missing Subnet not valid")
}

func (s *interfaceSuite) TestUnlinkSubnetNotLinked(c *gc.C) {
	_, iface := s.getServerAndNewInterface(c)
	err := iface.UnlinkSubnet(&subnet{ID: 42})
	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err.Error(), gc.Equals, "unlinked Subnet not valid")
}

func (s *interfaceSuite) TestUnlinkSubnetGood(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	// The changed information is there just for the test to show that the response
	// is parsed and the interface updated
	response := util.UpdateJSONMap(c, interfaceResponse, map[string]interface{}{
		"Name": "eth42",
	})
	server.AddPostResponse(iface.ResourceURI+"?op=unlink_subnet", http.StatusOK, response)
	err := iface.UnlinkSubnet(&subnet{ID: 1})
	c.Check(err, jc.ErrorIsNil)
	c.Check(iface.Name, gc.Equals, "eth42")

	request := server.LastRequest()
	form := request.PostForm
	// The link ID that contains Subnet 1 has an internal ID of 69.
	c.Assert(form.Get("ID"), gc.Equals, "69")
}

func (s *interfaceSuite) TestUnlinkSubnetMissing(c *gc.C) {
	_, iface := s.getServerAndNewInterface(c)
	err := iface.UnlinkSubnet(&subnet{ID: 1})
	c.Check(err, jc.Satisfies, util.IsBadRequestError)
}

func (s *interfaceSuite) TestUnlinkSubnetForbidden(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddPostResponse(iface.ResourceURI+"?op=unlink_subnet", http.StatusForbidden, "bad user")
	err := iface.UnlinkSubnet(&subnet{ID: 1})
	c.Check(err, jc.Satisfies, util.IsPermissionError)
	c.Check(err.Error(), gc.Equals, "bad user")
}

func (s *interfaceSuite) TestUnlinkSubnetUnknown(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddPostResponse(iface.ResourceURI+"?op=unlink_subnet", http.StatusMethodNotAllowed, "wat?")
	err := iface.UnlinkSubnet(&subnet{ID: 1})
	c.Check(err, jc.Satisfies, util.IsUnexpectedError)
	c.Assert(err.Error(), gc.Equals, "unexpected: ServerError: 405 Method Not Allowed (wat?)")
}

func (s *interfaceSuite) TestUpdateNoChangeNoRequest(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	count := server.RequestCount()
	err := iface.Update(UpdateInterfaceArgs{})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(server.RequestCount(), gc.Equals, count)
}

func (s *interfaceSuite) TestUpdateMissing(c *gc.C) {
	_, iface := s.getServerAndNewInterface(c)
	err := iface.Update(UpdateInterfaceArgs{Name: "eth2"})
	c.Check(err, jc.Satisfies, util.IsNoMatchError)
}

func (s *interfaceSuite) TestUpdateForbidden(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddPutResponse(iface.ResourceURI, http.StatusForbidden, "bad user")
	err := iface.Update(UpdateInterfaceArgs{Name: "eth2"})
	c.Check(err, jc.Satisfies, util.IsPermissionError)
	c.Check(err.Error(), gc.Equals, "bad user")
}

func (s *interfaceSuite) TestUpdateUnknown(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	server.AddPutResponse(iface.ResourceURI, http.StatusMethodNotAllowed, "wat?")
	err := iface.Update(UpdateInterfaceArgs{Name: "eth2"})
	c.Check(err, jc.Satisfies, util.IsUnexpectedError)
	c.Assert(err.Error(), gc.Equals, "unexpected: ServerError: 405 Method Not Allowed (wat?)")
}

func (s *interfaceSuite) TestUpdateGood(c *gc.C) {
	server, iface := s.getServerAndNewInterface(c)
	// The changed information is there just for the test to show that the response
	// is parsed and the interface updated
	response := util.UpdateJSONMap(c, interfaceResponse, map[string]interface{}{
		"Name": "eth42",
	})
	server.AddPutResponse(iface.ResourceURI, http.StatusOK, response)
	args := UpdateInterfaceArgs{
		Name:       "eth42",
		MACAddress: "c3-52-51-b4-50-cd",
		VLAN:       &vlan{ID: 13},
	}
	err := iface.Update(args)
	c.Check(err, jc.ErrorIsNil)
	c.Check(iface.Name, gc.Equals, "eth42")

	request := server.LastRequest()
	form := request.PostForm
	c.Assert(form.Get("Name"), gc.Equals, "eth42")
	c.Assert(form.Get("mac_address"), gc.Equals, "c3-52-51-b4-50-cd")
	c.Assert(form.Get("VLAN"), gc.Equals, "13")
}

const (
	interfacesResponse = "[" + interfaceResponse + "]"
	interfaceResponse  = `
{
    "effective_mtu": 1500,
    "mac_address": "52:54:00:c9:6a:45",
    "Children": ["eth0.1", "eth0.2"],
    "discovered": [],
    "params": "some params",
    "VLAN": {
        "resource_uri": "/MAAS/api/2.0/VLANs/1/",
        "ID": 1,
        "secondary_rack": null,
        "MTU": 1500,
        "primary_rack": "4y3h7n",
        "Name": "untagged",
        "Fabric": "Fabric-0",
        "dhcp_on": true,
        "VID": 0
    },
    "Name": "eth0",
    "Enabled": true,
    "Parents": ["bond0"],
    "ID": 40,
    "type": "physical",
    "resource_uri": "/MAAS/api/2.0/nodes/4y3ha6/interfaces/40/",
    "Tags": ["foo", "bar"],
    "Links": [
        {
            "ID": 69,
            "Mode": "auto",
            "Subnet": {
                "resource_uri": "/MAAS/api/2.0/Subnets/1/",
                "ID": 1,
                "rdns_mode": 2,
                "VLAN": {
                    "resource_uri": "/MAAS/api/2.0/VLANs/1/",
                    "ID": 1,
                    "secondary_rack": null,
                    "MTU": 1500,
                    "primary_rack": "4y3h7n",
                    "Name": "untagged",
                    "Fabric": "Fabric-0",
                    "dhcp_on": true,
                    "VID": 0
                },
                "dns_servers": [],
                "space": "space-0",
                "Name": "192.168.100.0/24",
                "gateway_ip": "192.168.100.1",
                "cidr": "192.168.100.0/24"
            }
        }
    ]
}
`
	interfaceNullsResponse = `
{
    "effective_mtu": 1500,
    "mac_address": null,
    "Children": ["eth0.1", "eth0.2"],
    "discovered": [],
    "params": "some params",
    "VLAN": null,
    "Name": "eth0",
    "Enabled": true,
    "Parents": ["bond0"],
    "ID": 40,
    "type": "physical",
    "resource_uri": "/MAAS/api/2.0/nodes/4y3ha6/interfaces/40/",
    "Tags": null,
    "Links": [
        {
            "ID": 69,
            "Mode": "auto",
            "Subnet": {
                "resource_uri": "/MAAS/api/2.0/Subnets/1/",
                "ID": 1,
                "rdns_mode": 2,
                "VLAN": {
                    "resource_uri": "/MAAS/api/2.0/VLANs/1/",
                    "ID": 1,
                    "secondary_rack": null,
                    "MTU": 1500,
                    "primary_rack": "4y3h7n",
                    "Name": "untagged",
                    "Fabric": "Fabric-0",
                    "dhcp_on": true,
                    "VID": 0
                },
                "dns_servers": [],
                "space": "space-0",
                "Name": "192.168.100.0/24",
                "gateway_ip": "192.168.100.1",
                "cidr": "192.168.100.0/24"
            }
        }
    ]
}
`
)
