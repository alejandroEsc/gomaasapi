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

	StaticRoutes() ([]staticRoute, error)

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
}
