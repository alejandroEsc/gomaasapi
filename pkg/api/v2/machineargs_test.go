package maasapiv2

import (
	"testing"

	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
)

func TestAllocateMachineArgs(t *testing.T) {
	for _, test := range []struct {
		args       AllocateMachineArgs
		err        string
		storage    string
		interfaces string
		notSubnets []string
	}{{
		args: AllocateMachineArgs{},
	}, {
		args: AllocateMachineArgs{
			Storage: []StorageSpec{{}},
		},
		err: "Storage: Size value 0 not valid",
	}, {
		args: AllocateMachineArgs{
			Storage: []StorageSpec{{Size: 200}, {Size: 400, Tags: []string{"ssd"}}},
		},
		storage: "200,400(ssd)",
	}, {
		args: AllocateMachineArgs{
			Storage: []StorageSpec{
				{Label: "foo", Size: 200},
				{Label: "foo", Size: 400, Tags: []string{"ssd"}},
			},
		},
		err: `reusing storage Label "foo" not valid`,
	}, {
		args: AllocateMachineArgs{
			Interfaces: []InterfaceSpec{{}},
		},
		err: "Interfaces: missing Label not valid",
	}, {
		args: AllocateMachineArgs{
			Interfaces: []InterfaceSpec{
				{Label: "foo", Space: "magic"},
				{Label: "bar", Space: "other"},
			},
		},
		interfaces: "foo:space=magic;bar:space=other",
	}, {
		args: AllocateMachineArgs{
			Interfaces: []InterfaceSpec{
				{Label: "foo", Space: "magic"},
				{Label: "foo", Space: "other"},
			},
		},
		err: `reusing interface Label "foo" not valid`,
	}, {
		args: AllocateMachineArgs{
			NotSpace: []string{""},
		},
		err: "empty NotSpace constraint not valid",
	}, {
		args: AllocateMachineArgs{
			NotSpace: []string{"foo"},
		},
		notSubnets: []string{"space:foo"},
	}, {
		args: AllocateMachineArgs{
			NotSpace: []string{"foo", "bar"},
		},
		notSubnets: []string{"space:foo", "space:bar"},
	}} {
		err := test.args.Validate()
		if test.err == "" {
			assert.Nil(t, err)
			assert.Equal(t, test.args.storage(), test.storage)
			assert.Equal(t, test.interfaces, test.args.interfaces())
			assert.EqualValues(t, test.args.notSubnets(), test.notSubnets)
		} else {
			assert.True(t, errors.IsNotValid(err))
			assert.Equal(t, err.Error(), test.err)
		}
	}
}
