package maasapiv2

import (
	"testing"

	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
)

func TestInterfaceSpec(t *testing.T) {
	for _, test := range []struct {
		spec InterfaceSpec
		err  string
		repr string
	}{{
		spec: InterfaceSpec{},
		err:  "missing Label not valid",
	}, {
		spec: InterfaceSpec{Label: "foo"},
		err:  "empty Space constraint not valid",
	}, {
		spec: InterfaceSpec{Label: "foo", Space: "magic"},
		repr: "foo:space=magic",
	}} {
		err := test.spec.Validate()
		if test.err == "" {
			assert.Nil(t, err)
			assert.Equal(t, test.spec.String(), test.repr)
		} else {
			assert.True(t, errors.IsNotValid(err))
			assert.Equal(t, err.Error(), test.err)
		}
	}
}
