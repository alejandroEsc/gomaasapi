package maasapiv2



type MachineOp string

const(

	// Comission begins commissioning process for a machine.
	Comission MachineOp = "comission"
	// Deploy an operating system to a machine.
	Deploy MachineOp = "deploy"
	// Details obtains various system details.
	Details MachineOp = "details"
	// GetCurtinConfig returns the rendered curtin configuration for the machine.
	GetCurtinConfig MachineOp = "get_curtin_config"
	// PowerParams obtain power parameters.
	PowerParams MachineOp = "power_parameters"
	// Abort a machine's current operation
	Abort MachineOp = "abort"
	// clear_default_gateways
	ClearDefaultGateways MachineOp = "clear_default_gateways"
	// ExitRescueMode exits rescue mode process for a machine.
	ExitRescueMode MachineOp = "exit_rescue_mode"
	// MarkBroken marks a node as 'broken'.
	MarkBroken MachineOp = "mark_broken"
	// MarkFixed mark a broken node as fixed and set its status as 'ready'.
	MarkFixed MachineOp = "mark_fixed"
	// MountSpecial Mount a special-purpose filesystem, like tmpfs.
	MountSpecial MachineOp = "mount_special"
	// PowerOFF to request Power off a node.
	PowerOFF MachineOp = "power_off"
	// PowerON Turn on a node.
	PowerON MachineOp = "power_on"
	// Release  a machine. Opposite of Machines.allocate.
	Release MachineOp = "release"
	// Begin rescue mode process for a machine.
	RescueMode MachineOp = "rescue_mode"
	// Reset a machine's configuration to its initial state.
	RestoreDefaultConfig MachineOp = "restore_default_configuration"
	// Reset a machine's networking options to its initial state.
	RestoreNetworkConfig MachineOp = "restore_networking_configuration"
	// Reset a machine's storage options to its initial state.
	RestoreStorageConfig MachineOp = "restore_storage_configuration"
	// Set key/value data for the current owner.
	SetOwnerData MachineOp = "set_owner_data"
	// Changes the storage layout on the machine.
	SetStorageLayout MachineOp = "set_storage_layout"
	// Unmount a special-purpose filesystem, like tmpfs.
	UnmountSpecial MachineOp = "unmount_special"



)


type MachinesOp string

const (
	// Allocate an available machine for deployment.
	Allocate MachinesOp = "allocate"

)