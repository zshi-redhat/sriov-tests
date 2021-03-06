package cluster

import (
	"fmt"

	sriovv1 "github.com/openshift/sriov-network-operator/pkg/apis/sriovnetwork/v1"
	testclient "github.com/openshift/sriov-tests/pkg/util/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnabledNodes provides info on sriov enabled nodes of the cluster.
type EnabledNodes struct {
	Nodes  []string
	States map[string]sriovv1.SriovNetworkNodeState
}

var (
	supportedDrivers = []string{"mlx5_core", "i40e", "ixgbe"}
)

// DiscoverSriov retrieves Sriov related information of a given cluster.
func DiscoverSriov(clients *testclient.ClientSet, operatorNamespace string) (*EnabledNodes, error) {
	nodeStates, err := clients.SriovNetworkNodeStates(operatorNamespace).List(metav1.ListOptions{})
	res := &EnabledNodes{}
	res.States = make(map[string]sriovv1.SriovNetworkNodeState)
	res.Nodes = make([]string, 0)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve note states %v", err)
	}

	for _, state := range nodeStates.Items {
		if !stateStable(state) {
			return nil, fmt.Errorf("Sync status still in progress")
		}

		node := state.Name
		for _, itf := range state.Status.Interfaces {
			if isDriverSupported(itf.Driver) {
				res.Nodes = append(res.Nodes, node)
				res.States[node] = state
				break
			}
		}
	}

	if len(res.Nodes) == 0 {
		return nil, fmt.Errorf("No sriov enabled node found")
	}
	return res, nil
}

// FindOneSriovDevice retrieves a valid sriov device for the given node.
func (n *EnabledNodes) FindOneSriovDevice(node string) (*sriovv1.InterfaceExt, error) {
	s, ok := n.States[node]
	if !ok {
		return nil, fmt.Errorf("Node %s not found", node)
	}
	for _, itf := range s.Status.Interfaces {
		if isDriverSupported(itf.Driver) {
			return &itf, nil
		}
	}

	return nil, fmt.Errorf("Unable to find sriov devices in node %s", node)
}

// SriovStable tells if all the node states are in sync (and the cluster is ready for another round of tests)
func SriovStable(operatorNamespace string, clients *testclient.ClientSet) (bool, error) {
	nodeStates, err := clients.SriovNetworkNodeStates(operatorNamespace).List(metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("Failed to fetch nodes state %v", err)
	}
	if len(nodeStates.Items) == 0 {
		return false, nil
	}
	for _, state := range nodeStates.Items {
		if !stateStable(state) {
			return false, nil
		}
	}
	return true, nil
}

func stateStable(state sriovv1.SriovNetworkNodeState) bool {
	switch state.Status.SyncStatus {
	case "Succeeded":
		return true
	case "":
		return true
	}
	return false
}

func isDriverSupported(driver string) bool {
	for _, supportedDriver := range supportedDrivers {
		if driver == supportedDriver {
			return true
		}
	}

	return false
}
