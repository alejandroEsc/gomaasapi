// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/juju/errors"
	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/util"
	"github.com/juju/utils/set"
	"github.com/juju/version"
	"github.com/stretchr/testify/assert"
)

var server *client.SimpleTestServer

const versionResponse = `{"version": "unknown", "subversion": "", "Capabilities": ["networks-management", "static-ipaddresses", "ipv6-deployment-ubuntu", "devices-management", "storage-deployment-ubuntu", "network-deployment-ubuntu"]}`

type constraintMatchInfo map[string][]int

func TestSupportedVersions(t *testing.T) {
	for _, apiVersion := range supportedAPIVersions {
		_, _, err := version.ParseMajorMinor(apiVersion)
		assert.Nil(t, err)
	}
}

func TestNewController(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()
	controller := getController(t)

	expectedCapabilities := set.NewStrings(
		"networks-management",
		"static-ipaddresses",
		"ipv6-deployment-ubuntu",
		"devices-management",
		"storage-deployment-ubuntu",
		"network-deployment-ubuntu",
	)

	capabilities := controller.Capabilities
	assert.Len(t, capabilities.Difference(expectedCapabilities), 0)
	assert.Len(t, expectedCapabilities.Difference(capabilities), 0)
}

func TestNewControllerBadAPIKeyFormat(t *testing.T) {
	server = client.NewSimpleServer()
	server.Start()
	defer server.Close()
	_, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "invalid",
	})

	assert.True(t, errors.IsNotValid(err))
}

func TestNewControllerNoSupport(t *testing.T) {
	server := client.NewSimpleServer()
	server.Start()
	defer server.Close()
	_, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	assert.True(t, util.IsUnsupportedVersionError(err))
}

func TestNewControllerBadCreds(t *testing.T) {
	server := client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusUnauthorized, "naughty")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.Start()
	defer server.Close()
	_, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	assert.True(t, util.IsPermissionError(err))
}

func TestNewControllerUnexpected(t *testing.T) {
	server := client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusConflict, "naughty")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.Start()
	defer server.Close()

	_, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	assert.True(t, util.IsUnexpectedError(err))
}

func TestNewControllerKnownVersion(t *testing.T) {
	server := client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, "naughty")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.Start()
	defer server.Close()

	// Using a server URL including the version should work.
	officialController, err := NewController(ControllerArgs{
		BaseURL: server.URL + "/api/2.0/",
		APIKey:  "fake:as:key",
	})
	assert.Nil(t, err)

	assert.Equal(t, officialController.APIVersion, version.Number{
		Major: 2,
		Minor: 0,
	})
}

func TestNewControllerUnsupportedVersionSpecified(t *testing.T) {
	// Ensure the server would actually respond to the version if it
	// was asked.
	server.AddGetResponse("/api/3.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.AddGetResponse("/api/3.0/version/", http.StatusOK, versionResponse)
	// Using a server URL including a version that isn't in the known
	// set should be denied.
	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL + "/api/3.0/",
		APIKey:  "fake:as:key",
	})
	assert.Nil(t, controller)
	assert.True(t, util.IsUnsupportedVersionError(err))
}

func TestNewControllerNotHidingErrors(t *testing.T) {
	// We should only treat 404 and 410 as "this version isn't
	// supported". Other errors should be returned up the stack
	// unchanged, so we don't confuse transient network errors with
	// version mismatches. lp:1667095
	server := client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, "underwater woman")
	server.AddGetResponse("/api/2.0/version/", http.StatusInternalServerError, "kablooey")
	server.Start()
	defer server.Close()

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	assert.Nil(t, controller)
	assert.EqualError(t, err, `ServerError: 500 Internal Server Error (kablooey)`)
}

func TestNewController410(t *testing.T) {
	// We should only treat 404 and 410 as "this version isn't
	// supported". Other errors should be returned up the stack
	// unchanged, so we don't confuse transient network errors with
	// version mismatches. lp:1667095
	server := client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, "the answer to all your prayers")
	server.AddGetResponse("/api/2.0/version/", http.StatusGone, "cya")
	server.Start()
	defer server.Close()

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	assert.Nil(t, controller)
	assert.True(t, util.IsUnsupportedVersionError(err))
}

