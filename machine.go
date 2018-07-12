// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	"fmt"
	"net/http"
	"net/url"

	"encoding/json"
	"github.com/juju/errors"
)

// MachineInterface represents a physical MachineInterface.
type Machine struct {
	Controller *controller

	ResourceURI string `json:"resource_uri,omitempty"`
	SystemID    string `json:"system_id,omitempty"`
	Hostname    string `json:"Hostname,omitempty"`
	FQDN        string `json:"FQDN,omitempty"`
	Tags        []string
	// OwnerData returns a copy of the key/value data stored for this
	// object.
	OwnerData       map[string]string `json:"owner_data,omitempty"`
	OperatingSystem string            `json:"osystem,omitempty"`
	DistroSeries    string            `json:"distro_series,omitempty"`
	Architecture    string            `json:"Architecture,omitempty"`
	Memory          int               `json:"Memory,omitempty"`
	CPUCount        int               `json:"cpu_count,omitempty"`
	IPAddresses     []string          `json:"ip_addresses,omitempty"`
	PowerState      string            `json:"power_state,omitempty"`
	// NOTE: consider some form of status struct
	StatusName    string `json:"status_name,omitempty"`
	StatusMessage string `json:"status_message,omitempty"`
	// BootInterface returns the interface that was used to boot the MachineInterface.
	BootInterface *MachineNetworkInterface `json:"boot_interface,omitempty"`
	// InterfaceSet returns all the interfaces for the MachineInterface.
	InterfaceSet []*MachineNetworkInterface `json:"interface_set,omitempty"`
	Zone         *zone                      `json:"Zone,omitempty"`
	// Don't really know the difference between these two lists:

	// PhysicalBlockDevice returns the physical block device for the MachineInterface
	// that matches the ID specified. If there is no match, nil is returned.
	PhysicalBlockDevices []*BlockDevice `json:"physicalblockdevice_set,omitempty"`
	// BlockDevices returns all the physical and virtual block devices on the MachineInterface.
	BlockDevices []*BlockDevice `json:"blockdevice_set,omitempty"`
}

func (m *Machine) updateFrom(other *Machine) {
	m.ResourceURI = other.ResourceURI
	m.SystemID = other.SystemID
	m.Hostname = other.Hostname
	m.FQDN = other.FQDN
	m.OperatingSystem = other.OperatingSystem
	m.DistroSeries = other.DistroSeries
	m.Architecture = other.Architecture
	m.Memory = other.Memory
	m.CPUCount = other.CPUCount
	m.IPAddresses = other.IPAddresses
	m.PowerState = other.PowerState
	m.StatusName = other.StatusName
	m.StatusMessage = other.StatusMessage
	m.Zone = other.Zone
	m.Tags = other.Tags
	m.OwnerData = other.OwnerData
}

// CreatemachineDeviceArgs is an argument structure for Machine.CreateDevice.
// Only InterfaceName and MACAddress fields are required, the others are only
// used if set. If Subnet and VLAN are both set, Subnet.VLAN() must match the
// given VLAN. On failure, returns an error satisfying errors.IsNotValid().
type CreateMachineDeviceArgs struct {
	Hostname      string
	InterfaceName string
	MACAddress    string
	Subnet        *subnet
	VLAN          *vlan
}

// MachineNetworkInterface implements Machine.
func (m *Machine) Interface(id int) *MachineNetworkInterface {
	for _, iface := range m.InterfaceSet {
		if iface.ID == id {
			iface.Controller = m.Controller
			return iface
		}
	}
	return nil
}

// PhysicalBlockDevice implements Machine.
func (m *Machine) PhysicalBlockDevice(id int) *BlockDevice {
	for _, blockDevice := range m.PhysicalBlockDevices {
		if blockDevice.ID == id {
			return blockDevice
		}
	}
	return nil
}

// BlockDevice implements Machine.
func (m *Machine) BlockDevice(id int) *BlockDevice {
	for _, blockDevice := range m.BlockDevices {
		if blockDevice.ID == id {
			return blockDevice
		}
	}
	return nil
}

// Devices implements Machine.
func (m *Machine) Devices(args DevicesArgs) ([]device, error) {
	// Perhaps in the future, MAAS will give us a way to query just for the
	// devices for a particular Parent.
	devices, err := m.Controller.Devices(args)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []device
	for _, d := range devices {
		if d.Parent == m.SystemID {
			result = append(result, d)
		}
	}
	return result, nil
}

// StartArgs is an argument struct for passing parameters to the Machine.Start
// method.
type StartArgs struct {
	// UserData needs to be Base64 encoded user data for cloud-init.
	UserData     string
	DistroSeries string
	Kernel       string
	Comment      string
}

