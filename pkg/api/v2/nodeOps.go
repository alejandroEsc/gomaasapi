package maasapiv2

type NodeOp string

const (
	NodeDetails         NodeOp = "details"
	NodePowerParameters NodeOp = "power_parameters"
)

type NodesOp string

const (
	SetZone NodesOp = "set_zone"
)
