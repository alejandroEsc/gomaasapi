// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var err error

func TestReadBlockDevicesBadSchema(t *testing.T) {
	var b BlockDevice
	err = json.Unmarshal([]byte("wat?"), &b)
	assert.Error(t, err)
}

func TestReadBlockDevices(t *testing.T) {
	var blockdevices []BlockDevice
	err = json.Unmarshal([]byte(blockdevicesResponse), &blockdevices)

	assert.Equal(t, err, nil, "unmarshal error should be nil")
	assert.Len(t, blockdevices, 1)

	blockdevice := blockdevices[0]

	assert.Equal(t, blockdevice.ID, 34)
	assert.Equal(t, blockdevice.Name, "sda")
	assert.Equal(t, blockdevice.Path, "/dev/disk/by-dname/sda")
	assert.Equal(t, blockdevice.IDPath, "/dev/disk/by-ID/ata-QEMU_HARDDISK_QM00001")
	assert.Equal(t, blockdevice.UsedFor, "MBR partitioned with 1 Partition")

	assert.Equal(t, blockdevice.Tags, []string{"rotary"})
	assert.Equal(t, blockdevice.BlockSize, uint64(4096))
	assert.Equal(t, blockdevice.UsedSize, uint64(8586788864))
	assert.Equal(t, blockdevice.Size, uint64(8589934592))

	partitions := blockdevice.Partitions
	assert.Len(t, partitions, 1)
	partition := partitions[0]

	assert.Equal(t, partition.ID, 1)
	assert.Equal(t, partition.UsedFor, "ext4 formatted Filesystem mounted at /")
}

func TestReadBlockDevicesWithNulls(t *testing.T) {
	var blockdevices []BlockDevice
	err = json.Unmarshal([]byte(blockdevicesWithNullsResponse), &blockdevices)
	assert.Equal(t, err, nil, "unmarshal error should be nil")

	assert.Len(t, blockdevices, 1)
	blockdevice := blockdevices[0]

	assert.Equal(t, blockdevice.Model, "")
	assert.Equal(t, blockdevice.IDPath, "")
}

const (
	blockdevicesResponse = `
[
    {
        "Path": "/dev/disk/by-dname/sda",
        "Name": "sda",
        "used_for": "MBR partitioned with 1 Partition",
        "Partitions": [
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
        ],
        "Filesystem": null,
        "id_path": "/dev/disk/by-ID/ata-QEMU_HARDDISK_QM00001",
        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/34/",
        "ID": 34,
        "serial": "QM00001",
        "type": "physical",
        "block_size": 4096,
        "used_size": 8586788864,
        "available_size": 0,
        "partition_table_type": "MBR",
        "UUID": null,
        "Size": 8589934592,
        "Model": "QEMU HARDDISK",
        "Tags": [
            "rotary"
        ]
    }
]
`
	blockdevicesWithNullsResponse = `
[
    {
        "Path": "/dev/disk/by-dname/sda",
        "Name": "sda",
        "used_for": "MBR partitioned with 1 Partition",
        "Partitions": [],
        "Filesystem": null,
        "id_path": null,
        "resource_uri": "/MAAS/api/2.0/nodes/4y3ha3/blockdevices/34/",
        "ID": 34,
        "serial": null,
        "type": "physical",
        "block_size": 4096,
        "used_size": 8586788864,
        "available_size": 0,
        "partition_table_type": null,
        "UUID": null,
        "Size": 8589934592,
        "Model": null,
        "Tags": []
    }
]
`
)
