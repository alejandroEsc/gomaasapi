// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"net/http"

	"encoding/json"

	"github.com/juju/errors"
	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/util"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestReadInterfacesBadSchema(t *testing.T) {
	var b MachineNetworkInterface
	err = json.Unmarshal([]byte("wat?"), &b)
	assert.Error(t, err)
}

func TestReadInterfacesNulls(t *testing.T) {
	var iface MachineNetworkInterface
	err = json.Unmarshal([]byte(interfaceNullsResponse), &iface)

	assert.Nil(t, err)

	assert.Equal(t, iface.MACAddress, "")
	assert.Equal(t, iface.Tags, []string{})
	assert.Nil(t, iface.VLAN)
}

func checkInterface(t *testing.T, iface *MachineNetworkInterface) {
	assert.Equal(t, iface.ID,  40)
	assert.Equal(t, iface.Name, "eth0")
	assert.Equal(t, iface.Type, "physical")
	assert.True(t, iface.Enabled)
	assert.Contains(t, iface.Tags, []string{"foo", "bar"})

	assert.Equal(t, iface.MACAddress,  "52:54:00:c9:6a:45")
	assert.Equal(t, iface.EffectiveMTU,  1500)

	assert.Contains(t, iface.Parents, []string{"bond0"})
	assert.Contains(t, iface.Children, []string{"eth0.1", "eth0.2"})

	vlan := iface.VLAN
	assert.NotNil(t, vlan)
	assert.Equal(t, vlan.Name,  "untagged")

	links := iface.Links
	assert.Len(t, links,  1)
	assert.Equal(t, links[0].ID, 69)
}

func  TestReadInterfaces(t *testing.T) {
	var interfaces []MachineNetworkInterface
	err = json.Unmarshal([]byte(interfacesResponse), &interfaces)
	assert.Nil(t, err)
	assert.Len(t, interfaces, 1)
	checkInterface(t, &interfaces[0])
}

func  TestReadInterface(t *testing.T) {
	var iface MachineNetworkInterface
	err = json.Unmarshal([]byte(interfacesResponse), &iface)
	assert.Nil(t, err)
	checkInterface(t, &iface)
}

func  getServerAndNewInterface(t *testing.T) (*client.SimpleTestServer, *MachineNetworkInterface) {
	server, controller := createTestServerController(t)
	server.AddGetResponse("/api/2.0/devices/", http.StatusOK, devicesResponse)
	devices, err := controller.Devices(DevicesArgs{})
	assert.Nil(t, err)
	device := devices[0]
	server.AddPostResponse(device.interfacesURI()+"?op=create_physical", http.StatusOK, interfaceResponse)
	iface, err := device.CreateInterface(minimalCreateInterfaceArgs())
	assert.Nil(t, err)
	return server, iface
}

func  TestInterfaceDelete(t *testing.T) {
	server, iface := getServerAndDevice(t)
	// Successful delete is 204 - StatusNoContent - We hope, would be consistent
	// with device deletions.
	server.AddDeleteResponse(iface.ResourceURI, http.StatusNoContent, "")
	err := iface.Delete()
	assert.Nil(t, err)
}

func  TestDelete404(t *testing.T) {
	_, iface := getServerAndDevice(t)
	// No Path, so 404
	err := iface.Delete()
	assert.True(t, util.IsNoMatchError(err))
}

func  TestDeleteForbidden(t *testing.T) {
	server, iface := getServerAndDevice(t)
	server.AddDeleteResponse(iface.ResourceURI, http.StatusForbidden, "")
	err := iface.Delete()
	assert.True(t, util.IsPermissionError(err))
}

func  TestDeleteUnknown(t *testing.T) {
	server, iface := getServerAndDevice(t)
	server.AddDeleteResponse(iface.ResourceURI, http.StatusConflict, "")
	err := iface.Delete()
	assert.True(t, util.IsUnexpectedError(err))
}

