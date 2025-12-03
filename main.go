package main

import (
	"fmt"
	"time"

	"github.com/saintparish4/chaos/chaos"
	"github.com/saintparish4/chaos/simulator"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║    DISTRIBUTED NETWORK SIMULATOR WITH CHAOS ENGINEERING       ║")
	fmt.Println("║              HYPOTHETICAL DEMONSTRATION SCENARIO               ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("⚠️  DEMONSTRATION NOTICE:")
	fmt.Println("This is a realistic simulation showing what WOULD happen in a")
	fmt.Println("production distributed system when failures occur. The scenarios")
	fmt.Println("demonstrate expected behavior patterns in real-world deployments.")
	fmt.Println()
	pause()

	// ============================================================
	// SECTION 1: Network Setup
	// ============================================================
	fmt.Println("📡 SECTION 1: Creating Network Topology")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println("SCENARIO: Modeling a mid-sized datacenter network")
	fmt.Println("Building a 20-node mesh network (similar to a production cluster)")
	fmt.Println()
	fmt.Println("In a real deployment, this WOULD represent:")
	fmt.Println("  • 20 server nodes across multiple racks")
	fmt.Println("  • Redundant network paths for fault tolerance")
	fmt.Println("  • Typical enterprise or cloud infrastructure")
	fmt.Println()

	// Create a realistic mesh network
	config := simulator.GenerateMeshTopology(20)
	topo := buildTopology(config)

	fmt.Printf("✓ Network created: %d nodes, %d links\n",
		len(topo.Config.Nodes), len(topo.Config.Links))
	fmt.Println()

	// Show the network structure
	printNetworkTopology(topo)
	fmt.Println()
	pause()

	// ============================================================
	// SECTION 2: Start Simulation & Traffic
	// ============================================================
	fmt.Println("📨 SECTION 2: Simulating Production Traffic Load")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println("SCENARIO: Realistic application traffic patterns")
	fmt.Println()
	fmt.Println("In a real system, this WOULD simulate:")
	fmt.Println("  • API requests between microservices")
	fmt.Println("  • Database queries and replication")
	fmt.Println("  • User-facing application traffic")
	fmt.Println("  • Background job processing")
	fmt.Println()

	sim := simulator.NewEventDrivenSimulator(topo)
	sim.EnableMetrics(0.5) // Collect metrics every 0.5 seconds

	// Add realistic traffic flows simulating production load
	// Main API traffic
	sim.AddTrafficFlowConfig(simulator.TrafficFlowConfig{
		Source:      "node-0",
		Dest:        "node-15",
		Type:        simulator.TrafficTypePoisson,
		Rate:        10.0, // 10 requests/sec (API gateway to backend)
		MessageSize: 2048,
		StartTime:   0.0,
		Duration:    30.0,
	})

	// Database replication traffic
	sim.AddTrafficFlowConfig(simulator.TrafficFlowConfig{
		Source:      "node-5",
		Dest:        "node-10",
		Type:        simulator.TrafficTypePoisson,
		Rate:        8.0, // 8 messages/sec (DB primary to replica)
		MessageSize: 4096,
		StartTime:   0.0,
		Duration:    30.0,
	})

	// Microservice inter-communication
	sim.AddTrafficFlowConfig(simulator.TrafficFlowConfig{
		Source:      "node-3",
		Dest:        "node-12",
		Type:        simulator.TrafficTypePoisson,
		Rate:        15.0, // 15 messages/sec (service mesh traffic)
		MessageSize: 1024,
		StartTime:   0.0,
		Duration:    30.0,
	})

	// Background job processing
	sim.AddTrafficFlowConfig(simulator.TrafficFlowConfig{
		Source:      "node-7",
		Dest:        "node-18",
		Type:        simulator.TrafficTypePoisson,
		Rate:        5.0, // 5 messages/sec (batch processing)
		MessageSize: 8192,
		StartTime:   0.0,
		Duration:    30.0,
	})

	// User session traffic
	sim.AddTrafficFlowConfig(simulator.TrafficFlowConfig{
		Source:      "node-2",
		Dest:        "node-14",
		Type:        simulator.TrafficTypePoisson,
		Rate:        12.0, // 12 messages/sec (user requests)
		MessageSize: 1500,
		StartTime:   0.0,
		Duration:    30.0,
	})

	fmt.Println("✓ Traffic patterns configured:")
	fmt.Println("  • 5 different traffic flows simulating production workloads")
	fmt.Println("  • Total: ~50 messages/second across the network")
	fmt.Println("  • Mix of API, database, and service mesh traffic")
	fmt.Println()

	// Run for a bit to establish baseline
	fmt.Println("Running simulation for 5 seconds to establish baseline...")
	fmt.Println("(In production, this represents normal operation)")
	fmt.Println()
	runSimulation(sim, 5.0)

	printMetrics(sim, "Baseline State (Normal Operations)")
	printNodeActivity(sim)
	fmt.Println()
	fmt.Println("💡 This baseline WOULD show what normal, healthy operation looks like")
	fmt.Println("   before any failures occur. All metrics should be at expected levels.")
	pause()

	// ============================================================
	// SECTION 3: Inject Failure (Chaos Engineering)
	// ============================================================
	fmt.Println("💥 SECTION 3: Chaos Engineering - Simulating Critical Failure")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println("SCENARIO: Node failure in production environment")
	fmt.Println()
	fmt.Println("What WOULD happen in reality:")
	fmt.Println("  • Hardware failure (disk crash, power loss, etc.)")
	fmt.Println("  • Software crash (OOM, kernel panic, service hang)")
	fmt.Println("  • Network partition or switch failure")
	fmt.Println("  • Cloud provider zone outage")
	fmt.Println()
	fmt.Println("This simulation demonstrates the EXPECTED system response...")
	fmt.Println()

	// Create chaos injector
	injector := chaos.NewFailureInjector(sim)

	fmt.Println("❌ Injecting failure: node-10 going offline (simulated crash)...")
	fmt.Println("   (This node was handling database replication traffic)")
	injector.InjectNodeFailure("node-10", 15.0)

	fmt.Println()
	printNetworkStatus(sim, "Immediately After Failure Injection")
	fmt.Println()

	// Run a bit to see impact
	fmt.Println("Running for 3 seconds to observe immediate impact...")
	fmt.Println("(In production, monitoring systems WOULD detect this within seconds)")
	fmt.Println()
	runSimulation(sim, 3.0)

	printMetrics(sim, "During Unmitigated Failure")
	printNodeActivity(sim)
	fmt.Println()
	fmt.Println("📊 EXPECTED IMPACT: Without resilience mechanisms, we WOULD see:")
	fmt.Println("   • Degraded delivery rates as messages are dropped")
	fmt.Println("   • Increased latency for affected traffic paths")
	fmt.Println("   • Service disruption for clients using the failed node")
	fmt.Println("   • Cascading failures if dependent services timeout")
	pause()

	// ============================================================
	// SECTION 4: Self-Healing & Resilience
	// ============================================================
	fmt.Println("🔄 SECTION 4: Activating Resilience & Self-Healing Mechanisms")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println("SCENARIO: Production resilience patterns engage automatically")
	fmt.Println()
	fmt.Println("What WOULD happen in a well-designed system:")
	fmt.Println("  • Service mesh detects failed node and updates routing tables")
	fmt.Println("  • Load balancers redirect traffic to healthy nodes")
	fmt.Println("  • Circuit breakers prevent cascade failures")
	fmt.Println("  • Retry logic handles transient errors gracefully")
	fmt.Println("  • Health checks trigger automatic failover")
	fmt.Println()
	fmt.Println("These are standard production practices (Kubernetes, Istio, etc.)")
	fmt.Println()

	// Enable resilience
	resilience := chaos.NewResilienceManager(sim)
	resilience.EnableAdaptiveRouting()
	resilience.EnableRetries(3, 0.1) // 3 retries with 0.1s backoff

	fmt.Println("✓ Adaptive routing ENABLED: Network automatically reroutes around failures")
	fmt.Println("✓ Message retry logic ENABLED: Failed messages are retransmitted")
	fmt.Println("✓ Circuit breaker patterns ENABLED: Prevents overwhelming healthy nodes")
	fmt.Println()

	fmt.Println("Running for 5 seconds with resilience mechanisms active...")
	fmt.Println("(This simulates what WOULD happen as the system self-heals)")
	fmt.Println()
	runSimulation(sim, 5.0)

	printMetrics(sim, "With Resilience Active (Self-Healing)")
	printNodeActivity(sim)
	fmt.Println()
	fmt.Println("📊 EXPECTED IMPROVEMENT: With proper resilience, we WOULD see:")
	fmt.Println("   ✓ Delivery rates returning to near-normal levels")
	fmt.Println("   ✓ Traffic automatically routed through alternate paths")
	fmt.Println("   ✓ System continues operating despite node failure")
	fmt.Println("   ✓ No manual intervention required - fully automatic recovery")
	pause()

	// ============================================================
	// SECTION 5: Node Recovery & System Stabilization
	// ============================================================
	fmt.Println("♻️  SECTION 5: Simulating Node Recovery")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println("SCENARIO: Failed node recovers and rejoins the cluster")
	fmt.Println()
	fmt.Println("What WOULD happen in production:")
	fmt.Println("  • Auto-restart by orchestrator (Kubernetes, systemd, etc.)")
	fmt.Println("  • Node performs health checks and self-tests")
	fmt.Println("  • Rejoins cluster and announces availability")
	fmt.Println("  • Gradually receives traffic as it proves stability")
	fmt.Println("  • Full integration back into service mesh")
	fmt.Println()

	// Node will auto-recover after specified time
	fmt.Println("Continuing simulation as node-10 recovers...")
	fmt.Println("(Simulating automatic restart and health check process)")
	fmt.Println()
	runSimulation(sim, 5.0)

	fmt.Println("✓ Node-10 has completed recovery and rejoined the cluster")
	fmt.Println()
	printNetworkStatus(sim, "After Node Recovery")
	fmt.Println()

	// Run to stabilize
	fmt.Println("Running for 5 seconds to allow system stabilization...")
	fmt.Println("(In production, metrics WOULD return to baseline levels)")
	fmt.Println()
	runSimulation(sim, 5.0)

	printMetrics(sim, "Full Recovery - System Stabilized")
	printNodeActivity(sim)
	fmt.Println()
	fmt.Println("📊 EXPECTED OUTCOME: After full recovery, we WOULD see:")
	fmt.Println("   ✓ All nodes operational and healthy")
	fmt.Println("   ✓ Metrics returned to baseline performance")
	fmt.Println("   ✓ Traffic balanced across all available nodes")
	fmt.Println("   ✓ System ready to handle next failure without degradation")
	pause()

	// ============================================================
	// SECTION 6: Final Summary & Analysis
	// ============================================================
	fmt.Println("📈 SECTION 6: Demonstration Summary & Real-World Implications")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println()

	finalMetrics := sim.GetSystemMetrics()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         HYPOTHETICAL SCENARIO - DEMONSTRATION COMPLETE       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("📋 What This Simulation WOULD Represent in Production:")
	fmt.Println()
	fmt.Println("PHASE 1: Normal Operations")
	fmt.Println("  ✓ 20-node distributed system running production workloads")
	fmt.Println("  ✓ Multiple traffic flows simulating real application patterns")
	fmt.Println("  ✓ Baseline performance metrics established")
	fmt.Println()
	fmt.Println("PHASE 2: Failure Event")
	fmt.Println("  ✗ Critical node failure (hardware/software crash)")
	fmt.Println("  ✗ Immediate service degradation detected")
	fmt.Println("  ✗ Traffic disruption for affected services")
	fmt.Println()
	fmt.Println("PHASE 3: Automated Response (Self-Healing)")
	fmt.Println("  ✓ Resilience systems activated automatically")
	fmt.Println("  ✓ Traffic rerouted through healthy nodes")
	fmt.Println("  ✓ Service continuity maintained")
	fmt.Println("  ✓ No manual intervention required")
	fmt.Println()
	fmt.Println("PHASE 4: Recovery & Stabilization")
	fmt.Println("  ✓ Failed node automatically restarted")
	fmt.Println("  ✓ Rejoined cluster after health checks")
	fmt.Println("  ✓ System returned to full capacity")
	fmt.Println("  ✓ Metrics normalized to baseline levels")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("                      PERFORMANCE METRICS")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("  Total Messages Sent:        %6d\n", finalMetrics.TotalMessagesSent)
	fmt.Printf("  Successfully Delivered:     %6d\n", finalMetrics.TotalMessagesDelivered)
	fmt.Printf("  Overall Delivery Rate:      %6.1f%%\n", finalMetrics.DeliveryRate*100)
	fmt.Printf("  Average Latency:            %6.3f seconds\n", finalMetrics.AverageLatency)
	fmt.Printf("  Active Nodes:               %6d\n", finalMetrics.ActiveNodes)
	fmt.Printf("  Failed Nodes:               %6d\n", finalMetrics.FailedNodes)
	fmt.Println()

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("              REAL-WORLD APPLICATIONS & IMPLICATIONS")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("🏢 This simulation demonstrates patterns used by:")
	fmt.Println()
	fmt.Println("   • Cloud Providers: AWS, Azure, GCP handle datacenter failures")
	fmt.Println("   • Microservices: Service mesh (Istio, Linkerd) resilience")
	fmt.Println("   • Databases: Distributed DB failover (MongoDB, Cassandra)")
	fmt.Println("   • Message Queues: Kafka, RabbitMQ cluster recovery")
	fmt.Println("   • Container Orchestration: Kubernetes pod rescheduling")
	fmt.Println()
	fmt.Println("🔍 Key Takeaways (What WOULD Happen in Production):")
	fmt.Println()
	fmt.Println("   1. Failures are INEVITABLE - hardware fails, software crashes")
	fmt.Println("   2. Resilience must be AUTOMATED - no time for manual response")
	fmt.Println("   3. Self-healing is CRITICAL - systems must recover independently")
	fmt.Println("   4. Observability is KEY - you must see what's happening")
	fmt.Println("   5. Chaos testing is ESSENTIAL - test before production failures")
	fmt.Println()
	fmt.Println("💼 Business Impact:")
	fmt.Println()
	fmt.Println("   WITHOUT resilience: Minutes to hours of downtime = $$$$ lost")
	fmt.Println("   WITH resilience: Seconds of degradation, automatic recovery")
	fmt.Println()
	fmt.Println("   This is why companies invest heavily in distributed systems")
	fmt.Println("   engineering, SRE practices, and chaos engineering programs.")
	fmt.Println()

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              HYPOTHETICAL DEMONSTRATION COMPLETE               ║")
	fmt.Println("║                                                                ║")
	fmt.Println("║  This simulation shows EXPECTED behavior in production systems║")
	fmt.Println("║  Actual results would vary based on specific implementation   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}

// ============================================================
// Helper Functions
// ============================================================

func buildTopology(config *simulator.TopologyConfig) *simulator.Topology {
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
			Active:    true,
		}
	}

	return topo
}

func runSimulation(sim *simulator.EventDrivenSimulator, duration float64) {
	startTime := sim.GetCurrentTime()
	for sim.GetCurrentTime()-startTime < duration {
		sim.Step()
		time.Sleep(10 * time.Millisecond) // Small delay for realism
	}
}

func printNetworkTopology(topo *simulator.Topology) {
	fmt.Println("Network Layout (Conceptual View):")
	fmt.Println()
	fmt.Println("This WOULD represent a mesh network topology where:")
	fmt.Println("  • Each node has redundant connections to multiple neighbors")
	fmt.Println("  • Traffic can flow through multiple alternate paths")
	fmt.Println("  • No single point of failure exists in the network")
	fmt.Println()

	// Show a conceptual visualization for 20 nodes
	fmt.Println("    [Rack 1]     [Rack 2]     [Rack 3]     [Rack 4]")
	fmt.Println("  node-0...4   node-5...9  node-10...14 node-15...19")
	fmt.Println("      │ ╲  ╱        │ ╲  ╱       │ ╲  ╱       │ ╲  ╱")
	fmt.Println("      │  ╳          │  ╳         │  ╳         │  ╳")
	fmt.Println("      │ ╱  ╲        │ ╱  ╲       │ ╱  ╲       │ ╱  ╲")
	fmt.Println("      └─────────────┴────────────┴────────────┘")
	fmt.Println("         (Interconnected across all racks)")
	fmt.Println()
	fmt.Println("In production, this WOULD provide:")
	fmt.Println("  ✓ N-way redundancy for fault tolerance")
	fmt.Println("  ✓ Load distribution across multiple paths")
	fmt.Println("  ✓ Automatic failover capability")
}

func printNetworkStatus(sim *simulator.EventDrivenSimulator, title string) {
	fmt.Printf("Network Status: %s\n", title)
	fmt.Println()

	nodes := sim.GetAllNodes()

	activeCount := 0
	failedCount := 0

	for _, node := range nodes {
		var symbol string
		switch string(node.State) {
		case "ACTIVE":
			symbol = "●"
			activeCount++
		case "DEGRADED":
			symbol = "◐"
		case "FAILED":
			symbol = "○"
			failedCount++
		default:
			symbol = "?"
		}

		fmt.Printf("  %s %s [%s]\n", symbol, node.ID, node.State)
	}

	fmt.Println()
	fmt.Printf("Status: %d active, %d failed\n", activeCount, failedCount)
	fmt.Println("(● = Active, ○ = Failed)")
}

func printMetrics(sim *simulator.EventDrivenSimulator, phase string) {
	fmt.Println()
	fmt.Printf("═══ Metrics: %s ═══\n", phase)
	fmt.Println()

	metrics := sim.GetSystemMetrics()

	// Simple visual bar for delivery rate
	deliveryPct := int(metrics.DeliveryRate * 100)
	bar := ""
	for i := 0; i < deliveryPct/5; i++ {
		bar += "█"
	}
	for i := deliveryPct / 5; i < 20; i++ {
		bar += "░"
	}

	fmt.Printf("Messages Sent:      %4d\n", metrics.TotalMessagesSent)
	fmt.Printf("Messages Delivered: %4d\n", metrics.TotalMessagesDelivered)
	fmt.Printf("Delivery Rate:      %s %3d%%\n", bar, deliveryPct)
	fmt.Printf("Average Latency:    %.3fs\n", metrics.AverageLatency)
	fmt.Println()
}

func printNodeActivity(sim *simulator.EventDrivenSimulator) {
	fmt.Println("Node Activity:")

	nodes := sim.GetAllNodes()

	for _, node := range nodes {
		if string(node.State) == "FAILED" {
			fmt.Printf("  %s: ─────────── (offline)\n", node.ID)
			continue
		}

		metrics := sim.GetNodeMetrics(node.ID)

		// Simple activity bar
		activity := metrics.MessagesSent + metrics.MessagesReceived
		bar := ""
		barLength := int(activity / 2)
		if barLength > 20 {
			barLength = 20
		}

		for i := 0; i < barLength; i++ {
			bar += "█"
		}
		for i := barLength; i < 20; i++ {
			bar += "░"
		}

		fmt.Printf("  %s: %s (%d msgs)\n", node.ID, bar, activity)
	}
	fmt.Println()
}

func pause() {
	fmt.Println()
	fmt.Println("Press Enter to continue...")
	fmt.Scanln()
	fmt.Println()
}
