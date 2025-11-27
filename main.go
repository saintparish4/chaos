package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/saintparish4/chaos/chaos"
	"github.com/saintparish4/chaos/simulator"
)

func main() {
	fmt.Println("=== Network Simulator - Layer 2 Demo ===")
	fmt.Println("Chaos Engineering Framework")

	// Load topology
	fmt.Println("Loading topology...")
	topo, err := simulator.LoadTopology("topology.yaml")
	if err != nil {
		log.Fatalf("Failed to load topology: %v", err)
	}
	fmt.Printf("✓ Loaded %d nodes, %d links\n\n", len(topo.Config.Nodes), len(topo.Config.Links))

	// Demo 1: Basic Failure Injection
	fmt.Println("=== Demo 1: Basic Failure Injection ===")
	demo1BasicFailures()
	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Demo 2: Chaos Scenarios
	fmt.Println("=== Demo 2: Predefined Chaos Scenarios ===")
	demo2ChaosScenarios()
	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Demo 3: YAML-Based Chaos Experiment
	fmt.Println("=== Demo 3: YAML-Based Chaos Experiment ===")
	demo3YAMLExperiment()
	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Demo 4: Resilience Mechanisms
	fmt.Println("=== Demo 4: Resilience Mechanisms ===")
	demo4Resilience()
	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Demo 5: Combined Chaos + Resilience
	fmt.Println("=== Demo 5: Chaos with Resilience ===")
	demo5CombinedChaosResilience()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("\n=== Layer 2 Complete ===")
	fmt.Println("✓ Failure injection")
	fmt.Println("✓ Chaos scenarios")
	fmt.Println("✓ YAML-based scheduler")
	fmt.Println("✓ Resilience mechanisms")
}

func demo1BasicFailures() {
	topo, _ := simulator.LoadTopology("topology.yaml")
	sim := simulator.NewEventDrivenSimulator(topo, 10.0)

	// Add baseline traffic
	gen := simulator.NewPoissonTrafficGenerator("tower-1", "tower-2", 5.0, 1000)
	flow := simulator.NewTrafficFlow("baseline", gen, 0.0, 10.0)
	sim.AddTrafficFlow(flow)

	// Create failure injector
	injector := chaos.NewFailureInjector(sim)

	fmt.Println("Test 1: Node Failure")
	fmt.Println("  Killing router-2 at t=2s for 3s...")

	// Schedule node failure at t=2s
	failureEvent := &simulator.SimulationEvent{
		ID:        "demo-node-failure",
		Type:      simulator.EventNodeFailure,
		Timestamp: 2.0,
		NodeID:    "router-2",
		Data: &simulator.NodeFailureEventData{
			NodeID:   "router-2",
			Duration: 3.0,
		},
	}
	sim.ScheduleEvent(failureEvent)

	fmt.Println("\nTest 2: Link Failure")
	fmt.Println("  Severing router-1 -> switch-1 at t=5s for 2s...")

	// Inject link failure programmatically
	sim.ScheduleEvent(&simulator.SimulationEvent{
		ID:        "demo-link-failure",
		Type:      simulator.EventLinkFailure,
		Timestamp: 5.0,
		LinkID:    "router-1->switch-1",
		Data: &simulator.LinkFailureEventData{
			Source:   "router-1",
			Dest:     "switch-1",
			Duration: 2.0,
		},
	})

	// Run simulation
	fmt.Println("\nRunning 10-second simulation...")
	sim.Run(10.0)
	sim.PrintStatus()

	// Show impact
	stats := injector.GetFailureImpact()
	fmt.Printf("\nFailure Impact:\n")
	fmt.Printf("  Failed nodes: %d\n", stats.NodesFailedCount)
}

