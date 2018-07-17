// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/juju/errors"
	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/util"
	"github.com/juju/loggo"
	"github.com/juju/utils/set"
	"github.com/juju/version"
)

var (
	logger = loggo.GetLogger("maas")
	// The supported versions should be ordered from most desirable version to
	// least as they will be tried in order.
	supportedAPIVersions = []string{"2.0", "2.1", "2.3", "2.4"}
	// Current request number. Informational only for logging.
	requestNumber int64
)

// Controller represents an API connection to a maas. Since the API
// is restful, there is no long held connection to the API server, but instead
// HTTP calls are made and JSON response structures parsed.
type Controller struct {
	Client     *client.MAASClient
	APIVersion version.Number
	// Capabilities returns a set of Capabilities as defined by the string
	// constants.
	Capabilities set.Strings
}

// ControllerArgs is an argument struct for passing the required parameters
// to the NewController method.
type ControllerArgs struct {
	BaseURL string
	APIKey  string
}

// NewController creates an authenticated Client to the maas API, and
// checks the Capabilities of the server. If the BaseURL specified
// includes the API version, that version of the API will be used,
// otherwise the ControllerInterface will use the highest supported version
// available.
//
// If the APIKey is not valid, a NotValid error is returned.
// If the credentials are incorrect, a PermissionError is returned.
func NewController(args ControllerArgs) (*Controller, error) {
	base, apiVersion, includesVersion := client.SplitVersionedURL(args.BaseURL)
	if includesVersion {
		if !SupportedVersion(apiVersion) {
			return nil, util.NewUnsupportedVersionError("version %s", apiVersion)
		}
		return NewControllerWithVersion(base, apiVersion, args.APIKey)
	}
	return NewControllerUnknownVersion(args)
}

func SupportedVersion(value string) bool {
	for _, version := range supportedAPIVersions {
		if value == version {
			return true
		}
	}
	return false
}

func NewControllerWithVersion(baseURL, apiVersion, apiKey string) (*Controller, error) {
	major, minor, err := version.ParseMajorMinor(apiVersion)
	// We should not Get an error here. See the test.
	if err != nil {
		return nil, errors.Errorf("bad version defined in supported versions: %q", apiVersion)
	}
	client, err := client.NewAuthenticatedMAASClient(client.AddAPIVersionToURL(baseURL, apiVersion), apiKey)
	if err != nil {
		// If the credentials aren't valid, return now.
		if errors.IsNotValid(err) {
			return nil, err
		}
		// Any other error attempting to create the authenticated Client
		// is an unexpected error and return now.
		return nil, util.NewUnexpectedError(err)
	}
	controllerVersion := version.Number{
		Major: major,
		Minor: minor,
	}
	controller := &Controller{Client: client, APIVersion: controllerVersion}
	controller.Capabilities, err = controller.readAPIVersionInfo()
	if err != nil {
		logger.Debugf("nread version failed: %#v", err)
		return nil, err
	}

	if err := controller.checkCreds(); err != nil {
		return nil, err
	}
	return controller, nil
}

func NewControllerUnknownVersion(args ControllerArgs) (*Controller, error) {
	// For now we don't need to test multiple versions. It is expected that at
	// some time in the future, we will try the most up to date version and then
	// work our way backwards.
	for _, apiVersion := range supportedAPIVersions {
		controller, err := NewControllerWithVersion(args.BaseURL, apiVersion, args.APIKey)
		switch {
		case err == nil:
			return controller, nil
		case util.IsUnsupportedVersionError(err):
			// This will only come back from readAPIVersionInfo for 410/404.
			continue
		default:
			return nil, err
		}
	}

	return nil, util.NewUnsupportedVersionError("ControllerInterface at %s does not support any of %s", args.BaseURL, supportedAPIVersions)
}

func (c *Controller) checkCreds() error {
	if _, err := c.Get("users", "whoami", nil); err != nil {
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			if svrErr.StatusCode == http.StatusUnauthorized {
				return errors.Wrap(err, util.NewPermissionError(svrErr.BodyMessage))
			}
		}
		return util.NewUnexpectedError(err)
	}
	return nil
}

