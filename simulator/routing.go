package simulator

import (
	"container/heap"
	"math"
)

// PriorityQueue implements heap.Interface for Dijkstra's algorithm
type PriorityQueue struct {
	items []*PQItem
}

type PQItem struct {
	nodeID   string
	priority int
	index    int
}

func (pq PriorityQueue) Len() int { return len(pq.items) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq.items[i].priority < pq.items[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].index = i
	pq.items[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(pq.items)
	item := x.(*PQItem)
	item.index = n
	pq.items = append(pq.items, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	item.index = -1
	pq.items = old[0 : n-1]
	return item
}

// DijkstraShortestPath computes shortest paths from soure to all other nodes
func DijkstraShortestPath(topo *Topology, source string) map[string]PathInfo {
	dist := make(map[string]int)
	prev := make(map[string]string)
	visited := make(map[string]bool)

	// Initialize distances
	for nodeID := range topo.Nodes {
		dist[nodeID] = math.MaxInt32
	}
	dist[source] = 0

	// Priority queue
	pq := &PriorityQueue{items: make([]*PQItem, 0)}
	heap.Init(pq)
	heap.Push(pq, &PQItem{nodeID: source, priority: 0})

	for pq.Len() > 0 {
		item := heap.Pop(pq).(*PQItem)
		current := item.nodeID

		if visited[current] {
			continue
		}
		visited[current] = true

		// Examine neighbors
		for _, neighbor := range topo.GetNeighbors(current) {
			if visited[neighbor] {
				continue
			}

			link := topo.GetLink(current, neighbor)
			if link == nil {
				continue
			}

			// Use latency as edge weight
			alt := dist[current] + link.Latency
			if alt < dist[neighbor] {
				dist[neighbor] = alt
				prev[neighbor] = current
				heap.Push(pq, &PQItem{nodeID: neighbor, priority: alt})
			}
		}
	}

	// Build path info
	result := make(map[string]PathInfo)
	for dest := range topo.Nodes {
		if dest == source {
			result[dest] = PathInfo{
				Destination: dest,
				NextHop:     source,
				Distance:    0,
				Path:        []string{source},
			}
			continue
		}

		path := reconstructPath(prev, source, dest)
		nextHop := ""
		if len(path) > 1 {
			nextHop = path[1]
		}

		result[dest] = PathInfo{
			Destination: dest,
			NextHop:     nextHop,
			Distance:    dist[dest],
			Path:        path,
		}
	}
	return result
}

// PathInfo contains routing information for a destination
type PathInfo struct {
	Destination string
	NextHop     string
	Distance    int
	Path        []string
}

// reconstructPath builds the path from source to dest using prev map
func reconstructPath(prev map[string]string, source, dest string) []string {
	if _, exists := prev[dest]; !exists && dest != source {
		return nil // No path exists
	}

	path := make([]string, 0)
	current := dest

	for current != "" {
		path = append([]string{current}, path...)
		if current == source {
			break
		}
		current = prev[current]
	}

	return path
}

// ComputeAllRoutingTables computes routing tables for all nodes
func ComputeAllRoutingTables(topo *Topology) map[string]map[string]string {
	allTables := make(map[string]map[string]string)

	for nodeID := range topo.Nodes {
		paths := DijkstraShortestPath(topo, nodeID)
		routingTable := make(map[string]string)

		for dest, pathInfo := range paths {
			if dest != nodeID && pathInfo.NextHop != "" {
				routingTable[dest] = pathInfo.NextHop
			}
		}

		allTables[nodeID] = routingTable
	}

	return allTables
}
