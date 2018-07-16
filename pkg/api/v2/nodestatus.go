// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package maasapiv2

const (
	// NodeStatus* Values represent the vocabulary of a Node‘s possible statuses.

	// The Node has been created and has a system ID assigned to it.
	NodeStatusDeclared = "0"

	//Testing and other commissioning steps are taking place.
	NodeStatusCommissioning = "1"

	// Smoke or burn-in testing has a found a problem.
	NodeStatusFailedTests = "2"

	// The Node can’t be contacted.
	NodeStatusMissing = "3"

	// The Node is in the general pool ready to be deployed.
	NodeStatusReady = "4"

	// The Node is ready for named deployment.
	NodeStatusReserved = "5"

	// The Node is powering a service from a charm or is ready for use with a fresh Ubuntu install.
	NodeStatusDeployed = "6"

	// The Node has been removed from service manually until an admin overrides the retirement.
	NodeStatusRetired = "7"

	// The Node is broken: a step in the Node lifecyle failed. More details
	// can be found in the Node's event log.
	NodeStatusBroken = "8"

	// The Node is being installed.
	NodeStatusDeploying = "9"

	// The Node has been allocated to a user and is ready for deployment.
	NodeStatusAllocated = "10"

	// The deployment of the Node failed.
	NodeStatusFailedDeployment = "11"

	// The Node is powering down after a release request.
	NodeStatusReleasing = "12"

	// The releasing of the Node failed.
	NodeStatusFailedReleasing = "13"

	// The Node is erasing its disks.
	NodeStatusDiskErasing = "14"

	// The Node failed to erase its disks.
	NodeStatusFailedDiskErasing = "15"
)
