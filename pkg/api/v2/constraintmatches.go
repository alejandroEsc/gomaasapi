package maasapiv2

// ConstraintMatches provides a way for the caller of AllocateMachine to determine
//.how the allocated MachineInterface matched the storage and interfaces constraints specified.
// The labels that were used in the constraints are the keys in the maps.
type ConstraintMatches struct {
	// NetworkInterface is a mapping of the constraint Label specified to the Interfaces
	// that match that constraint.
	Interfaces map[string][]NetworkInterface

	// Storage is a mapping of the constraint Label specified to the BlockDevices
	// that match that constraint.
	Storage map[string][]BlockDevice
}
