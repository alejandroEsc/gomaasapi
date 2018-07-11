// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"encoding/base64"
	"net/http"
	"net/url"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

type file struct {
	Controller *controller

	ResourceURI string `json:"resource_uri,omitempty"`
	// Filename is the Name of the file. No Path, just the Filename.
	Filename string
	// AnonymousURL is a URL that can be used to retrieve the conents of the
	// file without credentials.
	AnonymousURI *url.URL
	Content      string
}

// Delete implements File.
func (f *file) Delete() error {
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

// ReadAll implements File.
func (f *file) ReadAll() ([]byte, error) {
	if f.Content == "" {
		return f.readFromServer()
	}
	bytes, err := base64.StdEncoding.DecodeString(f.Content)
	if err != nil {
		return nil, NewUnexpectedError(err)
	}
	return bytes, nil
}

func (f *file) readFromServer() ([]byte, error) {
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

func readFiles(controllerVersion version.Number, source interface{}) ([]*file, error) {
	readFunc, err := getFileDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, errors.Trace(err)
	}

	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "file base schema check failed")
	}
	valid := coerced.([]interface{})
	return readFileList(valid, readFunc)
}

func readFile(controllerVersion version.Number, source interface{}) (*file, error) {
	readFunc, err := getFileDeserializationFunc(controllerVersion)
	if err != nil {
		return nil, errors.Trace(err)
	}

	checker := schema.StringMap(schema.Any())
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "file base schema check failed")
	}
	valid := coerced.(map[string]interface{})
	return readFunc(valid)
}

func getFileDeserializationFunc(controllerVersion version.Number) (fileDeserializationFunc, error) {
	var deserialisationVersion version.Number
	for v := range fileDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, NewUnsupportedVersionError("no file read func for version %s", controllerVersion)
	}
	return fileDeserializationFuncs[deserialisationVersion], nil
}

// readFileList expects the Values of the sourceList to be string maps.
func readFileList(sourceList []interface{}, readFunc fileDeserializationFunc) ([]*file, error) {
	result := make([]*file, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, NewDeserializationError("unexpected value for file %d, %T", i, value)
		}
		file, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "file %d", i)
		}
		result = append(result, file)
	}
	return result, nil
}

type fileDeserializationFunc func(map[string]interface{}) (*file, error)

var fileDeserializationFuncs = map[version.Number]fileDeserializationFunc{
	twoDotOh: file_2_0,
}

func file_2_0(source map[string]interface{}) (*file, error) {
	fields := schema.Fields{
		"resource_uri":      schema.String(),
		"Filename":          schema.String(),
		"anon_resource_uri": schema.String(),
		"Content":           schema.String(),
	}
	defaults := schema.Defaults{
		"Content": "",
	}
	checker := schema.FieldMap(fields, defaults)
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "file 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	anonURI, err := url.ParseRequestURI(valid["anon_resource_uri"].(string))
	if err != nil {
		return nil, NewUnexpectedError(err)
	}

	result := &file{
		ResourceURI:  valid["resource_uri"].(string),
		Filename:     valid["Filename"].(string),
		AnonymousURI: anonURI,
		Content:      valid["Content"].(string),
	}
	return result, nil
}
