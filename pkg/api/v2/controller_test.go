// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/set"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/util"
	"github.com/stretchr/testify/assert"
)

const (
	// Capability constants.
	NetworksManagement      = "networks-management"
	StaticIPAddresses       = "static-ipaddresses"
	IPv6DeploymentUbuntu    = "ipv6-deployment-ubuntu"
	DevicesManagement       = "devices-management"
	StorageDeploymentUbuntu = "storage-deployment-ubuntu"
	NetworkDeploymentUbuntu = "network-deployment-ubuntu"
)

var (
	server *client.SimpleTestServer
	versionResponse = `{"version": "unknown", "subversion": "", "Capabilities": ["networks-management", "static-ipaddresses", "ipv6-deployment-ubuntu", "devices-management", "storage-deployment-ubuntu", "network-deployment-ubuntu"]}`

)

func TestSupportedVersions(t *testing.T) {
	for _, apiVersion := range supportedAPIVersions {
		_, _, err := version.ParseMajorMinor(apiVersion)
		assert.Nil(t, err)
	}
}

func TestNewController(t *testing.T) {
	controller := getController()

	expectedCapabilities := set.NewStrings(
		NetworksManagement,
		StaticIPAddresses,
		IPv6DeploymentUbuntu,
		DevicesManagement,
		StorageDeploymentUbuntu,
		NetworkDeploymentUbuntu,
	)

	capabilities := controller.Capabilities
	assert.Len(t, capabilities.Difference(expectedCapabilities), 0)
	assert.Len(t, expectedCapabilities.Difference(capabilities), 0)
}

func TestNewControllerBadAPIKeyFormat(t *testing.T) {
	server := client.NewSimpleServer()
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
	c.Assert(err, jc.Satisfies, util.IsUnsupportedVersionError)
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
	// Using a server URL including the version should work.
	officialController, err := NewController(ControllerArgs{
		BaseURL: server.URL + "/api/2.0/",
		APIKey:  "fake:as:key",
	})
	assert.Nil(t, err)

	c.Assert(officialController.APIVersion, gc.Equals, version.Number{
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
	assert.EqualError(t, err, `ServerError: 500 Internal Server Error \(kablooey\)`)
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
	server := client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, "the answer to all your prayers")
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, "<html><head>")
	server.Start()
	defer server.Close()

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	assert.Nil(t, controller)
	assert.True(t, util.IsUnsupportedVersionError(err))
}

func TestBootResources(t *testing.T) {
	controller := getController()
	resources, err := controller.BootResources()
	assert.Nil(t, err)
	assert.Len(t, resources, 5)
}

func TestDevices(t *testing.T) {
	controller := getController()
	devices, err := controller.Devices(DevicesArgs{})
	assert.Nil(t, err)
	assert.Len(t, devices, 5)
}

