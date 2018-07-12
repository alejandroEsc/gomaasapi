// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync/atomic"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/schema"
	"github.com/juju/utils/set"
	"github.com/juju/version"
)

var (
	logger = loggo.GetLogger("maas")
	// The supported versions should be ordered from most desirable version to
	// least as they will be tried in order.
	supportedAPIVersions = []string{"2.0"}
	// Each of the api versions that change the request or response structure
	// for any given call should have a value defined for easy definition of
	// the deserialization functions.
	twoDotOh = version.Number{Major: 2, Minor: 0}
	// Current request number. Informational only for logging.
	requestNumber int64
)

// ControllerArgs is an argument struct for passing the required parameters
// to the NewController method.
type ControllerArgs struct {
	BaseURL string
	APIKey  string
}

// NewController creates an authenticated Client to the MAAS API, and
// checks the Capabilities of the server. If the BaseURL specified
// includes the API version, that version of the API will be used,
// otherwise the ControllerInterface will use the highest supported version
// available.
//
// If the APIKey is not valid, a NotValid error is returned.
// If the credentials are incorrect, a PermissionError is returned.
func NewController(args ControllerArgs) (*controller, error) {
	base, apiVersion, includesVersion := SplitVersionedURL(args.BaseURL)
	if includesVersion {
		if !supportedVersion(apiVersion) {
			return nil, NewUnsupportedVersionError("version %s", apiVersion)
		}
		return newControllerWithVersion(base, apiVersion, args.APIKey)
	}
	return newControllerUnknownVersion(args)
}

func supportedVersion(value string) bool {
	for _, version := range supportedAPIVersions {
		if value == version {
			return true
		}
	}
	return false
}

func newControllerWithVersion(baseURL, apiVersion, apiKey string) (*controller, error) {
	major, minor, err := version.ParseMajorMinor(apiVersion)
	// We should not get an error here. See the test.
	if err != nil {
		return nil, errors.Errorf("bad version defined in supported versions: %q", apiVersion)
	}
	client, err := NewAuthenticatedClient(AddAPIVersionToURL(baseURL, apiVersion), apiKey)
	if err != nil {
		// If the credentials aren't valid, return now.
		if errors.IsNotValid(err) {
			return nil, errors.Trace(err)
		}
		// Any other error attempting to create the authenticated Client
		// is an unexpected error and return now.
		return nil, NewUnexpectedError(err)
	}
	controllerVersion := version.Number{
		Major: major,
		Minor: minor,
	}
	controller := &controller{Client: client, APIVersion: controllerVersion}
	controller.Capabilities, err = controller.readAPIVersionInfo()
	if err != nil {
		logger.Debugf("read version failed: %#v", err)
		return nil, errors.Trace(err)
	}

	if err := controller.checkCreds(); err != nil {
		return nil, errors.Trace(err)
	}
	return controller, nil
}

func newControllerUnknownVersion(args ControllerArgs) (*controller, error) {
	// For now we don't need to test multiple versions. It is expected that at
	// some time in the future, we will try the most up to date version and then
	// work our way backwards.
	for _, apiVersion := range supportedAPIVersions {
		controller, err := newControllerWithVersion(args.BaseURL, apiVersion, args.APIKey)
		switch {
		case err == nil:
			return controller, nil
		case IsUnsupportedVersionError(err):
			// This will only come back from readAPIVersionInfo for 410/404.
			continue
		default:
			return nil, errors.Trace(err)
		}
	}

	return nil, NewUnsupportedVersionError("ControllerInterface at %s does not support any of %s", args.BaseURL, supportedAPIVersions)
}

// Controller represents an API connection to a MAAS ControllerInterface. Since the API
// is restful, there is no long held connection to the API server, but instead
// HTTP calls are made and JSON response structures parsed.
type controller struct {
	Client     *Client
	APIVersion version.Number
	// Capabilities returns a set of Capabilities as defined by the string
	// constants.
	Capabilities set.Strings
}

// BootResources implements ControllerInterface.
func (c *controller) BootResources() ([]*bootResource, error) {
	source, err := c.get("boot-resources")
	if err != nil {
		return nil, NewUnexpectedError(err)
	}

	var resources []*bootResource
	err = json.Unmarshal(source, &resources)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return resources, nil
}

