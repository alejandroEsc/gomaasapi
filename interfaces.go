// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import "github.com/juju/utils/set"

const (
	// Capability constants.
	NetworksManagement      = "networks-management"
	StaticIPAddresses       = "static-ipaddresses"
	IPv6DeploymentUbuntu    = "ipv6-deployment-ubuntu"
	DevicesManagement       = "devices-management"
	StorageDeploymentUbuntu = "storage-deployment-ubuntu"
	NetworkDeploymentUbuntu = "network-deployment-ubuntu"
)

// Controller represents an API connection to a MAAS Controller. Since the API
// is restful, there is no long held connection to the API server, but instead
// HTTP calls are made and JSON response structures parsed.
type Controller interface {

	// Capabilities returns a set of Capabilities as defined by the string
	// constants.
	Capabilities() set.Strings

	BootResources() ([]bootResource, error)

	// Fabrics returns the list of Fabrics defined in the MAAS Controller.
	Fabrics() ([]fabric, error)

	// Spaces returns the list of Spaces defined in the MAAS Controller.
	Spaces() ([]space, error)

	// StaticRoutes returns the list of StaticRoutes defined in the MAAS Controller.
	StaticRoutes() ([]staticRoute, error)

	// Zones lists all the zones known to the MAAS Controller.
	Zones() ([]zone, error)

	// Machines returns a list of machines that match the params.
	Machines(MachinesArgs) ([]Machine, error)

	// AllocateMachine will attempt to allocate a Machine to the user.
	// If successful, the allocated Machine is returned.
	AllocateMachine(AllocateMachineArgs) (Machine, ConstraintMatches, error)

	// ReleaseMachines will stop the specified machines, and release them
	// from the user making them available to be allocated again.
	ReleaseMachines(ReleaseMachinesArgs) error

	// Devices returns a list of devices that match the params.
	Devices(DevicesArgs) ([]Device, error)

	// CreateDevice creates and returns a new Device.
	CreateDevice(CreateDeviceArgs) (Device, error)

	// Files returns all the files that match the specified prefix.
	Files(prefix string) ([]File, error)

	// Return a single file by its Filename.
	GetFile(filename string) (File, error)

	// AddFile adds or replaces the Content of the specified Filename.
	// If or when the MAAS api is able to return metadata about a single
	// file without sending the Content of the file, we can return a File
	// instance here too.
	AddFile(AddFileArgs) error
}

// File represents a file stored in the MAAS Controller.
type File interface {
	// Delete removes the file from the MAAS Controller.
	Delete() error
	// ReadAll returns the Content of the file.
	ReadAll() ([]byte, error)
}

type Device interface {
	// CreateInterface will create a physical interface for this Machine.
	CreateInterface(CreateInterfaceArgs) (MachineNetworkInterface, error)
	// Delete will remove this Device.
	Delete() error
}

type Machine interface {
	OwnerDataHolder

	// Devices returns a list of devices that match the params and have
	// this Machine as the Parent.
	Devices(DevicesArgs) ([]Device, error)

	InterfaceSet() []MachineNetworkInterface
	// MachineNetworkInterface returns the interface for the Machine that matches the ID
	// specified. If there is no match, nil is returned.
	Interface(id int) MachineNetworkInterface
	// BlockDevice returns the block device for the Machine that matches the
	// ID specified. If there is no match, nil is returned.
	BlockDevice(id int) blockdevice

	Zone() zone

	// Start the Machine and install the operating system specified in the args.
	Start(StartArgs) error

	// CreateDevice creates a new Device with this Machine as the Parent.
	// The device will have one interface that is linked to the specified Subnet.
	CreateDevice(CreateMachineDeviceArgs) (Device, error)
}

// MachineNetworkInterface represents a physical or virtual network interface on a Machine.
type MachineNetworkInterface interface {
	// Params is a JSON field, and defaults to an empty string, but is almost
	// always a JSON object in practice. Gleefully ignoring it until we need it.

	// Update the Name, mac address or VLAN.
	Update(UpdateInterfaceArgs) error

	// Delete this interface.
	Delete() error

	// LinkSubnet will attempt to make this interface available on the specified
	// Subnet.
	LinkSubnet(LinkSubnetArgs) error

	// UnlinkSubnet will remove the Link to the Subnet, and release the IP
	// address associated if there is one.
	UnlinkSubnet(subnet) error
}

// OwnerDataHolder represents any MAAS object that can store key/value
// data.
type OwnerDataHolder interface {
	// OwnerData returns a copy of the key/value data stored for this
	// object.
	OwnerData() map[string]string

	// SetOwnerData updates the key/value data stored for this object
	// with the Values passed in. Existing keys that aren't specified
	// in the map passed in will be left in place; to clear a key set
	// its value to "". All Owner data is cleared when the object is
	// released.
	SetOwnerData(map[string]string) error
}