func TestDevicesArgs(t *testing.T) {
	controller := getController()
	// This will fail with a 404 due to the test server not having something  at
	// that address, but we don't care, all we want to do is capture the request
	// and make sure that all the Values were set.
	controller.Devices(DevicesArgs{
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

func TestCreateDevice(t *testing.T) {
	server.AddPostResponse("/api/2.0/devices/?op=", http.StatusOK, deviceResponse)
	controller := getController()
	device, err := controller.CreateDevice(CreateDeviceArgs{
		MACAddresses: []string{"a-mac-address"},
	})
	assert.Nil(t, err)
	assert.Equal(t, device.SystemID,"4y3haf")
}

func TestCreateDeviceMissingAddress(t *testing.T) {
	controller := getController()
	_, err := controller.CreateDevice(CreateDeviceArgs{})
	c.Assert(err, jc.Satisfies, util.IsBadRequestError)
	c.Assert(err.Error(), gc.Equals, "at least one MAC address must be specified")
}

func TestCreateDeviceBadRequest(t *testing.T) {
	server.AddPostResponse("/api/2.0/devices/?op=", http.StatusBadRequest, "some error")
	controller := getController()
	_, err := controller.CreateDevice(CreateDeviceArgs{
		MACAddresses: []string{"a-mac-address"},
	})
	c.Assert(err, jc.Satisfies, util.IsBadRequestError)
	c.Assert(err.Error(), gc.Equals, "some error")
}

func TestCreateDeviceArgs(t *testing.T) {
	server.AddPostResponse("/api/2.0/devices/?op=", http.StatusOK, deviceResponse)
	controller := getController()
	// Create an arg structure that sets all the Values.
	args := CreateDeviceArgs{
		Hostname:     "foobar",
		MACAddresses: []string{"an-address"},
		Domain:       "a domain",
		Parent:       "Parent",
	}
	_, err := controller.CreateDevice(args)
	assert.Nil(t, err)

	request := server.LastRequest()
	// There should be one entry in the form Values for each of the args.
	c.Assert(request.PostForm, gc.HasLen, 4)
}

func TestFabrics(t *testing.T) {
	controller := getController()
	fabrics, err := controller.Fabrics()
	assert.Nil(t, err)
	c.Assert(fabrics, gc.HasLen, 2)
}

func TestSpaces(t *testing.T) {
	controller := getController()
	spaces, err := controller.Spaces()
	assert.Nil(t, err)
	c.Assert(spaces, gc.HasLen, 1)
}

func TestStaticRoutes(t *testing.T) {
	controller := getController()
	staticRoutes, err := controller.StaticRoutes()
	assert.Nil(t, err)
	c.Assert(staticRoutes, gc.HasLen, 1)
}

func TestZones(t *testing.T) {
	controller := getController()
	zones, err := controller.Zones()
	assert.Nil(t, err)
	c.Assert(zones, gc.HasLen, 2)
}

func TestMachines(t *testing.T) {
	controller := getController()
	machines, err := controller.Machines(MachinesArgs{})
	assert.Nil(t, err)
	c.Assert(machines, gc.HasLen, 3)
}

func TestMachinesFilter(t *testing.T) {
	controller := getController()
	machines, err := controller.Machines(MachinesArgs{
		Hostnames: []string{"untasted-markita"},
	})
	assert.Nil(t, err)
	c.Assert(machines, gc.HasLen, 1)
	c.Assert(machines[0].Hostname, gc.Equals, "untasted-markita")
}

func TestMachinesFilterWithOwnerData(t *testing.T) {
	controller := getController()
	machines, err := controller.Machines(MachinesArgs{
		Hostnames: []string{"untasted-markita"},
		OwnerData: map[string]string{
			"fez": "jim crawford",
		},
	})
	assert.Nil(t, err)
	c.Assert(machines, gc.HasLen, 0)
}

func TestMachinesFilterWithOwnerData_MultipleMatches(t *testing.T) {
	controller := getController()
	machines, err := controller.Machines(MachinesArgs{
		OwnerData: map[string]string{
			"braid": "jonathan blow",
		},
	})
	assert.Nil(t, err)
	c.Assert(machines, gc.HasLen, 2)
	c.Assert(machines[0].Hostname, gc.Equals, "lowlier-glady")
	c.Assert(machines[1].Hostname, gc.Equals, "icier-nina")
}

func TestMachinesFilterWithOwnerData_RequiresAllMatch(t *testing.T) {
	controller := getController()
	machines, err := controller.Machines(MachinesArgs{
		OwnerData: map[string]string{
			"braid":          "jonathan blow",
			"frog-fractions": "jim crawford",
		},
	})
	assert.Nil(t, err)
	c.Assert(machines, gc.HasLen, 1)
	c.Assert(machines[0].Hostname, gc.Equals, "lowlier-glady")
}

func TestMachinesArgs(t *testing.T) {
	controller := getController()
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
	c.Assert(request.URL.Query(), gc.HasLen, 6)
}

func TestStorageSpec(t *testing.T) {
	for i, test := range []struct {
		spec StorageSpec
		err  string
		repr string
	}{{
		spec: StorageSpec{},
		err:  "Size value 0 not valid",
	}, {
		spec: StorageSpec{Size: -10},
		err:  "Size value -10 not valid",
	}, {
		spec: StorageSpec{Size: 200},
		repr: "200",
	}, {
		spec: StorageSpec{Label: "foo", Size: 200},
		repr: "foo:200",
	}, {
		spec: StorageSpec{Size: 200, Tags: []string{"foo", ""}},
		err:  "empty tag not valid",
	}, {
		spec: StorageSpec{Size: 200, Tags: []string{"foo"}},
		repr: "200(foo)",
	}, {
		spec: StorageSpec{Label: "omg", Size: 200, Tags: []string{"foo", "bar"}},
		repr: "omg:200(foo,bar)",
	}} {
		c.Logf("test %d", i)
		err := test.spec.Validate()
		if test.err == "" {
			assert.Nil(t, err)
			c.Assert(test.spec.String(), gc.Equals, test.repr)
		} else {
			c.Assert(err, jc.Satisfies, errors.IsNotValid)
			c.Assert(err.Error(), gc.Equals, test.err)
		}
	}
}

func TestInterfaceSpec(t *testing.T) {
	for i, test := range []struct {
		spec InterfaceSpec
		err  string
		repr string
	}{{
		spec: InterfaceSpec{},
		err:  "missing Label not valid",
	}, {
		spec: InterfaceSpec{Label: "foo"},
		err:  "empty Space constraint not valid",
	}, {
		spec: InterfaceSpec{Label: "foo", Space: "magic"},
		repr: "foo:space=magic",
	}} {
		c.Logf("test %d", i)
		err := test.spec.Validate()
		if test.err == "" {
			c.Check(err, jc.ErrorIsNil)
			c.Check(test.spec.String(), gc.Equals, test.repr)
		} else {
			c.Check(err, jc.Satisfies, errors.IsNotValid)
			c.Check(err.Error(), gc.Equals, test.err)
		}
	}
}

func TestAllocateMachineArgs(t *testing.T) {
	for i, test := range []struct {
		args       AllocateMachineArgs
		err        string
		storage    string
		interfaces string
		notSubnets []string
	}{{
		args: AllocateMachineArgs{},
	}, {
		args: AllocateMachineArgs{
			Storage: []StorageSpec{{}},
		},
		err: "Storage: Size value 0 not valid",
	}, {
		args: AllocateMachineArgs{
			Storage: []StorageSpec{{Size: 200}, {Size: 400, Tags: []string{"ssd"}}},
		},
		storage: "200,400(ssd)",
	}, {
		args: AllocateMachineArgs{
			Storage: []StorageSpec{
				{Label: "foo", Size: 200},
				{Label: "foo", Size: 400, Tags: []string{"ssd"}},
			},
		},
		err: `reusing storage Label "foo" not valid`,
	}, {
		args: AllocateMachineArgs{
			Interfaces: []InterfaceSpec{{}},
		},
		err: "Interfaces: missing Label not valid",
	}, {
		args: AllocateMachineArgs{
			Interfaces: []InterfaceSpec{
				{Label: "foo", Space: "magic"},
				{Label: "bar", Space: "other"},
			},
		},
		interfaces: "foo:space=magic;bar:space=other",
	}, {
		args: AllocateMachineArgs{
			Interfaces: []InterfaceSpec{
				{Label: "foo", Space: "magic"},
				{Label: "foo", Space: "other"},
			},
		},
		err: `reusing interface Label "foo" not valid`,
	}, {
		args: AllocateMachineArgs{
			NotSpace: []string{""},
		},
		err: "empty NotSpace constraint not valid",
	}, {
		args: AllocateMachineArgs{
			NotSpace: []string{"foo"},
		},
		notSubnets: []string{"space:foo"},
	}, {
		args: AllocateMachineArgs{
			NotSpace: []string{"foo", "bar"},
		},
		notSubnets: []string{"space:foo", "space:bar"},
	}} {
		c.Logf("test %d", i)
		err := test.args.Validate()
		if test.err == "" {
			c.Check(err, jc.ErrorIsNil)
			c.Check(test.args.storage(), gc.Equals, test.storage)
			c.Check(test.args.interfaces(), gc.Equals, test.interfaces)
			c.Check(test.args.notSubnets(), jc.DeepEquals, test.notSubnets)
		} else {
			c.Check(err, jc.Satisfies, errors.IsNotValid)
			c.Check(err.Error(), gc.Equals, test.err)
		}
	}
}

type constraintMatchInfo map[string][]int

func addAllocateResponse(t *testing.T, status int, interfaceMatches, storageMatches constraintMatchInfo) {
	constraints := make(map[string]interface{})
	if interfaceMatches != nil {
		constraints["interfaces"] = interfaceMatches
	}
	if storageMatches != nil {
		constraints["storage"] = storageMatches
	}
	allocateJSON := util.UpdateJSONMap(c, machineResponse, map[string]interface{}{
		"constraints_by_type": constraints,
	})
	server.AddPostResponse("/api/2.0/machines/?op=allocate", status, allocateJSON)
}

func TestAllocateMachine(t *testing.T) {
	s.addAllocateResponse(c, http.StatusOK, nil, nil)
	controller := getController()
	machine, _, err := controller.AllocateMachine(AllocateMachineArgs{})
	assert.Nil(t, err)
	c.Assert(machine.SystemID, gc.Equals, "4y3ha3")
}

func TestAllocateMachineInterfacesMatch(t *testing.T) {
	s.addAllocateResponse(c, http.StatusOK, constraintMatchInfo{
		"database": []int{35, 99},
	}, nil)
	controller := getController()
	_, match, err := controller.AllocateMachine(AllocateMachineArgs{
		// This isn't actually used, but here to show how it should be used.
		Interfaces: []InterfaceSpec{{
			Label: "database",
			Space: "space-0",
		}},
	})
	assert.Nil(t, err)
	c.Assert(match.Interfaces, gc.HasLen, 1)
	ifaces := match.Interfaces["database"]
	c.Assert(ifaces, gc.HasLen, 2)
	c.Assert(ifaces[0].ID, gc.Equals, 35)
	c.Assert(ifaces[1].ID, gc.Equals, 99)
}

func TestAllocateMachineInterfacesMatchMissing(t *testing.T) {
	// This should never happen, but if it does it is a clear indication of a
	// bug somewhere.
	s.addAllocateResponse(c, http.StatusOK, constraintMatchInfo{
		"database": []int{40},
	}, nil)
	controller := getController()
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{
		Interfaces: []InterfaceSpec{{
			Label: "database",
			Space: "space-0",
		}},
	})
	c.Assert(err, jc.Satisfies, util.IsDeserializationError)
}

func TestAllocateMachineStorageMatches(t *testing.T) {
	s.addAllocateResponse(c, http.StatusOK, nil, constraintMatchInfo{
		"root": []int{34, 98},
	})
	controller := getController()
	_, match, err := controller.AllocateMachine(AllocateMachineArgs{
		Storage: []StorageSpec{{
			Label: "root",
			Size:  50,
			Tags:  []string{"hefty", "tangy"},
		}},
	})
	assert.Nil(t, err)
	c.Assert(match.Storage, gc.HasLen, 1)
	storages := match.Storage["root"]
	c.Assert(storages, gc.HasLen, 2)
	c.Assert(storages[0].ID, gc.Equals, 34)
	c.Assert(storages[1].ID, gc.Equals, 98)
}

func TestAllocateMachineStorageLogicalMatches(t *testing.T) {
	server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusOK, machineResponse)
	controller := getController()
	machine, matches, err := controller.AllocateMachine(AllocateMachineArgs{
		Storage: []StorageSpec{{
			Tags: []string{"raid0"},
		}},
	})
	assert.Nil(t, err)
	var virtualDeviceID = 23

	//matches storage must contain the "raid0" virtual block device
	c.Assert(matches.Storage["0"][0], gc.Equals, machine.BlockDevice(virtualDeviceID))
}

func TestAllocateMachineStorageMatchMissing(t *testing.T) {
	// This should never happen, but if it does it is a clear indication of a
	// bug somewhere.
	s.addAllocateResponse(c, http.StatusOK, nil, constraintMatchInfo{
		"root": []int{50},
	})
	controller := getController()
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{
		Storage: []StorageSpec{{
			Label: "root",
			Size:  50,
			Tags:  []string{"hefty", "tangy"},
		}},
	})
	c.Assert(err, jc.Satisfies, util.IsDeserializationError)
}

