// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

import (
	"fmt"
	"net/http"

	"encoding/json"

	"github.com/juju/errors"
	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/util"
)

// NetworkInterface represents a physical or virtual network interface on a MachineInterface.
type NetworkInterface struct {
	Controller   *Controller `json:"-"`
	ResourceURI  string      `json:"resource_uri,omitempty"`
	ID           int         `json:"ID,omitempty"`
	Name         string      `json:"Name,omitempty"`
	Type         string      `json:"type,omitempty"`
	Enabled      bool        `json:"Enabled,omitempty"`
	Tags         []string    `json:"Tags,omitempty"`
	VLAN         *VLAN       `json:"VLAN,omitempty"`
	Links        []*Link     `json:"Links,omitempty"`
	MACAddress   string      `json:"mac_address,omitempty"`
	EffectiveMTU int         `json:"effective_mtu,omitempty"`
	Parents      []string    `json:"Parents,omitempty"`
	Children     []string    `json:"Children,omitempty"`
}

func (i *NetworkInterface) updateFrom(other *NetworkInterface) {
	i.ResourceURI = other.ResourceURI
	i.ID = other.ID
	i.Name = other.Name
	i.Type = other.Type
	i.Enabled = other.Enabled
	i.Tags = other.Tags
	i.VLAN = other.VLAN
	i.Links = other.Links
	i.MACAddress = other.MACAddress
	i.EffectiveMTU = other.EffectiveMTU
	i.Parents = other.Parents
	i.Children = other.Children
}

// UpdateInterfaceArgs is an argument struct for calling NetworkInterface.Update.
type UpdateInterfaceArgs struct {
	Name       string
	MACAddress string
	VLAN       *VLAN
}

func (a *UpdateInterfaceArgs) vlanID() int {
	if a.VLAN == nil {
		return 0
	}
	return a.VLAN.ID
}

// Update the Name, mac address or VLAN.
func (i *NetworkInterface) Update(args UpdateInterfaceArgs) error {
	var empty UpdateInterfaceArgs

	if args == empty {
		return nil
	}

	params := util.NewURLParams()
	params.MaybeAdd("Name", args.Name)
	params.MaybeAdd("mac_address", args.MACAddress)
	params.MaybeAddInt("VLAN", args.vlanID())

	source, err := i.Controller.Put(i.ResourceURI, params.Values)
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

	var response NetworkInterface
	err = json.Unmarshal(source, &response)
	if err != nil {
		return errors.Trace(err)
	}
	i.updateFrom(&response)
	return nil
}

// Delete this interface.
func (i *NetworkInterface) Delete() error {
	err := i.Controller.Delete(i.ResourceURI)
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

// InterfaceLinkMode is the type of the various Link Mode constants used for
// LinkSubnetArgs.
type InterfaceLinkMode string

// LinkSubnet will attempt to make this interface available on the specified
// Subnet.
func (i *NetworkInterface) LinkSubnet(args LinkSubnetArgs) error {
	if err := args.Validate(); err != nil {
		return errors.Trace(err)
	}
	params := util.NewURLParams()
	params.Values.Add("Mode", string(args.Mode))
	params.Values.Add("Subnet", fmt.Sprint(args.Subnet.ID))
	params.MaybeAdd("ip_address", args.IPAddress)
	params.MaybeAddBool("default_gateway", args.DefaultGateway)
	source, err := i.Controller.Post(i.ResourceURI, "link_subnet", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound, http.StatusBadRequest:
				return errors.Wrap(err, util.NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, util.NewPermissionError(svrErr.BodyMessage))
			case http.StatusServiceUnavailable:
				return errors.Wrap(err, util.NewCannotCompleteError(svrErr.BodyMessage))
			}
		}
		return util.NewUnexpectedError(err)
	}

	var response NetworkInterface
	err = json.Unmarshal(source, &response)
	if err != nil {
		return errors.Trace(err)
	}

	i.updateFrom(&response)
	return nil
}

func (i *NetworkInterface) linkForSubnet(st *Subnet) *Link {
	for _, link := range i.Links {
		if s := link.Subnet; s != nil && s.ID == st.ID {
			return link
		}
	}
	return nil
}

// UnlinkSubnet will remove the Link to the Subnet, and release the IP
// address associated if there is one.
func (i *NetworkInterface) UnlinkSubnet(s *Subnet) error {
	if s == nil {
		return errors.NotValidf("missing Subnet")
	}
	link := i.linkForSubnet(s)
	if link == nil {
		return errors.NotValidf("unlinked Subnet")
	}
	params := util.NewURLParams()
	params.Values.Add("ID", fmt.Sprint(link.ID))
	source, err := i.Controller.Post(i.ResourceURI, "unlink_subnet", params.Values)
	if err != nil {
		if svrErr, ok := errors.Cause(err).(client.ServerError); ok {
			switch svrErr.StatusCode {
			case http.StatusNotFound, http.StatusBadRequest:
				return errors.Wrap(err, util.NewBadRequestError(svrErr.BodyMessage))
			case http.StatusForbidden:
				return errors.Wrap(err, util.NewPermissionError(svrErr.BodyMessage))
			}
		}
		return util.NewUnexpectedError(err)
	}

	var response NetworkInterface
	err = json.Unmarshal(source, &response)
	if err != nil {
		return errors.Trace(err)
	}

	i.updateFrom(&response)

	return nil
}