func TestNewController404(t *testing.T) {
	// We should only treat 404 and 410 as "this version isn't
	// supported". Other errors should be returned up the stack
	// unchanged, so we don't confuse transient network errors with
	// version mismatches. lp:1667095
	server := client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, "the answer to all your prayers")
	server.AddGetResponse("/api/2.0/version/", http.StatusNotFound, "huh?")
	server.Start()
	defer server.Close()

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	assert.Nil(t, controller)
	assert.True(t, util.IsUnsupportedVersionError(err))
}

func TestNewControllerWith194Bug(t *testing.T) {
	// 1.9.4 has a bug where if you ask for /api/2.0/version/ without
	// being logged in (rather than OAuth connection) it redirects you
	// to the login page. This is fixed in 1.9.5, but we should work
	// around it anyway. https://bugs.launchpad.net/maas/+bug/1583715
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, "the answer to all your prayers")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, "<html><head>")

	server.Start()
	defer server.Close()

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	assert.Nil(t, controller)
	assert.True(t, util.IsDeserializationError(err))
}

func TestControllerBootResources(t *testing.T) {
	//t.Skipf("unknown failure %s", `Expected nil, but got: &url.Error{Op:"Get", URL:"http://127.0.0.1:53393/api/2.0/version/", Err:(*net.OpError)(0xc4202dc2d0)}`)
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/boot-resources/", http.StatusOK, bootResourcesResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	//Controller, err := NewController(ControllerArgs{
	//	BaseURL: server.URL,
	//	APIKey:  "fake:as:key",
	//})
	//assert.Nil(t, Controller)

	controller := getController(t)
	resources, err := controller.BootResources()
	assert.Nil(t, err)
	assert.Len(t, resources, 5)
}

func TestControllerNodes(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/nodes/", http.StatusOK, devicesResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	devices, err := controller.Nodes(NodesArgs{})
	assert.Nil(t, err)
	assert.Len(t, devices, 1)
}

func TestControllerNodesArgs(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/nodes/", http.StatusOK, devicesResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	// This will fail with a 404 due to the test server not having something  at
	// that address, but we don't care, all we want to do is capture the request
	// and make sure that all the Values were set.
	controller.Nodes(NodesArgs{
		Hostname:     []string{"untasted-markita"},
		MACAddresses: []string{"something"},
		SystemIDs:    []string{"something-else"},
		Domain:       "magic",
		Zone:         "foo",
		AgentName:    "agent 42",
	})
	request := server.LastRequest()
	// There should be one entry in the form Values for each of the args.
	assert.Len(t, request.URL.Query(), 6)
}

func TestControllerCreateNode(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/nodes/", http.StatusOK, devicesResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.AddPostResponse("/api/2.0/nodes/?op=", http.StatusOK, deviceResponse)
	server.Start()
	defer server.Close()

	controller := getController(t)
	device, err := controller.CreateNode(CreateNodeArgs{
		MACAddresses: []string{"a-mac-address"},
	})
	assert.Nil(t, err)
	assert.Equal(t, device.SystemID, "4y3haf")
}

func TestControllerCreateNodeMissingAddress(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/nodes/", http.StatusOK, devicesResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	_, err := controller.CreateNode(CreateNodeArgs{})
	assert.True(t, util.IsBadRequestError(err))
	assert.Equal(t, err.Error(), "at least one MAC address must be specified")
}

func TestControllerCreateNodeBadRequest(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddPostResponse("/api/2.0/nodes/?op=", http.StatusBadRequest, "some error")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	_, err := controller.CreateNode(CreateNodeArgs{
		MACAddresses: []string{"a-mac-address"},
	})
	assert.True(t, util.IsBadRequestError(err))
	assert.Equal(t, err.Error(), "some error")
}

