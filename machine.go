// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

// Machine represents a physical Machine.
type machine struct {
	Controller *controller

	ResourceURI string

	SystemID  string
	Hostname  string
	FQDN      string
	Tags      []string
	OwnerData map[string]string

	OperatingSystem string
	DistroSeries    string
	Architecture    string
	Memory          int
	CPUCount        int

	IPAddresses []string
	PowerState  string

	// NOTE: consider some form of status struct
	StatusName    string
	StatusMessage string

	// BootInterface returns the interface that was used to boot the Machine.
	BootInterface *MachineNetworkInterface
	// InterfaceSet returns all the interfaces for the Machine.
	InterfaceSet []*MachineNetworkInterface
	Zone         *zone

	// Don't really know the difference between these two lists:

	// PhysicalBlockDevice returns the physical block device for the Machine
	// that matches the ID specified. If there is no match, nil is returned.
	PhysicalBlockDevices []*blockdevice
	// BlockDevices returns all the physical and virtual block devices on the Machine.
	BlockDevices []*blockdevice
}

func (m *machine) updateFrom(other *machine) {
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

// CreatemachineDeviceArgs is an argument structure for machine.CreateDevice.
// Only InterfaceName and MACAddress fields are required, the others are only
// used if set. If Subnet and VLAN are both set, Subnet.VLAN() must match the
// given VLAN. On failure, returns an error satisfying errors.IsNotValid().
type CreateMachineDeviceArgs struct {
	Hostname      string
	InterfaceName string
	MACAddress    string
	Subnet        subnet
	VLAN          vlan
}

// MachineNetworkInterface implements machine.
func (m *machine) Interface(id int) MachineNetworkInterface {
	for _, iface := range m.InterfaceSet {
		if iface.ID == id {
			iface.Controller = m.Controller
			return iface
		}
	}
	return nil
}

// PhysicalBlockDevice implements machine.
func (m *machine) PhysicalBlockDevice(id int) *blockdevice {
	for _, blockDevice := range m.PhysicalBlockDevices {
		if blockDevice.ID == id {
			return blockDevice
		}
	}
	return nil
}

// BlockDevice implements machine.
func (m *machine) BlockDevice(id int) *blockdevice {
	for _, blockDevice := range m.BlockDevices {
		if blockDevice.ID == id {
			return blockDevice
		}
	}
	return nil
}

// Devices implements machine.
func (m *machine) Devices(args DevicesArgs) ([]Device, error) {
	// Perhaps in the future, MAAS will give us a way to query just for the
	// devices for a particular Parent.
	devices, err := m.Controller.Devices(args)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []Device
	for _, device := range devices {
		if device.Parent == m.SystemID {
			result = append(result, device)
		}
	}
	return result, nil
}

// StartArgs is an argument struct for passing parameters to the machine.Start
// method.
type StartArgs struct {
	// UserData needs to be Base64 encoded user data for cloud-init.
	UserData     string
	DistroSeries string
	Kernel       string
	Comment      string
}

// Start implements machine.
func (m *machine) Start(args StartArgs) error {
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

	machine, err := readmachine(m.Controller.APIVersion, result)
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
			a.Subnet.CIDR, a.Subnet.VLAN.ID, a.VLAN.ID(),
		)
		return errors.NewNotValid(nil, msg)
	}

	return nil
}

// CreateDevice implements machine
func (m *machine) CreateDevice(args CreateMachineDeviceArgs) (_ Device, err error) {
	if err := args.Validate(); err != nil {
		return nil, errors.Trace(err)
	}
	device, err := m.Controller.CreateDevice(CreateDeviceArgs{
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
			if innerErr := device.Delete(); innerErr != nil {
				logger.Warningf("could not delete device %q", device.SystemID())
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
	interfaces := device.InterfaceSet()
	if count := len(interfaces); count != 1 {
		err := errors.Errorf("unexpected interface count for device: %d", count)
		return nil, NewUnexpectedError(err)
	}
	iface := interfaces[0]
	nameToUse := args.InterfaceName

	if err := m.updateDeviceInterface(iface, nameToUse, vlanToUse); err != nil {
		return nil, errors.Trace(err)
	}

	if args.Subnet == nil {
		// Nothing further to update.
		return device, nil
	}

	if err := m.linkDeviceInterfaceToSubnet(iface, args.Subnet); err != nil {
		return nil, errors.Trace(err)
	}

	return device, nil
}

func (m *machine) updateDeviceInterface(iface MachineNetworkInterface, nameToUse string, vlanToUse VLAN) error {
	updateArgs := UpdateInterfaceArgs{}
	updateArgs.Name = nameToUse

	if vlanToUse != nil {
		updateArgs.VLAN = vlanToUse
	}

	if err := iface.Update(updateArgs); err != nil {
		return errors.Annotatef(err, "updating device interface %q failed", iface.Name())
	}

	return nil
}

func (m *machine) linkDeviceInterfaceToSubnet(iface MachineNetworkInterface, subnetToUse Subnet) error {
	err := iface.LinkSubnet(LinkSubnetArgs{
		Mode:   LinkModeStatic,
		Subnet: subnetToUse,
	})
	if err != nil {
		return errors.Annotatef(
			err, "linking device interface %q to Subnet %q failed",
			iface.Name(), subnetToUse.CIDR())
	}

	return nil
}

// SetOwnerData implements OwnerDataHolder.
func (m *machine) SetOwnerData(ownerData map[string]string) error {
	params := make(url.Values)
	for key, value := range ownerData {
		params.Add(key, value)
	}
	result, err := m.Controller.post(m.ResourceURI, "set_owner_data", params)
	if err != nil {
		return errors.Trace(err)
	}
	machine, err := readmachine(m.Controller.APIVersion, result)
	if err != nil {
		return errors.Trace(err)
	}
	m.updateFrom(machine)
	return nil
}

func readmachine(controllerVersion version.Number, source interface{}) (*machine, error) {
	readFunc, err := getmachineDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, errors.Trace(err)
	}

	checker := schema.StringMap(schema.Any())
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "machine base schema check failed")
	}
	valid := coerced.(map[string]interface{})
	return readFunc(valid)
}

