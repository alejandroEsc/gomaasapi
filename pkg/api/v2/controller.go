// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync/atomic"

	"io"

	"github.com/juju/errors"
	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/util"
	"github.com/juju/loggo"
	"github.com/juju/schema"
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

// BootResources implements ControllerInterface.
func (c *Controller) BootResources() ([]*BootResource, error) {
	source, err := c.Get("boot-resources", "", nil)
	if err != nil {
		return nil, util.NewUnexpectedError(err)
	}

	var resources []*BootResource
	err = json.Unmarshal(source, &resources)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

// Fabrics returns the list of Fabrics defined in the maas ControllerInterface.
func (c *Controller) Fabrics() ([]Fabric, error) {
	source, err := c.Get("fabrics", "", nil)
	if err != nil {
		return nil, util.NewUnexpectedError(err)
	}

	var fabrics []Fabric
	err = json.Unmarshal(source, &fabrics)
	if err != nil {
		return nil, err
	}

	return fabrics, nil
}

// Spaces returns the list of Spaces defined in the maas ControllerInterface.
func (c *Controller) Spaces() ([]Space, error) {
	source, err := c.Get("spaces", "", nil)
	if err != nil {
		return nil, util.NewUnexpectedError(err)
	}

	var spaces []Space
	err = json.Unmarshal(source, &spaces)
	if err != nil {
		return nil, err
	}

	return spaces, nil
}

// StaticRoutes returns the list of StaticRoutes defined in the maas ControllerInterface.
func (c *Controller) StaticRoutes() ([]StaticRoute, error) {
	source, err := c.Get("static-routes", "", nil)
	if err != nil {
		return nil, util.NewUnexpectedError(err)
	}
	var staticRoutes []StaticRoute
	err = json.Unmarshal(source, &staticRoutes)
	if err != nil {
		return nil, err
	}

	return staticRoutes, nil
}

// Zones lists all the zones known to the maas ControllerInterface.
func (c *Controller) Zones() ([]Zone, error) {
	source, err := c.Get("zones", "", nil)
	if err != nil {
		return nil, util.NewUnexpectedError(err)
	}
	var zones []Zone
	err = json.Unmarshal(source, &zones)
	if err != nil {
		return nil, err
	}
	return zones, nil
}

// Nodes returns a list of devices that match the params.
func (c *Controller) Nodes(args NodesArgs) ([]Node, error) {
	params := NodesParams(args)
	source, err := c.Get("nodes", "", params.Values)
	if err != nil {
		return nil, util.NewUnexpectedError(err)
	}

	results := make([]Node, 0)
	var nodes []Node
	err = json.Unmarshal(source, &nodes)
	if err != nil {
		return nil, err
	}
	for _, d := range nodes {
		d.Controller = c
		results = append(results, d)
	}
	return results, nil
}

// CreateNode creates and returns a new NodeInterface.
func (c *Controller) CreateNode(args CreateNodeArgs) (*Node, error) {
	// There must be at least one mac address.
	if len(args.MACAddresses) == 0 {
		return nil, util.NewBadRequestError("at least one MAC address must be specified")
	}
	params := CreateNodesParams(args)
	result, err := c.Post("nodes", "", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			if svrErr.StatusCode == http.StatusBadRequest {
				return nil, errors.Wrap(err, util.NewBadRequestError(svrErr.BodyMessage))
			}
		}
		// Translate http errors.
		return nil, util.NewUnexpectedError(err)
	}

	var d Node

	iSet := make([]*NetworkInterface, 0)
	err = json.Unmarshal(result, &d)
	if err != nil {
		return nil, err
	}
	d.Controller = c

	for _, i := range d.InterfaceSet {
		i.Controller = c
		iSet = append(iSet, i)
	}

	d.InterfaceSet = iSet

	return &d, nil
}

// Machines returns a list of machines that match the params.
func (c *Controller) Machines(args MachinesArgs) ([]Machine, error) {
	params := MachinesParams(args)
	// At the moment the maas API doesn't support filtering by Owner
	// data so we do that ourselves below.
	source, err := c.Get("machines", "", params.Values)
	if err != nil {
		return nil, util.NewUnexpectedError(err)
	}
	result := make([]Machine, 0)
	var machines []Machine
	err = json.Unmarshal(source, &machines)
	if err != nil {
		return nil, err
	}

	for _, m := range machines {
		if ownerDataMatches(m.OwnerData, args.OwnerData) {
			m.Controller = c

			resultIface := make([]*NetworkInterface, 0)
			for _, i := range m.InterfaceSet {
				i.Controller = c
				resultIface = append(resultIface, i)
			}
			m.InterfaceSet = resultIface

			result = append(result, m)
		}
	}

	return result, nil
}

func ownerDataMatches(ownerData, filter map[string]string) bool {
	for key, value := range filter {
		if ownerData[key] != value {
			return false
		}
	}
	return true
}

// AllocateMachine will attempt to allocate a MachineInterface to the user.
// If successful, the allocated MachineInterface is returned.
// Returns an error that satisfies IsNoMatchError if the requested
// constraints cannot be met.
func (c *Controller) AllocateMachine(args AllocateMachineArgs) (*Machine, ConstraintMatches, error) {
	var matches ConstraintMatches
	params := AllocateMachinesParams(args)
	result, err := c.Post("machines", "allocate", params.Values)
	if err != nil {
		// A 409 Status code is "No Matching Machines"
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			if svrErr.StatusCode == http.StatusConflict {
				return nil, matches, errors.Wrap(err, util.NewNoMatchError(svrErr.BodyMessage))
			}
		}
		// Translate http errors.
		return nil, matches, util.NewUnexpectedError(err)
	}

	var machine *Machine
	var source map[string]interface{}
	err = json.Unmarshal(result, &machine)
	if err != nil {
		return nil, matches, err
	}
	machine.Controller = c

	err = json.Unmarshal(result, &source)
	if err != nil {
		return nil, matches, err
	}

	// Parse the constraint matches.
	matches, err = parseAllocateConstraintsResponse(source, machine)
	if err != nil {
		return nil, matches, err
	}

	return machine, matches, nil
}

