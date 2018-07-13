// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/juju/errors"
	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/util"
	"github.com/stretchr/testify/assert"
)

func TestReadMachinesBadSchema(t *testing.T) {
	var m Machine
	err = json.Unmarshal([]byte("wat?"), &m)
	assert.Error(t, err)
}

func TestReadMachines(t *testing.T) {
	var machines []Machine
	err = json.Unmarshal([]byte(machineResponse), &machines)
	assert.Nil(t, err)
	assert.Len(t, machines, 3)

	machine := machines[0]

	assert.Equal(t, machine.SystemID, "4y3ha3")
	assert.Equal(t, machine.Hostname, "untasted-markita")
	assert.Equal(t, machine.FQDN, "untasted-markita.maas")
	assert.Contains(t, machine.Tags, []string{"virtual", "magic"})
	assert.EqualValues(t, machine.OwnerData, map[string]string{
		"fez":            "phil fish",
		"frog-fractions": "jim crawford",
	})

	assert.Contains(t, machine.IPAddresses, []string{"192.168.100.4"})
	assert.Equal(t, machine.Memory, 1024)
	assert.Equal(t, machine.CPUCount, 1)
	assert.Equal(t, machine.PowerState, "on")
	assert.Equal(t, machine.Zone.Name, "default")
	assert.Equal(t, machine.OperatingSystem, "ubuntu")
	assert.Equal(t, machine.DistroSeries, "trusty")
	assert.Equal(t, machine.Architecture, "amd64/generic")
	assert.Equal(t, machine.StatusName, "Deployed")
	assert.Equal(t, machine.StatusMessage, "From 'Deploying' to 'Deployed'")

	bootInterface := machine.BootInterface
	assert.NotNil(t, bootInterface)
	assert.Equal(t, bootInterface.Name, "eth0")

	interfaceSet := machine.InterfaceSet
	assert.Len(t, interfaceSet, 2)
	id := interfaceSet[0].ID
	assert.EqualValues(t, machine.Interface(id), interfaceSet[0])
	assert.Nil(t, machine.Interface(id+5))

	blockDevices := machine.BlockDevices
	assert.Len(t, blockDevices, 3)
	assert.Equal(t, blockDevices[0].Name, "sda")
	assert.Equal(t, blockDevices[1].Name, "sdb")
	assert.Equal(t, blockDevices[2].Name, "md0")

	blockDevices = machine.PhysicalBlockDevices
	assert.Len(t, blockDevices, 2)
	assert.Equal(t, blockDevices[0].Name, "sda")
	assert.Equal(t, blockDevices[1].Name, "sdb")

	id = blockDevices[0].ID
	assert.EqualValues(t, machine.PhysicalBlockDevice(id), blockDevices[0])
	assert.Nil(t, machine.PhysicalBlockDevice(id+5))
}

func TestReadMachinesNilValues(t *testing.T) {
	j := util.ParseJSON(t, machinesResponse)
	data := j.([]interface{})[0].(map[string]interface{})
	data["Architecture"] = nil
	data["status_message"] = nil
	data["boot_interface"] = nil

	jr, err := json.Marshal(data)
	assert.Nil(t, err)

	var machines []Machine
	err = json.Unmarshal(jr, &machines)
	assert.Nil(t, err)
	assert.Len(t, machines, 3)
	machine := machines[0]
	assert.Equal(t, machine.Architecture, "")
	assert.Equal(t, machine.StatusMessage, "")
	assert.Nil(t, machine.BootInterface)
}

func getServerAndMachine(t *testing.T) (*client.SimpleTestServer, *Machine) {
	server, controller := createTestServerController(t)
	// Just have machines return one MachineInterface
	server.AddGetResponse("/api/2.0/machines/", http.StatusOK, "["+machineResponse+"]")
	machines, err := controller.Machines(MachinesArgs{})
	assert.Nil(t, err)
	assert.Len(t, machines, 1)
	machine := machines[0]
	server.ResetRequests()
	return server, &machine
}

func TestStart(t *testing.T) {
	server, machine := getServerAndMachine(t)
	response := util.UpdateJSONMap(t, machineResponse, map[string]interface{}{
		"status_name":    "Deploying",
		"status_message": "for testing",
	})
	server.AddPostResponse(machine.ResourceURI+"?op=deploy", http.StatusOK, response)

	err := machine.Start(StartArgs{
		UserData:     "userdata",
		DistroSeries: "trusty",
		Kernel:       "kernel",
		Comment:      "a comment",
	})
	assert.Nil(t, err)
	assert.Equal(t, machine.StatusName, "Deploying")
	assert.Equal(t, machine.StatusMessage, "for testing")

	request := server.LastRequest()
	// There should be one entry in the form Values for each of the args.
	form := request.PostForm
	assert.Len(t, form, 4)
	assert.Equal(t, form.Get("user_data"), "userdata")
	assert.Equal(t, form.Get("distro_series"), "trusty")
	assert.Equal(t, form.Get("hwe_kernel"), "kernel")
	assert.Equal(t, form.Get("comment"), "a comment")
}