func TestControllerCreateNodeArgs(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddPostResponse("/api/2.0/nodes/?op=", http.StatusOK, deviceResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	// Create an arg structure that sets all the Values.
	args := CreateNodeArgs{
		Hostname:     "foobar",
		MACAddresses: []string{"an-address"},
		Domain:       "a domain",
		Parent:       "Parent",
	}
	_, err := controller.CreateNode(args)
	assert.Nil(t, err)

	request := server.LastRequest()
	// There should be one entry in the form Values for each of the args.
	assert.Len(t, request.PostForm, 4)
}

func TestControllerFabrics(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/fabrics/", http.StatusOK, fabricResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	fabrics, err := controller.Fabrics()
	assert.Nil(t, err)
	assert.Len(t, fabrics, 2)
}

func TestControllerSpaces(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/spaces/", http.StatusOK, spacesResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	spaces, err := controller.Spaces()
	assert.Nil(t, err)
	assert.Len(t, spaces, 1)
}

func TestControllerStaticRoutes(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/static-routes/", http.StatusOK, staticRoutesResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	staticRoutes, err := controller.StaticRoutes()
	assert.Nil(t, err)
	assert.Len(t, staticRoutes, 1)
}

func TestControllerZones(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/zones/", http.StatusOK, zoneResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	zones, err := controller.Zones()
	assert.Nil(t, err)
	assert.Len(t, zones, 2)
}

func TestControllerMachines(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/machines/", http.StatusOK, machinesResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	machines, err := controller.Machines(MachinesArgs{})
	assert.Nil(t, err)
	assert.Len(t, machines, 3)
}

func TestControllerMachinesFilter(t *testing.T) {
	hostName := "untasted-markita"
	response := "[" + machineResponse + "]"
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/machines/?Hostname="+hostName, http.StatusOK, response)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	machines, err := controller.Machines(MachinesArgs{
		Hostnames: []string{hostName},
	})
	assert.Nil(t, err)
	assert.Len(t, machines, 1)
	assert.Equal(t, machines[0].Hostname, hostName)
}

func TestControllerMachinesFilterWithOwnerData(t *testing.T) {
	server = client.NewSimpleServer()
	//server.AddGetResponse("/api/2.0/machines/", http.StatusOK, machinesResponse)
	server.AddGetResponse("/api/2.0/machines/?Hostname=untasted-markita", http.StatusOK, "["+machineResponse+"]")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	machines, err := controller.Machines(MachinesArgs{
		Hostnames: []string{"untasted-markita"},
		OwnerData: map[string]string{
			"fez": "jim crawford",
		},
	})
	assert.Nil(t, err)
	assert.Len(t, machines, 0)
}

func TestControllerMachinesFilterWithOwnerData_MultipleMatches(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/machines/", http.StatusOK, machinesResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	machines, err := controller.Machines(MachinesArgs{
		OwnerData: map[string]string{
			"braid": "jonathan blow",
		},
	})
	assert.Nil(t, err)
	assert.Len(t, machines, 2)
	assert.Equal(t, machines[0].Hostname, "lowlier-glady")
	assert.Equal(t, machines[1].Hostname, "icier-nina")
}

func TestControllerMachinesFilterWithOwnerData_RequiresAllMatch(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/machines/", http.StatusOK, machinesResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	machines, err := controller.Machines(MachinesArgs{
		OwnerData: map[string]string{
			"braid":          "jonathan blow",
			"frog-fractions": "jim crawford",
		},
	})
	assert.Nil(t, err)
	assert.Len(t, machines, 1)
	assert.Equal(t, machines[0].Hostname, "lowlier-glady")
}

func TestControllerMachinesArgs(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/machines/?Hostname=untasted-markita", http.StatusOK, "["+machineResponse+"]")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	// This will fail with a 404 due to the test server not having something  at
	// that address, but we don't care, all we want to do is capture the request
	// and make sure that all the Values were set.
	controller.Machines(MachinesArgs{
		Hostnames:    []string{"untasted-markita"},
		MACAddresses: []string{"something"},
		SystemIDs:    []string{"something-else"},
		Domain:       "magic",
		Zone:         "foo",
		AgentName:    "agent 42",
	})
	request := server.LastRequest()
	// There should be one entry in the form Values for each of the args.
	assert.Len(t, request.URL.Query(), 6)
}