// Fabrics returns the list of Fabrics defined in the MAAS ControllerInterface.
func (c *controller) Fabrics() ([]fabric, error) {
	source, err := c.get("fabrics")
	if err != nil {
		return nil, NewUnexpectedError(err)
	}

	var fabrics []fabric
	err = json.Unmarshal(source, &fabrics)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return fabrics, nil
}

// Spaces returns the list of Spaces defined in the MAAS ControllerInterface.
func (c *controller) Spaces() ([]space, error) {
	source, err := c.get("spaces")
	if err != nil {
		return nil, NewUnexpectedError(err)
	}

	var spaces []space
	err = json.Unmarshal(source, &spaces)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return spaces, nil
}

// StaticRoutes returns the list of StaticRoutes defined in the MAAS ControllerInterface.
func (c *controller) StaticRoutes() ([]staticRoute, error) {
	source, err := c.get("static-routes")
	if err != nil {
		return nil, NewUnexpectedError(err)
	}
	var staticRoutes []staticRoute
	err = json.Unmarshal(source, &staticRoutes)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return staticRoutes, nil
}

// Zones lists all the zones known to the MAAS ControllerInterface.
func (c *controller) Zones() ([]zone, error) {
	source, err := c.get("zones")
	if err != nil {
		return nil, NewUnexpectedError(err)
	}
	var zones []zone
	err = json.Unmarshal(source, &zones)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return zones, nil
}

// DevicesArgs is a argument struct for selecting Devices.
// Only devices that match the specified criteria are returned.
type DevicesArgs struct {
	Hostname     []string
	MACAddresses []string
	SystemIDs    []string
	Domain       string
	Zone         string
	AgentName    string
}

// Devices returns a list of devices that match the params.
func (c *controller) Devices(args DevicesArgs) ([]device, error) {
	params := NewURLParams()
	params.MaybeAddMany("Hostname", args.Hostname)
	params.MaybeAddMany("mac_address", args.MACAddresses)
	params.MaybeAddMany("ID", args.SystemIDs)
	params.MaybeAdd("domain", args.Domain)
	params.MaybeAdd("Zone", args.Zone)
	params.MaybeAdd("agent_name", args.AgentName)
	source, err := c.getQuery("devices", params.Values)
	if err != nil {
		return nil, NewUnexpectedError(err)
	}

	var devices []device
	err = json.Unmarshal(source, &devices)
	if err != nil {
		return nil, errors.Trace(err)
	}
	for _, d := range devices {
		d.Controller = c
	}
	return devices, nil
}

// CreateDeviceArgs is a argument struct for passing information into CreateDevice.
type CreateDeviceArgs struct {
	Hostname     string
	MACAddresses []string
	Domain       string
	Parent       string
}

// CreateDevice creates and returns a new DeviceInterface.
func (c *controller) CreateDevice(args CreateDeviceArgs) (*device, error) {
	// There must be at least one mac address.
	if len(args.MACAddresses) == 0 {
		return nil, NewBadRequestError("at least one MAC address must be specified")
	}
	params := NewURLParams()
	params.MaybeAdd("Hostname", args.Hostname)
	params.MaybeAdd("domain", args.Domain)
	params.MaybeAddMany("mac_addresses", args.MACAddresses)
	params.MaybeAdd("Parent", args.Parent)
	result, err := c.post("devices", "", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			if svrErr.StatusCode == http.StatusBadRequest {
				return nil, errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			}
		}
		// Translate http errors.
		return nil, NewUnexpectedError(err)
	}

	var d device
	err = json.Unmarshal(result, &d)
	if err != nil {
		return nil, errors.Trace(err)
	}
	d.Controller = c
	return &d, nil
}

// MachinesArgs is a argument struct for selecting Machines.
// Only machines that match the specified criteria are returned.
type MachinesArgs struct {
	Hostnames    []string
	MACAddresses []string
	SystemIDs    []string
	Domain       string
	Zone         string
	AgentName    string
	OwnerData    map[string]string
}