func  TestLinkSubnetArgs(t *testing.T) {
	for _, test := range []struct {
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
		err := test.args.Validate()
		if test.errText == "" {
			assert.Nil(t,err)
		} else {
			assert.True(t, errors.IsNotValid(err))
			assert.Equal(t,err.Error(), test.errText)
		}
	}
}

func  TestLinkSubnetValidates(t *testing.T) {
	_, iface := getServerAndNewInterface(t)
	err := iface.LinkSubnet(LinkSubnetArgs{})
	assert.True(t, errors.IsNotValid(err))
	assert.Equal(t, err.Error(),"missing Mode not valid")
}

func  TestLinkSubnetGood(t *testing.T) {
	server, iface := getServerAndNewInterface(t)
	// The changed information is there just for the test to show that the response
	// is parsed and the interface updated
	response := util.UpdateJSONMap(t, interfaceResponse, map[string]interface{}{
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
	assert.Nil(t, err)
	assert.Equal(t, iface.Name, "eth42")

	request := server.LastRequest()
	form := request.PostForm
	assert.Equal(t, form.Get("Mode"),  "STATIC")
	assert.Equal(t, form.Get("Subnet"), "42")
	assert.Equal(t, form.Get("ip_address"),  "10.10.10.10")
	assert.Equal(t, form.Get("default_gateway"),  "true")
}

func  TestLinkSubnetMissing(t *testing.T) {
	_, iface := getServerAndNewInterface(t)
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &subnet{ID: 42},
	}
	err := iface.LinkSubnet(args)
	assert.True(t, util.IsBadRequestError(err))
}

func  TestLinkSubnetForbidden(t *testing.T) {
	server, iface := getServerAndNewInterface(t)
	server.AddPostResponse(iface.ResourceURI+"?op=link_subnet", http.StatusForbidden, "bad user")
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &subnet{ID: 42},
	}
	err := iface.LinkSubnet(args)
	assert.True(t, util.IsPermissionError(err))
	assert.Equal(t, err.Error(),  "bad user")
}

func  TestLinkSubnetNoAddressesAvailable(t *testing.T) {
	server, iface := getServerAndNewInterface(t)
	server.AddPostResponse(iface.ResourceURI+"?op=link_subnet", http.StatusServiceUnavailable, "no addresses")
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &subnet{ID: 42},
	}
	err := iface.LinkSubnet(args)
	assert.True(t, util.IsCannotCompleteError(err))
	assert.Equal(t, err.Error(),  "no addresses")
}

func  TestLinkSubnetUnknown(t *testing.T) {
	server, iface := getServerAndNewInterface(t)
	server.AddPostResponse(iface.ResourceURI+"?op=link_subnet", http.StatusMethodNotAllowed, "wat?")
	args := LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: &subnet{ID: 42},
	}
	err := iface.LinkSubnet(args)
	assert.True(t, util.IsUnexpectedError(err))
	assert.Equal(t, err.Error(), "unexpected: ServerError: 405 Method Not Allowed (wat?)")
}

func  TestUnlinkSubnetValidates(t *testing.T) {
	_, iface := getServerAndNewInterface(t)
	err := iface.UnlinkSubnet(nil)
	assert.True(t, errors.IsNotValid(err))
	assert.Equal(t, err.Error(),  "missing Subnet not valid")
}

func  TestUnlinkSubnetNotLinked(t *testing.T) {
	_, iface := getServerAndNewInterface(t)
	err := iface.UnlinkSubnet(&subnet{ID: 42})
	assert.True(t, errors.IsNotValid(err))
	assert.Equal(t, err.Error(),  "unlinked Subnet not valid")
}

func  TestUnlinkSubnetGood(t *testing.T) {
	server, iface := getServerAndNewInterface(t)
	// The changed information is there just for the test to show that the response
	// is parsed and the interface updated
	response := util.UpdateJSONMap(t, interfaceResponse, map[string]interface{}{
		"Name": "eth42",
	})
	server.AddPostResponse(iface.ResourceURI+"?op=unlink_subnet", http.StatusOK, response)
	err := iface.UnlinkSubnet(&subnet{ID: 1})
	assert.Nil(t, err)
	assert.Equal(t, iface.Name,  "eth42")

	request := server.LastRequest()
	form := request.PostForm
	// The link ID that contains Subnet 1 has an internal ID of 69.
	assert.Equal(t, form.Get("ID"), "69")
}