// ReleaseMachines will stop the specified machines, and release them
// from the user making them available to be allocated again.
// Release multiple machines at once. Returns
//  - BadRequestError if any of the machines cannot be found
//  - PermissionError if the user does not have permission to release any of the machines
//  - CannotCompleteError if any of the machines could not be released due to their current state
func (c *Controller) ReleaseMachines(args ReleaseMachinesArgs) error {
	params := ReleaseMachinesParams(args)
	_, err := c.Post("machines", "release", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusBadRequest:
				return errors.Wrap(err, util.NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, util.NewPermissionError(svrErr.BodyMessage))
			case http.StatusConflict:
				return errors.Wrap(err, util.NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return util.NewUnexpectedError(err)
	}

	return nil
}

// getFiles returns all the files that match the specified prefix.
func (c *Controller) getFiles(prefix string) ([]File, error) {
	params := util.NewURLParams()
	params.MaybeAdd("prefix", prefix)
	source, err := c.Get("files", "", params.Values)
	if err != nil {
		return nil, util.NewUnexpectedError(err)
	}

	var files []File
	results := make([]File, 0)
	err = json.Unmarshal(source, &files)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		f.Controller = c
		results = append(results, f)
	}
	return results, nil
}

// GetFile returns a single File by its Filename.
func (c *Controller) GetFile(filename string) (*File, error) {
	if filename == "" {
		return nil, errors.NotValidf("missing Filename")
	}
	source, err := c.Get("files/"+filename, "", nil)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			if svrErr.StatusCode == http.StatusNotFound {
				return nil, errors.Wrap(err, util.NewNoMatchError(svrErr.BodyMessage))
			}
		}
		return nil, util.NewUnexpectedError(err)
	}
	var file File
	err = json.Unmarshal(source, &file)
	if err != nil {
		return nil, err
	}
	file.Controller = c
	return &file, nil
}

// AddFile adds or replaces the Content of the specified Filename.
// If or when the maas api is able to return metadata about a single
// File without sending the Content of the File, we can return a FileInterface
// instance here too.
func (c *Controller) AddFile(args AddFileArgs) error {
	if err := args.Validate(); err != nil {
		return err
	}
	fileContent := args.Content
	if fileContent == nil {
		content, err := ioutil.ReadAll(io.LimitReader(args.Reader, args.Length))
		if err != nil {
			return errors.Annotatef(err, "cannot read File Content")
		}
		fileContent = content
	}
	params := url.Values{"Filename": {args.Filename}}
	_, err := c.PostFile("files", "", params, fileContent)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			if svrErr.StatusCode == http.StatusBadRequest {
				return errors.Wrap(err, util.NewBadRequestError(svrErr.BodyMessage))
			}
		}
		return util.NewUnexpectedError(err)
	}
	return nil
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

func parseAllocateConstraintsResponse(source interface{}, machine *Machine) (ConstraintMatches, error) {
	var empty ConstraintMatches
	matchFields := schema.Fields{
		"storage":    schema.StringMap(schema.List(schema.ForceInt())),
		"interfaces": schema.StringMap(schema.List(schema.ForceInt())),
	}
	matchDefaults := schema.Defaults{
		"storage":    schema.Omit,
		"interfaces": schema.Omit,
	}
	fields := schema.Fields{
		"constraints_by_type": schema.FieldMap(matchFields, matchDefaults),
	}
	checker := schema.FieldMap(fields, nil) // no defaults
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return empty, util.WrapWithDeserializationError(err, "allocation constraints response schema check failed")
	}
	valid := coerced.(map[string]interface{})
	constraintsMap := valid["constraints_by_type"].(map[string]interface{})
	result := ConstraintMatches{
		Interfaces: make(map[string][]NetworkInterface),
		Storage:    make(map[string][]BlockDevice),
	}

	if interfaceMatches, found := constraintsMap["interfaces"]; found {
		matches := convertConstraintMatches(interfaceMatches)
		for label, ids := range matches {
			interfaces := make([]NetworkInterface, len(ids))
			for index, id := range ids {
				iface := machine.Interface(id)
				if iface == nil {
					return empty, util.NewDeserializationError("constraint match interface %q: %d does not match an interface for the MachineInterface", label, id)
				}
				interfaces[index] = *iface
			}
			result.Interfaces[label] = interfaces
		}
	}

	if storageMatches, found := constraintsMap["storage"]; found {
		matches := convertConstraintMatches(storageMatches)
		for label, ids := range matches {
			blockDevices := make([]BlockDevice, len(ids))
			for index, id := range ids {
				blockDevice := machine.BlockDevice(id)
				if blockDevice == nil {
					return empty, util.NewDeserializationError("constraint match storage %q: %d does not match a block node for the MachineInterface", label, id)
				}
				blockDevices[index] = *blockDevice
			}
			result.Storage[label] = blockDevices
		}
	}
	return result, nil
}

func convertConstraintMatches(source interface{}) map[string][]int {
	// These casts are all safe because of the schema check.
	result := make(map[string][]int)
	matchMap := source.(map[string]interface{})
	for label, values := range matchMap {
		items := values.([]interface{})
		result[label] = make([]int, len(items))
		for index, value := range items {
			result[label][index] = value.(int)
		}
	}
	return result
}
