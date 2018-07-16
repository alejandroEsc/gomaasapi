package maasapiv2

import "github.com/juju/errors"

const (
	// LinkModeDHCP - Bring the interface up with DHCP on the given Subnet. Only
	// one Subnet can be set to DHCP. If the Subnet is managed this interface
	// will pull from the dynamic IP range.
	LinkModeDHCP InterfaceLinkMode = "DHCP"

	// LinkModeStatic - Bring the interface up with a STATIC IP address on the
	// given Subnet. Any number of STATIC Links can exist on an interface.
	LinkModeStatic InterfaceLinkMode = "STATIC"

	// LinkModeLinkUp - Bring the interface up only on the given Subnet. No IP
	// address will be assigned to this interface. The interface cannot have any
	// current DHCP or STATIC Links.
	LinkModeLinkUp InterfaceLinkMode = "LINK_UP"
)

// LinkSubnetArgs is an argument struct for passing parameters to
// the MachineNetworkInterface.LinkSubnet method.
type LinkSubnetArgs struct {
	// Mode is used to describe how the address is provided for the Link.
	// Required field.
	Mode InterfaceLinkMode
	// Subnet is the Subnet to link to. Required field.
	Subnet *subnet
	// IPAddress is only valid when the Mode is set to LinkModeStatic. If
	// not specified with a Mode of LinkModeStatic, an IP address from the
	// Subnet will be auto selected.
	IPAddress string
	// DefaultGateway will set the gateway IP address for the Subnet as the
	// default gateway for the Machine or node the interface belongs to.
	// Option can only be used with Mode LinkModeStatic.
	DefaultGateway bool
}

// Validate ensures that the Mode and Subnet are set, and that the other options
// are consistent with the Mode.
func (a *LinkSubnetArgs) Validate() error {
	switch a.Mode {
	case LinkModeDHCP, LinkModeLinkUp, LinkModeStatic:
	case "":
		return errors.NotValidf("missing Mode")
	default:
		return errors.NotValidf("unknown Mode value (%q)", a.Mode)
	}
	if a.Subnet == nil {
		return errors.NotValidf("missing Subnet")
	}
	if a.IPAddress != "" && a.Mode != LinkModeStatic {
		return errors.NotValidf("setting IP Address when Mode is not LinkModeStatic")
	}
	if a.DefaultGateway && a.Mode != LinkModeStatic {
		return errors.NotValidf("specifying DefaultGateway for Mode %q", a.Mode)
	}
	return nil
}