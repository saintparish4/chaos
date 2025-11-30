package simulator

import (
	"testing"
)

func TestGenerateRingTopology(t *testing.T) {
	numNodes := 5
	config := GenerateRingTopology(numNodes)

	if len(config.Nodes) != numNodes {
		t.Errorf("Expected %d nodes, got %d", numNodes, len(config.Nodes))
	}

	if len(config.Links) != numNodes {
		t.Errorf("Expected %d links, got %d", numNodes, len(config.Links))
	}

	// Verify ring structure
	for i := 0; i < numNodes; i++ {
		link := config.Links[i]
		expectedSource := config.Nodes[i].ID
		expectedDest := config.Nodes[(i+1)%numNodes].ID

		if link.Source != expectedSource {
			t.Errorf("Link %d: expected source %s, got %s", i, expectedSource, link.Source)
		}
		if link.Dest != expectedDest {
			t.Errorf("Link %d: expected dest %s, got %s", i, expectedDest, link.Dest)
		}
	}
}

func TestGenerateMeshTopology(t *testing.T) {
	numNodes := 4
	config := GenerateMeshTopology(numNodes)

	if len(config.Nodes) != numNodes {
		t.Errorf("Expected %d nodes, got %d", numNodes, len(config.Nodes))
	}

	// Full mesh should have n*(n-1) directed links
	expectedLinks := numNodes * (numNodes - 1)
	if len(config.Links) != expectedLinks {
		t.Errorf("Expected %d links, got %d", expectedLinks, len(config.Links))
	}
}

func TestGenerateTreeTopology(t *testing.T) {
	depth := 3
	expectedNodes := 7 // 2^3 - 1
	config := GenerateTreeTopology(depth)

	if len(config.Nodes) != expectedNodes {
		t.Errorf("Expected %d nodes, got %d", expectedNodes, len(config.Nodes))
	}

	// Tree with n nodes has n-1 edges (in a rooted tree)
	// Binary tree has 2*(n-leaves) edges
	if len(config.Links) < 1 {
		t.Error("Expected at least 1 link in tree")
	}
}

func TestTopologyValidation(t *testing.T) {
	// Valid topology
	validConfig := &TopologyConfig{
		Nodes: []NodeConfig{
			{ID: "node-1", Type: NodeTypeRouter, Capacity: 1000},
			{ID: "node-2", Type: NodeTypeRouter, Capacity: 1000},
		},
		Links: []LinkConfig{
			{Source: "node-1", Dest: "node-2", Bandwidth: 1000000, Latency: 10},
		},
	}

	topo := &Topology{
		Config:  *validConfig,
		AdjList: make(map[string][]string),
		Links:   make(map[string]map[string]*Link),
		Nodes:   make(map[string]*NodeConfig),
	}

	for i := range validConfig.Nodes {
		node := &validConfig.Nodes[i]
		topo.Nodes[node.ID] = node
	}

	for _, linkCfg := range validConfig.Links {
		topo.AdjList[linkCfg.Source] = append(topo.AdjList[linkCfg.Source], linkCfg.Dest)
	}

	err := topo.Validate()
	if err != nil {
		t.Errorf("Valid topology failed validation: %v", err)
	}

	// Invalid topology - orphaned node
	invalidConfig := &TopologyConfig{
		Nodes: []NodeConfig{
			{ID: "node-1", Type: NodeTypeRouter, Capacity: 1000},
			{ID: "node-2", Type: NodeTypeRouter, Capacity: 1000},
			{ID: "orphan", Type: NodeTypeRouter, Capacity: 1000},
		},
		Links: []LinkConfig{
			{Source: "node-1", Dest: "node-2", Bandwidth: 1000000, Latency: 10},
		},
	}

	invalidTopo := &Topology{
		Config:  *invalidConfig,
		AdjList: make(map[string][]string),
		Links:   make(map[string]map[string]*Link),
		Nodes:   make(map[string]*NodeConfig),
	}

	for i := range invalidConfig.Nodes {
		node := &invalidConfig.Nodes[i]
		invalidTopo.Nodes[node.ID] = node
	}

	for _, linkCfg := range invalidConfig.Links {
		invalidTopo.AdjList[linkCfg.Source] = append(invalidTopo.AdjList[linkCfg.Source], linkCfg.Dest)
	}

	err = invalidTopo.Validate()
	if err == nil {
		t.Error("Expected validation error for orphaned node, got nil")
	}
}

