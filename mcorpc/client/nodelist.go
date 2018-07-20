package client

import "sync"

// NodeList is a list of nodes the client is interacting with and used
// to keep track of things like which have responded, still to respond etc
type NodeList struct {
	sync.RWMutex
	hosts map[string]struct{}
}

// NewNodeList creates a new initialized NodeList
func NewNodeList() *NodeList {
	return &NodeList{
		hosts: make(map[string]struct{}),
	}
}

// AddHosts appends the given nodes to the list of known nodes
func (n *NodeList) AddHosts(hosts ...string) {
	n.Lock()
	defer n.Unlock()

	for _, h := range hosts {
		n.hosts[h] = struct{}{}
	}
}

// Clear removes all nodes from the NodeList
func (n *NodeList) Clear() {
	n.Lock()
	defer n.Unlock()

	n.hosts = make(map[string]struct{})
}

// Count returns the number of nodes on the list
func (n *NodeList) Count() int {
	n.RLock()
	defer n.RUnlock()

	return len(n.hosts)
}

// Hosts returns the individual nodes on the list
func (n *NodeList) Hosts() []string {
	n.RLock()
	defer n.RUnlock()

	result := []string{}

	for k := range n.hosts {
		result = append(result, k)
	}

	return result
}

// DeleteIfKnown removes a node from the list if it's known, boolean result indicates if it was known
func (n *NodeList) DeleteIfKnown(host string) bool {
	n.Lock()
	defer n.Unlock()

	_, found := n.hosts[host]
	if found {
		delete(n.hosts, host)
	}

	return found
}

// Have determines if a node is known
func (n *NodeList) Have(host string) bool {
	n.RLock()
	defer n.RUnlock()

	_, ok := n.hosts[host]

	return ok
}

// HaveAny determines if any of the given nodes are known in a boolean OR fashion
func (n *NodeList) HaveAny(hosts ...string) bool {
	n.RLock()
	defer n.RUnlock()

	for _, host := range hosts {
		_, ok := n.hosts[host]
		if ok {
			return true
		}
	}

	return false
}
