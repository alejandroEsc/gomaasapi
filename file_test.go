// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	"net/http"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/version"
	gc "gopkg.in/check.v1"
)

type fileSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&fileSuite{})

func (*fileSuite) TestReadFilesBadSchema(c *gc.C) {
	_, err := readFiles(twoDotOh, "wat?")
	c.Check(err, jc.Satisfies, IsDeserializationError)
	c.Assert(err.Error(), gc.Equals, `File base schema check failed: expected list, got string("wat?")`)
}

func (*fileSuite) TestReadFiles(c *gc.C) {
	files, err := readFiles(twoDotOh, parseJSON(c, filesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(files, gc.HasLen, 2)
	file := files[0]
	c.Assert(file.Filename(), gc.Equals, "test")
}

func (*fileSuite) TestLowVersion(c *gc.C) {
	_, err := readFiles(version.MustParse("1.9.0"), parseJSON(c, filesResponse))
	c.Assert(err, jc.Satisfies, IsUnsupportedVersionError)
}

func (*fileSuite) TestHighVersion(c *gc.C) {
	files, err := readFiles(version.MustParse("2.1.9"), parseJSON(c, filesResponse))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(files, gc.HasLen, 2)
}

func (s *fileSuite) TestReadAllFromGetFile(c *gc.C) {
	// When get File is used, the response includes the body of the File
	// base64 encoded, so ReadAll just decodes it.
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/files/testing/", http.StatusOK, fileResponse)
	file, err := controller.GetFile("testing")
	c.Assert(err, jc.ErrorIsNil)
	content, err := file.ReadAll()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(string(content), gc.Equals, "this is a test\n")
}

func (s *fileSuite) TestReadAllFromFiles(c *gc.C) {
	// When get File is used, the response includes the body of the File
	// base64 encoded, so ReadAll just decodes it.
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/files/", http.StatusOK, filesResponse)
	server.AddGetResponse("/api/2.0/files/?Filename=test&op=get", http.StatusOK, "some Content\n")
	files, err := controller.Files("")
	c.Assert(err, jc.ErrorIsNil)
	file := files[0]
	content, err := file.ReadAll()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(string(content), gc.Equals, "some Content\n")
}

func (s *fileSuite) TestDeleteMissing(c *gc.C) {
	// If we get a File, but someone else deletes it first, we get a ...
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/files/testing/", http.StatusOK, fileResponse)
	file, err := controller.GetFile("testing")
	c.Assert(err, jc.ErrorIsNil)
	err = file.Delete()
	c.Assert(err, jc.Satisfies, IsNoMatchError)
}

func (s *fileSuite) TestDelete(c *gc.C) {
	// If we get a File, but someone else deletes it first, we get a ...
	server, controller := createTestServerController(c, s)
	server.AddGetResponse("/api/2.0/files/testing/", http.StatusOK, fileResponse)
	server.AddDeleteResponse("/api/2.0/files/testing/", http.StatusOK, "")
	file, err := controller.GetFile("testing")
	c.Assert(err, jc.ErrorIsNil)
	err = file.Delete()
	c.Assert(err, jc.Satisfies, IsNoMatchError)
}

var (
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