func readmachines(controllerVersion version.Number, source interface{}) ([]*machine, error) {
	readFunc, err := getmachineDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, errors.Trace(err)
	}

	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "machine base schema check failed")
	}
	valid := coerced.([]interface{})
	return readmachineList(valid, readFunc)
}

func getmachineDeserializationFunc(controllerVersion version.Number) (machineDeserializationFunc, error) {
	var deserialisationVersion version.Number
	for v := range machineDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no machine read func for version %s", controllerVersion)
	}
	return machineDeserializationFuncs[deserialisationVersion], nil
}

func readmachineList(sourceList []interface{}, readFunc machineDeserializationFunc) ([]*machine, error) {
	result := make([]*machine, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for machine %d, %T", i, value)
		}
		machine, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "machine %d", i)
		}
		result = append(result, machine)
	}
	return result, nil
}

type machineDeserializationFunc func(map[string]interface{}) (*machine, error)

var machineDeserializationFuncs = map[version.Number]machineDeserializationFunc{
	twoDotOh: machine_2_0,
}

func machine_2_0(source map[string]interface{}) (*machine, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),

		"system_id":  schema.String(),
		"Hostname":   schema.String(),
		"FQDN":       schema.String(),
		"tag_names":  schema.List(schema.String()),
		"owner_data": schema.StringMap(schema.String()),

		"osystem":       schema.String(),
		"distro_series": schema.String(),
		"Architecture":  schema.OneOf(schema.Nil(""), schema.String()),
		"Memory":        schema.ForceInt(),
		"cpu_count":     schema.ForceInt(),

		"ip_addresses":   schema.List(schema.String()),
		"power_state":    schema.String(),
		"status_name":    schema.String(),
		"status_message": schema.OneOf(schema.Nil(""), schema.String()),

		"boot_interface": schema.OneOf(schema.Nil(""), schema.StringMap(schema.Any())),
		"interface_set":  schema.List(schema.StringMap(schema.Any())),
		"Zone":           schema.StringMap(schema.Any()),

		"physicalblockdevice_set": schema.List(schema.StringMap(schema.Any())),
		"blockdevice_set":         schema.List(schema.StringMap(schema.Any())),
	}
	defaults := schema.Defaults{
		"Architecture": "",
	}
	checker := schema.FieldMap(fields, defaults)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "machine 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	var bootInterface *MachineNetworkInterface
	if ifaceMap, ok := valid["boot_interface"].(map[string]interface{}); ok {
		bootInterface, err = interface_2_0(ifaceMap)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	interfaceSet, err := readInterfaceList(valid["interface_set"].([]interface{}), interface_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}
	zone, err := zone_2_0(valid["Zone"].(map[string]interface{}))
	if err != nil {
		return nil, errors.Trace(err)
	}
	physicalBlockDevices, err := readBlockDeviceList(valid["physicalblockdevice_set"].([]interface{}), blockdevice_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}
	blockDevices, err := readBlockDeviceList(valid["blockdevice_set"].([]interface{}), blockdevice_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}
	architecture, _ := valid["Architecture"].(string)
	statusMessage, _ := valid["status_message"].(string)
	result := &machine{
		ResourceURI: valid["resource_uri"].(string),

		SystemID:  valid["system_id"].(string),
		Hostname:  valid["Hostname"].(string),
		FQDN:      valid["FQDN"].(string),
		Tags:      convertToStringSlice(valid["tag_names"]),
		OwnerData: convertToStringMap(valid["owner_data"]),

		OperatingSystem: valid["osystem"].(string),
		DistroSeries:    valid["distro_series"].(string),
		Architecture:    architecture,
		Memory:          valid["Memory"].(int),
		CPUCount:        valid["cpu_count"].(int),

		IPAddresses:   convertToStringSlice(valid["ip_addresses"]),
		PowerState:    valid["power_state"].(string),
		StatusName:    valid["status_name"].(string),
		StatusMessage: statusMessage,

		BootInterface:        bootInterface,
		InterfaceSet:         interfaceSet,
		Zone:                 zone,
		PhysicalBlockDevices: physicalBlockDevices,
		BlockDevices:         blockDevices,
	}

	return result, nil
}

func convertToStringSlice(field interface{}) []string {
	if field == nil {
		return nil
	}
	fieldSlice := field.([]interface{})
	result := make([]string, len(fieldSlice))
	for i, value := range fieldSlice {
		result[i] = value.(string)
	}
	return result
}

func convertToStringMap(field interface{}) map[string]string {
	if field == nil {
		return nil
	}
	// This function is only called after a schema Coerce, so it's
	// safe.
	fieldMap := field.(map[string]interface{})
	result := make(map[string]string)
	for key, value := range fieldMap {
		result[key] = value.(string)
	}
	return result
}
