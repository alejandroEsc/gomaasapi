// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	"encoding/base64"
	"net/http"
	"net/url"

	"github.com/juju/errors"
)

type File struct {
	Controller  *controller
	ResourceURI string `json:"resource_uri,omitempty"`
	// Filename is the Name of the File. No Path, just the Filename.
	Filename string `json:"Filename,omitempty"`
	// AnonymousURL is a URL that can be used to retrieve the conents of the
	// File without credentials.
	AnonymousURI *url.URL `json:"anon_resource_uri,omitempty"`
	Content      string   `json:"Content,omitempty"`
}

// Delete implements FileInterface.
func (f *File) Delete() error {
	err := f.Controller.delete(f.ResourceURI)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound:
				return errors.Wrap(err, NewNoMatchError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}
	return nil
}

// ReadAll implements FileInterface.
func (f *File) ReadAll() ([]byte, error) {
	if f.Content == "" {
		return f.readFromServer()
	}
	bytes, err := base64.StdEncoding.DecodeString(f.Content)
	if err != nil {
		return nil, NewUnexpectedError(err)
	}
	return bytes, nil
}

func (f *File) readFromServer() ([]byte, error) {
	// If the Content is available, it is base64 encoded, so
	args := make(url.Values)
	args.Add("Filename", f.Filename)
	bytes, err := f.Controller._getRaw("files", "get", args)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound:
				return nil, errors.Wrap(err, NewNoMatchError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return nil, errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			}
		}
		return nil, NewUnexpectedError(err)
	}
	return bytes, nil
}

// FileInterface represents a File stored in the MAAS ControllerInterface.
type FileInterface interface {
	// Delete removes the File from the MAAS ControllerInterface.
	Delete() error
	// ReadAll returns the Content of the File.
	ReadAll() ([]byte, error)
}
