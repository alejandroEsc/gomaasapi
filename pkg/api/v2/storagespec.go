package maasapiv2

import (
	"fmt"
	"strings"

	"github.com/juju/errors"
)

// StorageSpec represents one element of storage constraints necessary
// to be satisfied to allocate a MachineInterface.
type StorageSpec struct {
	// Label is optional and an arbitrary string. Labels need to be unique
	// across the StorageSpec elements specified in the AllocateMachineArgs.
	Label string
	// Size is required and refers to the required minimum Size in GB.
	Size int
	// Zero or more Tags assocated to with the disks.
	Tags []string
}

// Validate ensures that there is a positive Size and that there are no Empty
// tag Values.
func (s *StorageSpec) Validate() error {
	if s.Size <= 0 {
		return errors.NotValidf("Size value %d", s.Size)
	}
	for _, v := range s.Tags {
		if v == "" {
			return errors.NotValidf("empty tag")
		}
	}
	return nil
}

// String returns the string representation of the storage spec.
func (s *StorageSpec) String() string {
	label := s.Label
	if label != "" {
		label += ":"
	}
	tags := strings.Join(s.Tags, ",")
	if tags != "" {
		tags = "(" + tags + ")"
	}
	return fmt.Sprintf("%s%d%s", label, s.Size, tags)
}
