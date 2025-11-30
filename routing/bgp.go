package routing

import (
	"fmt"
	"strings"

	"github.com/saintparish4/chaos/simulator"
)

// BGPProtocol implements AS path vector routing with loop prevention and policy preferences
type BGPProtocol struct {
	*RoutingProtocolBase
	AS               int
	ASPath           []int
	PolicyPreference map[string]int
}

func NewBGPProtocol(as int) *BGPProtocol {
	return &BGPProtocol{
		RoutingProtocolBase: NewRoutingProtocolBase("BGP"),
		AS:                  as,
		ASPath:              []int{as},
		PolicyPreference:    make(map[string]int),
	}
}

func (bgp *BGPProtocol) Initialize(nodeID string, topology *simulator.Topology) {
	bgp.NodeID = nodeID
	bgp.Topology = topology
	bgp.SetNeighbors(topology)
	bgp.UpdateRoute(nodeID, nodeID, 0, []string{nodeID})
	fmt.Printf("[BGP] Initialized on %s (AS %d)\n", nodeID, bgp.AS)
}

func (bgp *BGPProtocol) ProcessUpdate(update *RoutingUpdate) {
	bgp.Metrics.RecordUpdateReceived()

	for _, route := range update.Routes {
		if bgp.hasLoop(route.Path) {
			continue
		}

		newMetric := len(route.Path)

		if bgp.shouldAcceptRoute(route.Destination, newMetric, route.Path) {
			newPath := append([]string{bgp.NodeID}, route.Path...)
			bgp.UpdateRoute(route.Destination, update.SourceNode, newMetric, newPath)
			bgp.Metrics.RoutesLearned++
			fmt.Printf("[BGP] %s learned route to %s via %s (AS path len: %d)\n",
				bgp.NodeID, route.Destination, update.SourceNode, newMetric)
		}
	}
}

func (bgp *BGPProtocol) SendUpdates(neighbors []string) {
	routes := make([]RouteEntry, 0)

	bgp.mu.RLock()
	for dest, entry := range bgp.RoutingTable {
		if dest != bgp.NodeID && entry.Valid {
			routes = append(routes, entry)
		}
	}
	bgp.mu.RUnlock()

	if len(routes) == 0 {
		return
	}

	bgp.Metrics.RecordUpdateSent()
	bgp.Metrics.ProtocolOverhead += len(routes) * 100

	fmt.Printf("[BGP] %s announced %d routes to %d neighbors\n",
		bgp.NodeID, len(routes), len(neighbors))
}

func (bgp *BGPProtocol) shouldAcceptRoute(dest string, newMetric int, newPath []string) bool {
	bgp.mu.RLock()
	defer bgp.mu.RUnlock()

	existing, exists := bgp.RoutingTable[dest]

	if !exists || !existing.Valid {
		return true
	}

	if newMetric < existing.Metric {
		return true
	}

	if newMetric == existing.Metric {
		newPref := bgp.getPathPreference(newPath)
		existingPref := bgp.getPathPreference(existing.Path)
		return newPref > existingPref
	}

	return false
}

func (bgp *BGPProtocol) hasLoop(path []string) bool {
	seen := make(map[string]bool)
	for _, node := range path {
		if node == bgp.NodeID || seen[node] {
			return true
		}
		seen[node] = true
	}
	return false
}

func (bgp *BGPProtocol) getPathPreference(path []string) int {
	preference := 0
	for _, node := range path {
		if pref, exists := bgp.PolicyPreference[node]; exists {
			preference += pref
		}
	}
	return preference
}

func (bgp *BGPProtocol) SetPolicyPreference(node string, preference int) {
	bgp.mu.Lock()
	defer bgp.mu.Unlock()
	bgp.PolicyPreference[node] = preference
}

func (bgp *BGPProtocol) AnnounceRoute(dest string, metric int) {
	path := []string{bgp.NodeID, dest}
	bgp.UpdateRoute(dest, dest, metric, path)
	fmt.Printf("[BGP] %s announcing route to %s\n", bgp.NodeID, dest)
}

func (bgp *BGPProtocol) WithdrawRoute(dest string) {
	bgp.InvalidateRoute(dest)
	fmt.Printf("[BGP] %s withdrawing route to %s\n", bgp.NodeID, dest)
}

func (bgp *BGPProtocol) GetASPath(dest string) []string {
	bgp.mu.RLock()
	defer bgp.mu.RUnlock()

	if entry, exists := bgp.RoutingTable[dest]; exists {
		return entry.Path
	}
	return nil
}

func (bgp *BGPProtocol) PrintRoutingTable() {
	bgp.mu.RLock()
	defer bgp.mu.RUnlock()

	fmt.Printf("\n[BGP Routing Table - %s]\n", bgp.NodeID)
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("%-15s %-15s %-10s %-30s\n", "Destination", "Next Hop", "Metric", "AS Path")
	fmt.Println(strings.Repeat("-", 70))

	for dest, entry := range bgp.RoutingTable {
		if entry.Valid {
			pathStr := strings.Join(entry.Path, " -> ")
			fmt.Printf("%-15s %-15s %-10d %-30s\n",
				dest, entry.NextHop, entry.Metric, pathStr)
		}
	}
	fmt.Println(strings.Repeat("-", 70))
}