// Machines returns a list of machines that match the params.
func (c *controller) Machines(args MachinesArgs) ([]Machine, error) {
	params := NewURLParams()
	params.MaybeAddMany("Hostname", args.Hostnames)
	params.MaybeAddMany("mac_address", args.MACAddresses)
	params.MaybeAddMany("ID", args.SystemIDs)
	params.MaybeAdd("domain", args.Domain)
	params.MaybeAdd("Zone", args.Zone)
	params.MaybeAdd("agent_name", args.AgentName)
	// At the moment the MAAS API doesn't support filtering by Owner
	// data so we do that ourselves below.
	source, err := c.getQuery("machines", params.Values)
	if err != nil {
		return nil, NewUnexpectedError(err)
	}
	var machines []Machine
	err = json.Unmarshal(source, &machines)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var result []Machine
	for _, m := range machines {
		m.Controller = c
		if ownerDataMatches(m.OwnerData, args.OwnerData) {
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

// StorageSpec represents one element of storage constraints necessary
// to be satisfied to allocate a MachineInterface.
type StorageSpec struct {
	// Label is optional and an arbitrary string. Labels need to be unique
	// across the StorageSpec elements specified in the AllocateMachineArgs.
	Label string
	// Size is required and refers to the required minimum Size in GB.
	Size int
	// Zero or more Tags assocated to with the disks.
	Tags []string
}

// Validate ensures that there is a positive Size and that there are no Empty
// tag Values.
func (s *StorageSpec) Validate() error {
	if s.Size <= 0 {
		return errors.NotValidf("Size value %d", s.Size)
	}
	for _, v := range s.Tags {
		if v == "" {
			return errors.NotValidf("empty tag")
		}
	}
	return nil
}

// String returns the string representation of the storage spec.
func (s *StorageSpec) String() string {
	label := s.Label
	if label != "" {
		label += ":"
	}
	tags := strings.Join(s.Tags, ",")
	if tags != "" {
		tags = "(" + tags + ")"
	}
	return fmt.Sprintf("%s%d%s", label, s.Size, tags)
}

// InterfaceSpec represents one elemenet of network related constraints.
type InterfaceSpec struct {
	// Label is required and an arbitrary string. Labels need to be unique
	// across the InterfaceSpec elements specified in the AllocateMachineArgs.
	// The Label is returned in the ConstraintMatches response from
	// AllocateMachine.
	Label string
	Space string

	// NOTE: there are other interface spec Values that we are not exposing at
	// this stage that can be added on an as needed basis. Other possible Values are:
	//     'fabric_class', 'not_fabric_class',
	//     'subnet_cidr', 'not_subnet_cidr',
	//     'VID', 'not_vid',
	//     'Fabric', 'not_fabric',
	//     'Subnet', 'not_subnet',
	//     'Mode'
}

// Validate ensures that a Label is specified and that there is at least one
// Space or NotSpace value set.
func (a *InterfaceSpec) Validate() error {
	if a.Label == "" {
		return errors.NotValidf("missing Label")
	}
	// Perhaps at some stage in the future there will be other possible specs
	// supported (like VID, Subnet, etc), but until then, just space to check.
	if a.Space == "" {
		return errors.NotValidf("empty Space constraint")
	}
	return nil
}

// String returns the interface spec as MaaS requires it.
func (a *InterfaceSpec) String() string {
	return fmt.Sprintf("%s:space=%s", a.Label, a.Space)
}

// AllocateMachineArgs is an argument struct for passing args into MachineInterface.Allocate.
type AllocateMachineArgs struct {
	Hostname     string
	SystemId     string
	Architecture string
	MinCPUCount  int
	// MinMemory represented in MB.
	MinMemory int
	Tags      []string
	NotTags   []string
	Zone      string
	NotInZone []string
	// Storage represents the required disks on the MachineInterface. If any are specified
	// the first value is used for the root disk.
	Storage []StorageSpec
	// Interfaces represents a number of required interfaces on the MachineInterface.
	// Each InterfaceSpec relates to an individual network interface.
	Interfaces []InterfaceSpec
	// NotSpace is a MachineInterface level constraint, and applies to the entire MachineInterface
	// rather than specific interfaces.
	NotSpace  []string
	AgentName string
	Comment   string
	DryRun    bool
}

// Validate makes sure that any labels specifed in Storage or Interfaces
// are unique, and that the required specifications are valid.
func (a *AllocateMachineArgs) Validate() error {
	storageLabels := set.NewStrings()
	for _, spec := range a.Storage {
		if err := spec.Validate(); err != nil {
			return errors.Annotate(err, "Storage")
		}
		if spec.Label != "" {
			if storageLabels.Contains(spec.Label) {
				return errors.NotValidf("reusing storage Label %q", spec.Label)
			}
			storageLabels.Add(spec.Label)
		}
	}
	interfaceLabels := set.NewStrings()
	for _, spec := range a.Interfaces {
		if err := spec.Validate(); err != nil {
			return errors.Annotate(err, "Interfaces")
		}
		if interfaceLabels.Contains(spec.Label) {
			return errors.NotValidf("reusing interface Label %q", spec.Label)
		}
		interfaceLabels.Add(spec.Label)
	}
	for _, v := range a.NotSpace {
		if v == "" {
			return errors.NotValidf("empty NotSpace constraint")
		}
	}
	return nil
}

func (a *AllocateMachineArgs) storage() string {
	var values []string
	for _, spec := range a.Storage {
		values = append(values, spec.String())
	}
	return strings.Join(values, ",")
}

func (a *AllocateMachineArgs) interfaces() string {
	var values []string
	for _, spec := range a.Interfaces {
		values = append(values, spec.String())
	}
	return strings.Join(values, ";")
}

func (a *AllocateMachineArgs) notSubnets() []string {
	var values []string
	for _, v := range a.NotSpace {
		values = append(values, "space:"+v)
	}
	return values
}

// ConstraintMatches provides a way for the caller of AllocateMachine to determine
//.how the allocated MachineInterface matched the storage and interfaces constraints specified.
// The labels that were used in the constraints are the keys in the maps.
type ConstraintMatches struct {
	// MachineNetworkInterface is a mapping of the constraint Label specified to the Interfaces
	// that match that constraint.
	Interfaces map[string][]MachineNetworkInterface

	// Storage is a mapping of the constraint Label specified to the BlockDevices
	// that match that constraint.
	Storage map[string][]BlockDevice
}

// AllocateMachine will attempt to allocate a MachineInterface to the user.
// If successful, the allocated MachineInterface is returned.
// Returns an error that satisfies IsNoMatchError if the requested
// constraints cannot be met.
func (c *controller) AllocateMachine(args AllocateMachineArgs) (*Machine, ConstraintMatches, error) {
	var matches ConstraintMatches
	params := NewURLParams()
	params.MaybeAdd("Name", args.Hostname)
	params.MaybeAdd("system_id", args.SystemId)
	params.MaybeAdd("arch", args.Architecture)
	params.MaybeAddInt("cpu_count", args.MinCPUCount)
	params.MaybeAddInt("mem", args.MinMemory)
	params.MaybeAddMany("Tags", args.Tags)
	params.MaybeAddMany("not_tags", args.NotTags)
	params.MaybeAdd("storage", args.storage())
	params.MaybeAdd("interfaces", args.interfaces())
	params.MaybeAddMany("not_subnets", args.notSubnets())
	params.MaybeAdd("Zone", args.Zone)
	params.MaybeAddMany("not_in_zone", args.NotInZone)
	params.MaybeAdd("agent_name", args.AgentName)
	params.MaybeAdd("comment", args.Comment)
	params.MaybeAddBool("dry_run", args.DryRun)
	result, err := c.post("machines", "allocate", params.Values)
	if err != nil {
		// A 409 Status code is "No Matching Machines"
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			if svrErr.StatusCode == http.StatusConflict {
				return nil, matches, errors.Wrap(err, NewNoMatchError(svrErr.BodyMessage))
			}
		}
		// Translate http errors.
		return nil, matches, NewUnexpectedError(err)
	}

	var machine *Machine
	err = json.Unmarshal(result, &machine)
	if err != nil {
		return nil, matches, errors.Trace(err)
	}
	machine.Controller = c

	// Parse the constraint matches.
	matches, err = parseAllocateConstraintsResponse(result, machine)
	if err != nil {
		return nil, matches, errors.Trace(err)
	}

	return machine, matches, nil
}

// ReleaseMachinesArgs is an argument struct for passing the MachineInterface system IDs
// and an optional comment into the ReleaseMachines method.
type ReleaseMachinesArgs struct {
	SystemIDs []string
	Comment   string
}

// ReleaseMachines will stop the specified machines, and release them
// from the user making them available to be allocated again.
// Release multiple machines at once. Returns
//  - BadRequestError if any of the machines cannot be found
//  - PermissionError if the user does not have permission to release any of the machines
//  - CannotCompleteError if any of the machines could not be released due to their current state
func (c *controller) ReleaseMachines(args ReleaseMachinesArgs) error {
	params := NewURLParams()
	params.MaybeAddMany("machines", args.SystemIDs)
	params.MaybeAdd("comment", args.Comment)
	_, err := c.post("machines", "release", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusBadRequest:
				return errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			case http.StatusConflict:
				return errors.Wrap(err, NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}

	return nil
}

// Files returns all the files that match the specified prefix.
func (c *controller) Files(prefix string) ([]File, error) {
	params := NewURLParams()
	params.MaybeAdd("prefix", prefix)
	source, err := c.getQuery("files", params.Values)
	if err != nil {
		return nil, NewUnexpectedError(err)
	}

	var files []File
	err = json.Unmarshal(source, &files)
	if err != nil {
		return nil, errors.Trace(err)
	}

	for _, f := range files {
		f.Controller = c
	}

	return files, nil
}

// GetFile returns a single File by its Filename.
func (c *controller) GetFile(filename string) (*File, error) {
	if filename == "" {
		return nil, errors.NotValidf("missing Filename")
	}
	source, err := c.get("files/" + filename)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			if svrErr.StatusCode == http.StatusNotFound {
				return nil, errors.Wrap(err, NewNoMatchError(svrErr.BodyMessage))
			}
		}
		return nil, NewUnexpectedError(err)
	}
	var file File
	err = json.Unmarshal(source, &file)
	if err != nil {
		return nil, errors.Trace(err)
	}
	file.Controller = c
	return &file, nil
}

// AddFileArgs is a argument struct for passing information into AddFile.
// One of Content or (Reader, Length) must be specified.
type AddFileArgs struct {
	Filename string
	Content  []byte
	Reader   io.Reader
	Length   int64
}

// Validate checks to make sure the Filename has no slashes, and that one of
// Content or (Reader, Length) is specified.
func (a *AddFileArgs) Validate() error {
	dir, _ := path.Split(a.Filename)
	if dir != "" {
		return errors.NotValidf("paths in Filename %q", a.Filename)
	}
	if a.Filename == "" {
		return errors.NotValidf("missing Filename")
	}
	if a.Content == nil {
		if a.Reader == nil {
			return errors.NotValidf("missing Content or Reader")
		}
		if a.Length == 0 {
			return errors.NotValidf("missing Length")
		}
	} else {
		if a.Reader != nil {
			return errors.NotValidf("specifying Content and Reader")
		}
		if a.Length != 0 {
			return errors.NotValidf("specifying Length and Content")
		}
	}
	return nil
}

// AddFile adds or replaces the Content of the specified Filename.
// If or when the MAAS api is able to return metadata about a single
// File without sending the Content of the File, we can return a FileInterface
// instance here too.
func (c *controller) AddFile(args AddFileArgs) error {
	if err := args.Validate(); err != nil {
		return errors.Trace(err)
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
	_, err := c.postFile("files", "", params, fileContent)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			if svrErr.StatusCode == http.StatusBadRequest {
				return errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}
	return nil
}

func (c *controller) checkCreds() error {
	if _, err := c.getOp("users", "whoami"); err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			if svrErr.StatusCode == http.StatusUnauthorized {
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}
	return nil
}

func (c *controller) put(path string, params url.Values) ([]byte, error) {
	path = EnsureTrailingSlash(path)
	requestID := nextRequestID()
	logger.Tracef("request %x: PUT %s%s, params: %s", requestID, c.Client.APIURL, path, params.Encode())
	bytes, err := c.Client.Put(&url.URL{Path: path}, params)
	if err != nil {
		logger.Tracef("response %x: error: %q", requestID, err.Error())
		logger.Tracef("error detail: %#v", err)
		return nil, errors.Trace(err)
	}
	return bytes, nil
}

func (c *controller) post(path, op string, params url.Values) ([]byte, error) {
	bytes, err := c._postRaw(path, op, params, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return bytes, nil
}

func (c *controller) postFile(path, op string, params url.Values, fileContent []byte) ([]byte, error) {
	// Only one File is ever sent at a time.
	files := map[string][]byte{"File": fileContent}
	return c._postRaw(path, op, params, files)
}

func (c *controller) _postRaw(path, op string, params url.Values, files map[string][]byte) ([]byte, error) {
	path = EnsureTrailingSlash(path)
	requestID := nextRequestID()
	if logger.IsTraceEnabled() {
		opArg := ""
		if op != "" {
			opArg = "?op=" + op
		}
		logger.Tracef("request %x: POST %s%s%s, params=%s", requestID, c.Client.APIURL, path, opArg, params.Encode())
	}
	bytes, err := c.Client.Post(&url.URL{Path: path}, op, params, files)
	if err != nil {
		logger.Tracef("response %x: error: %q", requestID, err.Error())
		logger.Tracef("error detail: %#v", err)
		return nil, errors.Trace(err)
	}
	logger.Tracef("response %x: %s", requestID, string(bytes))
	return bytes, nil
}

func (c *controller) delete(path string) error {
	path = EnsureTrailingSlash(path)
	requestID := nextRequestID()
	logger.Tracef("request %x: DELETE %s%s", requestID, c.Client.APIURL, path)
	err := c.Client.Delete(&url.URL{Path: path})
	if err != nil {
		logger.Tracef("response %x: error: %q", requestID, err.Error())
		logger.Tracef("error detail: %#v", err)
		return errors.Trace(err)
	}
	logger.Tracef("response %x: complete", requestID)
	return nil
}

func (c *controller) getQuery(path string, params url.Values) ([]byte, error) {
	return c._get(path, "", params)
}

func (c *controller) get(path string) ([]byte, error) {
	return c._get(path, "", nil)
}

func (c *controller) getOp(path, op string) ([]byte, error) {
	return c._get(path, op, nil)
}

func (c *controller) _get(path, op string, params url.Values) ([]byte, error) {
	bytes, err := c._getRaw(path, op, params)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return bytes, nil
}

func (c *controller) _getRaw(path, op string, params url.Values) ([]byte, error) {
	path = EnsureTrailingSlash(path)
	requestID := nextRequestID()
	if logger.IsTraceEnabled() {
		var query string
		if params != nil {
			query = "?" + params.Encode()
		}
		logger.Tracef("request %x: GET %s%s%s", requestID, c.Client.APIURL, path, query)
	}
	bytes, err := c.Client.Get(&url.URL{Path: path}, op, params)
	if err != nil {
		logger.Tracef("response %x: error: %q", requestID, err.Error())
		logger.Tracef("error detail: %#v", err)
		return nil, errors.Trace(err)
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
	if serverErr, ok := errors.Cause(err).(ServerError); ok {
		code := serverErr.StatusCode
		return code == http.StatusNotFound || code == http.StatusGone
	}
	// Workaround for bug in MAAS 1.9.4 - instead of a 404 we get a
	// redirect to the HTML login page, which doesn't parse as JSON.
	// https://bugs.launchpad.net/maas/+bug/1583715
	if syntaxErr, ok := errors.Cause(err).(*json.SyntaxError); ok {
		message := "invalid character '<' looking for beginning of value"
		return syntaxErr.Offset == 1 && syntaxErr.Error() == message
	}
	return false
}

func (c *controller) readAPIVersionInfo() (set.Strings, error) {
	parsed, err := c.get("version")
	if indicatesUnsupportedVersion(err) {
		return nil, WrapWithUnsupportedVersionError(err)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	// As we care about other fields, add them.
	fields := schema.Fields{
		"Capabilities": schema.List(schema.String()),
	}
	checker := schema.FieldMap(fields, nil) // no defaults
	coerced, err := checker.Coerce(parsed, nil)
	if err != nil {
		return nil, WrapWithDeserializationError(err, "version response")
	}
	// For now, we don't append any subversion, but as it becomes used, we
	// should parse and check.

	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.
	capabilities := set.NewStrings()
	capabilityValues := valid["Capabilities"].([]interface{})
	for _, value := range capabilityValues {
		capabilities.Add(value.(string))
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
		return empty, WrapWithDeserializationError(err, "allocation constraints response schema check failed")
	}
	valid := coerced.(map[string]interface{})
	constraintsMap := valid["constraints_by_type"].(map[string]interface{})
	result := ConstraintMatches{
		Interfaces: make(map[string][]MachineNetworkInterface),
		Storage:    make(map[string][]BlockDevice),
	}

	if interfaceMatches, found := constraintsMap["interfaces"]; found {
		matches := convertConstraintMatches(interfaceMatches)
		for label, ids := range matches {
			interfaces := make([]MachineNetworkInterface, len(ids))
			for index, id := range ids {
				iface := machine.Interface(id)
				if iface == nil {
					return empty, NewDeserializationError("constraint match interface %q: %d does not match an interface for the MachineInterface", label, id)
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
					return empty, NewDeserializationError("constraint match storage %q: %d does not match a block device for the MachineInterface", label, id)
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
