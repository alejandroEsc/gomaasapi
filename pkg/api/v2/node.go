// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"fmt"
	"net/http"
	"strings"

	"encoding/json"

	"github.com/juju/errors"
	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/util"
)

// Nodes are now known as nodes...? should reconsider this struct.
// Device represents some form of node in MAAS.
type node struct {
	// TODO: add domain
	Controller  *controller `json:"-"`
	ResourceURI string      `json:"resource_uri,omitempty"`
	SystemID    string      `json:"system_id,omitempty"`
	Hostname    string      `json:"Hostname,omitempty"`
	FQDN        string      `json:"FQDN,omitempty"`
	// Parent returns the SystemID of the Parent. Most often this will be a
	// MachineInterface.
	Parent string `json:"Parent,omitempty"`
	// Owner is the username of the user that created the node.
	Owner       string   `json:"Owner,omitempty"`
	IPAddresses []string `json:"ip_addresses,omitempty"`
	// InterfaceSet returns all the interfaces for the NodeInterface.
	InterfaceSet []*MachineNetworkInterface `json:"interface_set,omitempty"`
	Zone         *zone                      `json:"Zone,omitempty"`
}

// CreateInterfaceArgs is an argument struct for passing parameters to
// the MachineInterface.CreateInterface method.
type CreateInterfaceArgs struct {
	// Name of the interface (required).
	Name string
	// MACAddress is the MAC address of the interface (required).
	MACAddress string
	// VLAN is the untagged VLAN the interface is connected to (required).
	VLAN *vlan
	// Tags to attach to the interface (optional).
	Tags []string
	// MTU - Maximum transmission unit. (optional)
	MTU int
	// AcceptRA - Accept router advertisements. (IPv6 only)
	AcceptRA bool
	// Autoconf - Perform stateless autoconfiguration. (IPv6 only)
	Autoconf bool
}

// Validate checks the required fields are set for the arg structure.
func (a *CreateInterfaceArgs) Validate() error {
	if a.Name == "" {
		return errors.NotValidf("missing Name")
	}
	if a.MACAddress == "" {
		return errors.NotValidf("missing MACAddress")
	}
	if a.VLAN == nil {
		return errors.NotValidf("missing VLAN")
	}
	return nil
}

// interfacesURI used to add interfaces for this node. The operations
// are on the nodes endpoint, not devices.
func (d *node) interfacesURI() string {
	return strings.Replace(d.ResourceURI, "devices", "nodes", 1) + "interfaces/"
}

// CreateInterface implements NodeInterface.
func (d *node) CreateInterface(args CreateInterfaceArgs) (*MachineNetworkInterface, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}
	params := util.NewURLParams()
	params.Values.Add("Name", args.Name)
	params.Values.Add("mac_address", args.MACAddress)
	params.Values.Add("VLAN", fmt.Sprint(args.VLAN.ID))
	params.MaybeAdd("Tags", strings.Join(args.Tags, ","))
	params.MaybeAddInt("MTU", args.MTU)
	params.MaybeAddBool("accept_ra", args.AcceptRA)
	params.MaybeAddBool("autoconf", args.Autoconf)

	uri := d.interfacesURI()
	result, err := d.Controller.post(uri, "create_physical", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound, http.StatusConflict:
				return nil, errors.Wrap(err, util.NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return nil, errors.Wrap(err, util.NewPermissionError(svrErr.BodyMessage))
			case http.StatusServiceUnavailable:
				return nil, errors.Wrap(err, util.NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return nil, util.NewUnexpectedError(err)
	}

	var iface MachineNetworkInterface
	err = json.Unmarshal(result, &iface)
	if err != nil {
		return nil, err
	}
	iface.Controller = d.Controller
	d.InterfaceSet = append(d.InterfaceSet, &iface)
	return &iface, nil
}

// Delete implements NodeInterface.
func (d *node) Delete() error {
	err := d.Controller.delete(d.ResourceURI)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound:
				return errors.Wrap(err, util.NewNoMatchError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, util.NewPermissionError(svrErr.BodyMessage))
			}
		}
		return util.NewUnexpectedError(err)
	}
	return nil
}

type NodeInterface interface {
	// CreateInterface will create a physical interface for this MachineInterface.
	CreateInterface(CreateInterfaceArgs) (*MachineNetworkInterface, error)
	// Delete will remove this NodeInterface.
	Delete() error
}