func TestDijkstraShortestPath(t *testing.T) {
	// Create simple 3-node line topology
	config := &TopologyConfig{
		Nodes: []NodeConfig{
			{ID: "A", Type: NodeTypeRouter, Capacity: 1000},
			{ID: "B", Type: NodeTypeRouter, Capacity: 1000},
			{ID: "C", Type: NodeTypeRouter, Capacity: 1000},
		},
		Links: []LinkConfig{
			{Source: "A", Dest: "B", Bandwidth: 1000000, Latency: 10},
			{Source: "B", Dest: "C", Bandwidth: 1000000, Latency: 10},
		},
	}

	topo := &Topology{
		Config:  *config,
		AdjList: make(map[string][]string),
		Links:   make(map[string]map[string]*Link),
		Nodes:   make(map[string]*NodeConfig),
	}

	for i := range config.Nodes {
		node := &config.Nodes[i]
		topo.Nodes[node.ID] = node
	}

	for _, linkCfg := range config.Links {
		topo.AdjList[linkCfg.Source] = append(topo.AdjList[linkCfg.Source], linkCfg.Dest)
		if topo.Links[linkCfg.Source] == nil {
			topo.Links[linkCfg.Source] = make(map[string]*Link)
		}
		topo.Links[linkCfg.Source][linkCfg.Dest] = &Link{
			Source:    linkCfg.Source,
			Dest:      linkCfg.Dest,
			Bandwidth: linkCfg.Bandwidth,
			Latency:   linkCfg.Latency,
		}
	}

	// Test shortest path from A
	paths := DijkstraShortestPath(topo, "A")

	// Check path to B
	if pathInfo, exists := paths["B"]; exists {
		if pathInfo.NextHop != "B" {
			t.Errorf("Expected next hop to B to be 'B', got '%s'", pathInfo.NextHop)
		}
		if pathInfo.Distance != 10 {
			t.Errorf("Expected distance to B to be 10, got %d", pathInfo.Distance)
		}
	} else {
		t.Error("No path found to B")
	}

	// Check path to C (should go through B)
	if pathInfo, exists := paths["C"]; exists {
		if pathInfo.NextHop != "B" {
			t.Errorf("Expected next hop to C to be 'B', got '%s'", pathInfo.NextHop)
		}
		if pathInfo.Distance != 20 {
			t.Errorf("Expected distance to C to be 20, got %d", pathInfo.Distance)
		}
		expectedPath := []string{"A", "B", "C"}
		if len(pathInfo.Path) != len(expectedPath) {
			t.Errorf("Expected path length %d, got %d", len(expectedPath), len(pathInfo.Path))
		}
	} else {
		t.Error("No path found to C")
	}
}

func TestComputeAllRoutingTables(t *testing.T) {
	config := GenerateRingTopology(3)

	topo := &Topology{
		Config:  *config,
		AdjList: make(map[string][]string),
		Links:   make(map[string]map[string]*Link),
		Nodes:   make(map[string]*NodeConfig),
	}

	for i := range config.Nodes {
		node := &config.Nodes[i]
		topo.Nodes[node.ID] = node
	}

	for _, linkCfg := range config.Links {
		topo.AdjList[linkCfg.Source] = append(topo.AdjList[linkCfg.Source], linkCfg.Dest)
		if topo.Links[linkCfg.Source] == nil {
			topo.Links[linkCfg.Source] = make(map[string]*Link)
		}
		topo.Links[linkCfg.Source][linkCfg.Dest] = &Link{
			Source:    linkCfg.Source,
			Dest:      linkCfg.Dest,
			Bandwidth: linkCfg.Bandwidth,
			Latency:   linkCfg.Latency,
		}
	}

	tables := ComputeAllRoutingTables(topo)

	if len(tables) != 3 {
		t.Errorf("Expected 3 routing tables, got %d", len(tables))
	}

	// Each node should have routes to 2 other nodes
	for nodeID, table := range tables {
		if len(table) != 2 {
			t.Errorf("Node %s: expected 2 routes, got %d", nodeID, len(table))
		}
	}
}
