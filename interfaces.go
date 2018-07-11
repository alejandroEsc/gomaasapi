// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

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

// ControllerInterface represents an API connection to a MAAS ControllerInterface. Since the API
// is restful, there is no long held connection to the API server, but instead
// HTTP calls are made and JSON response structures parsed.
type ControllerInterface interface {

	// Capabilities returns a set of Capabilities as defined by the string
	// constants.
	Capabilities() set.Strings

	BootResources() ([]bootResource, error)

	// Fabrics returns the list of Fabrics defined in the MAAS ControllerInterface.
	Fabrics() ([]fabric, error)

	// Spaces returns the list of Spaces defined in the MAAS ControllerInterface.
	Spaces() ([]space, error)

	// StaticRoutes returns the list of StaticRoutes defined in the MAAS ControllerInterface.
	StaticRoutes() ([]staticRoute, error)

	// Zones lists all the zones known to the MAAS ControllerInterface.
	Zones() ([]zone, error)

	// Machines returns a list of machines that match the params.
	Machines(MachinesArgs) ([]MachineInterface, error)

	// AllocateMachine will attempt to allocate a MachineInterface to the user.
	// If successful, the allocated MachineInterface is returned.
	AllocateMachine(AllocateMachineArgs) (MachineInterface, ConstraintMatches, error)

	// ReleaseMachines will stop the specified machines, and release them
	// from the user making them available to be allocated again.
	ReleaseMachines(ReleaseMachinesArgs) error

	// Devices returns a list of devices that match the params.
	Devices(DevicesArgs) ([]DeviceInterface, error)

	// CreateDevice creates and returns a new DeviceInterface.
	CreateDevice(CreateDeviceArgs) (DeviceInterface, error)

	// Files returns all the files that match the specified prefix.
	Files(prefix string) ([]FileInterface, error)

	// Return a single File by its Filename.
	GetFile(filename string) (FileInterface, error)

	// AddFile adds or replaces the Content of the specified Filename.
	// If or when the MAAS api is able to return metadata about a single
	// File without sending the Content of the File, we can return a FileInterface
	// instance here too.
	AddFile(AddFileArgs) error
}

// FileInterface represents a File stored in the MAAS ControllerInterface.
type FileInterface interface {
	// Delete removes the File from the MAAS ControllerInterface.
	Delete() error
	// ReadAll returns the Content of the File.
	ReadAll() ([]byte, error)
}

type DeviceInterface interface {
	// CreateInterface will create a physical interface for this MachineInterface.
	CreateInterface(CreateInterfaceArgs) (MachineNetworkInterface, error)
	// Delete will remove this DeviceInterface.
	Delete() error
}

type MachineInterface interface {
	OwnerDataHolderInterface

	// Devices returns a list of devices that match the params and have
	// this MachineInterface as the Parent.
	Devices(DevicesArgs) ([]DeviceInterface, error)

	InterfaceSet() []*MachineNetworkInterface
	// MachineNetworkInterface returns the interface for the MachineInterface that matches the ID
	// specified. If there is no match, nil is returned.
	Interface(id int) *MachineNetworkInterface
	// BlockDevice returns the block device for the MachineInterface that matches the
	// ID specified. If there is no match, nil is returned.
	BlockDevice(id int) blockdevice

	Zone() zone

	// Start the MachineInterface and install the operating system specified in the args.
	Start(StartArgs) error

	// CreateDevice creates a new DeviceInterface with this MachineInterface as the Parent.
	// The device will have one interface that is linked to the specified Subnet.
	CreateDevice(CreateMachineDeviceArgs) (DeviceInterface, error)
}

// MachineNetworkInterface represents a physical or virtual network interface on a MachineInterface.
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

// OwnerDataHolderInterface represents any MAAS object that can store key/value
// data.
type OwnerDataHolderInterface interface {
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
