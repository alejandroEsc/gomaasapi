// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package client

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/tidwall/gjson"
	"github.com/juju/gomaasapi/pkg/api/util"
)

const (
	// Our JSON processor distinguishes a MAASObject from a jsonMap by the fact
	// that it contains a key "resource_uri".  (A regular map might contain the
	// same key through sheer coincide, but never mind: you can still treat it
	// as a jsonMap and never notice the difference.)
	resourceURI = "resource_uri"
)

// MAASObject represents a MAAS object as returned by the MAAS API, such as a
// Node or a Tag.
// You can extract a MAASObject out of a JSONObject using
// JSONObject.GetMAASObject.  A MAAS API call will usually return either a
// MAASObject or a list of MAASObjects.  The list itself would be wrapped in
// a JSONObject, so if an API call returns a list of objects "l," you first
// obtain the array using l.GetArray().  Then, for each item "i" in the array,
// obtain the matching MAASObject using i.GetMAASObject().
type MAASObject struct {
	Values []byte
	Client MAASClient
	URI    *url.URL
}

// NewMAAS returns an interface to the MAAS API as a *MAASObject.
func NewMAAS(client MAASClient) *MAASObject {
	return &MAASObject{URI: client.APIURL, Client: client, Values: nil}
}

// MarshalJSON tells the standard json package how to serialize a MAASObject.
func (obj *MAASObject) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(obj.Values, "", "  ")
}

// GetSubObject returns a new MAASObject representing the API resource found
// at a given sub-path of the current object's resource URI.
func (obj MAASObject) GetSubObject(name string) *MAASObject {
	uri := obj.URI
	newURL := url.URL{Path: name}
	resUrl := uri.ResolveReference(&newURL)
	resUrl.Path = util.EnsureTrailingSlash(resUrl.Path)
	return &MAASObject{URI: resUrl, Client: obj.Client, Values: nil}
}

func marshalNode(node MAASObject) (string, error) {
	res, err := json.MarshalIndent(node.Values, "", "  ")
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// extractURI obtains the "resource_uri" string from a json map.
func extractURI(value []byte) (*url.URL, error) {
	uriEntry := gjson.Get(string(value), resourceURI)
	uri := uriEntry.String()
	resourceURL, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("resource_uri does not contain a valid URL: %v", uri)
	}
	return resourceURL, nil
}

// URL returns a full absolute URL (including network part) for this MAAS
// object on the API.
func (obj MAASObject) URL() *url.URL {
	return obj.Client.GetURL(obj.URI)
}

// GetField extracts a string field from this MAAS object.
func (obj MAASObject) GetField(name string) string {
	return gjson.Get(string(obj.Values), name).String()
}

// Get retrieves a fresh copy of this MAAS object from the API.
func (obj MAASObject) Get() (*MAASObject, error) {
	result, err := obj.Client.Get(obj.URI, "", url.Values{})
	if err != nil {
		return nil, err
	}

	uri, err := extractURI(obj.Values)
	if err != nil {
		return nil, err
	}

	return &MAASObject{Values: result, Client: obj.Client, URI: uri}, nil
}

// Post overwrites this object's existing value on the API with those given
// in "params."  It returns the object's new value as received from the API.
func (obj MAASObject) Post(params url.Values) (*MAASObject, error) {
	result, err := obj.Client.Post(obj.URI, "", params, nil)
	if err != nil {
		return nil, err
	}

	uri, err := extractURI(result)
	if err != nil {
		return &MAASObject{}, err
	}

	return &MAASObject{Values: result, Client: obj.Client, URI: uri}, nil
}

// Update modifies this object on the API, based on the Values given in
// "params."  It returns the object's new value as received from the API.
func (obj MAASObject) Update(params url.Values) (*MAASObject, error) {
	result, err := obj.Client.Put(obj.URI, params)
	if err != nil {
		return nil, err
	}

	uri, err := extractURI(result)
	if err != nil {
		return &MAASObject{}, err
	}

	return &MAASObject{Values: result, Client: obj.Client, URI: uri}, nil
}

// Delete removes this object on the API.
func (obj MAASObject) Delete() error {
	return obj.Client.Delete(obj.URI)
}

// CallGet invokes an idempotent API method on this object.
func (obj MAASObject) CallGet(operation string, params url.Values) (*MAASObject, error) {
	result, err := obj.Client.Get(obj.URI, operation, params)
	if err != nil {
		return nil, err
	}

	uri, err := extractURI(result)
	if err != nil {
		return &MAASObject{}, err
	}

	return &MAASObject{Values: result, Client: obj.Client, URI: uri}, nil
}

// CallPost invokes a non-idempotent API method on this object.
func (obj MAASObject) CallPost(operation string, params url.Values) (*MAASObject, error) {
	return obj.CallPostFiles(operation, params, nil)
}

// CallPostFiles invokes a non-idempotent API method on this object.  It is
// similar to CallPost but has an extra parameter, 'files', which should
// contain the files that will be uploaded to the API.
func (obj MAASObject) CallPostFiles(operation string, params url.Values, files map[string][]byte) (*MAASObject, error) {
	result, err := obj.Client.Post(obj.URI, operation, params, files)
	if err != nil {
		return nil, err
	}
	uri, err := extractURI(result)
	if err != nil {
		return &MAASObject{}, err
	}

	return &MAASObject{Values: result, Client: obj.Client, URI: uri}, nil
}
