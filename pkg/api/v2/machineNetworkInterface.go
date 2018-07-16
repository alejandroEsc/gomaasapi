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

// MachineNetworkInterface represents a physical or virtual network interface on a MachineInterface.
type MachineNetworkInterface struct {
	Controller   *controller `json:"-"`
	ResourceURI  string      `json:"resource_uri,omitempty"`
	ID           int         `json:"ID,omitempty"`
	Name         string      `json:"Name,omitempty"`
	Type         string      `json:"type,omitempty"`
	Enabled      bool        `json:"Enabled,omitempty"`
	Tags         []string    `json:"Tags,omitempty"`
	VLAN         *vlan       `json:"VLAN,omitempty"`
	Links        []*link     `json:"Links,omitempty"`
	MACAddress   string      `json:"mac_address,omitempty"`
	EffectiveMTU int         `json:"effective_mtu,omitempty"`
	Parents      []string    `json:"Parents,omitempty"`
	Children     []string    `json:"Children,omitempty"`
}

func (i *MachineNetworkInterface) updateFrom(other *MachineNetworkInterface) {
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

// UpdateInterfaceArgs is an argument struct for calling MachineNetworkInterface.Update.
type UpdateInterfaceArgs struct {
	Name       string
	MACAddress string
	VLAN       *vlan
}

func (a *UpdateInterfaceArgs) vlanID() int {
	if a.VLAN == nil {
		return 0
	}
	return a.VLAN.ID
}

// Update the Name, mac address or VLAN.
func (i *MachineNetworkInterface) Update(args UpdateInterfaceArgs) error {
	var empty UpdateInterfaceArgs

	if args == empty {
		return nil
	}

	params := util.NewURLParams()
	params.MaybeAdd("Name", args.Name)
	params.MaybeAdd("mac_address", args.MACAddress)
	params.MaybeAddInt("VLAN", args.vlanID())

	source, err := i.Controller.put(i.ResourceURI, params.Values)
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

	var response MachineNetworkInterface
	err = json.Unmarshal(source, &response)
	if err != nil {
		return errors.Trace(err)
	}
	i.updateFrom(&response)
	return nil
}

// Delete this interface.
func (i *MachineNetworkInterface) Delete() error {
	err := i.Controller.delete(i.ResourceURI)
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

// InterfaceLinkMode is the type of the various link Mode constants used for
// LinkSubnetArgs.
type InterfaceLinkMode string



// LinkSubnet will attempt to make this interface available on the specified
// Subnet.
func (i *MachineNetworkInterface) LinkSubnet(args LinkSubnetArgs) error {
	if err := args.Validate(); err != nil {
		return errors.Trace(err)
	}
	params := util.NewURLParams()
	params.Values.Add("Mode", string(args.Mode))
	params.Values.Add("Subnet", fmt.Sprint(args.Subnet.ID))
	params.MaybeAdd("ip_address", args.IPAddress)
	params.MaybeAddBool("default_gateway", args.DefaultGateway)
	source, err := i.Controller.post(i.ResourceURI, "link_subnet", params.Values)
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

	var response MachineNetworkInterface
	err = json.Unmarshal(source, &response)
	if err != nil {
		return errors.Trace(err)
	}

	i.updateFrom(&response)
	return nil
}

func (i *MachineNetworkInterface) linkForSubnet(st *subnet) *link {
	for _, link := range i.Links {
		if s := link.Subnet; s != nil && s.ID == st.ID {
			return link
		}
	}
	return nil
}

// UnlinkSubnet will remove the Link to the Subnet, and release the IP
// address associated if there is one.
func (i *MachineNetworkInterface) UnlinkSubnet(s *subnet) error {
	if s == nil {
		return errors.NotValidf("missing Subnet")
	}
	link := i.linkForSubnet(s)
	if link == nil {
		return errors.NotValidf("unlinked Subnet")
	}
	params := util.NewURLParams()
	params.Values.Add("ID", fmt.Sprint(link.ID))
	source, err := i.Controller.post(i.ResourceURI, "unlink_subnet", params.Values)
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

	var response MachineNetworkInterface
	err = json.Unmarshal(source, &response)
	if err != nil {
		return errors.Trace(err)
	}

	i.updateFrom(&response)

	return nil
}
