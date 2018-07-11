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

// OwnerDataHolderInterface represents any MAAS object that can store key/value
// data.
type OwnerDataHolderInterface interface {
	// SetOwnerData updates the key/value data stored for this object
	// with the Values passed in. Existing keys that aren't specified
	// in the map passed in will be left in place; to clear a key set
	// its value to "". All Owner data is cleared when the object is
	// released.
	SetOwnerData(map[string]string) error
}