func  TestUnlinkSubnetMissing(t *testing.T) {
	_, iface := getServerAndNewInterface(t)
	err := iface.UnlinkSubnet(&subnet{ID: 1})
	assert.True(t, util.IsBadRequestError(err))
}

func  TestUnlinkSubnetForbidden(t *testing.T) {
	server, iface := getServerAndNewInterface(t)
	server.AddPostResponse(iface.ResourceURI+"?op=unlink_subnet", http.StatusForbidden, "bad user")
	err := iface.UnlinkSubnet(&subnet{ID: 1})
	assert.True(t, util.IsPermissionError(err))
	assert.Equal(t, err.Error(), "bad user")
}

func  TestUnlinkSubnetUnknown(t *testing.T) {
	server, iface := getServerAndNewInterface(t)
	server.AddPostResponse(iface.ResourceURI+"?op=unlink_subnet", http.StatusMethodNotAllowed, "wat?")
	err := iface.UnlinkSubnet(&subnet{ID: 1})
	assert.True(t, util.IsUnexpectedError(err))
	assert.Equal(t, err.Error(),"unexpected: ServerError: 405 Method Not Allowed (wat?)")
}

func TestInterfaceUpdateNoChangeNoRequest(t *testing.T) {
	server, iface := getServerAndNewInterface(t)
	count := server.RequestCount()
	err := iface.Update(UpdateInterfaceArgs{})
	assert.Nil(t, err)
	assert.Equal(t, server.RequestCount(), count)
}

func TestInterfaceUpdateMissing(t *testing.T) {
	_, iface := getServerAndNewInterface(t)
	err := iface.Update(UpdateInterfaceArgs{Name: "eth2"})
	assert.True(t, util.IsNoMatchError(err))
}

func  TestInterfaceUpdateForbidden(t *testing.T) {
	server, iface := getServerAndNewInterface(t)
	server.AddPutResponse(iface.ResourceURI, http.StatusForbidden, "bad user")
	err := iface.Update(UpdateInterfaceArgs{Name: "eth2"})
	assert.True(t, util.IsPermissionError(err))
	assert.Equal(t, err.Error(), "bad user")
}

func TestInterfaceUpdateUnknown(t *testing.T) {
	server, iface := getServerAndNewInterface(t)
	server.AddPutResponse(iface.ResourceURI, http.StatusMethodNotAllowed, "wat?")
	err := iface.Update(UpdateInterfaceArgs{Name: "eth2"})
	assert.True(t, util.IsUnexpectedError(err))
	assert.Equal(t, err.Error(), "unexpected: ServerError: 405 Method Not Allowed (wat?)")
}

func TestUpdateGood(t *testing.T) {
	server, iface := getServerAndNewInterface(t)
	// The changed information is there just for the test to show that the response
	// is parsed and the interface updated
	response := util.UpdateJSONMap(t, interfaceResponse, map[string]interface{}{
		"Name": "eth42",
	})
	server.AddPutResponse(iface.ResourceURI, http.StatusOK, response)
	args := UpdateInterfaceArgs{
		Name:       "eth42",
		MACAddress: "c3-52-51-b4-50-cd",
		VLAN:       &vlan{ID: 13},
	}
	err := iface.Update(args)
	assert.Nil(t, err)
	assert.Equal(t, iface.Name, "eth42")

	request := server.LastRequest()
	form := request.PostForm
	assert.Equal(t, form.Get("Name"), "eth42")
	assert.Equal(t, form.Get("mac_address"), "c3-52-51-b4-50-cd")
	assert.Equal(t, form.Get("VLAN"), "13")
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
