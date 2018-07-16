package node

import "github.com/juju/gomaasapi/pkg/api/util"

// CreateNodeArgs is a argument struct for passing information into CreateNode.
type CreateNodeArgs struct {
	Hostname     string
	MACAddresses []string
	Domain       string
	Parent       string
}

// NodesArgs is a argument struct for selecting Nodes.
// Only devices that match the specified criteria are returned.
type NodesArgs struct {
	Hostname     []string
	MACAddresses []string
	SystemIDs    []string
	Domain       string
	Zone         string
	AgentName    string
}

func NodesParams(args NodesArgs) *util.URLParams {
	params := util.NewURLParams()
	params.MaybeAddMany("hostname", args.Hostname)
	params.MaybeAddMany("mac_address", args.MACAddresses)
	params.MaybeAddMany("id", args.SystemIDs)
	params.MaybeAdd("domain", args.Domain)
	params.MaybeAdd("zone", args.Zone)
	params.MaybeAdd("agent_name", args.AgentName)
	return params
}

func CreateNodesParams(args CreateNodeArgs) *util.URLParams {
	params := util.NewURLParams()
	params.MaybeAdd("Hostname", args.Hostname)
	params.MaybeAdd("domain", args.Domain)
	params.MaybeAddMany("mac_addresses", args.MACAddresses)
	params.MaybeAdd("Parent", args.Parent)
	return params
}
