package maasapiv2

import (
	"fmt"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/gomaasapi/pkg/api/util"
	"github.com/juju/utils/set"
)

// MachinesArgs is a argument struct for selecting Machines.
// Only machines that match the specified criteria are returned.
type MachinesArgs struct {
	Hostnames    []string
	MACAddresses []string
	SystemIDs    []string
	Domain       string
	Zone         string
	AgentName    string
	OwnerData    map[string]string
}

// ReleaseMachinesArgs is an argument struct for passing the MachineInterface system IDs
// and an optional comment into the ReleaseMachines method.
type ReleaseMachinesArgs struct {
	SystemIDs []string
	Comment   string
}

// AllocateMachineArgs is an argument struct for passing args into MachineInterface.Allocate.
type AllocateMachineArgs struct {
	Hostname     string
	SystemId     string
	Architecture string
	MinCPUCount  int
	// MinMemory represented in MB.
	MinMemory int
	Tags      []string
	NotTags   []string
	Zone      string
	NotInZone []string
	// Storage represents the required disks on the MachineInterface. If any are specified
	// the first value is used for the root disk.
	Storage []StorageSpec
	// Interfaces represents a number of required interfaces on the MachineInterface.
	// Each InterfaceSpec relates to an individual network interface.
	Interfaces []InterfaceSpec
	// NotSpace is a MachineInterface level constraint, and applies to the entire MachineInterface
	// rather than specific interfaces.
	NotSpace  []string
	AgentName string
	Comment   string
	DryRun    bool
}

// DeployMachineArgs is an argument struct for passing parameters to the Machine.Deploy
// method.
type DeployMachineArgs struct {
	// UserData needs to be Base64 encoded user data for cloud-init.
	UserData     string
	DistroSeries string
	Kernel       string
	Comment      string
}



// CreatemachineDeviceArgs is an argument structure for Machine.CreateNode.
// Only InterfaceName and MACAddress fields are required, the others are only
// used if set. If Subnet and VLAN are both set, Subnet.VLAN() must match the
// given VLAN. On failure, returns an error satisfying errors.IsNotValid().
type CreateMachineNodeArgs struct {
	Hostname      string
	InterfaceName string
	MACAddress    string
	Subnet        *subnet
	VLAN          *vlan
}

// Validate ensures that all required Values are non-emtpy.
func (a *CreateMachineNodeArgs) Validate() error {
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

// Validate makes sure that any labels specifed in Storage or Interfaces
// are unique, and that the required specifications are valid.
func (a *AllocateMachineArgs) Validate() error {
	storageLabels := set.NewStrings()
	for _, spec := range a.Storage {
		if err := spec.Validate(); err != nil {
			return errors.Annotate(err, "Storage")
		}
		if spec.Label != "" {
			if storageLabels.Contains(spec.Label) {
				return errors.NotValidf("reusing storage Label %q", spec.Label)
			}
			storageLabels.Add(spec.Label)
		}
	}
	interfaceLabels := set.NewStrings()
	for _, spec := range a.Interfaces {
		if err := spec.Validate(); err != nil {
			return errors.Annotate(err, "Interfaces")
		}
		if interfaceLabels.Contains(spec.Label) {
			return errors.NotValidf("reusing interface Label %q", spec.Label)
		}
		interfaceLabels.Add(spec.Label)
	}
	for _, v := range a.NotSpace {
		if v == "" {
			return errors.NotValidf("empty NotSpace constraint")
		}
	}
	return nil
}

func (a *AllocateMachineArgs) storage() string {
	var values []string
	for _, spec := range a.Storage {
		values = append(values, spec.String())
	}
	return strings.Join(values, ",")
}

func (a *AllocateMachineArgs) interfaces() string {
	var values []string
	for _, spec := range a.Interfaces {
		values = append(values, spec.String())
	}
	return strings.Join(values, ";")
}

func (a *AllocateMachineArgs) notSubnets() []string {
	var values []string
	for _, v := range a.NotSpace {
		values = append(values, "space:"+v)
	}
	return values
}

func machinesParams(args MachinesArgs) *util.URLParams {
	params := util.NewURLParams()
	params.MaybeAddMany("Hostname", args.Hostnames)
	params.MaybeAddMany("mac_address", args.MACAddresses)
	params.MaybeAddMany("ID", args.SystemIDs)
	params.MaybeAdd("domain", args.Domain)
	params.MaybeAdd("Zone", args.Zone)
	params.MaybeAdd("agent_name", args.AgentName)
	return params
}

func allocateMachinesParams(args AllocateMachineArgs) *util.URLParams {
	params := util.NewURLParams()
	params.MaybeAdd("Name", args.Hostname)
	params.MaybeAdd("system_id", args.SystemId)
	params.MaybeAdd("arch", args.Architecture)
	params.MaybeAddInt("cpu_count", args.MinCPUCount)
	params.MaybeAddInt("mem", args.MinMemory)
	params.MaybeAddMany("Tags", args.Tags)
	params.MaybeAddMany("not_tags", args.NotTags)
	params.MaybeAdd("storage", args.storage())
	params.MaybeAdd("interfaces", args.interfaces())
	params.MaybeAddMany("not_subnets", args.notSubnets())
	params.MaybeAdd("Zone", args.Zone)
	params.MaybeAddMany("not_in_zone", args.NotInZone)
	params.MaybeAdd("agent_name", args.AgentName)
	params.MaybeAdd("comment", args.Comment)
	params.MaybeAddBool("dry_run", args.DryRun)
	return params
}

func releaseMachinesParams(args ReleaseMachinesArgs) *util.URLParams {
	params := util.NewURLParams()
	params.MaybeAddMany("machines", args.SystemIDs)
	params.MaybeAdd("comment", args.Comment)
	return params
}

func startMachineParams(args DeployMachineArgs) *util.URLParams {
	params := util.NewURLParams()
	params.MaybeAdd("user_data", args.UserData)
	params.MaybeAdd("distro_series", args.DistroSeries)
	params.MaybeAdd("hwe_kernel", args.Kernel)
	params.MaybeAdd("comment", args.Comment)
	return params
}