func TestControllerAllocateMachine(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	addAllocateResponse(t, http.StatusOK, nil, nil)
	controller := getController(t)
	machine, _, err := controller.AllocateMachine(AllocateMachineArgs{})
	assert.Nil(t, err)
	assert.Equal(t, machine.SystemID, "4y3ha3")
}

func TestControllerAllocateMachineInterfacesMatch(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	addAllocateResponse(t, http.StatusOK, constraintMatchInfo{
		"database": []int{35, 99},
	}, nil)
	controller := getController(t)
	_, match, err := controller.AllocateMachine(AllocateMachineArgs{
		// This isn't actually used, but here to show how it should be used.
		Interfaces: []InterfaceSpec{{
			Label: "database",
			Space: "Space-0",
		}},
	})
	assert.Nil(t, err)
	assert.Len(t, match.Interfaces, 1)
	ifaces := match.Interfaces["database"]
	assert.Len(t, ifaces, 2)
	assert.Equal(t, ifaces[0].ID, 35)
	assert.Equal(t, ifaces[1].ID, 99)
}

func TestControllerAllocateMachineInterfacesMatchMissing(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()
	// This should never happen, but if it does it is a clear indication of a
	// bug somewhere.
	addAllocateResponse(t, http.StatusOK, constraintMatchInfo{
		"database": []int{40},
	}, nil)

	controller := getController(t)
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{
		Interfaces: []InterfaceSpec{{
			Label: "database",
			Space: "Space-0",
		}},
	})
	assert.True(t, util.IsDeserializationError(err))
}

func TestControllerAllocateMachineStorageMatches(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	addAllocateResponse(t, http.StatusOK, nil, constraintMatchInfo{
		"root": []int{34, 98},
	})
	controller := getController(t)
	_, match, err := controller.AllocateMachine(AllocateMachineArgs{
		Storage: []StorageSpec{{
			Label: "root",
			Size:  50,
			Tags:  []string{"hefty", "tangy"},
		}},
	})
	assert.Nil(t, err)
	assert.Len(t, match.Storage, 1)
	storages := match.Storage["root"]
	assert.Len(t, storages, 2)
	assert.Equal(t, storages[0].ID, 34)
	assert.Equal(t, storages[1].ID, 98)
}

func TestControllerAllocateMachineStorageLogicalMatches(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusOK, machineResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	machine, matches, err := controller.AllocateMachine(AllocateMachineArgs{
		Storage: []StorageSpec{{
			Tags: []string{"raid0"},
		}},
	})
	assert.Nil(t, err)
	var virtualDeviceID = 23

	//matches storage must contain the "raid0" virtual block node
	assert.EqualValues(t, matches.Storage["0"][0], *machine.BlockDevice(virtualDeviceID))
}

func TestControllerAllocateMachineStorageMatchMissing(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	// This should never happen, but if it does it is a clear indication of a
	// bug somewhere.
	addAllocateResponse(t, http.StatusOK, nil, constraintMatchInfo{
		"root": []int{50},
	})
	controller := getController(t)
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{
		Storage: []StorageSpec{{
			Label: "root",
			Size:  50,
			Tags:  []string{"hefty", "tangy"},
		}},
	})
	assert.True(t, util.IsDeserializationError(err))
}