func TestStartMachineNotFound(t *testing.T) {
	server, machine := getServerAndMachine(t)
	server.AddPostResponse(machine.ResourceURI+"?op=deploy", http.StatusNotFound, "can't find MachineInterface")
	err := machine.Start(StartArgs{})
	assert.True(t, util.IsBadRequestError(err))
	assert.Equal(t, err.Error(), "can't find MachineInterface")
}

func TestStartMachineConflict(t *testing.T) {
	server, machine := getServerAndMachine(t)
	server.AddPostResponse(machine.ResourceURI+"?op=deploy", http.StatusConflict, "MachineInterface not allocated")
	err := machine.Start(StartArgs{})
	assert.True(t, util.IsBadRequestError(err))
	assert.Equal(t, err.Error(), "MachineInterface not allocated")
}

func TestStartMachineForbidden(t *testing.T) {
	server, machine := getServerAndMachine(t)
	server.AddPostResponse(machine.ResourceURI+"?op=deploy", http.StatusForbidden, "MachineInterface not yours")
	err := machine.Start(StartArgs{})
	assert.True(t, util.IsPermissionError(err))
	assert.Equal(t, err.Error(), "MachineInterface not yours")
}

func TestStartMachineServiceUnavailable(t *testing.T) {
	server, machine := getServerAndMachine(t)
	server.AddPostResponse(machine.ResourceURI+"?op=deploy", http.StatusServiceUnavailable, "no ip addresses available")
	err := machine.Start(StartArgs{})
	assert.True(t, util.IsCannotCompleteError(err))
	assert.Equal(t, err.Error(), "no ip addresses available")
}

func TestStartMachineUnknown(t *testing.T) {
	server, machine := getServerAndMachine(t)
	server.AddPostResponse(machine.ResourceURI+"?op=deploy", http.StatusMethodNotAllowed, "wat?")
	err := machine.Start(StartArgs{})
	assert.True(t, util.IsUnexpectedError(err))
	assert.Equal(t, err.Error(), "unexpected: ServerError: 405 Method Not Allowed (wat?)")
}

func TestDevices(t *testing.T) {
	server, machine := getServerAndMachine(t)
	server.AddGetResponse("/api/2.0/devices/", http.StatusOK, devicesResponse)
	devices, err := machine.Devices(DevicesArgs{})
	assert.Nil(t, err)
	assert.Len(t, devices, 1)
	assert.Equal(t, devices[0].Parent, machine.SystemID)
}

func TestDevicesNone(t *testing.T) {
	server, machine := getServerAndMachine(t)
	response := util.UpdateJSONMap(t, deviceResponse, map[string]interface{}{
		"Parent": "other",
	})
	server.AddGetResponse("/api/2.0/devices/", http.StatusOK, "["+response+"]")
	devices, err := machine.Devices(DevicesArgs{})
	assert.Nil(t, err)
	assert.Len(t, devices, 0)
}

func TestCreateMachineDeviceArgsValidate(t *testing.T) {
	for _, test := range []struct {
		args    CreateMachineDeviceArgs
		errText string
	}{{
		errText: "missing InterfaceName not valid",
	}, {
		args: CreateMachineDeviceArgs{
			InterfaceName: "eth1",
		},
		errText: `missing MACAddress not valid`,
	}, {
		args: CreateMachineDeviceArgs{
			InterfaceName: "eth1",
			MACAddress:    "something",
			Subnet: &subnet{
				CIDR: "1.2.3.4/5",
				VLAN: &vlan{ID: 42},
			},
			VLAN: &vlan{ID: 10},
		},
		errText: `given Subnet "1.2.3.4/5" on VLAN 42 does not match given VLAN 10`,
	}, {
		args: CreateMachineDeviceArgs{
			Hostname:      "is-optional",
			InterfaceName: "eth1",
			MACAddress:    "something",
			Subnet:        nil,
			VLAN:          &vlan{},
		},
	}, {
		args: CreateMachineDeviceArgs{
			InterfaceName: "eth1",
			MACAddress:    "something",
			Subnet:        &subnet{},
			VLAN:          nil,
		},
	}, {
		args: CreateMachineDeviceArgs{
			InterfaceName: "eth1",
			MACAddress:    "something",
			Subnet:        nil,
			VLAN:          nil,
		},
	}} {
		err := test.args.Validate()
		if test.errText == "" {
			assert.Nil(t, err)
		} else {
			assert.True(t, errors.IsNotValid(err))
			assert.Equal(t, err.Error(), test.errText)
		}
	}
}

func TestCreateDeviceValidates(t *testing.T) {
	_, machine := getServerAndMachine(t)
	_, err := machine.CreateDevice(CreateMachineDeviceArgs{})
	assert.True(t, errors.IsNotValid(err))
	assert.Equal(t, err.Error(), "missing InterfaceName not valid")
}

