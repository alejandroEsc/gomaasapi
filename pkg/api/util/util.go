// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package util

import (
	"strings"
	"encoding/json"
	"testing"
	"github.com/stretchr/testify/assert"
)

// JoinURLs joins a base URL and a subpath together.
// Regardless of whether baseURL ends in a trailing slash (or even multiple
// trailing slashes), or whether there are any leading slashes at the begining
// of Path, the two will always be joined together by a single slash.
func JoinURLs(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

// EnsureTrailingSlash appends a slash at the end of the given string unless
// there already is one.
// This is used to create the kind of normalized URLs that Django expects.
// (to avoid Django's redirection when an URL does not ends with a slash.)
func EnsureTrailingSlash(URL string) string {
	if strings.HasSuffix(URL, "/") {
		return URL
	}
	return URL + "/"
}


// ParseJSON parses a string source a map to be used in testing
func ParseJSON(t *testing.T, source string) interface{} {
	var parsed interface{}
	err := json.Unmarshal([]byte(source), &parsed)
	assert.Nil(t, err)
	return parsed
}

// UpdateJSONMap updates a json with changes
func UpdateJSONMap(t *testing.T, source string, changes map[string]interface{}) string {
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(source), &parsed)
	assert.Nil(t, err)
	for key, value := range changes {
		parsed[key] = value
	}
	bytes, err := json.Marshal(parsed)
	assert.Nil(t, err)
	return string(bytes)
}
