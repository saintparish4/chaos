package routing

import (
	"container/heap"
	"fmt"
	"math"
	"strings"

	"github.com/saintparish4/chaos/simulator"
)

// OSPFProtocol implements link-state routing with Dijkstra SPF calculation
type OSPFProtocol struct {
	*RoutingProtocolBase
	LinkStateDB map[string]*LinkStateAdvertisement
	SequenceNum int
	Area        int
}

// LinkStateAdvertisement represents an OSPF LSA
type LinkStateAdvertisement struct {
	OriginNode  string
	SequenceNum int
	Links       []LinkInfo
	Age         int
}

// LinkInfo represents link information in an LSA
type LinkInfo struct {
	Neighbor string
	Cost     int
}

func NewOSPFProtocol(area int) *OSPFProtocol {
	return &OSPFProtocol{
		RoutingProtocolBase: NewRoutingProtocolBase("OSPF"),
		LinkStateDB:         make(map[string]*LinkStateAdvertisement),
		SequenceNum:         0,
		Area:                area,
	}
}

func (ospf *OSPFProtocol) Initialize(nodeID string, topology *simulator.Topology) {
	ospf.NodeID = nodeID
	ospf.Topology = topology
	ospf.SetNeighbors(topology)
	ospf.generateLSA()
	ospf.runSPF()
	fmt.Printf("[OSPF] Initialized on %s (Area %d)\n", nodeID, ospf.Area)
}

func (ospf *OSPFProtocol) ProcessUpdate(update *RoutingUpdate) {
	ospf.Metrics.RecordUpdateReceived()

	for _, route := range update.Routes {
		lsa := &LinkStateAdvertisement{
			OriginNode:  route.Destination,
			SequenceNum: route.Metric,
			Age:         0,
		}

		if ospf.shouldAcceptLSA(lsa) {
			ospf.LinkStateDB[lsa.OriginNode] = lsa
			ospf.runSPF()
			ospf.Metrics.ProtocolOverhead += 200
		}
	}
}

func (ospf *OSPFProtocol) SendUpdates(neighbors []string) {
	ospf.generateLSA()

	ospf.Metrics.RecordUpdateSent()
	ospf.Metrics.ProtocolOverhead += len(neighbors) * 200

	fmt.Printf("[OSPF] %s flooded LSA (seq %d) to %d neighbors\n",
		ospf.NodeID, ospf.SequenceNum, len(neighbors))
}

func (ospf *OSPFProtocol) generateLSA() {
	ospf.mu.Lock()
	defer ospf.mu.Unlock()

	ospf.SequenceNum++

	links := make([]LinkInfo, 0, len(ospf.Neighbors))
	for _, neighbor := range ospf.Neighbors {
		cost := ospf.getLinkCost(ospf.NodeID, neighbor)
		links = append(links, LinkInfo{
			Neighbor: neighbor,
			Cost:     cost,
		})
	}

	ospf.LinkStateDB[ospf.NodeID] = &LinkStateAdvertisement{
		OriginNode:  ospf.NodeID,
		SequenceNum: ospf.SequenceNum,
		Links:       links,
		Age:         0,
	}
}

func (ospf *OSPFProtocol) shouldAcceptLSA(newLSA *LinkStateAdvertisement) bool {
	ospf.mu.RLock()
	defer ospf.mu.RUnlock()

	existing, exists := ospf.LinkStateDB[newLSA.OriginNode]
	if !exists {
		return true
	}
	return newLSA.SequenceNum > existing.SequenceNum
}

func (ospf *OSPFProtocol) runSPF() {
	ospf.mu.Lock()
	defer ospf.mu.Unlock()

	graph := ospf.buildGraph()
	distances, previous := ospf.dijkstra(graph, ospf.NodeID)

	ospf.RoutingTable = make(map[string]RouteEntry)

	for dest, dist := range distances {
		if dest == ospf.NodeID {
			continue
		}

		nextHop := ospf.findNextHop(previous, dest)
		if nextHop != "" {
			ospf.RoutingTable[dest] = RouteEntry{
				Destination: dest,
				NextHop:     nextHop,
				Metric:      dist,
				Valid:       true,
			}
		}
	}

	ospf.Metrics.UpdateRoutingTableSize(len(ospf.RoutingTable))
	ospf.Metrics.RoutesLearned = len(ospf.RoutingTable)
}