func TestCreateDevice(t *testing.T) {
	server, machine := getServerAndMachine(t)
	// The createDeviceResponse returns a single interface with the Name "eth0".
	server.AddPostResponse("/api/2.0/devices/?op=", http.StatusOK, createDeviceResponse)
	updateInterfaceResponse := util.UpdateJSONMap(t, interfaceResponse, map[string]interface{}{
		"Name":         "eth4",
		"Links":        []interface{}{},
		"resource_uri": "/MAAS/api/2.0/nodes/4y3haf/interfaces/48/",
	})
	server.AddPutResponse("/MAAS/api/2.0/nodes/4y3haf/interfaces/48/", http.StatusOK, updateInterfaceResponse)
	linkSubnetResponse := util.UpdateJSONMap(t, interfaceResponse, map[string]interface{}{
		"Name":         "eth4",
		"resource_uri": "/MAAS/api/2.0/nodes/4y3haf/interfaces/48/",
	})
	server.AddPostResponse("/MAAS/api/2.0/nodes/4y3haf/interfaces/48/?op=link_subnet", http.StatusOK, linkSubnetResponse)
	subnet := machine.BootInterface.Links[0].Subnet
	device, err := machine.CreateDevice(CreateMachineDeviceArgs{
		InterfaceName: "eth4",
		MACAddress:    "fake-mac-address",
		Subnet:        subnet,
		VLAN:          subnet.VLAN,
	})
	assert.Nil(t, err)
	assert.Equal(t, device.InterfaceSet[0].Name, "eth4")
	assert.Equal(t, device.InterfaceSet[0].VLAN.ID, subnet.VLAN.ID)
}

func TestCreateDeviceWithoutSubnetOrVLAN(t *testing.T) {
	server, machine := getServerAndMachine(t)
	// The createDeviceResponse returns a single interface with the Name "eth0".
	server.AddPostResponse("/api/2.0/devices/?op=", http.StatusOK, createDeviceResponse)
	updateInterfaceResponse := util.UpdateJSONMap(t, interfaceResponse, map[string]interface{}{
		"Name":         "eth4",
		"Links":        []interface{}{},
		"resource_uri": "/MAAS/api/2.0/nodes/4y3haf/interfaces/48/",
	})
	server.AddPutResponse("/MAAS/api/2.0/nodes/4y3haf/interfaces/48/", http.StatusOK, updateInterfaceResponse)
	device, err := machine.CreateDevice(CreateMachineDeviceArgs{
		InterfaceName: "eth4",
		MACAddress:    "fake-mac-address",
		Subnet:        nil,
		VLAN:          nil,
	})
	assert.Nil(t, err)
	assert.Equal(t, device.InterfaceSet[0].Name, "eth4")
	// No specifc Subnet or VLAN should be set.
	assert.Equal(t, device.InterfaceSet[0].VLAN.ID, 1) // set in interfaceResponse
	assert.Len(t, device.InterfaceSet[0].Links, 0)     // set above
}

func TestCreateDeviceWithVLANOnly(t *testing.T) {
	server, machine := getServerAndMachine(t)
	// The createDeviceResponse returns a single interface with the Name "eth0".
	server.AddPostResponse("/api/2.0/devices/?op=", http.StatusOK, createDeviceResponse)
	updateInterfaceResponse := util.UpdateJSONMap(t, interfaceResponse, map[string]interface{}{
		"Name": "eth4",
		"VLAN": map[string]interface{}{
			"ID":           42,
			"resource_uri": "/MAAS/api/2.0/VLANs/42/",
			"VID":          1234,
			"Fabric":       "live",
			"dhcp_on":      false,
			"MTU":          9001,
		},
		"Links":        []interface{}{},
		"resource_uri": "/MAAS/api/2.0/nodes/4y3haf/interfaces/48/",
	})
	server.AddPutResponse("/MAAS/api/2.0/nodes/4y3haf/interfaces/48/", http.StatusOK, updateInterfaceResponse)
	device, err := machine.CreateDevice(CreateMachineDeviceArgs{
		InterfaceName: "eth4",
		MACAddress:    "fake-mac-address",
		Subnet:        nil,
		VLAN:          &vlan{ID: 42},
	})
	assert.Nil(t, err)
	assert.Equal(t, device.InterfaceSet[0].Name, "eth4")
	// VLAN should be set.
	assert.Equal(t, device.InterfaceSet[0].VLAN.ID, 42)
}

