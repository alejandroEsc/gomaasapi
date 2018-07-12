// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	"encoding/json"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

func parseJSON(c *gc.C, source string) interface{} {
	var parsed interface{}
	err := json.Unmarshal([]byte(source), &parsed)
	c.Assert(err, jc.ErrorIsNil)
	return parsed
}

func updateJSONMap(c *gc.C, source string, changes map[string]interface{}) string {
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(source), &parsed)
	c.Assert(err, jc.ErrorIsNil)
	for key, value := range changes {
		parsed[key] = value
	}
	bytes, err := json.Marshal(parsed)
	c.Assert(err, jc.ErrorIsNil)
	return string(bytes)
}