func TestControllerAllocateMachineArgsForm(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	addAllocateResponse(t, http.StatusOK, nil, nil)
	controller := getController(t)
	// Create an arg structure that sets all the Values.
	args := AllocateMachineArgs{
		Hostname:     "foobar",
		SystemId:     "some_id",
		Architecture: "amd64",
		MinCPUCount:  42,
		MinMemory:    20000,
		Tags:         []string{"good"},
		NotTags:      []string{"bad"},
		Storage:      []StorageSpec{{Label: "root", Size: 200}},
		Interfaces:   []InterfaceSpec{{Label: "default", Space: "magic"}},
		NotSpace:     []string{"special"},
		Zone:         "magic",
		NotInZone:    []string{"not-magic"},
		AgentName:    "agent 42",
		Comment:      "testing",
		DryRun:       true,
	}
	_, _, err := controller.AllocateMachine(args)
	assert.Nil(t, err)

	request := server.LastRequest()
	// There should be one entry in the form Values for each of the args.
	form := request.PostForm
	assert.Len(t, form, 15)
	// Positive Space check.
	assert.Equal(t, form.Get("interfaces"), "default:space=magic")
	// Negative Space check.
	assert.Equal(t, form.Get("not_subnets"), "space:special")
}

func TestControllerAllocateMachineNoMatch(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusConflict, "boo")
	server.Start()
	defer server.Close()

	controller := getController(t)
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{})
	assert.True(t, util.IsNoMatchError(err))
}

func TestControllerAllocateMachineUnexpected(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusBadRequest, "boo")
	controller := getController(t)
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{})
	assert.True(t, util.IsUnexpectedError(err))
}

func TestControllerReleaseMachines(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusOK, "[]")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
		Comment:   "all good",
	})
	assert.Nil(t, err)

	request := server.LastRequest()
	// There should be one entry in the form Values for each of the args.
	assert.Equal(t, request.PostForm["machines"], []string{"this", "that"})
	assert.EqualValues(t, request.PostForm.Get("comment"), "all good")
}

func TestControllerReleaseMachinesBadRequest(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusBadRequest, "unknown machines")
	controller := getController(t)
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	assert.True(t, util.IsBadRequestError(err))
	assert.Equal(t, err.Error(), "unknown machines")
}

func TestControllerReleaseMachinesForbidden(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusForbidden, "bzzt denied")
	controller := getController(t)
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	assert.True(t, util.IsPermissionError(err))
	assert.Equal(t, err.Error(), "bzzt denied")
}

func TestControllerReleaseMachinesConflict(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusConflict, "MachineInterface busy")
	controller := getController(t)
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	assert.True(t, util.IsCannotCompleteError(err))
	assert.Equal(t, err.Error(), "MachineInterface busy")
}

func TestControllerReleaseMachinesUnexpected(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusBadGateway, "wat")
	controller := getController(t)
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	assert.True(t, util.IsUnexpectedError(err))
	assert.Equal(t, err.Error(), "unexpected: ServerError: 502 Bad Gateway (wat)")
}

func TestControllerFiles(t *testing.T) {
	t.Skip("skip until testing on real server.")
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/files/", http.StatusOK, filesResponse)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	files, err := controller.getFiles("")
	assert.Nil(t, err)
	assert.Len(t, files, 2)

	file := files[0]
	assert.Equal(t, file.Filename, "test")

	assert.Equal(t, file.AnonymousURI.Scheme, "http")
	assert.Equal(t, file.AnonymousURI.RequestURI(), "/MAAS/api/2.0/files/?op=get_by_key&key=3afba564-fb7d-11e5-932f-52540051bf22")
}

func TestControllerGetFile(t *testing.T) {
	t.Skip("skip until testing on real server.")
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	server.AddGetResponse("/api/2.0/files/testing/", http.StatusOK, fileResponse)
	controller := getController(t)
	file, err := controller.GetFile("testing")
	assert.Nil(t, err)

	assert.Equal(t, file.Filename, "testing")

	assert.Nil(t, err)
	assert.Equal(t, file.AnonymousURI.Scheme, "http")
	assert.Equal(t, file.AnonymousURI.RequestURI(), "/MAAS/api/2.0/files/?op=get_by_key&key=88e64b76-fb82-11e5-932f-52540051bf22")
}

func TestControllerGetFileMissing(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	_, err := controller.GetFile("missing")
	assert.True(t, util.IsNoMatchError(err))
}

