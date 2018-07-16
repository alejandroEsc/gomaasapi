// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadAndCloseReturnsEmptyStringForNil(t *testing.T) {
	data, err := readAndClose(nil)
	assert.Nil(t, err)
	assert.Equal(t, string(data), "")
}

func TestReadAndCloseReturnsContents(t *testing.T) {
	content := "Stream contents."
	stream := ioutil.NopCloser(strings.NewReader(content))

	data, err := readAndClose(stream)
	assert.Nil(t, err)

	assert.Equal(t, string(data), content)
}

func TestClientdispatchRequestReturnsServerError(t *testing.T) {
	URI := "/some/url/?param1=test"
	expectedResult := "expected:result"
	server := newSingleServingServer(URI, expectedResult, http.StatusBadRequest)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	assert.Nil(t, err)
	request, err := http.NewRequest("GET", server.URL+URI, nil)

	result, err := client.dispatchRequest(request)

	expectedErrorString := fmt.Sprintf("ServerError: 400 Bad Request (%v)", expectedResult)
	assert.EqualError(t, err, expectedErrorString)

	svrError, ok := GetServerError(err)
	assert.True(t, ok)
	assert.Equal(t, svrError.StatusCode, 400)
	assert.Equal(t, string(result), expectedResult)
}

func TestClientdispatchRequestRetries503(t *testing.T) {
	URI := "/some/url/?param1=test"
	server := newFlakyServer(URI, 503, NumberOfRetries)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	assert.Nil(t, err)
	content := "Content"
	request, err := http.NewRequest("GET", server.URL+URI, ioutil.NopCloser(strings.NewReader(content)))

	_, err = client.dispatchRequest(request)

	assert.Nil(t, err)
	assert.Equal(t, *server.nbRequests, NumberOfRetries+1)
	expectedRequestsContent := make([][]byte, NumberOfRetries+1)
	for i := 0; i < NumberOfRetries+1; i++ {
		expectedRequestsContent[i] = []byte(content)
	}
	assert.EqualValues(t, *server.requests, expectedRequestsContent)
}

func TestClientdispatchRequestDoesntRetry200(t *testing.T) {
	URI := "/some/url/?param1=test"
	server := newFlakyServer(URI, 200, 10)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	assert.Nil(t, err)

	request, err := http.NewRequest("GET", server.URL+URI, nil)

	_, err = client.dispatchRequest(request)

	assert.Nil(t, err)
	assert.Equal(t, *server.nbRequests, 1)
}

func TestClientdispatchRequestRetriesIsLimited(t *testing.T) {
	URI := "/some/url/?param1=test"
	// Make the server return 503 responses NumberOfRetries + 1 times.
	server := newFlakyServer(URI, 503, NumberOfRetries+1)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	assert.Nil(t, err)
	request, err := http.NewRequest("GET", server.URL+URI, nil)

	_, err = client.dispatchRequest(request)

	assert.Equal(t, *server.nbRequests, NumberOfRetries+1)
	svrError, ok := GetServerError(err)
	assert.True(t, ok)
	assert.Equal(t, svrError.StatusCode, 503)
}

func TestClientDispatchRequestReturnsNonServerError(t *testing.T) {
	client, err := NewAnonymousClient("/foo", "1.0")
	assert.Nil(t, err)
	// Create a bad request that will fail to dispatch.
	request, err := http.NewRequest("GET", "/", nil)
	assert.Nil(t, err)

	result, err := client.dispatchRequest(request)
	assert.NotNil(t, err)
	// This type of failure is an error, but not a ServerError.
	_, ok := GetServerError(err)
	assert.False(t, ok)
	// For this kind of error, result is guaranteed to be nil.
	assert.Nil(t, result)
}