func TestAllocateMachineArgsForm(t *testing.T) {
	s.addAllocateResponse(c, http.StatusOK, nil, nil)
	controller := getController()
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
	c.Assert(form, gc.HasLen, 15)
	// Positive space check.
	c.Assert(form.Get("interfaces"), gc.Equals, "default:space=magic")
	// Negative space check.
	c.Assert(form.Get("not_subnets"), gc.Equals, "space:special")
}

func TestAllocateMachineNoMatch(t *testing.T) {
	server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusConflict, "boo")
	controller := getController()
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{})
	c.Assert(err, jc.Satisfies, util.IsNoMatchError)
}

func TestAllocateMachineUnexpected(t *testing.T) {
	server.AddPostResponse("/api/2.0/machines/?op=allocate", http.StatusBadRequest, "boo")
	controller := getController()
	_, _, err := controller.AllocateMachine(AllocateMachineArgs{})
	c.Assert(err, jc.Satisfies, util.IsUnexpectedError)
}

func TestReleaseMachines(t *testing.T) {
	server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusOK, "[]")
	controller := getController()
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
		Comment:   "all good",
	})
	assert.Nil(t, err)

	request := server.LastRequest()
	// There should be one entry in the form Values for each of the args.
	c.Assert(request.PostForm["machines"], jc.SameContents, []string{"this", "that"})
	c.Assert(request.PostForm.Get("comment"), gc.Equals, "all good")
}