func TestControllerAddFileArgsValidate(t *testing.T) {
	reader := bytes.NewBufferString("test")
	for _, test := range []struct {
		args    AddFileArgs
		errText string
	}{{
		errText: "missing Filename not valid",
	}, {
		args:    AddFileArgs{Filename: "/foo"},
		errText: `paths in Filename "/foo" not valid`,
	}, {
		args:    AddFileArgs{Filename: "a/foo"},
		errText: `paths in Filename "a/foo" not valid`,
	}, {
		args:    AddFileArgs{Filename: "foo.txt"},
		errText: `missing Content or Reader not valid`,
	}, {
		args: AddFileArgs{
			Filename: "foo.txt",
			Reader:   reader,
		},
		errText: `missing Length not valid`,
	}, {
		args: AddFileArgs{
			Filename: "foo.txt",
			Reader:   reader,
			Length:   4,
		},
	}, {
		args: AddFileArgs{
			Filename: "foo.txt",
			Content:  []byte("foo"),
			Reader:   reader,
		},
		errText: `specifying Content and Reader not valid`,
	}, {
		args: AddFileArgs{
			Filename: "foo.txt",
			Content:  []byte("foo"),
			Length:   20,
		},
		errText: `specifying Length and Content not valid`,
	}, {
		args: AddFileArgs{
			Filename: "foo.txt",
			Content:  []byte("foo"),
		},
	}} {
		err := test.args.Validate()
		if test.errText == "" {
			assert.Nil(t, err)
		} else {
			assert.True(t, errors.IsNotValid(err))
			assert.EqualError(t, err, test.errText)
		}
	}
}

func TestControllerAddFileValidates(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	controller := getController(t)
	err := controller.AddFile(AddFileArgs{})
	assert.True(t, errors.IsNotValid(err))
}

func TestControllerAddFileContent(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	server.AddPostResponse("/api/2.0/files/?op=", http.StatusOK, "")
	controller := getController(t)
	err := controller.AddFile(AddFileArgs{
		Filename: "foo.txt",
		Content:  []byte("foo"),
	})
	assert.Nil(t, err)

	request := server.LastRequest()
	assertFile(t, request, "foo.txt", "foo")
}

func TestControllerAddFileReader(t *testing.T) {
	server = client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.Start()
	defer server.Close()

	reader := bytes.NewBufferString("test\n extra over length ignored")
	server.AddPostResponse("/api/2.0/files/?op=", http.StatusOK, "")
	controller := getController(t)
	err := controller.AddFile(AddFileArgs{
		Filename: "foo.txt",
		Reader:   reader,
		Length:   5,
	})
	assert.Nil(t, err)

	request := server.LastRequest()
	assertFile(t, request, "foo.txt", "test\n")
}

func assertFile(t *testing.T, request *http.Request, filename, content string) {
	form := request.Form
	assert.Equal(t, form.Get("Filename"), filename)

	fileHeader := request.MultipartForm.File["File"][0]
	f, err := fileHeader.Open()
	assert.Nil(t, err)
	bytes, err := ioutil.ReadAll(f)
	assert.Nil(t, err)
	assert.Equal(t, string(bytes), content)
}

// createTestServerController creates a ControllerInterface backed on to a test server
// that has sufficient knowledge of versions and users to be able to create a
// valid ControllerInterface.
func createTestServerController(t *testing.T) (*client.SimpleTestServer, *Controller) {
	server := client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.Start()

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	assert.Nil(t, err)
	return server, controller
}

func getController(t *testing.T) *Controller {
	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	assert.Nil(t, err)
	return controller
}

func addAllocateResponse(t *testing.T, status int, interfaceMatches, storageMatches constraintMatchInfo) {
	constraints := make(map[string]interface{})
	if interfaceMatches != nil {
		constraints["interfaces"] = interfaceMatches
	}
	if storageMatches != nil {
		constraints["storage"] = storageMatches
	}
	allocateJSON := util.UpdateJSONMap(t, machineResponse, map[string]interface{}{
		"constraints_by_type": constraints,
	})
	server.AddPostResponse("/api/2.0/machines/?op=allocate", status, allocateJSON)
}