func (ospf *OSPFProtocol) buildGraph() map[string]map[string]int {
	graph := make(map[string]map[string]int)

	for nodeID, lsa := range ospf.LinkStateDB {
		if graph[nodeID] == nil {
			graph[nodeID] = make(map[string]int)
		}
		for _, link := range lsa.Links {
			graph[nodeID][link.Neighbor] = link.Cost
			if graph[link.Neighbor] == nil {
				graph[link.Neighbor] = make(map[string]int)
			}
		}
	}

	return graph
}

func (ospf *OSPFProtocol) dijkstra(graph map[string]map[string]int, source string) (map[string]int, map[string]string) {
	distances := make(map[string]int)
	previous := make(map[string]string)
	visited := make(map[string]bool)

	for node := range graph {
		distances[node] = math.MaxInt32
	}
	distances[source] = 0

	pq := &PriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &PQItem{node: source, priority: 0})

	for pq.Len() > 0 {
		item := heap.Pop(pq).(*PQItem)
		current := item.node

		if visited[current] {
			continue
		}
		visited[current] = true

		for neighbor, cost := range graph[current] {
			if visited[neighbor] {
				continue
			}

			newDist := distances[current] + cost
			if newDist < distances[neighbor] {
				distances[neighbor] = newDist
				previous[neighbor] = current
				heap.Push(pq, &PQItem{node: neighbor, priority: newDist})
			}
		}
	}

	return distances, previous
}

func (ospf *OSPFProtocol) findNextHop(previous map[string]string, dest string) string {
	if previous[dest] == "" {
		return ""
	}

	current := dest
	for {
		prev := previous[current]
		if prev == ospf.NodeID {
			return current
		}
		if prev == "" {
			return ""
		}
		current = prev
	}
}

func (ospf *OSPFProtocol) getLinkCost(source, dest string) int {
	if ospf.Topology == nil {
		return 1
	}

	if links, exists := ospf.Topology.Links[source]; exists {
		if link, exists := links[dest]; exists {
			return link.Latency * 1000
		}
	}

	return 1
}

func (ospf *OSPFProtocol) PrintRoutingTable() {
	ospf.mu.RLock()
	defer ospf.mu.RUnlock()

	fmt.Printf("\n[OSPF Routing Table - %s]\n", ospf.NodeID)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-15s %-15s %-10s\n", "Destination", "Next Hop", "Cost")
	fmt.Println(strings.Repeat("-", 60))

	for dest, entry := range ospf.RoutingTable {
		if entry.Valid {
			fmt.Printf("%-15s %-15s %-10d\n", dest, entry.NextHop, entry.Metric)
		}
	}
	fmt.Println(strings.Repeat("-", 60))
}

func (ospf *OSPFProtocol) PrintLinkStateDB() {
	ospf.mu.RLock()
	defer ospf.mu.RUnlock()

	fmt.Printf("\n[OSPF Link-State Database - %s]\n", ospf.NodeID)
	fmt.Println(strings.Repeat("-", 60))

	for nodeID, lsa := range ospf.LinkStateDB {
		fmt.Printf("Node %s (seq %d):\n", nodeID, lsa.SequenceNum)
		for _, link := range lsa.Links {
			fmt.Printf("  -> %s (cost %d)\n", link.Neighbor, link.Cost)
		}
	}
	fmt.Println(strings.Repeat("-", 60))
}

// PriorityQueue implements heap.Interface for Dijkstra
type PriorityQueue []*PQItem

type PQItem struct {
	node     string
	priority int
	index    int
}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PQItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}
