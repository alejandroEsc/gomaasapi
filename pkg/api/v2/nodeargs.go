package maasapiv2

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

func nodesParams(args NodesArgs) *util.URLParams {
	params := util.NewURLParams()
	params.MaybeAddMany("Hostname", args.Hostname)
	params.MaybeAddMany("mac_address", args.MACAddresses)
	params.MaybeAddMany("ID", args.SystemIDs)
	params.MaybeAdd("domain", args.Domain)
	params.MaybeAdd("Zone", args.Zone)
	params.MaybeAdd("agent_name", args.AgentName)
	return params
}

func createNodesParams(args CreateNodeArgs) *util.URLParams {
	params := util.NewURLParams()
	params.MaybeAdd("Hostname", args.Hostname)
	params.MaybeAdd("domain", args.Domain)
	params.MaybeAddMany("mac_addresses", args.MACAddresses)
	params.MaybeAdd("Parent", args.Parent)
	return params
}
