// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package util

import (
	"strings"

	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNoMatchError(t *testing.T) {
	err := NewNoMatchError("foo")
	assert.NotNil(t, err)
	assert.True(t, IsNoMatchError(err))
}

func TestUnexpectedError(t *testing.T) {
	err := errors.New("wat")
	err = NewUnexpectedError(err)
	assert.NotNil(t, err)
	assert.True(t, IsUnexpectedError(err))
	assert.Equal(t, err.Error(), "unexpected: wat")
}

func TestUnsupportedVersionError(t *testing.T) {
	err := NewUnsupportedVersionError("foo %d", 42)
	assert.NotNil(t, err)
	assert.True(t, IsUnsupportedVersionError(err))
	assert.Equal(t, err.Error(), "foo 42")
}

func TestWrapWithUnsupportedVersionError(t *testing.T) {
	err := WrapWithUnsupportedVersionError(errors.New("bad"))
	assert.NotNil(t, err)
	assert.True(t, IsUnsupportedVersionError(err))
	assert.Equal(t, err.Error(), "unsupported version: bad")
	stack := errors.ErrorStack(err)
	assert.Len(t, strings.Split(stack, "\n"), 2)
}

func TestDeserializationError(t *testing.T) {
	err := NewDeserializationError("foo %d", 42)
	assert.NotNil(t, err)
	assert.True(t, IsDeserializationError(err))
	assert.Equal(t, err.Error(), "foo 42")
}

func TestWrapWithDeserializationError(t *testing.T) {
	err := errors.New("base error")
	err = WrapWithDeserializationError(err, "foo %d", 42)
	assert.NotNil(t, err)
	assert.True(t, IsDeserializationError(err))
	assert.Equal(t, err.Error(), "foo 42: base error")
	stack := errors.ErrorStack(err)
	assert.Len(t, strings.Split(stack, "\n"), 2)
}

func TestBadRequestError(t *testing.T) {
	err := NewBadRequestError("omg")
	assert.NotNil(t, err)
	assert.True(t, IsBadRequestError(err))
	assert.Equal(t, err.Error(), "omg")
}

func TestPermissionError(t *testing.T) {
	err := NewPermissionError("naughty")
	assert.NotNil(t, err)
	assert.True(t, IsPermissionError(err))
	assert.Equal(t, err.Error(), "naughty")
}

func TestCannotCompleteError(t *testing.T) {
	err := NewCannotCompleteError("server says no")
	assert.NotNil(t, err)
	assert.True(t, IsCannotCompleteError(err))
	assert.Equal(t, err.Error(), "server says no")
}
