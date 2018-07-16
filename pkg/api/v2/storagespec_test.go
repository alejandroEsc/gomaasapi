package maasapiv2

import (
	"testing"

	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
)

func TestStorageSpec(t *testing.T) {
	for _, test := range []struct {
		spec StorageSpec
		err  string
		repr string
	}{{
		spec: StorageSpec{},
		err:  "Size value 0 not valid",
	}, {
		spec: StorageSpec{Size: -10},
		err:  "Size value -10 not valid",
	}, {
		spec: StorageSpec{Size: 200},
		repr: "200",
	}, {
		spec: StorageSpec{Label: "foo", Size: 200},
		repr: "foo:200",
	}, {
		spec: StorageSpec{Size: 200, Tags: []string{"foo", ""}},
		err:  "empty tag not valid",
	}, {
		spec: StorageSpec{Size: 200, Tags: []string{"foo"}},
		repr: "200(foo)",
	}, {
		spec: StorageSpec{Label: "omg", Size: 200, Tags: []string{"foo", "bar"}},
		repr: "omg:200(foo,bar)",
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
