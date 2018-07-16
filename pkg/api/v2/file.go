// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/base64"
	"net/http"
	"net/url"

	"encoding/json"
	"fmt"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/util"
	. "github.com/juju/gomaasapi/pkg/api/v2/controller"
)

type File struct {
	Controller  *Controller `json:"-"`
	ResourceURI string      `json:"resource_uri,string,omitempty"`
	// Filename is the Name of the File. No Path, just the Filename.
	Filename string `json:"Filename,string,omitempty"`
	// AnonymousURI is a URL that can be used to retrieve the contents of the
	// File without credentials.
	AnonymousURI *url.URL `json:"anon_resource_uri,string,omitempty"`
	Content      string   `json:"Content,string,omitempty"`
}

// Delete implements FileInterface.
func (f *File) Delete() error {
	err := f.Controller.Delete(f.ResourceURI)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound:
				return errors.Wrap(err, util.NewNoMatchError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, util.NewPermissionError(svrErr.BodyMessage))
			}
		}
		return util.NewUnexpectedError(err)
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
		//return nil, util.NewUnexpectedError(fmt.Errorf("content is %s.",f.Content))
		return nil, fmt.Errorf("err: %s, content is %s.", err.Error(), f.Content)
	}
	return bytes, nil
}

func (f *File) get(path, op string, params url.Values) ([]byte, error) {
	return f.Controller.Get(path, op, params)
}

func (f *File) readFromServer() ([]byte, error) {
	// If the Content is available, it is base64 encoded, so
	args := make(url.Values)
	args.Add("Filename", f.Filename)
	bytes, err := f.get("files", "Get", args)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound:
				return nil, errors.Wrap(err, util.NewNoMatchError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return nil, errors.Wrap(err, util.NewPermissionError(svrErr.BodyMessage))
			}
		}
		return nil, err
		//return nil, util.NewUnexpectedError(err)
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

func (f *File) UnmarshalJSON(j []byte) error {
	var rawStrings map[string]string

	err := json.Unmarshal(j, &rawStrings)
	if err != nil {
		return err
	}

	for k, v := range rawStrings {
		switch strings.ToLower(k) {
		case "resource_uri":
			f.ResourceURI = v
		case "filename":
			f.Filename = v
		case "anon_resource_uri":
			u, err := url.Parse(v)
			if err != nil {
				return err
			}
			f.AnonymousURI = u
		case "content":
			f.Content = v
		}
	}

	return nil
}