// Start implements Machine.
func (m *Machine) Start(args StartArgs) error {
	params := NewURLParams()
	params.MaybeAdd("user_data", args.UserData)
	params.MaybeAdd("distro_series", args.DistroSeries)
	params.MaybeAdd("hwe_kernel", args.Kernel)
	params.MaybeAdd("comment", args.Comment)
	result, err := m.Controller.post(m.ResourceURI, "deploy", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound, http.StatusConflict:
				return errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			case http.StatusServiceUnavailable:
				return errors.Wrap(err, NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}

	var machine *Machine
	err = json.Unmarshal(result, &machine)
	if err != nil {
		return errors.Trace(err)
	}

	m.updateFrom(machine)
	return nil
}

// Validate ensures that all required Values are non-emtpy.
func (a *CreateMachineDeviceArgs) Validate() error {
	if a.InterfaceName == "" {
		return errors.NotValidf("missing InterfaceName")
	}

	if a.MACAddress == "" {
		return errors.NotValidf("missing MACAddress")
	}

	if a.Subnet != nil && a.VLAN != nil && a.Subnet.VLAN != a.VLAN {
		msg := fmt.Sprintf(
			"given Subnet %q on VLAN %d does not match given VLAN %d",
			a.Subnet.CIDR, a.Subnet.VLAN.ID, a.VLAN.ID,
		)
		return errors.NewNotValid(nil, msg)
	}

	return nil
}

// CreateDevice implements Machine
func (m *Machine) CreateDevice(args CreateMachineDeviceArgs) (*device, error) {
	if err := args.Validate(); err != nil {
		return nil, errors.Trace(err)
	}
	d, err := m.Controller.CreateDevice(CreateDeviceArgs{
		Hostname:     args.Hostname,
		MACAddresses: []string{args.MACAddress},
		Parent:       m.SystemID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	defer func(err *error) {
		// If there is an error return, at least try to delete the device we just created.
		if *err != nil {
			if innerErr := d.Delete(); innerErr != nil {
				logger.Warningf("could not delete device %q", d.SystemID)
			}
		}
	}(&err)

	// Update the VLAN to use for the interface, if given.
	vlanToUse := args.VLAN
	if vlanToUse == nil && args.Subnet != nil {
		vlanToUse = args.Subnet.VLAN
	}

	// There should be one interface created for each MAC Address, and since we
	// only specified one, there should just be one response.
	interfaces := d.InterfaceSet
	if count := len(interfaces); count != 1 {
		err := errors.Errorf("unexpected interface count for device: %d", count)
		return nil, NewUnexpectedError(err)
	}
	iface := interfaces[0]
	nameToUse := args.InterfaceName

	if err := m.updateDeviceInterface(*iface, nameToUse, vlanToUse); err != nil {
		return nil, errors.Trace(err)
	}

	if args.Subnet == nil {
		// Nothing further to update.
		return d, nil
	}

	if err := m.linkDeviceInterfaceToSubnet(*iface, args.Subnet); err != nil {
		return nil, errors.Trace(err)
	}

	return d, nil
}

func (m *Machine) updateDeviceInterface(iface MachineNetworkInterface, nameToUse string, vlanToUse *vlan) error {
	updateArgs := UpdateInterfaceArgs{}
	updateArgs.Name = nameToUse

	if vlanToUse != nil {
		updateArgs.VLAN = vlanToUse
	}

	if err := iface.Update(updateArgs); err != nil {
		return errors.Annotatef(err, "updating device interface %q failed", iface.Name)
	}

	return nil
}

func (m *Machine) linkDeviceInterfaceToSubnet(iface MachineNetworkInterface, subnetToUse *subnet) error {
	err := iface.LinkSubnet(LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: subnetToUse,
	})
	if err != nil {
		return errors.Annotatef(
			err, "linking device interface %q to Subnet %q failed",
			iface.Name, subnetToUse.CIDR)
	}

	return nil
}

// SetOwnerData updates the key/value data stored for this object
// with the Values passed in. Existing keys that aren't specified
// in the map passed in will be left in place; to clear a key set
// its value to "". All Owner data is cleared when the object is
// released.
func (m *Machine) SetOwnerData(ownerData map[string]string) error {
	params := make(url.Values)
	for key, value := range ownerData {
		params.Add(key, value)
	}
	result, err := m.Controller.post(m.ResourceURI, "set_owner_data", params)
	if err != nil {
		return errors.Trace(err)
	}

	var machine *Machine
	err = json.Unmarshal(result, &machine)
	if err != nil {
		return errors.Trace(err)
	}

	m.updateFrom(machine)
	return nil
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
	BlockDevice(id int) BlockDevice

	// Start the MachineInterface and install the operating system specified in the args.
	Start(StartArgs) error

	// CreateDevice creates a new DeviceInterface with this MachineInterface as the Parent.
	// The device will have one interface that is linked to the specified Subnet.
	CreateDevice(CreateMachineDeviceArgs) (DeviceInterface, error)
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