func TestCreateDeviceTriesToDeleteDeviceOnError(t *testing.T) {
	server, machine := getServerAndMachine(t)
	// The createDeviceResponse returns a single interface with the Name "eth0".
	server.AddPostResponse("/api/2.0/devices/?op=", http.StatusOK, createDeviceResponse)
	updateInterfaceResponse := util.UpdateJSONMap(t, interfaceResponse, map[string]interface{}{
		"Name":         "eth4",
		"Links":        []interface{}{},
		"resource_uri": "/MAAS/api/2.0/nodes/4y3haf/interfaces/48/",
	})
	server.AddPutResponse("/MAAS/api/2.0/nodes/4y3haf/interfaces/48/", http.StatusOK, updateInterfaceResponse)
	server.AddPostResponse("/MAAS/api/2.0/nodes/4y3haf/interfaces/48/?op=link_subnet", http.StatusServiceUnavailable, "no addresses")
	// We'll ignore that that it fails to delete, all we care about testing is that it tried.
	subnet := machine.BootInterface.Links[0].Subnet
	_, err := machine.CreateDevice(CreateMachineDeviceArgs{
		InterfaceName: "eth4",
		MACAddress:    "fake-mac-address",
		Subnet:        subnet,
	})
	assert.True(t, util.IsCannotCompleteError(err))

	request := server.LastRequest()
	assert.Equal(t, request.Method, "DELETE")
	assert.Equal(t, request.RequestURI, "/MAAS/api/2.0/devices/4y3haf/")
}

func TestOwnerDataCopies(t *testing.T) {
	machine := Machine{OwnerData: make(map[string]string)}
	ownerData := machine.OwnerData
	ownerData["sad"] = "Children"
	assert.Contains(t, machine.OwnerData, map[string]string{})
}

func TestSetOwnerData(t *testing.T) {
	server, machine := getServerAndMachine(t)
	server.AddPostResponse(machine.ResourceURI+"?op=set_owner_data", 200, machineWithOwnerData(`{"returned": "data"}`))
	err := machine.SetOwnerData(map[string]string{
		"draco": "malfoy",
		"empty": "", // Check that empty strings get passed along.
	})
	assert.Nil(t, err)
	assert.Contains(t, machine.OwnerData, map[string]string{"returned": "data"})
	form := server.LastRequest().PostForm
	// Looking at the map directly so we can tell the difference
	// between no value and an explicit empty string.
	assert.Contains(t, form["draco"], []string{"malfoy"})
	assert.Contains(t, form["empty"], []string{""})
}

func machineWithOwnerData(data string) string {
	return fmt.Sprintf(machineOwnerDataTemplate, data)
}

