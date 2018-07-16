// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewParamsNonNilValues(t *testing.T) {
	params := NewURLParams()
	assert.NotNil(t, params.Values)
}

func TestNewMaybeAddEmpty(t *testing.T) {
	params := NewURLParams()
	params.MaybeAdd("foo", "")
	assert.Equal(t, params.Values.Encode(), "")
}

func TestNewMaybeAddWithValue(t *testing.T) {
	params := NewURLParams()
	params.MaybeAdd("foo", "bar")
	assert.Equal(t, params.Values.Encode(), "foo=bar")
}

func TestNewMaybeAddIntZero(t *testing.T) {
	params := NewURLParams()
	params.MaybeAddInt("foo", 0)
	assert.Equal(t, params.Values.Encode(), "")
}

func TestNewMaybeAddIntWithValue(t *testing.T) {
	params := NewURLParams()
	params.MaybeAddInt("foo", 42)
	assert.Equal(t, params.Values.Encode(), "foo=42")
}

func TestNewMaybeAddBoolFalse(t *testing.T) {
	params := NewURLParams()
	params.MaybeAddBool("foo", false)
	assert.Equal(t, params.Values.Encode(), "")
}

func TestNewMaybeAddBoolTrue(t *testing.T) {
	params := NewURLParams()
	params.MaybeAddBool("foo", true)
	assert.Equal(t, params.Values.Encode(), "foo=true")
}

func TestNewMaybeAddManyNil(t *testing.T) {
	params := NewURLParams()
	params.MaybeAddMany("foo", nil)
	assert.Equal(t, params.Values.Encode(), "")
}

func TestNewMaybeAddManyValues(t *testing.T) {
	params := NewURLParams()
	params.MaybeAddMany("foo", []string{"two", "", "values"})
	assert.Equal(t, params.Values.Encode(), "foo=two&foo=values")
}
