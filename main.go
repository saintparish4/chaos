package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/saintparish4/chaos/simulator"
)

func main() {
	fmt.Println("=== Network Simulator - Layer 1 Demo ===")
	fmt.Println("Discrete Event Simulation with Traffic Generation")

	// Load topology
	fmt.Println("Loading topology...")
	topo, err := simulator.LoadTopology("topology.yaml")
	if err != nil {
		log.Fatalf("Failed to load topology: %v", err)
	}
	fmt.Printf("✓ Loaded %d nodes, %d links\n\n", len(topo.Config.Nodes), len(topo.Config.Links))

	// Create event-driven simulator with 10x speed
	fmt.Println("Creating event-driven simulator (10x speed)...")
	sim := simulator.NewEventDrivenSimulator(topo, 10.0)
	fmt.Println("✓ Simulator initialized")
	fmt.Println("✓ Routing tables computed")
	fmt.Println("✓ Simulated links created with bandwidth/latency modeling")

	// Demo 1: Poisson Traffic Generation
	fmt.Println("=== Demo 1: Poisson Traffic ===")
	poissonGen := simulator.NewPoissonTrafficGenerator(
		"tower-1", // source
		"tower-2", // destination
		5.0,       // 5 messages per second
		1000,      // 1000 bytes per message
	)
	poissonFlow := simulator.NewTrafficFlow(
		"poisson-1",
		poissonGen,
		0.0, // start time
		5.0, // end time
	)
	sim.AddTrafficFlow(poissonFlow)
	fmt.Printf("Added: %s\n", poissonGen.GetDescription())

	// Run simulation for 5 seconds
	fmt.Println("\nRunning simulation for 5 seconds...")
	err = sim.Run(5.0)
	if err != nil {
		log.Fatalf("Simulation failed: %v", err)
	}

	// Print results
	sim.PrintStatus()

	// Show link metrics
	fmt.Println("\n=== Link Metrics ===")
	link := sim.GetLink("router-1", "router-2")
	if link != nil {
		sent, dropped, lossRate, avgLatency, utilization := link.GetMetrics().GetStats()
		fmt.Printf("router-1 -> router-2:\n")
		fmt.Printf("  Messages: sent=%d, dropped=%d, loss=%.2f%%\n", sent, dropped, lossRate*100)
		fmt.Printf("  Avg Latency: %.3fs\n", avgLatency)
		fmt.Printf("  Utilization: %.1f%%\n", utilization*100)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))

	// Demo 2: Bursty Traffic with Multiple Flows
	fmt.Println("\n=== Demo 2: Bursty Traffic with Multiple Flows ===")

	// Create new simulator
	sim2 := simulator.NewEventDrivenSimulator(topo, 50.0) // 50x speed

	// Add bursty traffic flow
	burstyGen := simulator.NewBurstyTrafficGenerator(
		"router-1", // source
		"switch-2", // destination
		20.0,       // 20 messages per second during burst
		10,         // 10 messages per burst
		2.0,        // burst every 2 seconds
		500,        // 500 bytes per message
	)
	burstyFlow := simulator.NewTrafficFlow("bursty-1", burstyGen, 0.0, 10.0)
	sim2.AddTrafficFlow(burstyFlow)
	fmt.Printf("Added: %s\n", burstyGen.GetDescription())

	// Add constant background traffic
	constantGen := simulator.NewConstantTrafficGenerator(
		"switch-1",
		"tower-2",
		2.0, // 2 messages per second
		200, // 200 bytes
	)
	constantFlow := simulator.NewTrafficFlow("constant-1", constantGen, 0.0, 10.0)
	sim2.AddTrafficFlow(constantFlow)
	fmt.Printf("Added: %s\n", constantGen.GetDescription())

	// Add another Poisson flow
	poissonGen2 := simulator.NewPoissonTrafficGenerator("tower-2", "tower-1", 3.0, 800)
	poissonFlow2 := simulator.NewTrafficFlow("poisson-2", poissonGen2, 0.0, 10.0)
	sim2.AddTrafficFlow(poissonFlow2)
	fmt.Printf("Added: %s\n", poissonGen2.GetDescription())

	fmt.Println("\nRunning simulation for 10 seconds...")
	err = sim2.Run(10.0)
	if err != nil {
		log.Fatalf("Simulation failed: %v", err)
	}

	sim2.PrintStatus()

	// Show system metrics over time
	fmt.Println("\n=== System Metrics Summary ===")
	summary := sim2.GetMetrics().System.GetSummary()
	fmt.Println(summary.String())

	// Show throughput samples
	throughputSamples := sim2.GetMetrics().System.Throughput.GetRecent(5)
	fmt.Println("\nLast 5 Throughput Samples:")
	for i, sample := range throughputSamples {
		fmt.Printf("  %d. t=%.2fs: %.2f msg/s\n", i+1, sample.Timestamp, sample.Value)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))

	// Demo 3: Link Congestion and Packet Drops
	fmt.Println("\n=== Demo 3: Link Congestion & Packet Drops ===")

	sim3 := simulator.NewEventDrivenSimulator(topo, 100.0) // 100x speed

	// Configure a link with packet drops
	link3 := sim3.GetLink("switch-1", "tower-1")
	if link3 != nil {
		link3.SetDropRate(0.1) // 10% random drop rate
		link3.SetDropOnCongestion(true)
		fmt.Println("Configured switch-1 -> tower-1 with 10% drop rate")
	}

	// Generate heavy traffic to cause congestion
	heavyGen := simulator.NewPoissonTrafficGenerator(
		"switch-1",
		"tower-1",
		50.0, // 50 messages per second (high load)
		2000, // 2000 bytes
	)
	heavyFlow := simulator.NewTrafficFlow("heavy-1", heavyGen, 0.0, 5.0)
	sim3.AddTrafficFlow(heavyFlow)
	fmt.Printf("Added: %s\n", heavyGen.GetDescription())

	fmt.Println("\nRunning simulation for 5 seconds...")
	err = sim3.Run(5.0)
	if err != nil {
		log.Fatalf("Simulation failed: %v", err)
	}

	sim3.PrintStatus()

	// Show link with drops
	if link3 != nil {
		sent, dropped, lossRate, _, utilization := link3.GetMetrics().GetStats()
		fmt.Printf("\nswitch-1 -> tower-1 (with drops):\n")
		fmt.Printf("  Sent: %d, Dropped: %d\n", sent, dropped)
		fmt.Printf("  Loss Rate: %.2f%%\n", lossRate*100)
		fmt.Printf("  Utilization: %.1f%%\n", utilization*100)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))

	// Demo 4: Different Simulation Speeds
	fmt.Println("\n=== Demo 4: Simulation Speed Comparison ===")

	speeds := []float64{1.0, 10.0, 100.0, 1000.0}
	duration := 2.0 // 2 seconds of simulated time

	for _, speed := range speeds {
		simSpeed := simulator.NewEventDrivenSimulator(topo, speed)

		// Add simple traffic
		gen := simulator.NewPoissonTrafficGenerator("router-1", "router-3", 5.0, 500)
		flow := simulator.NewTrafficFlow("test", gen, 0.0, duration)
		simSpeed.AddTrafficFlow(flow)

		fmt.Printf("\nSpeed %.0fx: ", speed)
		simSpeed.Run(duration)
		fmt.Printf("Processed %d events\n", simSpeed.GetMetrics().System.TotalMessages.Size())
	}

	fmt.Println("\n" + strings.Repeat("=", 50))

	// Demo 5: Node and Link Failures
	fmt.Println("\n=== Demo 5: Node and Link Failures ===")

	sim5 := simulator.NewEventDrivenSimulator(topo, 10.0)

	// Add traffic
	gen5 := simulator.NewPoissonTrafficGenerator("tower-1", "tower-2", 10.0, 1000)
	flow5 := simulator.NewTrafficFlow("traffic", gen5, 0.0, 10.0)
	sim5.AddTrafficFlow(flow5)

	// Schedule node failure at t=3s, recover at t=7s
	nodeFailure := &simulator.SimulationEvent{
		ID:        "fail-router-2",
		Type:      simulator.EventNodeFailure,
		Timestamp: 3.0,
		NodeID:    "router-2",
		Data: &simulator.NodeFailureEventData{
			NodeID:   "router-2",
			Duration: 4.0, // Fail for 4 seconds
		},
	}
	sim5.ScheduleEvent(nodeFailure)
	fmt.Println("Scheduled: router-2 failure at t=3s (recover at t=7s)")

	// Schedule link failure at t=2s, recover at t=5s
	linkFailure := &simulator.SimulationEvent{
		ID:        "fail-link",
		Type:      simulator.EventLinkFailure,
		Timestamp: 2.0,
		LinkID:    "router-1->router-2",
		Data: &simulator.LinkFailureEventData{
			Source:   "router-1",
			Dest:     "router-2",
			Duration: 3.0, // Fail for 3 seconds
		},
	}
	sim5.ScheduleEvent(linkFailure)
	fmt.Println("Scheduled: router-1->router-2 link failure at t=2s (recover at t=5s)")

	fmt.Println("\nRunning simulation for 10 seconds...")
	err = sim5.Run(10.0)
	if err != nil {
		log.Fatalf("Simulation failed: %v", err)
	}

	sim5.PrintStatus()

	// Show delivery rate impact
	summary5 := sim5.GetMetrics().System.GetSummary()
	fmt.Printf("\nImpact of failures:\n")
	fmt.Printf("  Delivery Rate: %.1f%% (failures caused packet loss)\n", summary5.AvgDeliveryRate*100)

	fmt.Println("\n" + strings.Repeat("=", 50))

	fmt.Println("\n=== Layer 1 Complete ===")
	fmt.Println("✓ Discrete event simulation")
	fmt.Println("✓ Priority queue for events")
	fmt.Println("✓ Simulation clock with configurable speed")
	fmt.Println("✓ Network link simulation (bandwidth, latency, queuing)")
	fmt.Println("✓ Packet drop simulation")
	fmt.Println("✓ Traffic generation (Poisson, Bursty, Constant)")
	fmt.Println("✓ Multiple traffic flows")
	fmt.Println("✓ Time-series metrics collection")
	fmt.Println("✓ System, node, and link metrics")
	fmt.Println("✓ Node and link failure handling")
}