const (
	machineOwnerDataTemplate = `
	{
        "netboot": false,
        "constraints_by_type": {
          "storage": {
              "0": [
                  23
              ]
          }
         },
        "system_id": "4y3ha3",
        "ip_addresses": [
            "192.168.100.4"
        ],
        "Memory": 1024,
        "cpu_count": 1,
        "hwe_kernel": "hwe-t",
        "status_action": "",
        "osystem": "ubuntu",
        "node_type_name": "MachineInterface",
        "macaddress_set": [
            {
                "mac_address": "52:54:00:55:b6:80"
            }
        ],
        "special_filesystems": [],
        "status": 6,
        "virtualblockdevice_set": [
            {
                "block_size": 512,
                "serial": null,
                "Path": "/dev/disk/by-dname/md0",
                "system_id": "xc3e6q",
                "available_size": 256599130112,
                "Size": 256599130112,
                "UUID": "b76de3fd-d05f-4a3f-b515-189de53d6c03",
                "Tags": [
                    "raid0"
                ],
                "used_size": 0,
                "Name": "md0",
                "type": "virtual",
                "filesystem": null,
                "used_for": "Unused",
                "Partitions": [],
                "ID": 23,
                "partition_table_type": null,
                "Model": null,
                "id_path": null,
                "resource_uri": "/MAAS/api/2.0/nodes/xc3e6q/blockdevices/23/"
            }
         ],

        "physicalblockdevice_set": [
            {
                "Path": "/dev/disk/by-dname/sda",
                "Name": "sda",
                "used_for": "MBR partitioned with 1 partition",
                "Partitions": [
                    {
                        "bootable": false,
                        "ID": 1,
                        "Path": "/dev/disk/by-dname/sda-part1",
                        "filesystem": {
                            "Type": "ext4",
                            "mount_point": "/",
                            "Label": "root",
                            "mount_options": null,
                            "UUID": "fcd7745e-f1b5-4f5d-9575-9b0bb796b752"
                        },
                        "type": "partition",
                        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/34/partition/1",
                        "UUID": "6199b7c9-b66f-40f6-a238-a938a58a0adf",
                        "used_for": "ext4 formatted filesystem mounted at /",
                        "Size": 8581545984
                    }
                ],
                "filesystem": null,
                "id_path": "/dev/disk/by-ID/ata-QEMU_HARDDISK_QM00001",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/34/",
                "ID": 34,
                "serial": "QM00001",
                "type": "physical",
                "block_size": 4096,
                "used_size": 8586788864,
                "available_size": 0,
                "partition_table_type": "MBR",
                "UUID": null,
                "Size": 8589934592,
                "Model": "QEMU HARDDISK",
                "Tags": [
                    "rotary"
                ]
            },
            {
                "Path": "/dev/disk/by-dname/sdb",
                "Name": "sdb",
                "used_for": "MBR partitioned with 1 partition",
                "Partitions": [
                    {
                        "bootable": false,
                        "ID": 101,
                        "Path": "/dev/disk/by-dname/sdb-part1",
                        "filesystem": {
                            "Type": "ext4",
                            "mount_point": "/home",
                            "Label": "home",
                            "mount_options": null,
                            "UUID": "fcd7745e-f1b5-4f5d-9575-9b0bb796b753"
                        },
                        "type": "partition",
                        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/98/partition/101",
                        "UUID": "6199b7c9-b66f-40f6-a238-a938a58a0ae0",
                        "used_for": "ext4 formatted filesystem mounted at /home",
                        "Size": 8581545984
                    }
                ],
                "filesystem": null,
                "id_path": "/dev/disk/by-ID/ata-QEMU_HARDDISK_QM00002",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/98/",
                "ID": 98,
                "serial": "QM00002",
                "type": "physical",
                "block_size": 4096,
                "used_size": 8586788864,
                "available_size": 0,
                "partition_table_type": "MBR",
                "UUID": null,
                "Size": 8589934592,
                "Model": "QEMU HARDDISK",
                "Tags": [
                    "rotary"
                ]
            }
        ],
        "interface_set": [
            {
                "effective_mtu": 1500,
                "mac_address": "52:54:00:55:b6:80",
                "Children": [],
                "discovered": [],
                "params": "",
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
                "Parents": [],
                "ID": 35,
                "type": "physical",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/interfaces/35/",
                "Tags": [],
                "Links": [
                    {
                        "ID": 82,
                        "ip_address": "192.168.100.4",
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
                        },
                        "Mode": "auto"
                    }
                ]
            },
            {
                "effective_mtu": 1500,
                "mac_address": "52:54:00:55:b6:81",
                "Children": [],
                "discovered": [],
                "params": "",
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
                "Parents": [],
                "ID": 99,
                "type": "physical",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/interfaces/99/",
                "Tags": [],
                "Links": [
                    {
                        "ID": 83,
                        "ip_address": "192.168.100.5",
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
                        },
                        "Mode": "auto"
                    }
                ]
            }
        ],
        "resource_uri": "/MAAS/api/2.0/machines/4y3ha3/",
        "Hostname": "untasted-markita",
        "status_name": "Deployed",
        "min_hwe_kernel": "",
        "address_ttl": null,
        "boot_interface": {
            "effective_mtu": 1500,
            "mac_address": "52:54:00:55:b6:80",
            "Children": [],
            "discovered": [],
            "params": "",
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
            "Parents": [],
            "ID": 35,
            "type": "physical",
            "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/interfaces/35/",
            "Tags": [],
            "Links": [
                {
                    "ID": 82,
                    "ip_address": "192.168.100.4",
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
                    },
                    "Mode": "auto"
                }
            ]
        },
        "power_state": "on",
        "Architecture": "amd64/generic",
        "power_type": "virsh",
        "distro_series": "trusty",
        "tag_names": [
           "virtual", "magic"
        ],
        "disable_ipv4": false,
        "status_message": "From 'Deploying' to 'Deployed'",
        "swap_size": null,
        "blockdevice_set": [
            {
                "Path": "/dev/disk/by-dname/sda",
                "partition_table_type": "MBR",
                "Name": "sda",
                "used_for": "MBR partitioned with 1 partition",
                "Partitions": [
                    {
                        "bootable": false,
                        "ID": 1,
                        "Path": "/dev/disk/by-dname/sda-part1",
                        "filesystem": {
                            "Type": "ext4",
                            "mount_point": "/",
                            "Label": "root",
                            "mount_options": null,
                            "UUID": "fcd7745e-f1b5-4f5d-9575-9b0bb796b752"
                        },
                        "type": "partition",
                        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/34/partition/1",
                        "UUID": "6199b7c9-b66f-40f6-a238-a938a58a0adf",
                        "used_for": "ext4 formatted filesystem mounted at /",
                        "Size": 8581545984
                    }
                ],
                "filesystem": null,
                "id_path": "/dev/disk/by-ID/ata-QEMU_HARDDISK_QM00001",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/34/",
                "ID": 34,
                "serial": "QM00001",
                "block_size": 4096,
                "type": "physical",
                "used_size": 8586788864,
                "Tags": [
                    "rotary"
                ],
                "available_size": 0,
                "UUID": null,
                "Size": 8589934592,
                "Model": "QEMU HARDDISK"
            },
            {
                "Path": "/dev/disk/by-dname/sdb",
                "Name": "sdb",
                "used_for": "MBR partitioned with 1 partition",
                "Partitions": [
                    {
                        "bootable": false,
                        "ID": 101,
                        "Path": "/dev/disk/by-dname/sdb-part1",
                        "filesystem": {
                            "Type": "ext4",
                            "mount_point": "/home",
                            "Label": "home",
                            "mount_options": null,
                            "UUID": "fcd7745e-f1b5-4f5d-9575-9b0bb796b753"
                        },
                        "type": "partition",
                        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/98/partition/101",
                        "UUID": "6199b7c9-b66f-40f6-a238-a938a58a0ae0",
                        "used_for": "ext4 formatted filesystem mounted at /home",
                        "Size": 8581545984
                    }
                ],
                "filesystem": null,
                "id_path": "/dev/disk/by-ID/ata-QEMU_HARDDISK_QM00002",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/98/",
                "ID": 98,
                "serial": "QM00002",
                "type": "physical",
                "block_size": 4096,
                "used_size": 8586788864,
                "available_size": 0,
                "partition_table_type": "MBR",
                "UUID": null,
                "Size": 8589934592,
                "Model": "QEMU HARDDISK",
                "Tags": [
                    "rotary"
                ]
            },
            {
                "Tags": [
                    "raid0"
                ],
                "used_size": 0,
                "Path": "/dev/disk/by-dname/md0",
                "serial": null,
                "available_size": 256599130112,
                "system_id": "xc3e6q",
                "UUID": "b76de3fd-d05f-4a3f-b515-189de53d6c03",
                "block_size": 512,
                "Size": 256599130112,
                "type": "virtual",
                "filesystem": null,
                "used_for": "Unused",
                "Partitions": [],
                "ID": 23,
                "Name": "md0",
                "partition_table_type": null,
                "Model": null,
                "id_path": null,
                "resource_uri": "/MAAS/api/2.0/nodes/xc3e6q/blockdevices/23/"
            }
        ],
        "Zone": {
            "Description": "",
            "resource_uri": "/MAAS/api/2.0/zones/default/",
            "Name": "default"
        },
        "FQDN": "untasted-markita.maas",
        "storage": 8589.934592,
        "node_type": 0,
        "boot_disk": null,
        "Owner": "thumper",
        "domain": {
            "ID": 0,
            "Name": "maas",
            "resource_uri": "/MAAS/api/2.0/domains/0/",
            "resource_record_count": 0,
            "ttl": null,
            "authoritative": true
        },
        "owner_data": %s
    }
`

	createDeviceResponse = `
{
	"Zone": {
		"Description": "",
		"resource_uri": "/MAAS/api/2.0/zones/default/",
		"Name": "default"
	},
	"domain": {
		"resource_record_count": 0,
		"resource_uri": "/MAAS/api/2.0/domains/0/",
		"authoritative": true,
		"Name": "maas",
		"ttl": null,
		"ID": 0
	},
	"node_type_name": "DeviceInterface",
	"address_ttl": null,
	"Hostname": "furnacelike-brittney",
	"node_type": 1,
	"resource_uri": "/MAAS/api/2.0/devices/4y3haf/",
	"ip_addresses": ["192.168.100.11"],
	"Owner": "thumper",
	"tag_names": [],
	"FQDN": "furnacelike-brittney.maas",
	"system_id": "4y3haf",
	"Parent": "4y3ha3",
	"interface_set": [
		{
			"resource_uri": "/MAAS/api/2.0/nodes/4y3haf/interfaces/48/",
			"type": "physical",
			"mac_address": "78:f0:f1:16:a7:46",
			"params": "",
			"discovered": null,
			"effective_mtu": 1500,
			"ID": 48,
			"Children": [],
			"Links": [],
			"Name": "eth0",
			"VLAN": {
				"secondary_rack": null,
				"dhcp_on": true,
				"Fabric": "Fabric-0",
				"MTU": 1500,
				"primary_rack": "4y3h7n",
				"resource_uri": "/MAAS/api/2.0/VLANs/1/",
				"external_dhcp": null,
				"Name": "untagged",
				"ID": 1,
				"VID": 0
			},
			"Tags": [],
			"Parents": [],
			"Enabled": true
		}
	]
}
`
)

