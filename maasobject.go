// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

const (
	// Our JSON processor distinguishes a MAASObject from a jsonMap by the fact
	// that it contains a key "resource_uri".  (A regular map might contain the
	// same key through sheer coincide, but never mind: you can still treat it
	// as a jsonMap and never notice the difference.)
	resourceURI = "resource_uri"
)

var (
	noResourceURI                 = errors.New("not a MAAS object: no 'resource_uri' key")
	_              json.Marshaler = (*MAASObject)(nil)
	NotImplemented                = errors.New("Not implemented")
)

// NewMAAS returns an interface to the MAAS API as a *MAASObject.
func NewMAAS(client Client) (*MAASObject, error) {
	attrs := map[string]interface{}{resourceURI: client.APIURL.String()}
	return newJSONMAASObject(attrs, client)
}

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
	Client Client
	URI    *url.URL
}

// newJSONMAASObject creates a new MAAS object.  It will panic if the given map
// does not contain a valid URL for the 'resource_uri' key.
func newJSONMAASObject(jmap map[string]interface{}, client Client) (*MAASObject, error) {
	obj, err := maasify(client, jmap).GetMAASObject()
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

// MarshalJSON tells the standard json package how to serialize a MAASObject.
func (obj *MAASObject) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(obj.Values, "", "  ")
}

func marshalNode(node MAASObject) (string, error) {
	res, err := json.MarshalIndent(node.Values, "", "  ")
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// extractURI obtains the "resource_uri" string from a JSONObject map.
func extractURI(attrs map[string]JSONObject) (*url.URL, error) {
	uriEntry, ok := attrs[resourceURI]
	if !ok {
		return nil, noResourceURI
	}
	uri, err := uriEntry.GetString()
	if err != nil {
		return nil, fmt.Errorf("invalid resource_uri: %v", uri)
	}
	resourceURL, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("resource_uri does not contain a valid URL: %v", uri)
	}
	return resourceURL, nil
}

// JSONObject getter for a MAAS object.  From a decoding perspective, a
// MAASObject is just like a map except it contains a key "resource_uri", and
// it keeps track of the Client you got it from so that you can invoke API
// methods directly on their MAAS objects.
func (obj JSONObject) GetMAASObject() (MAASObject, error) {
	attrs, err := obj.GetMap()
	if err != nil {
		return MAASObject{}, err
	}
	uri, err := extractURI(attrs)
	if err != nil {
		return MAASObject{}, err
	}
	return MAASObject{Values: attrs, Client: obj.client, URI: uri}, nil
}

// GetField extracts a string field from this MAAS object.
func (obj MAASObject) GetField(name string) (string, error) {
	return obj.Values[name].GetString()
}

// URL returns a full absolute URL (including network part) for this MAAS
// object on the API.
func (obj MAASObject) URL() *url.URL {
	return obj.Client.GetURL(obj.URI)
}

// GetMap returns all of the object's attributes in the form of a map.
func (obj MAASObject) GetMap() map[string]JSONObject {
	return obj.Values
}

// GetSubObject returns a new MAASObject representing the API resource found
// at a given sub-Path of the current object's resource URI.
func (obj MAASObject) GetSubObject(name string) (*MAASObject, error) {
	newURL := url.URL{Path: name}
	resUrl := obj.URI.ResolveReference(&newURL)
	resUrl.Path = EnsureTrailingSlash(resUrl.Path)
	input := map[string]interface{}{resourceURI: resUrl.String()}
	return newJSONMAASObject(input, obj.Client)
}

// Get retrieves a fresh copy of this MAAS object from the API.
func (obj MAASObject) Get() (MAASObject, error) {
	result, err := obj.Client.Get(obj.URI, "", url.Values{})
	if err != nil {
		return MAASObject{}, err
	}
	jsonObj, err := Parse(obj.Client, result)
	if err != nil {
		return MAASObject{}, err
	}
	return jsonObj.GetMAASObject()
}

// Post overwrites this object's existing value on the API with those given
// in "params."  It returns the object's new value as received from the API.
func (obj MAASObject) Post(params url.Values) (JSONObject, error) {
	result, err := obj.Client.Post(obj.URI, "", params, nil)
	if err != nil {
		return JSONObject{}, err
	}
	return Parse(obj.Client, result)
}

// Update modifies this object on the API, based on the Values given in
// "params."  It returns the object's new value as received from the API.
func (obj MAASObject) Update(params url.Values) (MAASObject, error) {
	result, err := obj.Client.Put(obj.URI, params)
	if err != nil {
		return MAASObject{}, err
	}
	jsonObj, err := Parse(obj.Client, result)
	if err != nil {
		return MAASObject{}, err
	}
	return jsonObj.GetMAASObject()
}

// Delete removes this object on the API.
func (obj MAASObject) Delete() error {
	return obj.Client.Delete(obj.URI)
}

// CallGet invokes an idempotent API method on this object.
func (obj MAASObject) CallGet(operation string, params url.Values) (JSONObject, error) {
	result, err := obj.Client.Get(obj.URI, operation, params)
	if err != nil {
		return JSONObject{}, err
	}
	return Parse(obj.Client, result)
}

// CallPost invokes a non-idempotent API method on this object.
func (obj MAASObject) CallPost(operation string, params url.Values) (JSONObject, error) {
	return obj.CallPostFiles(operation, params, nil)
}

// CallPostFiles invokes a non-idempotent API method on this object.  It is
// similar to CallPost but has an extra parameter, 'files', which should
// contain the files that will be uploaded to the API.
func (obj MAASObject) CallPostFiles(operation string, params url.Values, files map[string][]byte) (JSONObject, error) {
	result, err := obj.Client.Post(obj.URI, operation, params, files)
	if err != nil {
		return JSONObject{}, err
	}
	return Parse(obj.Client, result)
}