func TestClientdispatchRequestSignsRequest(t *testing.T) {
	URI := "/some/url/?param1=test"
	expectedResult := "expected:result"
	server := newSingleServingServer(URI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAuthenticatedMAASClient(server.URL, "the:api:key")
	assert.Nil(t, err)
	request, err := http.NewRequest("GET", server.URL+URI, nil)
	assert.Nil(t, err)

	result, err := client.dispatchRequest(request)

	assert.Nil(t, err)
	assert.Equal(t, string(result), expectedResult)

	authHeader := (*server.requestHeader)["Authorization"][0]
	assert.Contains(t, authHeader, "OAuth")
}

func TestClientGetFormatsGetParameters(t *testing.T) {
	URI, err := url.Parse("/some/url")
	assert.Nil(t, err)
	expectedResult := "expected:result"
	params := url.Values{"test": {"123"}}
	fullURI := URI.String() + "?test=123"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	assert.Nil(t, err)

	result, err := client.Get(URI, "", params)

	assert.Nil(t, err)
	assert.Equal(t, string(result), expectedResult)
}

func TestClientGetFormatsOperationAsGetParameter(t *testing.T) {
	URI, err := url.Parse("/some/url")
	assert.Nil(t, err)
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=list"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	assert.Nil(t, err)

	result, err := client.Get(URI, "list", nil)

	assert.Nil(t, err)
	assert.Equal(t, string(result), expectedResult)
}

func TestClientPostSendsRequestWithParams(t *testing.T) {
	URI, err := url.Parse("/some/url")
	assert.Nil(t, err)
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=list"
	params := url.Values{"test": {"123"}}
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	assert.Nil(t, err)

	result, err := client.Post(URI, "list", params, nil)

	assert.Nil(t, err)
	assert.Equal(t, string(result), expectedResult)
	postedValues, err := url.ParseQuery(*server.requestContent)
	assert.Nil(t, err)
	expectedPostedValues, err := url.ParseQuery("test=123")
	assert.Nil(t, err)
	assert.EqualValues(t, postedValues, expectedPostedValues)
}

// extractFileContent extracts from the request built using 'requestContent',
// 'requestHeader' and 'requestURL', the File named 'Filename'.
func extractFileContent(requestContent string, requestHeader *http.Header, requestURL string, filename string) ([]byte, error) {
	// Recreate the request from server.requestContent to use the parsing
	// utility from the http package (http.Request.FormFile).
	request, err := http.NewRequest("POST", requestURL, bytes.NewBufferString(requestContent))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", requestHeader.Get("Content-Type"))
	file, _, err := request.FormFile("testfile")
	if err != nil {
		return nil, err
	}
	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return fileContent, nil
}

func TestClientPostSendsMultipartRequest(t *testing.T) {
	URI, err := url.Parse("/some/url")
	assert.Nil(t, err)
	expectedResult := "expected:result"
	fullURI := URI.String() + "?op=add"
	server := newSingleServingServer(fullURI, expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	assert.Nil(t, err)
	fileContent := []byte("Content")
	files := map[string][]byte{"testfile": fileContent}

	result, err := client.Post(URI, "add", nil, files)

	assert.Nil(t, err)
	assert.Equal(t, string(result), expectedResult)
	receivedFileContent, err := extractFileContent(*server.requestContent, server.requestHeader, fullURI, "testfile")
	assert.Nil(t, err)
	assert.EqualValues(t, receivedFileContent, fileContent)
}

func TestClientPutSendsRequest(t *testing.T) {
	URI, err := url.Parse("/some/url")
	assert.Nil(t, err)
	expectedResult := "expected:result"
	params := url.Values{"test": {"123"}}
	server := newSingleServingServer(URI.String(), expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	assert.Nil(t, err)

	result, err := client.Put(URI, params)

	assert.Nil(t, err)
	assert.Equal(t, string(result), expectedResult)
	assert.Equal(t, *server.requestContent, "test=123")
}

func TestClientDeleteSendsRequest(t *testing.T) {
	URI, err := url.Parse("/some/url")
	assert.Nil(t, err)
	expectedResult := "expected:result"
	server := newSingleServingServer(URI.String(), expectedResult, http.StatusOK)
	defer server.Close()
	client, err := NewAnonymousClient(server.URL, "1.0")
	assert.Nil(t, err)

	err = client.Delete(URI)
	assert.Nil(t, err)
}

func TestNewAnonymousClientEnsuresTrailingSlash(t *testing.T) {
	client, err := NewAnonymousClient("http://cmd.com/", "1.0")
	assert.Nil(t, err)
	expectedURL, err := url.Parse("http://cmd.com/api/1.0/")
	assert.Nil(t, err)
	assert.EqualValues(t, client.APIURL, expectedURL)
}

func TestNewAuthenticatedClientEnsuresTrailingSlash(t *testing.T) {
	client, err := NewAuthenticatedMAASClient("http://cmd.com/api/1.0", "a:b:c")
	assert.Nil(t, err)
	expectedURL, err := url.Parse("http://cmd.com/api/1.0/")
	assert.Nil(t, err)
	assert.EqualValues(t, client.APIURL, expectedURL)
}

func TestNewAuthenticatedClientParsesApiKey(t *testing.T) {
	// NewAuthenticatedMAASClient returns a plainTextOAuthSigneri configured
	// to use the given API key.
	consumerKey := "consumerKey"
	tokenKey := "tokenKey"
	tokenSecret := "tokenSecret"
	keyElements := []string{consumerKey, tokenKey, tokenSecret}
	apiKey := strings.Join(keyElements, ":")

	client, err := NewAuthenticatedMAASClient("http://cmd.com/api/1.0/", apiKey)

	assert.Nil(t, err)
	signer := client.Signer.(*plainTextOAuthSigner)
	assert.Equal(t, signer.token.ConsumerKey, consumerKey)
	assert.Equal(t, signer.token.TokenKey, tokenKey)
	assert.Equal(t, signer.token.TokenSecret, tokenSecret)
}

func TestNewAuthenticatedClientFailsIfInvalidKey(t *testing.T) {
	client, err := NewAuthenticatedMAASClient("", "invalid-key")

	assert.Contains(t, err.Error(), "invalid API key")
	assert.Nil(t, client)
}

func TestAddAPIVersionToURL(t *testing.T) {
	addVersion := AddAPIVersionToURL
	assert.Equal(t, addVersion("http://cmd.com/maas", "1.0"), "http://cmd.com/maas/api/1.0/")
	assert.Equal(t, addVersion("http://cmd.com/maas/", "2.0"), "http://cmd.com/maas/api/2.0/")
}

func TestSplitVersionedURL(t *testing.T) {
	check := func(url, expectedBase, expectedVersion string, expectedResult bool) {
		base, version, ok := SplitVersionedURL(url)
		assert.Equal(t, ok, expectedResult)
		assert.Equal(t, base, expectedBase)
		assert.Equal(t, version, expectedVersion)
	}
	check("http://maas.server/maas", "http://maas.server/maas", "", false)
	check("http://maas.server/maas/api/3.0", "http://maas.server/maas/", "3.0", true)
	check("http://maas.server/maas/api/3.0/", "http://maas.server/maas/", "3.0", true)
	check("http://maas.server/maas/api/maas", "http://maas.server/maas/api/maas", "", false)
}