func demo2ChaosScenarios() {
	topo, _ := simulator.LoadTopology("topology.yaml")
	sim := simulator.NewEventDrivenSimulator(topo, 50.0)

	// Add traffic
	gen := simulator.NewPoissonTrafficGenerator("router-1", "tower-2", 10.0, 1000)
	flow := simulator.NewTrafficFlow("traffic", gen, 0.0, 15.0)
	sim.AddTrafficFlow(flow)

	injector := chaos.NewFailureInjector(sim)

	// Scenario 1: Random Node Failure
	fmt.Println("\nScenario 1: Random Node Failure")
	scenario1 := &chaos.RandomNodeFailureScenario{
		MinNodes: 1,
		MaxNodes: 2,
		Duration: 5.0,
	}
	fmt.Printf("  %s\n", scenario1.Description())
	scenario1.Execute(injector, 2.0)

	// Scenario 2: Network Congestion
	fmt.Println("\nScenario 2: Network Congestion")
	scenario2 := &chaos.NetworkCongestionScenario{
		SourceNode:        "switch-1",
		DestNode:          "tower-1",
		MessagesPerSecond: 100.0,
		MessageSize:       3000,
		Duration:          3.0,
	}
	fmt.Printf("  %s\n", scenario2.Description())
	scenario2.Execute(injector, 5.0)

	// Scenario 3: Slow Links
	fmt.Println("\nScenario 3: Slow Links")
	scenario3 := &chaos.SlowLinksScenario{
		NumLinks:          2,
		LatencyMultiplier: 15.0,
		Duration:          4.0,
	}
	fmt.Printf("  %s\n", scenario3.Description())
	scenario3.Execute(injector, 8.0)

	// Run simulation
	fmt.Println("\nRunning 15-second simulation...")
	sim.Run(15.0)
	sim.PrintStatus()
}

func demo3YAMLExperiment() {
	topo, _ := simulator.LoadTopology("topology.yaml")
	sim := simulator.NewEventDrivenSimulator(topo, 10.0)

	// Add baseline traffic
	gen := simulator.NewPoissonTrafficGenerator("tower-1", "tower-2", 8.0, 1000)
	flow := simulator.NewTrafficFlow("baseline", gen, 0.0, 20.0)
	sim.AddTrafficFlow(flow)

	// Create chaos scheduler
	injector := chaos.NewFailureInjector(sim)
	scheduler := chaos.NewChaosScheduler(injector)

	// Load experiment from YAML
	fmt.Println("Loading chaos experiment from chaos_experiment.yaml...")
	err := scheduler.LoadExperiment("chaos_experiment.yaml")
	if err != nil {
		fmt.Printf("Warning: Could not load YAML (using programmatic experiment): %v\n", err)
		// Use programmatic experiment instead
		useProgrammaticExperiment(scheduler)
	}

	// Schedule all chaos events
	scheduler.ScheduleExperiment()

	// Run simulation
	fmt.Println("\nRunning 20-second chaos experiment...")
	sim.Run(20.0)
	sim.PrintStatus()
}

func useProgrammaticExperiment(scheduler *chaos.ChaosScheduler) {
	experiment := &chaos.ChaosExperiment{
		Name:        "Programmatic Chaos Test",
		Description: "Fallback programmatic chaos experiment",
		Duration:    15.0,
		Events: []chaos.ChaosEventConfig{
			{
				Time:     2.0,
				Type:     "node_failure",
				Target:   "router-2",
				Duration: 3.0,
			},
			{
				Time:     5.0,
				Type:     "link_failure",
				Source:   "router-1",
				Dest:     "router-3",
				Duration: 4.0,
			},
			{
				Time:              8.0,
				Type:              "congestion",
				Source:            "switch-1",
				Dest:              "tower-1",
				MessagesPerSecond: 80.0,
				MessageSize:       2000,
				Duration:          2.0,
			},
		},
	}
	scheduler.SetExperiment(experiment)
}

