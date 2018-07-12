// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package client

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"

	"testing"
	"github.com/stretchr/testify/assert"
)

func makeFakeResourceURI() string {
	return "http://cmd.com/" + fmt.Sprint(rand.Int31())
}

// makeFakeMAASObject creates a MAASObject for some imaginary resource.
// There is no actual HTTP service or resource attached.
// serviceURL is the base URL of the service, and ResourceURI is the Path for
// the object, relative to serviceURL.
func makeFakeMAASObject(serviceURL, resourcePath string) MAASObject {
	baseURL, err := url.Parse(serviceURL)
	if err != nil {
		panic(fmt.Errorf("creation of fake object failed: %v", err))
	}
	uri := serviceURL + resourcePath
	input := map[string]interface{}{resourceURI: uri}
	j, err := json.Marshal(input)
	if err != nil {
		panic(fmt.Errorf("creation of fake object failed: %v", err))
	}

	resourceURL, err := url.Parse(uri)
	if err != nil {
		panic(fmt.Errorf("creation of fake object failed: %v", err))
	}

	client := MAASClient{APIURL: baseURL}
	return MAASObject{URI: resourceURL, Client: client, Values: j}
}

// Passing GetSubObject a relative Path effectively concatenates that Path to
// the original object's resource URI.
func TestGetSubObjectRelative(t *testing.T) {
	obj := makeFakeMAASObject("http://cmd.com/", "a/resource/")

	subObj := obj.GetSubObject("test")
	subURL := subObj.URL()

	// URI ends with a slash and subName starts with one, but the two paths
	// should be concatenated as "http://example.com/a/resource/test/".
	expectedSubURL, err := url.Parse("http://cmd.com/a/resource/test/")
	assert.Nil(t, err)
	assert.EqualValues(t, subURL, expectedSubURL)
}

// Passing GetSubObject an absolute Path effectively substitutes that Path for
// the Path component in the original object's resource URI.
func TestGetSubObjectAbsolute(t *testing.T) {
	obj := makeFakeMAASObject("http://cmd.com/", "a/resource/")

	subObj := obj.GetSubObject("/b/test")
	subURL := subObj.URL()

	expectedSubURL, err := url.Parse("http://cmd.com/b/test/")
	assert.Nil(t, err)
	assert.EqualValues(t, subURL, expectedSubURL)
}

// An absolute Path passed to GetSubObject is rooted at the server root, not
// at the service root.  So every absolute resource URI must repeat the part
// of the Path that leads to the service root.  This does not double that part
// of the URI.
func TestGetSubObjectAbsoluteDoesNotDoubleServiceRoot(t *testing.T) {
	obj := makeFakeMAASObject("http://cmd.com/service", "a/resource/")

	subObj := obj.GetSubObject("/service/test")
	subURL := subObj.URL()

	// The "/service" part is not repeated; it must be included.
	expectedSubURL, err := url.Parse("http://cmd.com/service/test/")
	assert.Nil(t, err)
	assert.EqualValues(t, subURL, expectedSubURL)
}

// The argument to GetSubObject is a relative Path, not a URL.  So it won't
// take a query part.  The special characters that mark a query are escaped
// so they are recognized as parts of the Path.
func TestGetSubObjectTakesPathNotURL(t *testing.T) {
	obj := makeFakeMAASObject("http://cmd.com/", "x/")

	subObj := obj.GetSubObject("/y?z")

	assert.Equal(t, subObj.URL().String(), "http://cmd.com/y%3Fz/")
}

func TestGetField(t *testing.T) {
	uri := "http://cmd.com/a/resource"
	fieldName := "field Name"
	fieldValue := "a value"
	input := map[string]interface{}{
		resourceURI: uri, fieldName: fieldValue,
	}
	j, err := json.Marshal(input)
	assert.Nil(t, err)

	resourceURL, err := url.Parse(uri)
	assert.Nil(t, err)
	obj := MAASObject{URI: resourceURL, Client: MAASClient{}, Values: j}

	value := obj.GetField(fieldName)
	assert.Equal(t, value, fieldValue)
}


func TestNewMAASUsesBaseURLFromClient(t *testing.T) {
	baseURLString := "https://server.com:888/"
	baseURL, _ := url.Parse(baseURLString)
	client := MAASClient{APIURL: baseURL}
	maas := NewMAAS(client)
	URL := maas.URL()
	assert.EqualValues(t, URL, baseURL)
}
