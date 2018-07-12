// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

type partition struct {
	ResourceURI string      `json:"resource_uri,omitempty"`
	ID          int         `json:"ID,omitempty"`
	Path        string      `json:"Path,omitempty"`
	UUID        string      `json:"UUID,omitempty"`
	UsedFor     string      `json:"used_for,omitempty"`
	Size        uint64      `json:"Size,omitempty"`
	FileSystem  *filesystem `json:"filesystem,omitempty"`
}