func TestReleaseMachinesBadRequest(t *testing.T) {
	server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusBadRequest, "unknown machines")
	controller := getController()
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	c.Assert(err, jc.Satisfies, util.IsBadRequestError)
	c.Assert(err.Error(), gc.Equals, "unknown machines")
}

func TestReleaseMachinesForbidden(t *testing.T) {
	server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusForbidden, "bzzt denied")
	controller := getController()
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	c.Assert(err, jc.Satisfies, util.IsPermissionError)
	c.Assert(err.Error(), gc.Equals, "bzzt denied")
}

func TestReleaseMachinesConflict(t *testing.T) {
	server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusConflict, "MachineInterface busy")
	controller := getController()
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	c.Assert(err, jc.Satisfies, util.IsCannotCompleteError)
	c.Assert(err.Error(), gc.Equals, "MachineInterface busy")
}

func TestReleaseMachinesUnexpected(t *testing.T) {
	server.AddPostResponse("/api/2.0/machines/?op=release", http.StatusBadGateway, "wat")
	controller := getController()
	err := controller.ReleaseMachines(ReleaseMachinesArgs{
		SystemIDs: []string{"this", "that"},
	})
	c.Assert(err, jc.Satisfies, util.IsUnexpectedError)
	c.Assert(err.Error(), gc.Equals, "unexpected: ServerError: 502 Bad Gateway (wat)")
}

