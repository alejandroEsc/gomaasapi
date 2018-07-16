// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"
	"testing"

	"github.com/juju/gomaasapi/pkg/api/util"
	"github.com/stretchr/testify/assert"
)

func TestReadPartitionsBadSchema(t *testing.T) {
	var p Partition
	err = json.Unmarshal([]byte("wat?"), &p)
	assert.Error(t, err)
}

func TestReadPartitions(t *testing.T) {
	var partitions []Partition
	err = json.Unmarshal([]byte(partitionsResponse), &partitions)
	assert.Nil(t, err)

	assert.Len(t, partitions, 1)
	partition := partitions[0]

	assert.Equal(t, partition.ID, 1)
	assert.Equal(t, partition.Path, "/dev/disk/by-dname/sda-part1")
	assert.Equal(t, partition.UUID, "6199b7c9-b66f-40f6-a238-a938a58a0adf")
	assert.Equal(t, partition.UsedFor, "ext4 formatted Filesystem mounted at /")
	assert.Equal(t, partition.Size, uint64(8581545984))

	fs := partition.FileSystem
	assert.NotNil(t, fs)
	assert.Equal(t, fs.Type, "ext4")
	assert.Equal(t, fs.MountPoint, "/")
}

func TestReadPartitionsNilUUID(t *testing.T) {
	j := util.ParseJSON(t, partitionsResponse)
	j.([]interface{})[0].(map[string]interface{})["UUID"] = nil

	jr, err := json.Marshal(j)
	assert.Nil(t, err)

	var partitions []Partition
	err = json.Unmarshal(jr, &partitions)

	assert.Nil(t, err)
	assert.Len(t, partitions, 1)
	partition := partitions[0]
	assert.Equal(t, partition.UUID, "")
}

const partitionsResponse = `
[
    {
        "bootable": false,
        "ID": 1,
        "Path": "/dev/disk/by-dname/sda-part1",
        "Filesystem": {
            "Type": "ext4",
            "mount_point": "/",
            "Label": "root",
            "mount_options": null,
            "UUID": "fcd7745e-f1b5-4f5d-9575-9b0bb796b752"
        },
        "type": "Partition",
        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/34/Partition/1",
        "UUID": "6199b7c9-b66f-40f6-a238-a938a58a0adf",
        "used_for": "ext4 formatted Filesystem mounted at /",
        "Size": 8581545984
    }
]
`
