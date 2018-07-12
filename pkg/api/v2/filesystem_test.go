// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"github.com/juju/gomaasapi/pkg/api/util"
	"testing"
)

func  TestParse2_0(t *testing.T) {
	source := map[string]interface{}{
		"Type":        "ext4",
		"mount_point": "/",
		"Label":       "root",
		"UUID":        "fake-UUID",
	}

	j, err := json.Marshal(source)
	c.Assert(err, jc.ErrorIsNil)

	var fs filesystem
	err = json.Unmarshal(j, &fs)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(fs.Type, gc.Equals, "ext4")
	c.Check(fs.MountPoint, gc.Equals, "/")
	c.Check(fs.Label, gc.Equals, "root")
	c.Check(fs.UUID, gc.Equals, "fake-UUID")
}

func  TestParse2_Defaults(t *testing.T) {
	source := map[string]interface{}{
		"Type":        "ext4",
		"mount_point": nil,
		"Label":       nil,
		"UUID":        "fake-UUID",
	}
	j, err := json.Marshal(source)
	c.Assert(err, jc.ErrorIsNil)

	var fs filesystem
	err = json.Unmarshal(j, &fs)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(fs.Type, gc.Equals, "ext4")
	c.Check(fs.MountPoint, gc.Equals, "")
	c.Check(fs.Label, gc.Equals, "")
	c.Check(fs.UUID, gc.Equals, "fake-UUID")
}

func  TestParse2_0BadSchema(t *testing.T) {
	source := map[string]interface{}{
		"mount_point": "/",
		"Label":       "root",
		"UUID":        "fake-UUID",
	}
	j, err := json.Marshal(source)
	c.Assert(err, jc.ErrorIsNil)

	var fs filesystem
	err = json.Unmarshal(j, &fs)

	c.Assert(err, jc.Satisfies, util.IsDeserializationError)
}