func TestFiles(t *testing.T) {
	controller := getController()
	files, err := controller.Files("")
	assert.Nil(t, err)
	c.Assert(files, gc.HasLen, 2)

	file := files[0]
	c.Assert(file.Filename, gc.Equals, "test")

	c.Assert(file.AnonymousURI.Scheme, gc.Equals, "http")
	c.Assert(file.AnonymousURI.RequestURI(), gc.Equals, "/MAAS/api/2.0/files/?op=get_by_key&key=3afba564-fb7d-11e5-932f-52540051bf22")
}

func TestGetFile(t *testing.T) {
	server.AddGetResponse("/api/2.0/files/testing/", http.StatusOK, fileResponse)
	controller := getController()
	file, err := controller.GetFile("testing")
	assert.Nil(t, err)

	c.Assert(file.Filename, gc.Equals, "testing")

	assert.Nil(t, err)
	c.Assert(file.AnonymousURI.Scheme, gc.Equals, "http")
	c.Assert(file.AnonymousURI.RequestURI(), gc.Equals, "/MAAS/api/2.0/files/?op=get_by_key&key=88e64b76-fb82-11e5-932f-52540051bf22")
}

func TestGetFileMissing(t *testing.T) {
	controller := getController()
	_, err := controller.GetFile("missing")
	assert.True(t, util.IsNoMatchError(err))
}

