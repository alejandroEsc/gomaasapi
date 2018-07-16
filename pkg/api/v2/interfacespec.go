package maasapiv2

import (
	"github.com/juju/errors"
	"fmt"
)

// InterfaceSpec represents one element of network related constraints.
type InterfaceSpec struct {
	// Label is required and an arbitrary string. Labels need to be unique
	// across the InterfaceSpec elements specified in the AllocateMachineArgs.
	// The Label is returned in the ConstraintMatches response from
	// AllocateMachine.
	Label string
	Space string

	// NOTE: there are other interface spec Values that we are not exposing at
	// this stage that can be added on an as needed basis. Other possible Values are:
	//     'fabric_class', 'not_fabric_class',
	//     'subnet_cidr', 'not_subnet_cidr',
	//     'VID', 'not_vid',
	//     'Fabric', 'not_fabric',
	//     'Subnet', 'not_subnet',
	//     'Mode'
}

// Validate ensures that a Label is specified and that there is at least one
// Space or NotSpace value set.
func (a *InterfaceSpec) Validate() error {
	if a.Label == "" {
		return errors.NotValidf("missing Label")
	}
	// Perhaps at some stage in the future there will be other possible specs
	// supported (like VID, Subnet, etc), but until then, just space to check.
	if a.Space == "" {
		return errors.NotValidf("empty Space constraint")
	}
	return nil
}
// String returns the interface spec as MaaS requires it.
func (a *InterfaceSpec) String() string {
	return fmt.Sprintf("%s:space=%s", a.Label, a.Space)
}