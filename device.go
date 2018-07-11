// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	"fmt"
	"net/http"
	"strings"

	"encoding/json"
	"github.com/juju/errors"
)

// Device represents some form of device in MAAS.
type device struct {
	// TODO: add domain
	Controller  *controller
	ResourceURI string `json:"resource_uri,omitempty"`
	SystemID    string `json:"system_id,omitempty"`
	Hostname    string `json:"Hostname,omitempty"`
	FQDN        string `json:"FQDN,omitempty"`
	// Parent returns the SystemID of the Parent. Most often this will be a
	// MachineInterface.
	Parent string `json:"Parent,omitempty"`
	// Owner is the username of the user that created the device.
	Owner       string   `json:"Owner,omitempty"`
	IPAddresses []string `json:"ip_addresses,omitempty"`
	// InterfaceSet returns all the interfaces for the DeviceInterface.
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

// interfacesURI used to add interfaces for this device. The operations
// are on the nodes endpoint, not devices.
func (d *device) interfacesURI() string {
	return strings.Replace(d.ResourceURI, "devices", "nodes", 1) + "interfaces/"
}

// CreateInterface implements DeviceInterface.
func (d *device) CreateInterface(args CreateInterfaceArgs) (MachineNetworkInterface, error) {
	if err := args.Validate(); err != nil {
		return nil, errors.Trace(err)
	}
	params := NewURLParams()
	params.Values.Add("Name", args.Name)
	params.Values.Add("mac_address", args.MACAddress)
	params.Values.Add("VLAN", fmt.Sprint(args.VLAN.ID))
	params.MaybeAdd("Tags", strings.Join(args.Tags, ","))
	params.MaybeAddInt("MTU", args.MTU)
	params.MaybeAddBool("accept_ra", args.AcceptRA)
	params.MaybeAddBool("autoconf", args.Autoconf)
	result, err := d.Controller.post(d.interfacesURI(), "create_physical", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound, http.StatusConflict:
				return nil, errors.Wrap(err, NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return nil, errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			case http.StatusServiceUnavailable:
				return nil, errors.Wrap(err, NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return nil, NewUnexpectedError(err)
	}

	var iface Interface
	err = json.Unmarshal(result, &iface)
	if err != nil {
		return nil, errors.Trace(err)
	}
	iface.Controller = d.Controller

	// TODO: add to the interfaces for the device when the interfaces are returned.
	// lp:bug 1567213.
	return iface, nil
}

// Delete implements DeviceInterface.
func (d *device) Delete() error {
	err := d.Controller.delete(d.ResourceURI)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound:
				return errors.Wrap(err, NewNoMatchError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, NewPermissionError(svrErr.BodyMessage))
			}
		}
		return NewUnexpectedError(err)
	}
	return nil
}

type DeviceInterface interface {
	// CreateInterface will create a physical interface for this MachineInterface.
	CreateInterface(CreateInterfaceArgs) (MachineNetworkInterface, error)
	// Delete will remove this DeviceInterface.
	Delete() error
}
