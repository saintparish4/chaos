//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"

	"github.com/saintparish4/chaos/simulator"
)

// examples.go
//
// This file contains example functions demonstrating various simulator features.
// Build with: go run examples.go
// The +build ignore tag excludes this from normal builds to avoid conflicts.

// Example 1: Load topology and send messages
func example1() {
	fmt.Println("=== Example 1: Load Topology and Send Messages ===")

	// Load topology from file
	topo, err := simulator.LoadTopology("topology.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Create simulator
	sim := simulator.NewNetworkSimulator(topo)

	// Send a message
	msgID, err := sim.SendMessage("router-1", "tower-2", []byte("Hello!"))
	if err != nil {
		log.Fatal(err)
	}

	// Process the network
	for i := 0; i < 10; i++ {
		sim.ProcessAllNodes()
	}

	// Check message delivery
	track, _ := sim.GetMessageTrack(msgID)
	if track.Delivered {
		fmt.Printf("✓ Message delivered!\n")
		fmt.Printf("  Path: %v\n", track.Path)
		fmt.Printf("  Hops: %d\n", track.HopCount)
		fmt.Printf("  Latency: %v\n", track.EndTime.Sub(track.StartTime))
	}
}

// Example 2: Create a ring topology programmatically
func example2() {
	fmt.Println("\n=== Example 2: Programmatic Ring Topology ===")

	// Generate a 5-node ring
	config := simulator.GenerateRingTopology(5)

	// Build topology
	topo := &simulator.Topology{
		Config:  *config,
		AdjList: make(map[string][]string),
		Links:   make(map[string]map[string]*simulator.Link),
		Nodes:   make(map[string]*simulator.NodeConfig),
	}

	// Populate topology structures
	for i := range config.Nodes {
		node := &config.Nodes[i]
		topo.Nodes[node.ID] = node
	}

	for _, linkCfg := range config.Links {
		topo.AdjList[linkCfg.Source] = append(topo.AdjList[linkCfg.Source], linkCfg.Dest)
		if topo.Links[linkCfg.Source] == nil {
			topo.Links[linkCfg.Source] = make(map[string]*simulator.Link)
		}
		topo.Links[linkCfg.Source][linkCfg.Dest] = &simulator.Link{
			Source:    linkCfg.Source,
			Dest:      linkCfg.Dest,
			Bandwidth: linkCfg.Bandwidth,
			Latency:   linkCfg.Latency,
		}
	}

	// Validate
	if err := topo.Validate(); err != nil {
		log.Fatal(err)
	}

	// Create simulator
	sim := simulator.NewNetworkSimulator(topo)

	// Send message around the ring
	msgID, _ := sim.SendMessage("node-0", "node-4", []byte("Ring message"))

	// Process
	for i := 0; i < 10; i++ {
		sim.ProcessAllNodes()
	}

	// Check result
	track, _ := sim.GetMessageTrack(msgID)
	if track.Delivered {
		fmt.Printf("✓ Message traveled around ring\n")
		fmt.Printf("  Path: %v\n", track.Path)
	}
}

// Example 3: Test node failure
func example3() {
	fmt.Println("\n=== Example 3: Node Failure Handling ===")

	// Create simple topology
	config := simulator.GenerateRingTopology(4)
	topo := &simulator.Topology{
		Config:  *config,
		AdjList: make(map[string][]string),
		Links:   make(map[string]map[string]*simulator.Link),
		Nodes:   make(map[string]*simulator.NodeConfig),
	}

	for i := range config.Nodes {
		node := &config.Nodes[i]
		topo.Nodes[node.ID] = node
	}

	for _, linkCfg := range config.Links {
		topo.AdjList[linkCfg.Source] = append(topo.AdjList[linkCfg.Source], linkCfg.Dest)
		if topo.Links[linkCfg.Source] == nil {
			topo.Links[linkCfg.Source] = make(map[string]*simulator.Link)
		}
		topo.Links[linkCfg.Source][linkCfg.Dest] = &simulator.Link{
			Source:    linkCfg.Source,
			Dest:      linkCfg.Dest,
			Bandwidth: linkCfg.Bandwidth,
			Latency:   linkCfg.Latency,
		}
	}

	sim := simulator.NewNetworkSimulator(topo)

	// Fail a node
	node, _ := sim.GetNode("node-1")
	node.SetState("FAILED")
	fmt.Printf("Failed node: node-1\n")

	// Try to send message through failed node
	msgID, _ := sim.SendMessage("node-0", "node-1", []byte("Test"))

	// Process
	for i := 0; i < 5; i++ {
		sim.ProcessAllNodes()
	}

	// Check if message was dropped
	track, _ := sim.GetMessageTrack(msgID)
	if !track.Delivered {
		fmt.Printf("✓ Message correctly rejected by failed node\n")
	}
}

// Example 4: Compute shortest paths
func example4() {
	fmt.Println("\n=== Example 4: Shortest Path Computation ===")

	topo, _ := simulator.LoadTopology("topology.yaml")

	// Compute paths from router-1
	paths := simulator.DijkstraShortestPath(topo, "router-1")

	fmt.Println("Shortest paths from router-1:")
	for dest, info := range paths {
		if dest != "router-1" && info.NextHop != "" {
			fmt.Printf("  To %s: next hop=%s, distance=%d, path=%v\n",
				dest, info.NextHop, info.Distance, info.Path)
		}
	}
}

// Example 5: Monitor network metrics
func example5() {
	fmt.Println("\n=== Example 5: Network Metrics ===")

	topo, _ := simulator.LoadTopology("topology.yaml")
	sim := simulator.NewNetworkSimulator(topo)

	// Send multiple messages
	for i := 0; i < 10; i++ {
		sim.SendMessage("router-1", "tower-1", []byte("Traffic"))
	}

	// Process
	for i := 0; i < 20; i++ {
		sim.ProcessAllNodes()
	}

	// Display metrics
	fmt.Println("Node Metrics:")
	for _, node := range sim.GetAllNodes() {
		sent, received, dropped, queueDepth := node.GetMetrics()
		fmt.Printf("  %s: sent=%d, recv=%d, dropped=%d, queue=%d\n",
			node.ID, sent, received, dropped, queueDepth)
	}
}

// Example 6: Build custom topology
func example6() {
	fmt.Println("\n=== Example 6: Custom Topology ===")

	// Create custom topology config
	config := &simulator.TopologyConfig{
		Nodes: []simulator.NodeConfig{
			{ID: "core-1", Type: "router", Capacity: 2000},
			{ID: "core-2", Type: "router", Capacity: 2000},
			{ID: "edge-1", Type: "switch", Capacity: 1000},
			{ID: "edge-2", Type: "switch", Capacity: 1000},
		},
		Links: []simulator.LinkConfig{
			// Core backbone
			{Source: "core-1", Dest: "core-2", Bandwidth: 10000000, Latency: 2},
			{Source: "core-2", Dest: "core-1", Bandwidth: 10000000, Latency: 2},
			// Edge connections
			{Source: "core-1", Dest: "edge-1", Bandwidth: 5000000, Latency: 5},
			{Source: "edge-1", Dest: "core-1", Bandwidth: 5000000, Latency: 5},
			{Source: "core-2", Dest: "edge-2", Bandwidth: 5000000, Latency: 5},
			{Source: "edge-2", Dest: "core-2", Bandwidth: 5000000, Latency: 5},
		},
	}

	// Build topology
	topo := &simulator.Topology{
		Config:  *config,
		AdjList: make(map[string][]string),
		Links:   make(map[string]map[string]*simulator.Link),
		Nodes:   make(map[string]*simulator.NodeConfig),
	}

	for i := range config.Nodes {
		node := &config.Nodes[i]
		topo.Nodes[node.ID] = node
	}

	for _, linkCfg := range config.Links {
		topo.AdjList[linkCfg.Source] = append(topo.AdjList[linkCfg.Source], linkCfg.Dest)
		if topo.Links[linkCfg.Source] == nil {
			topo.Links[linkCfg.Source] = make(map[string]*simulator.Link)
		}
		topo.Links[linkCfg.Source][linkCfg.Dest] = &simulator.Link{
			Source:    linkCfg.Source,
			Dest:      linkCfg.Dest,
			Bandwidth: linkCfg.Bandwidth,
			Latency:   linkCfg.Latency,
		}
	}

	topo.Validate()

	fmt.Printf("✓ Created custom topology with %d nodes and %d links\n",
		len(config.Nodes), len(config.Links))

	// Test it
	sim := simulator.NewNetworkSimulator(topo)
	msgID, _ := sim.SendMessage("edge-1", "edge-2", []byte("Cross-network"))

	for i := 0; i < 10; i++ {
		sim.ProcessAllNodes()
	}

	track, _ := sim.GetMessageTrack(msgID)
	if track.Delivered {
		fmt.Printf("✓ Message routed through core: %v\n", track.Path)
	}
}

func main() {
	example1()
	example2()
	example3()
	example4()
	example5()
	example6()

	fmt.Println("\n=== All Examples Complete ===")
}
