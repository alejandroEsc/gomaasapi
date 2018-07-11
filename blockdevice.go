// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

type BlockDevice struct {
	ResourceURI string `json:"resource_uri,omitempty"`

	ID      int      `json:"ID,omitempty"`
	Name    string   `json:"Name,omitempty"`
	Model   string   `json:"Model,omitempty"`
	IDPath  string   `json:"id_path,omitempty"`
	Path    string   `json:"Path,omitempty"`
	UsedFor string   `json:"used_for,omitempty"`
	Tags    []string `json:"Tags,omitempty"`

	BlockSize uint64 `json:"block_size,omitempty"`
	UsedSize  uint64 `json:"used_size,omitempty"`
	Size      uint64 `json:"Size,omitempty"`

	Partitions []*partition `json:"Partitions,omitempty"`
}
