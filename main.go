package main

import (
	"fmt"
	"log"
	"time"

	"github.com/saintparish4/chaos/simulator"
)

func main() {
	fmt.Println("=== Network Simulator ===")

	// Load topology from YAML file
	fmt.Println("Loading topology from topology.yaml...")
	topo, err := simulator.LoadTopology("topology.yaml")
	if err != nil {
		log.Fatalf("Failed to load topology: %v", err)
	}
	fmt.Printf("✓ Loaded topology with %d nodes and %d links\n\n",
		len(topo.Config.Nodes), len(topo.Config.Links))

	// Create network simulator
	fmt.Println("Creating network simulator...")
	sim := simulator.NewNetworkSimulator(topo)
	sim.Start()
	fmt.Println("✓ Network simulator initialized")
	fmt.Println("✓ Routing tables computed")

	// Display network status
	sim.PrintNetworkStatus()

	// Send some test messages
	fmt.Println("\n=== Sending Test Messages ===")

	testMessages := []struct {
		source string
		dest   string
		msg    string
	}{
		{"tower-1", "tower-2", "Hello from tower 1"},
		{"router-1", "router-3", "Direct router message"},
		{"switch-1", "switch-2", "Switch to switch"},
		{"tower-2", "router-1", "Tower to router"},
	}

	for i, test := range testMessages {
		msgID, err := sim.SendMessage(test.source, test.dest, []byte(test.msg))
		if err != nil {
			log.Printf("Failed to send message: %v", err)
			continue
		}
		fmt.Printf("%d. Sent message %s: %s -> %s\n", i+1, msgID[:20], test.source, test.dest)
	}

	// Process messages through the network
	fmt.Println("\n=== Processing Messages ===")
	for i := 0; i < 10; i++ {
		err := sim.ProcessAllNodes()
		if err != nil {
			log.Printf("Error processing nodes: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Show results
	sim.PrintNetworkStatus()
	sim.PrintMessageTracking()

	// Test programmatic topology generation
	fmt.Println("\n=== Testing Topology Generators ===")

	// Generate ring topology
	fmt.Println("\n1. Ring Topology (5 nodes):")
	ringConfig := simulator.GenerateRingTopology(5)
	fmt.Printf("   Nodes: %d, Links: %d\n", len(ringConfig.Nodes), len(ringConfig.Links))

	// Generate mesh topology
	fmt.Println("\n2. Mesh Topology (4 nodes):")
	meshConfig := simulator.GenerateMeshTopology(4)
	fmt.Printf("   Nodes: %d, Links: %d\n", len(meshConfig.Nodes), len(meshConfig.Links))

	// Generate tree topology
	fmt.Println("\n3. Tree Topology (depth 3):")
	treeConfig := simulator.GenerateTreeTopology(3)
	fmt.Printf("   Nodes: %d, Links: %d\n", len(treeConfig.Nodes), len(treeConfig.Links))

	// Test a generated topology
	fmt.Println("\n=== Testing Ring Topology ===")
	ringTopo := &simulator.Topology{
		Config:  *ringConfig,
		AdjList: make(map[string][]string),
		Links:   make(map[string]map[string]*simulator.Link),
		Nodes:   make(map[string]*simulator.NodeConfig),
	}

	// Build topology structures
	for i := range ringConfig.Nodes {
		node := &ringConfig.Nodes[i]
		ringTopo.Nodes[node.ID] = node
	}

	for _, linkCfg := range ringConfig.Links {
		ringTopo.AdjList[linkCfg.Source] = append(ringTopo.AdjList[linkCfg.Source], linkCfg.Dest)
		if ringTopo.Links[linkCfg.Source] == nil {
			ringTopo.Links[linkCfg.Source] = make(map[string]*simulator.Link)
		}
		ringTopo.Links[linkCfg.Source][linkCfg.Dest] = &simulator.Link{
			Source:    linkCfg.Source,
			Dest:      linkCfg.Dest,
			Bandwidth: linkCfg.Bandwidth,
			Latency:   linkCfg.Latency,
		}
	}

	ringTopo.Validate()
	ringSim := simulator.NewNetworkSimulator(ringTopo)
	ringSim.Start()

	// Send message around the ring
	msgID, err := ringSim.SendMessage("node-0", "node-3", []byte("Ring message"))
	if err != nil {
		log.Fatalf("Failed to send ring message: %v", err)
	}
	fmt.Printf("Sent message in ring: node-0 -> node-3\n")

	// Process
	for i := 0; i < 5; i++ {
		ringSim.ProcessAllNodes()
		time.Sleep(10 * time.Millisecond)
	}

	// Show tracking
	track, err := ringSim.GetMessageTrack(msgID)
	if err != nil {
		log.Printf("Failed to get message track: %v", err)
	} else if track.Delivered {
		fmt.Printf("✓ Message delivered via path: %v\n", track.Path)
		fmt.Printf("  Hops: %d, Latency: %v\n", track.HopCount, track.EndTime.Sub(track.StartTime))
	}

	fmt.Println("\n=== Layer 0 Complete ===")
	fmt.Println("✓ Topology loading and validation")
	fmt.Println("✓ Node implementation with state management")
	fmt.Println("✓ Message queuing")
	fmt.Println("✓ Dijkstra's shortest path routing")
	fmt.Println("✓ Message forwarding and delivery tracking")
	fmt.Println("✓ Programmatic topology generation (ring, mesh, tree)")
}