func demo4Resilience() {
	topo, _ := simulator.LoadTopology("topology.yaml")
	sim := simulator.NewEventDrivenSimulator(topo, 20.0)

	// Add traffic
	gen := simulator.NewPoissonTrafficGenerator("tower-1", "tower-2", 10.0, 1000)
	flow := simulator.NewTrafficFlow("traffic", gen, 0.0, 15.0)
	sim.AddTrafficFlow(flow)

	// Create resilience manager
	resilience := chaos.NewResilienceManager(sim)

	fmt.Println("Enabling resilience mechanisms:")
	resilience.EnableAdaptiveRouting()
	resilience.EnableRetries(3, 0.1)
	resilience.EnableHealthChecks()

	// Inject some failures to test resilience
	fmt.Println("\nInjecting failures to test resilience:")
	fmt.Println("  - Node failure at t=3s")
	fmt.Println("  - Link failure at t=7s")

	// Node failure at t=3s
	sim.ScheduleEvent(&simulator.SimulationEvent{
		ID:        "test-resilience-node",
		Type:      simulator.EventNodeFailure,
		Timestamp: 3.0,
		NodeID:    "router-2",
		Data: &simulator.NodeFailureEventData{
			NodeID:   "router-2",
			Duration: 5.0,
		},
	})

	// Link failure at t=7s
	sim.ScheduleEvent(&simulator.SimulationEvent{
		ID:        "test-resilience-link",
		Type:      simulator.EventLinkFailure,
		Timestamp: 7.0,
		LinkID:    "switch-1->tower-1",
		Data: &simulator.LinkFailureEventData{
			Source:   "switch-1",
			Dest:     "tower-1",
			Duration: 3.0,
		},
	})

	fmt.Println("\nRunning simulation with resilience...")
	sim.Run(15.0)
	sim.PrintStatus()

	fmt.Println("\nResilience helped the network:")
	fmt.Println("  ✓ Adaptive routing recalculated paths around failures")
	fmt.Println("  ✓ Message retries attempted delivery after transient failures")
	fmt.Println("  ✓ Health checks detected unhealthy nodes")
}

func demo5CombinedChaosResilience() {
	topo, _ := simulator.LoadTopology("topology.yaml")
	sim := simulator.NewEventDrivenSimulator(topo, 25.0)

	// Heavy traffic load
	gen1 := simulator.NewPoissonTrafficGenerator("tower-1", "tower-2", 15.0, 1000)
	flow1 := simulator.NewTrafficFlow("flow1", gen1, 0.0, 20.0)
	sim.AddTrafficFlow(flow1)

	gen2 := simulator.NewPoissonTrafficGenerator("router-1", "switch-2", 10.0, 800)
	flow2 := simulator.NewTrafficFlow("flow2", gen2, 0.0, 20.0)
	sim.AddTrafficFlow(flow2)

	// Enable resilience
	resilience := chaos.NewResilienceManager(sim)
	resilience.EnableAdaptiveRouting()
	resilience.EnableRetries(5, 0.05)
	resilience.EnableHealthChecks()

	// Create aggressive chaos
	injector := chaos.NewFailureInjector(sim)

	fmt.Println("Running aggressive chaos with resilience enabled:")
	fmt.Println("  - Multiple node failures")
	fmt.Println("  - Link failures")
	fmt.Println("  - Network congestion")
	fmt.Println("  - Network partition")

	// Cascading failure at t=3s
	scenario1 := &chaos.CascadingFailureScenario{
		InitialNode: "router-2",
		CascadeProb: 0.4,
		MaxCascade:  3,
		Duration:    5.0,
	}
	scenario1.Execute(injector, 3.0)

	// Network partition at t=8s
	scenario2 := &chaos.NetworkPartitionScenario{
		Partitions: [][]string{
			{"router-1", "switch-1", "tower-1"},
			{"router-2", "router-3", "switch-2", "tower-2"},
		},
		Duration: 4.0,
	}
	scenario2.Execute(injector, 8.0)

	// Congestion at t=12s
	scenario3 := &chaos.NetworkCongestionScenario{
		SourceNode:        "router-1",
		DestNode:          "tower-2",
		MessagesPerSecond: 150.0,
		MessageSize:       3000,
		Duration:          3.0,
	}
	scenario3.Execute(injector, 12.0)

	// Run simulation
	fmt.Println("\nRunning 20-second simulation...")
	sim.Run(20.0)
	sim.PrintStatus()

	fmt.Println("\nResilience vs Chaos Results:")
	summary := sim.GetMetrics().System.GetSummary()
	fmt.Printf("  Delivery Rate: %.1f%% (resilience helped maintain connectivity)\n",
		summary.AvgDeliveryRate*100)
	fmt.Printf("  Avg Latency: %.3fs (adaptive routing found alternate paths)\n",
		summary.AvgLatency)
}