var (
	machineResponse = machineWithOwnerData(`{
            "fez": "phil fish",
            "frog-fractions": "jim crawford"
        }
`)

	machinesResponse = "[" + machineResponse + `,
    {
        "netboot": true,
        "system_id": "4y3ha4",
        "ip_addresses": [],
        "virtualblockdevice_set": [],
        "Memory": 1024,
        "cpu_count": 1,
        "hwe_kernel": "",
        "status_action": "",
        "osystem": "",
        "node_type_name": "MachineInterface",
        "macaddress_set": [
            {
                "mac_address": "52:54:00:33:6b:2c"
            }
        ],
        "special_filesystems": [],
        "status": 4,
        "physicalblockdevice_set": [
            {
                "Path": "/dev/disk/by-dname/sda",
                "Name": "sda",
                "used_for": "MBR partitioned with 1 partition",
                "Partitions": [
                    {
                        "bootable": false,
                        "ID": 2,
                        "Path": "/dev/disk/by-dname/sda-part1",
                        "filesystem": {
                            "Type": "ext4",
                            "mount_point": "/",
                            "Label": "root",
                            "mount_options": null,
                            "UUID": "7a0e75a8-0bc6-456b-ac92-4769e97baf02"
                        },
                        "type": "partition",
                        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha4/blockdevices/35/partition/2",
                        "UUID": "6fe782cf-ad1a-4b31-8beb-333401b4d4bb",
                        "used_for": "ext4 formatted filesystem mounted at /",
                        "Size": 8581545984
                    }
                ],
                "filesystem": null,
                "id_path": "/dev/disk/by-ID/ata-QEMU_HARDDISK_QM00001",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha4/blockdevices/35/",
                "ID": 35,
                "serial": "QM00001",
                "type": "physical",
                "block_size": 4096,
                "used_size": 8586788864,
                "available_size": 0,
                "partition_table_type": "MBR",
                "UUID": null,
                "Size": 8589934592,
                "Model": "QEMU HARDDISK",
                "Tags": [
                    "rotary"
                ]
            }
        ],
        "interface_set": [
            {
                "effective_mtu": 1500,
                "mac_address": "52:54:00:33:6b:2c",
                "Children": [],
                "discovered": [],
                "params": "",
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
                "Parents": [],
                "ID": 39,
                "type": "physical",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha4/interfaces/39/",
                "Tags": [],
                "Links": [
                    {
                        "ID": 67,
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
        ],
        "resource_uri": "/MAAS/api/2.0/machines/4y3ha4/",
        "Hostname": "lowlier-glady",
        "status_name": "Ready",
        "min_hwe_kernel": "",
        "address_ttl": null,
        "boot_interface": {
            "effective_mtu": 1500,
            "mac_address": "52:54:00:33:6b:2c",
            "Children": [],
            "discovered": [],
            "params": "",
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
            "Parents": [],
            "ID": 39,
            "type": "physical",
            "resource_uri": "/MAAS/api/2.0/nodes/4y3ha4/interfaces/39/",
            "Tags": [],
            "Links": [
                {
                    "ID": 67,
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
        },
        "power_state": "off",
        "Architecture": "amd64/generic",
        "power_type": "virsh",
        "distro_series": "",
        "tag_names": [
            "virtual"
        ],
        "disable_ipv4": false,
        "status_message": "From 'Commissioning' to 'Ready'",
        "swap_size": null,
        "blockdevice_set": [
            {
                "Path": "/dev/disk/by-dname/sda",
                "partition_table_type": "MBR",
                "Name": "sda",
                "used_for": "MBR partitioned with 1 partition",
                "Partitions": [
                    {
                        "bootable": false,
                        "ID": 2,
                        "Path": "/dev/disk/by-dname/sda-part1",
                        "filesystem": {
                            "Type": "ext4",
                            "mount_point": "/",
                            "Label": "root",
                            "mount_options": null,
                            "UUID": "7a0e75a8-0bc6-456b-ac92-4769e97baf02"
                        },
                        "type": "partition",
                        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha4/blockdevices/35/partition/2",
                        "UUID": "6fe782cf-ad1a-4b31-8beb-333401b4d4bb",
                        "used_for": "ext4 formatted filesystem mounted at /",
                        "Size": 8581545984
                    }
                ],
                "filesystem": null,
                "id_path": "/dev/disk/by-ID/ata-QEMU_HARDDISK_QM00001",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha4/blockdevices/35/",
                "ID": 35,
                "serial": "QM00001",
                "block_size": 4096,
                "type": "physical",
                "used_size": 8586788864,
                "Tags": [
                    "rotary"
                ],
                "available_size": 0,
                "UUID": null,
                "Size": 8589934592,
                "Model": "QEMU HARDDISK"
            }
        ],
        "Zone": {
            "Description": "",
            "resource_uri": "/MAAS/api/2.0/zones/default/",
            "Name": "default"
        },
        "FQDN": "lowlier-glady.maas",
        "storage": 8589.934592,
        "node_type": 0,
        "boot_disk": null,
        "Owner": null,
        "domain": {
            "ID": 0,
            "Name": "maas",
            "resource_uri": "/MAAS/api/2.0/domains/0/",
            "resource_record_count": 0,
            "ttl": null,
            "authoritative": true
        },
        "owner_data": {
            "braid": "jonathan blow",
            "frog-fractions": "jim crawford"
        }
    },
    {
        "netboot": true,
        "system_id": "4y3ha6",
        "ip_addresses": [],
        "virtualblockdevice_set": [],
        "Memory": 1024,
        "cpu_count": 1,
        "hwe_kernel": "",
        "status_action": "",
        "osystem": "",
        "node_type_name": "MachineInterface",
        "macaddress_set": [
            {
                "mac_address": "52:54:00:c9:6a:45"
            }
        ],
        "special_filesystems": [],
        "status": 4,
        "physicalblockdevice_set": [
            {
                "Path": "/dev/disk/by-dname/sda",
                "Name": "sda",
                "used_for": "MBR partitioned with 1 partition",
                "Partitions": [
                    {
                        "bootable": false,
                        "ID": 3,
                        "Path": "/dev/disk/by-dname/sda-part1",
                        "filesystem": {
                            "Type": "ext4",
                            "mount_point": "/",
                            "Label": "root",
                            "mount_options": null,
                            "UUID": "f15b4e94-7dc3-460d-8838-0c299905c799"
                        },
                        "type": "partition",
                        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha6/blockdevices/36/partition/3",
                        "UUID": "a20ae130-bd8f-41b5-bdb3-47ab11a621b5",
                        "used_for": "ext4 formatted filesystem mounted at /",
                        "Size": 8581545984
                    }
                ],
                "filesystem": null,
                "id_path": "/dev/disk/by-ID/ata-QEMU_HARDDISK_QM00001",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha6/blockdevices/36/",
                "ID": 36,
                "serial": "QM00001",
                "type": "physical",
                "block_size": 4096,
                "used_size": 8586788864,
                "available_size": 0,
                "partition_table_type": "MBR",
                "UUID": null,
                "Size": 8589934592,
                "Model": "QEMU HARDDISK",
                "Tags": [
                    "rotary"
                ]
            }
        ],
        "interface_set": [
            {
                "effective_mtu": 1500,
                "mac_address": "52:54:00:c9:6a:45",
                "Children": [],
                "discovered": [],
                "params": "",
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
                "Parents": [],
                "ID": 40,
                "type": "physical",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha6/interfaces/40/",
                "Tags": [],
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
        ],
        "resource_uri": "/MAAS/api/2.0/machines/4y3ha6/",
        "Hostname": "icier-nina",
        "status_name": "Ready",
        "min_hwe_kernel": "",
        "address_ttl": null,
        "boot_interface": {
            "effective_mtu": 1500,
            "mac_address": "52:54:00:c9:6a:45",
            "Children": [],
            "discovered": [],
            "params": "",
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
            "Parents": [],
            "ID": 40,
            "type": "physical",
            "resource_uri": "/MAAS/api/2.0/nodes/4y3ha6/interfaces/40/",
            "Tags": [],
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
        },
        "power_state": "off",
        "Architecture": "amd64/generic",
        "power_type": "virsh",
        "distro_series": "",
        "tag_names": [
            "virtual"
        ],
        "disable_ipv4": false,
        "status_message": "From 'Commissioning' to 'Ready'",
        "swap_size": null,
        "blockdevice_set": [
            {
                "Path": "/dev/disk/by-dname/sda",
                "partition_table_type": "MBR",
                "Name": "sda",
                "used_for": "MBR partitioned with 1 partition",
                "Partitions": [
                    {
                        "bootable": false,
                        "ID": 3,
                        "Path": "/dev/disk/by-dname/sda-part1",
                        "filesystem": {
                            "Type": "ext4",
                            "mount_point": "/",
                            "Label": "root",
                            "mount_options": null,
                            "UUID": "f15b4e94-7dc3-460d-8838-0c299905c799"
                        },
                        "type": "partition",
                        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha6/blockdevices/36/partition/3",
                        "UUID": "a20ae130-bd8f-41b5-bdb3-47ab11a621b5",
                        "used_for": "ext4 formatted filesystem mounted at /",
                        "Size": 8581545984
                    }
                ],
                "filesystem": null,
                "id_path": "/dev/disk/by-ID/ata-QEMU_HARDDISK_QM00001",
                "resource_uri": "/MAAS/api/2.0/nodes/4y3ha6/blockdevices/36/",
                "ID": 36,
                "serial": "QM00001",
                "block_size": 4096,
                "type": "physical",
                "used_size": 8586788864,
                "Tags": [
                    "rotary"
                ],
                "available_size": 0,
                "UUID": null,
                "Size": 8589934592,
                "Model": "QEMU HARDDISK"
            }
        ],
        "Zone": {
            "Description": "",
            "resource_uri": "/MAAS/api/2.0/zones/default/",
            "Name": "default"
        },
        "FQDN": "icier-nina.maas",
        "storage": 8589.934592,
        "node_type": 0,
        "boot_disk": null,
        "Owner": null,
        "domain": {
            "ID": 0,
            "Name": "maas",
            "resource_uri": "/MAAS/api/2.0/domains/0/",
            "resource_record_count": 0,
            "ttl": null,
            "authoritative": true
        },
        "owner_data": {
            "braid": "jonathan blow",
            "fez": "phil fish"
        }
    }
]
`
)
