// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/base64"
	"net/url"

	"encoding/json"
	"fmt"
	"strings"

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

// UnmarshalJSON allows json.Unmarshal to properly unmarshal json
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
			bytes, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return fmt.Errorf("err: %s, content is %s.", err.Error(), v)
			}
			f.Content = string(bytes)
		}
	}

	return nil
}