func (c Controller) Put(path string, params url.Values) ([]byte, error) {
	path = util.EnsureTrailingSlash(path)
	requestID := nextRequestID()
	logger.Tracef("request %x: PUT %s%s, params: %s", requestID, c.Client.APIURL, path, params.Encode())
	bytes, err := c.Client.Put(&url.URL{Path: path}, params)
	if err != nil {
		logger.Tracef("response %x: error: %q", requestID, err.Error())
		logger.Tracef("error detail: %#v", err)
		return nil, err
	}
	return bytes, nil
}

func (c Controller) Post(path, op string, params url.Values) ([]byte, error) {
	bytes, err := c.postRaw(path, op, params, nil)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (c Controller) PostFile(path, op string, params url.Values, fileContent []byte) ([]byte, error) {
	// Only one File is ever sent at a time.
	files := map[string][]byte{"file": fileContent}
	return c.postRaw(path, op, params, files)
}

func (c *Controller) postRaw(path, op string, params url.Values, files map[string][]byte) ([]byte, error) {
	path = util.EnsureTrailingSlash(path)
	url := &url.URL{Path: path}
	requestID := nextRequestID()
	if logger.IsTraceEnabled() {
		opArg := ""
		if op != "" {
			opArg = "?op=" + op
		}
		logger.Tracef("request %x: Post %s%s%s, params=%s", requestID, c.Client.APIURL, path, opArg, params.Encode())
	}
	bytes, err := c.Client.Post(url, op, params, files)
	if err != nil {
		logger.Tracef("response %x: error: %q", requestID, err.Error())
		logger.Tracef("error detail: %#v", err)
		return nil, err
	}
	logger.Tracef("response %x: %s", requestID, string(bytes))
	return bytes, nil
}

func (c Controller) Delete(path string) error {
	path = util.EnsureTrailingSlash(path)
	url := &url.URL{Path: path}
	requestID := nextRequestID()
	logger.Tracef("request %x: DELETE %s%s", requestID, c.Client.APIURL, path)
	err := c.Client.Delete(url)
	if err != nil {
		logger.Tracef("response %x: error: %q", requestID, err.Error())
		logger.Tracef("error detail: %#v", err)
		return err
	}
	logger.Tracef("response %x: complete", requestID)
	return nil
}

func (c Controller) Get(path string, op string, params url.Values) ([]byte, error) {
	path = util.EnsureTrailingSlash(path)
	url := &url.URL{Path: path}
	requestID := nextRequestID()
	if logger.IsTraceEnabled() {
		var query string
		if params != nil {
			query = "?" + params.Encode()
		}
		logger.Tracef("request %x: Get %s%s%s", requestID, c.Client.APIURL, path, query)
	}
	bytes, err := c.Client.Get(url, op, params)
	if err != nil {
		logger.Tracef("response %x: error: %q", requestID, err.Error())
		logger.Tracef("error detail: %#v", err)
		return nil, err
	}
	logger.Tracef("response %x: %s", requestID, string(bytes))
	return bytes, nil
}

func nextRequestID() int64 {
	return atomic.AddInt64(&requestNumber, 1)
}

func indicatesUnsupportedVersion(err error) bool {
	if err == nil {
		return false
	}
	if serverErr, ok := errors.Cause(err).(client.ServerError); ok {
		code := serverErr.StatusCode
		return code == http.StatusNotFound || code == http.StatusGone
	}
	// Workaround for bug in maas 1.9.4 - instead of a 404 we Get a
	// redirect to the HTML login page, which doesn't parse as JSON.
	// https://bugs.launchpad.net/maas/+bug/1583715
	if syntaxErr, ok := errors.Cause(err).(*json.SyntaxError); ok {
		message := "invalid character '<' looking for beginning of value"
		return syntaxErr.Offset == 1 && syntaxErr.Error() == message
	}
	return false
}

func (c *Controller) readAPIVersionInfo() (set.Strings, error) {
	parsedBytes, err := c.Get("version", "", nil)
	if indicatesUnsupportedVersion(err) {
		return nil, util.WrapWithUnsupportedVersionError(err)
	} else if err != nil {
		return nil, err
	}

	var version Version
	err = json.Unmarshal(parsedBytes, &version)
	if err != nil {
		return nil, util.WrapWithDeserializationError(err, "unmarshal error")
	}

	capabilities := set.NewStrings()
	for _, value := range version.Capabilities {
		capabilities.Add(value)
	}

	return capabilities, nil
}


