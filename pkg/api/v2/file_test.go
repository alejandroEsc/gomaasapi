// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"net/http"
	"testing"

	"encoding/json"

	"github.com/juju/gomaasapi/pkg/api/util"
	"github.com/stretchr/testify/assert"

)


func TestReadFilesBadSchema(t *testing.T) {
	var f File
	err = json.Unmarshal([]byte("wat?"), &f)
	assert.Error(t, err)
}

func TestReadFiles(t *testing.T) {
	var files []File
	err = json.Unmarshal([]byte(filesResponse), &files)
	assert.Nil(t, err)
	c.Assert(files, gc.HasLen, 2)
	file := files[0]
	c.Assert(file.Filename, gc.Equals, "test")
}

func TestReadAllFromGetFile(t *testing.T) {
	// When get File is used, the response includes the body of the File
	// base64 encoded, so ReadAll just decodes it.
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/files/testing/", http.StatusOK, fileResponse)
	file, err := controller.GetFile("testing")
	assert.Nil(t, err)
	content, err := file.ReadAll()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(string(content), gc.Equals, "this is a test\n")
}

func TestReadAllFromFiles(t *testing.T) {
	// When get File is used, the response includes the body of the File
	// base64 encoded, so ReadAll just decodes it.
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/files/", http.StatusOK, filesResponse)
	server.AddGetResponse("/api/2.0/files/?Filename=test&op=get", http.StatusOK, "some Content\n")
	files, err := controller.Files("")
	assert.Nil(t, err)
	file := files[0]
	content, err := file.ReadAll()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(string(content), gc.Equals, "some Content\n")
}

func TestDeleteMissing(t *testing.T) {
	// If we get a File, but someone else deletes it first, we get a ...
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/files/testing/", http.StatusOK, fileResponse)
	file, err := controller.GetFile("testing")
	assert.Nil(t, err)
	err = file.Delete()
	c.Assert(err, jc.Satisfies, util.IsNoMatchError)
}

func TestDelete(t *testing.T) {
	// If we get a File, but someone else deletes it first, we get a ...
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/files/testing/", http.StatusOK, fileResponse)
	server.AddDeleteResponse("/api/2.0/files/testing/", http.StatusOK, "")
	file, err := controller.GetFile("testing")
	assert.Nil(t, err)
	err = file.Delete()
	c.Assert(err, jc.Satisfies, util.IsNoMatchError)
}

const (
	fileResponse = `
{
    "resource_uri": "/MAAS/api/2.0/files/testing/",
    "Content": "dGhpcyBpcyBhIHRlc3QK",
    "anon_resource_uri": "/MAAS/api/2.0/files/?op=get_by_key&key=88e64b76-fb82-11e5-932f-52540051bf22",
    "Filename": "testing"
}
`
	filesResponse = `
[
    {
        "resource_uri": "/MAAS/api/2.0/files/test/",
        "anon_resource_uri": "/MAAS/api/2.0/files/?op=get_by_key&key=3afba564-fb7d-11e5-932f-52540051bf22",
        "Filename": "test"
    },
    {
        "resource_uri": "/MAAS/api/2.0/files/test-File.txt/",
        "anon_resource_uri": "/MAAS/api/2.0/files/?op=get_by_key&key=69913e62-fad2-11e5-932f-52540051bf22",
        "Filename": "test-File.txt"
    }
]
`
)