func TestAddFileArgsValidate(t *testing.T) {
	reader := bytes.NewBufferString("test")
	for i, test := range []struct {
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
			assert.True(t,  errors.IsNotValid(err))
			assert.EqualError(t, err, test.errText)
		}
	}
}

func TestAddFileValidates(t *testing.T) {
	controller := getController()
	err := controller.AddFile(AddFileArgs{})
	assert.True(t, errors.IsNotValid(err))
}

func TestAddFileContent(t *testing.T) {
	server.AddPostResponse("/api/2.0/files/?op=", http.StatusOK, "")
	controller := getController()
	err := controller.AddFile(AddFileArgs{
		Filename: "foo.txt",
		Content:  []byte("foo"),
	})
	assert.Nil(t, err)

	request := server.LastRequest()
	assertFile(t, request, "foo.txt", "foo")
}

func TestAddFileReader(t *testing.T) {
	reader := bytes.NewBufferString("test\n extra over length ignored")
	server.AddPostResponse("/api/2.0/files/?op=", http.StatusOK, "")
	controller := getController()
	err := controller.AddFile(AddFileArgs{
		Filename: "foo.txt",
		Reader:   reader,
		Length:   5,
	})
	assert.Nil(t, err)

	request := server.LastRequest()
	assertFile(t, request, "foo.txt", "test\n")
}


type cleanup interface {
	AddCleanup(func(*gc.C))
}

func assertFile(t *testing.T, request *http.Request, filename, content string) {
	form := request.Form
	assert.Equal(t, form.Get("Filename"), filename)

	fileHeader := request.MultipartForm.File["File"][0]
	f, err := fileHeader.Open()
	assert.Nil(t, err)
	bytes, err := ioutil.ReadAll(f)
	assert.Nil(t, err)
	assert.Equal(t,string(bytes), content)
}


// createTestServerController creates a ControllerInterface backed on to a test server
// that has sufficient knowledge of versions and users to be able to create a
// valid ControllerInterface.
func createTestServerController(t *testing.T, suite cleanup) (*client.SimpleTestServer, *controller) {
	server := client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.Start()
	suite.AddCleanup(func(*gc.C) { server.Close() })

	controller, err := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	assert.Nil(t, err)
	return server, controller
}

func setUpTest() {
	server := client.NewSimpleServer()
	server.AddGetResponse("/api/2.0/boot-resources/", http.StatusOK, bootResourcesResponse)
	server.AddGetResponse("/api/2.0/devices/", http.StatusOK, devicesResponse)
	server.AddGetResponse("/api/2.0/fabrics/", http.StatusOK, fabricResponse)
	server.AddGetResponse("/api/2.0/files/", http.StatusOK, filesResponse)
	server.AddGetResponse("/api/2.0/machines/", http.StatusOK, machinesResponse)
	server.AddGetResponse("/api/2.0/machines/?Hostname=untasted-markita", http.StatusOK, "["+machineResponse+"]")
	server.AddGetResponse("/api/2.0/spaces/", http.StatusOK, spacesResponse)
	server.AddGetResponse("/api/2.0/static-routes/", http.StatusOK, staticRoutesResponse)
	server.AddGetResponse("/api/2.0/users/?op=whoami", http.StatusOK, `"captain awesome"`)
	server.AddGetResponse("/api/2.0/version/", http.StatusOK, versionResponse)
	server.AddGetResponse("/api/2.0/zones/", http.StatusOK, zoneResponse)
	server.Start()
}

func getController() *controller {
	controller, _ := NewController(ControllerArgs{
		BaseURL: server.URL,
		APIKey:  "fake:as:key",
	})
	return controller
}