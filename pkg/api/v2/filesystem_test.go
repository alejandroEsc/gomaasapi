// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse2_0(t *testing.T) {
	source := map[string]interface{}{
		"Type":        "ext4",
		"mount_point": "/",
		"Label":       "root",
		"UUID":        "fake-UUID",
	}

	j, err := json.Marshal(source)
	assert.Nil(t, err)

	var fs filesystem
	err = json.Unmarshal(j, &fs)

	assert.Nil(t, err)
	assert.Equal(t, fs.Type, "ext4")
	assert.Equal(t, fs.MountPoint, "/")
	assert.Equal(t, fs.Label, "root")
	assert.Equal(t, fs.UUID, "fake-UUID")
}

func TestParse2_Defaults(t *testing.T) {
	source := map[string]interface{}{
		"Type":        "ext4",
		"mount_point": nil,
		"Label":       nil,
		"UUID":        "fake-UUID",
	}
	j, err := json.Marshal(source)
	assert.Nil(t, err)

	var fs filesystem
	err = json.Unmarshal(j, &fs)

	assert.Nil(t, err)
	assert.Equal(t, fs.Type, "ext4")
	assert.Equal(t, fs.MountPoint, "")
	assert.Equal(t, fs.Label, "")
	assert.Equal(t, fs.UUID, "fake-UUID")
}
