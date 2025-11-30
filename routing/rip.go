package routing

import (
	"fmt"
	"strings"

	"github.com/saintparish4/chaos/simulator"
)

// RIPProtocol implements distance-vector routing with split horizon and poison reverse
type RIPProtocol struct {
	*RoutingProtocolBase
	MaxHops      int
	UpdateTimer  int
	InvalidTimer int
	FlushTimer   int
}

const (
	RIPMaxHops        = 15
	RIPInfinity       = 16
	RIPUpdateInterval = 30
	RIPInvalidAfter   = 180
	RIPFlushAfter     = 240
)

func NewRIPProtocol() *RIPProtocol {
	return &RIPProtocol{
		RoutingProtocolBase: NewRoutingProtocolBase("RIP"),
		MaxHops:             RIPMaxHops,
		UpdateTimer:         RIPUpdateInterval,
		InvalidTimer:        RIPInvalidAfter,
		FlushTimer:          RIPFlushAfter,
	}
}

func (rip *RIPProtocol) Initialize(nodeID string, topology *simulator.Topology) {
	rip.NodeID = nodeID
	rip.Topology = topology
	rip.SetNeighbors(topology)

	rip.UpdateRoute(nodeID, nodeID, 0, []string{nodeID})

	for _, neighbor := range rip.Neighbors {
		rip.UpdateRoute(neighbor, neighbor, 1, []string{nodeID, neighbor})
	}

	fmt.Printf("[RIP] Initialized on %s\n", nodeID)
}

func (rip *RIPProtocol) ProcessUpdate(update *RoutingUpdate) {
	rip.Metrics.RecordUpdateReceived()

	sourceNode := update.SourceNode
	updated := false

	for _, route := range update.Routes {
		dest := route.Destination

		if dest == rip.NodeID {
			continue
		}

		newMetric := route.Metric + 1
		if newMetric >= RIPInfinity {
			newMetric = RIPInfinity
		}

		if rip.shouldUpdateRoute(dest, sourceNode, newMetric) {
			path := append([]string{rip.NodeID}, route.Path...)
			rip.UpdateRoute(dest, sourceNode, newMetric, path)
			updated = true
			rip.Metrics.RoutesLearned++
			fmt.Printf("[RIP] %s updated route to %s via %s (metric %d)\n",
				rip.NodeID, dest, sourceNode, newMetric)
		}
	}

	_ = updated // Would trigger update to neighbors in real implementation
}

func (rip *RIPProtocol) SendUpdates(neighbors []string) {
	for _, neighbor := range neighbors {
		routes := rip.prepareUpdateForNeighbor(neighbor)

		if len(routes) == 0 {
			continue
		}

		rip.Metrics.RecordUpdateSent()
		rip.Metrics.ProtocolOverhead += len(routes) * 20

		fmt.Printf("[RIP] %s sent %d routes to %s\n", rip.NodeID, len(routes), neighbor)
	}
}

func (rip *RIPProtocol) prepareUpdateForNeighbor(neighbor string) []RouteEntry {
	rip.mu.RLock()
	defer rip.mu.RUnlock()

	routes := make([]RouteEntry, 0)

	for dest, entry := range rip.RoutingTable {
		if dest == neighbor {
			continue
		}

		if entry.NextHop == neighbor {
			// Poison reverse: advertise as unreachable
			routes = append(routes, RouteEntry{
				Destination: dest,
				NextHop:     neighbor,
				Metric:      RIPInfinity,
				Path:        entry.Path,
				Valid:       false,
			})
		} else if entry.Valid {
			routes = append(routes, RouteEntry{
				Destination: dest,
				NextHop:     entry.NextHop,
				Metric:      entry.Metric,
				Path:        entry.Path,
				Valid:       true,
			})
		}
	}

	return routes
}

func (rip *RIPProtocol) shouldUpdateRoute(dest, sourceNode string, newMetric int) bool {
	rip.mu.RLock()
	defer rip.mu.RUnlock()

	existing, exists := rip.RoutingTable[dest]

	if !exists {
		return newMetric < RIPInfinity
	}

	if existing.NextHop == sourceNode {
		return true
	}

	return newMetric < existing.Metric
}

func (rip *RIPProtocol) InvalidateRoutesVia(nextHop string) {
	rip.mu.Lock()
	defer rip.mu.Unlock()

	count := 0
	for dest, entry := range rip.RoutingTable {
		if entry.NextHop == nextHop && entry.Valid {
			entry.Metric = RIPInfinity
			entry.Valid = false
			rip.RoutingTable[dest] = entry
			count++
		}
	}

	if count > 0 {
		fmt.Printf("[RIP] %s invalidated %d routes via %s\n", rip.NodeID, count, nextHop)
	}
}

func (rip *RIPProtocol) GarbageCollect() {
	rip.mu.Lock()
	defer rip.mu.Unlock()

	count := 0
	for dest, entry := range rip.RoutingTable {
		if !entry.Valid && dest != rip.NodeID {
			delete(rip.RoutingTable, dest)
			count++
		}
	}

	if count > 0 {
		fmt.Printf("[RIP] %s garbage collected %d routes\n", rip.NodeID, count)
		rip.Metrics.UpdateRoutingTableSize(len(rip.RoutingTable))
	}
}

func (rip *RIPProtocol) CountByHop() map[int]int {
	rip.mu.RLock()
	defer rip.mu.RUnlock()

	counts := make(map[int]int)
	for _, entry := range rip.RoutingTable {
		if entry.Valid {
			counts[entry.Metric]++
		}
	}
	return counts
}

func (rip *RIPProtocol) PrintRoutingTable() {
	rip.mu.RLock()
	defer rip.mu.RUnlock()

	fmt.Printf("\n[RIP Routing Table - %s]\n", rip.NodeID)
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("%-15s %-15s %-10s %-10s %-20s\n", "Destination", "Next Hop", "Hops", "Valid", "Path")
	fmt.Println(strings.Repeat("-", 70))

	for dest, entry := range rip.RoutingTable {
		validStr := "Yes"
		if !entry.Valid || entry.Metric >= RIPInfinity {
			validStr = "No"
		}

		pathStr := strings.Join(entry.Path, "->")
		if len(pathStr) > 20 {
			pathStr = pathStr[:17] + "..."
		}

		fmt.Printf("%-15s %-15s %-10d %-10s %-20s\n",
			dest, entry.NextHop, entry.Metric, validStr, pathStr)
	}
	fmt.Println(strings.Repeat("-", 70))

	hopCounts := make(map[int]int)
	validCount := 0
	for _, entry := range rip.RoutingTable {
		if entry.Valid && entry.Metric < RIPInfinity {
			hopCounts[entry.Metric]++
			validCount++
		}
	}

	fmt.Printf("Summary: %d valid routes", validCount)
	if len(hopCounts) > 0 {
		fmt.Printf(" (")
		first := true
		for hops := 0; hops <= RIPMaxHops; hops++ {
			if count, exists := hopCounts[hops]; exists {
				if !first {
					fmt.Printf(", ")
				}
				fmt.Printf("%d hop: %d", hops, count)
				first = false
			}
		}
		fmt.Printf(")")
	}
	fmt.Println()
}

func (rip *RIPProtocol) ComputeAverageHopCount() float64 {
	rip.mu.RLock()
	defer rip.mu.RUnlock()

	total := 0
	count := 0

	for _, entry := range rip.RoutingTable {
		if entry.Valid && entry.Metric < RIPInfinity {
			total += entry.Metric
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return float64(total) / float64(count)
}
