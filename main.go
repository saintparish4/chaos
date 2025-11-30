package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/saintparish4/chaos/congestion"
	"github.com/saintparish4/chaos/qos"
	"github.com/saintparish4/chaos/routing"
	"github.com/saintparish4/chaos/simulator"
)

func main() {
	fmt.Println("=== Network Simulator - Layer 3 Demo ===")
	fmt.Println("Advanced Network Features")

	// Demo 1: Routing Protocol Comparison
	fmt.Println("=== Demo 1: Routing Protocol Comparison ===")
	demo1RoutingProtocols()
	fmt.Println("\n" + strings.Repeat("=", 70))

	// Demo 2: BGP Policy Routing
	fmt.Println("=== Demo 2: BGP Policy Routing ===")
	demo2BGPPolicies()
	fmt.Println("\n" + strings.Repeat("=", 70))

	// Demo 3: TCP Congestion Control
	fmt.Println("=== Demo 3: TCP Congestion Control ===")
	demo3CongestionControl()
	fmt.Println("\n" + strings.Repeat("=", 70))

	// Demo 4: Quality of Service (QoS)
	fmt.Println("=== Demo 4: Quality of Service (QoS) ===")
	demo4QualityOfService()
	fmt.Println("\n" + strings.Repeat("=", 70))

	// Demo 5: Combined Advanced Features
	fmt.Println("=== Demo 5: Combined Advanced Features ===")
	demo5Combined()

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("\n=== Layer 3 Complete ===")
	fmt.Println("✓ Routing protocols (BGP, OSPF, RIP)")
	fmt.Println("✓ Congestion control (TCP-like)")
	fmt.Println("✓ Quality of Service (QoS)")
}

func demo1RoutingProtocols() {
	fmt.Println("Comparing BGP, OSPF, and RIP on the same topology")

	// Load topology
	topo, err := simulator.LoadTopology("topology.yaml")
	if err != nil {
		log.Printf("Warning: Could not load topology, using generated one\n")
		config := simulator.GenerateMeshTopology(6)
		topo = buildTopology(config)
	}

	// Create protocol manager
	_ = routing.NewProtocolManager(topo)

	// Test BGP
	fmt.Println("1. BGP (Border Gateway Protocol)")
	bgp := routing.NewBGPProtocol(65001)
	bgp.Initialize("router-1", topo)
	bgp.SetPolicyPreference("router-2", 100) // Prefer routes through router-2

	// Announce some routes
	bgp.AnnounceRoute("network-1", 1)
	bgp.AnnounceRoute("network-2", 2)

	// Simulate receiving updates
	update1 := &routing.RoutingUpdate{
		SourceNode: "router-2",
		Protocol:   "BGP",
		Routes: []routing.RouteEntry{
			{Destination: "network-3", Metric: 2, Path: []string{"router-2", "network-3"}},
		},
	}
	bgp.ProcessUpdate(update1)

	bgp.PrintRoutingTable()
	fmt.Printf("Metrics: %s\n\n", bgp.GetMetrics().GetSummary())

	// Test OSPF
	fmt.Println("2. OSPF (Open Shortest Path First)")
	ospf := routing.NewOSPFProtocol(0)
	ospf.Initialize("router-1", topo)

	// In OSPF, routes are calculated from link-state database
	ospf.PrintLinkStateDB()
	ospf.PrintRoutingTable()
	fmt.Printf("Metrics: %s\n\n", ospf.GetMetrics().GetSummary())

	// Test RIP
	fmt.Println("3. RIP (Routing Information Protocol)")
	rip := routing.NewRIPProtocol()
	rip.Initialize("router-1", topo)

	// Simulate receiving RIP updates
	update2 := &routing.RoutingUpdate{
		SourceNode: "router-2",
		Protocol:   "RIP",
		Routes: []routing.RouteEntry{
			{Destination: "router-3", Metric: 1, Path: []string{"router-2", "router-3"}},
			{Destination: "switch-1", Metric: 2, Path: []string{"router-2", "router-3", "switch-1"}},
		},
	}
	rip.ProcessUpdate(update2)

	rip.PrintRoutingTable()
	fmt.Printf("Metrics: %s\n", rip.GetMetrics().GetSummary())
	fmt.Printf("Average hop count: %.2f\n", rip.ComputeAverageHopCount())

	// Compare protocols
	fmt.Println("\n=== Protocol Comparison ===")
	fmt.Printf("%-10s %-15s %-15s %-20s\n", "Protocol", "Table Size", "Updates Sent", "Overhead (bytes)")
	fmt.Println("---------------------------------------------------------------")

	for name, proto := range map[string]routing.RoutingProtocol{
		"BGP":  bgp,
		"OSPF": ospf,
		"RIP":  rip,
	} {
		metrics := proto.GetMetrics()
		table := proto.GetRoutingTable()
		fmt.Printf("%-10s %-15d %-15d %-20d\n",
			name, len(table), metrics.UpdatesSent, metrics.ProtocolOverhead)
	}
}

func demo2BGPPolicies() {
	fmt.Println("Demonstrating BGP policy-based routing")

	config := simulator.GenerateRingTopology(5)
	topo := buildTopology(config)

	// Create multiple BGP routers with different AS numbers
	bgp1 := routing.NewBGPProtocol(65001)
	bgp1.Initialize("node-0", topo)

	bgp2 := routing.NewBGPProtocol(65002)
	bgp2.Initialize("node-1", topo)

	bgp3 := routing.NewBGPProtocol(65003)
	bgp3.Initialize("node-2", topo)

	// Set policy preferences
	fmt.Println("Setting BGP policies:")
	fmt.Println("  node-0: Prefers routes through node-1 (preference=100)")
	fmt.Println("  node-0: Avoids routes through node-2 (preference=-50)")

	bgp1.SetPolicyPreference("node-1", 100)
	bgp1.SetPolicyPreference("node-2", -50)

	// Announce routes
	bgp2.AnnounceRoute("dest-A", 1)
	bgp3.AnnounceRoute("dest-A", 1) // Same destination, different path

	// Simulate updates
	update1 := &routing.RoutingUpdate{
		SourceNode: "node-1",
		Protocol:   "BGP",
		Routes: []routing.RouteEntry{
			{Destination: "dest-A", Metric: 2, Path: []string{"node-1", "dest-A"}},
		},
	}
	update2 := &routing.RoutingUpdate{
		SourceNode: "node-2",
		Protocol:   "BGP",
		Routes: []routing.RouteEntry{
			{Destination: "dest-A", Metric: 2, Path: []string{"node-2", "dest-A"}},
		},
	}

	bgp1.ProcessUpdate(update1)
	bgp1.ProcessUpdate(update2)

	fmt.Println("Result after policy application:")
	bgp1.PrintRoutingTable()

	// Show AS path
	asPath := bgp1.GetASPath("dest-A")
	if len(asPath) > 0 {
		fmt.Printf("AS Path to dest-A: %v\n", asPath)
	}
}

func demo3CongestionControl() {
	fmt.Println("Demonstrating TCP-like congestion control")

	// Create congestion controller
	cc := congestion.NewCongestionController("conn-1", "client", "server")

	fmt.Println("Initial state:")
	fmt.Printf("  %s\n\n", cc.GetMetrics().String())

	// Simulate slow start
	fmt.Println("=== Slow Start Phase ===")
	for i := 0; i < 5; i++ {
		cc.OnAck(1)
		metrics := cc.GetMetrics()
		fmt.Printf("ACK %d: %s\n", i+1, metrics.String())
	}

	fmt.Println("\n=== Congestion Avoidance Phase ===")
	for i := 0; i < 5; i++ {
		cc.OnAck(1)
		metrics := cc.GetMetrics()
		fmt.Printf("ACK %d: %s\n", i+6, metrics.String())
	}

	fmt.Println("\n=== Packet Loss Event ===")
	cc.OnLoss()
	fmt.Printf("After loss: %s\n", cc.GetMetrics().String())

	fmt.Println("\n=== Recovery ===")
	for i := 0; i < 3; i++ {
		cc.OnAck(1)
		metrics := cc.GetMetrics()
		fmt.Printf("ACK %d: %s\n", i+1, metrics.String())
	}

	fmt.Println("\n=== Timeout Event ===")
	cc.OnTimeout()
	fmt.Printf("After timeout: %s\n", cc.GetMetrics().String())

	// Update RTT
	fmt.Println("\n=== RTT Measurement ===")
	cc.UpdateRTT(0.05) // 50ms
	cc.UpdateRTT(0.08) // 80ms
	cc.UpdateRTT(0.06) // 60ms
	fmt.Printf("Updated RTT: %s\n", cc.GetMetrics().String())

	// Fair queuing demonstration
	fmt.Println("\n=== Fair Queuing ===")
	fq := congestion.NewFairQueuingScheduler()

	// Enqueue packets from different flows
	for i := 0; i < 3; i++ {
		fq.Enqueue("flow-1", fmt.Sprintf("packet-1-%d", i))
		fq.Enqueue("flow-2", fmt.Sprintf("packet-2-%d", i))
		fq.Enqueue("flow-3", fmt.Sprintf("packet-3-%d", i))
	}

	fmt.Printf("Queue depth: %d packets\n", fq.GetQueueDepth())
	fmt.Println("Dequeuing (round-robin):")
	for i := 0; i < 9; i++ {
		flowID, packet := fq.Dequeue()
		if packet != nil {
			fmt.Printf("  %d. Flow %s: %v\n", i+1, flowID, packet)
		}
	}

	stats := fq.GetFlowStats()
	fmt.Println("\nFlow statistics:")
	for flowID, count := range stats {
		fmt.Printf("  %s: %d packets sent\n", flowID, count)
	}
}

func demo4QualityOfService() {
	fmt.Println("Demonstrating QoS with priority queues and traffic shaping")

	// Create priority queue
	fmt.Println("=== Priority Queue ===")
	pq := qos.NewPriorityQueue(10)

	// Enqueue packets with different priorities
	packets := []qos.Packet{
		{ID: "p1", Priority: qos.PriorityLow, Size: 1000, Source: "A", Dest: "B"},
		{ID: "p2", Priority: qos.PriorityHigh, Size: 500, Source: "C", Dest: "D"},
		{ID: "p3", Priority: qos.PriorityMedium, Size: 800, Source: "E", Dest: "F"},
		{ID: "p4", Priority: qos.PriorityHigh, Size: 600, Source: "G", Dest: "H"},
		{ID: "p5", Priority: qos.PriorityLow, Size: 1200, Source: "I", Dest: "J"},
	}

	fmt.Println("Enqueuing packets:")
	for _, p := range packets {
		pq.Enqueue(p)
		fmt.Printf("  %s (%s priority, %d bytes)\n", p.ID, p.Priority, p.Size)
	}

	depths := pq.GetDepthByPriority()
	fmt.Printf("\nQueue depths: High=%d, Medium=%d, Low=%d\n",
		depths[qos.PriorityHigh], depths[qos.PriorityMedium], depths[qos.PriorityLow])

	fmt.Println("\nDequeuing (strict priority):")
	for i := 0; i < 5; i++ {
		packet, ok := pq.Dequeue()
		if ok {
			fmt.Printf("  %d. %s (%s priority)\n", i+1, packet.ID, packet.Priority)
		}
	}

	// Traffic shaping
	fmt.Println("\n=== Traffic Shaping (Token Bucket) ===")
	shaper := qos.NewTrafficShaper(1000.0, 5000) // 1000 bytes/sec, 5000 byte burst

	fmt.Println("Testing token bucket with various packet sizes:")
	testSizes := []int{1000, 2000, 3000, 1500, 2500}
	for i, size := range testSizes {
		canSend := shaper.CanSend(size)
		status := "ALLOWED"
		if !canSend {
			status = "SHAPED/DROPPED"
		}
		fmt.Printf("  Packet %d (%d bytes): %s\n", i+1, size, status)

		if i < len(testSizes)-1 {
			time.Sleep(100 * time.Millisecond) // Allow token refill
		}
	}

	shaperStats := shaper.GetStats()
	fmt.Printf("\nShaper stats: Shaped=%d, Dropped=%d, Bytes=%d\n",
		shaperStats.PacketsShaped, shaperStats.PacketsDropped, shaperStats.BytesShaped)

	// Packet classification
	fmt.Println("\n=== Packet Classification ===")
	classifier := qos.NewPacketClassifier()

	// Add classification rules
	classifier.AddRule(qos.ClassificationRule{
		Name:        "VoIP Traffic",
		SourceMatch: "voip-server",
		DestMatch:   "*",
		Priority:    qos.PriorityHigh,
		DSCP:        46, // EF (Expedited Forwarding)
	})
	classifier.AddRule(qos.ClassificationRule{
		Name:        "Video Traffic",
		SourceMatch: "video-server",
		DestMatch:   "*",
		Priority:    qos.PriorityMedium,
		DSCP:        34, // AF41
	})
	classifier.AddRule(qos.ClassificationRule{
		Name:        "Best Effort",
		SourceMatch: "*",
		DestMatch:   "*",
		Priority:    qos.PriorityLow,
		DSCP:        0,
	})

	// Classify some packets
	testPackets := []struct{ source, dest string }{
		{"voip-server", "client-1"},
		{"video-server", "client-2"},
		{"web-server", "client-3"},
		{"voip-server", "client-4"},
	}

	fmt.Println("Classification results:")
	for _, tp := range testPackets {
		priority, dscp := classifier.Classify(tp.source, tp.dest)
		fmt.Printf("  %s -> %s: Priority=%s, DSCP=%d\n",
			tp.source, tp.dest, priority, dscp)
	}

	// QoS metrics
	fmt.Println("\n=== QoS Metrics ===")
	qosMetrics := qos.NewQoSMetrics()

	// Record some measurements
	qosMetrics.RecordDelay(qos.PriorityHigh, 0.010)   // 10ms
	qosMetrics.RecordDelay(qos.PriorityHigh, 0.012)   // 12ms
	qosMetrics.RecordDelay(qos.PriorityMedium, 0.050) // 50ms
	qosMetrics.RecordDelay(qos.PriorityMedium, 0.055) // 55ms
	qosMetrics.RecordDelay(qos.PriorityLow, 0.100)    // 100ms
	qosMetrics.RecordDelay(qos.PriorityLow, 0.120)    // 120ms

	// Record packet stats
	qosMetrics.RecordPacket(qos.PriorityHigh, false)   // Sent
	qosMetrics.RecordPacket(qos.PriorityHigh, false)   // Sent
	qosMetrics.RecordPacket(qos.PriorityMedium, false) // Sent
	qosMetrics.RecordPacket(qos.PriorityMedium, true)  // Dropped
	qosMetrics.RecordPacket(qos.PriorityLow, true)     // Dropped
	qosMetrics.RecordPacket(qos.PriorityLow, true)     // Dropped

	qosMetrics.PrintDetailedStats()
}

func demo5Combined() {
	fmt.Println("Combining routing, congestion control, and QoS")

	// Setup topology with routing
	config := simulator.GenerateMeshTopology(4)
	topo := buildTopology(config)

	// Use OSPF for routing
	ospf := routing.NewOSPFProtocol(0)
	ospf.Initialize("node-0", topo)

	fmt.Println("Using OSPF routing protocol")
	ospf.PrintRoutingTable()

	// Setup congestion control for multiple connections
	fmt.Println("\n=== Congestion Control ===")
	conn1 := congestion.NewCongestionController("conn-1", "node-0", "node-1")
	conn2 := congestion.NewCongestionController("conn-2", "node-0", "node-2")

	// Simulate traffic with different congestion patterns
	fmt.Println("Connection 1: Smooth traffic")
	for i := 0; i < 5; i++ {
		conn1.OnAck(1)
	}
	fmt.Printf("  Conn1: %s\n", conn1.GetMetrics().String())

	fmt.Println("\nConnection 2: Lossy traffic")
	for i := 0; i < 3; i++ {
		conn2.OnAck(1)
	}
	conn2.OnLoss()
	conn2.OnAck(1)
	fmt.Printf("  Conn2: %s\n", conn2.GetMetrics().String())

	// Setup QoS
	fmt.Println("\n=== QoS Classification ===")
	classifier := qos.NewPacketClassifier()
	classifier.AddRule(qos.ClassificationRule{
		Name:        "Connection 1",
		SourceMatch: "node-0",
		DestMatch:   "node-1",
		Priority:    qos.PriorityHigh,
		DSCP:        46,
	})
	classifier.AddRule(qos.ClassificationRule{
		Name:        "Connection 2",
		SourceMatch: "node-0",
		DestMatch:   "node-2",
		Priority:    qos.PriorityMedium,
		DSCP:        0,
	})

	// Classify traffic
	priority1, dscp1 := classifier.Classify("node-0", "node-1")
	priority2, dscp2 := classifier.Classify("node-0", "node-2")

	fmt.Printf("Conn1 classification: Priority=%s, DSCP=%d\n", priority1, dscp1)
	fmt.Printf("Conn2 classification: Priority=%s, DSCP=%d\n", priority2, dscp2)

	// Priority queue with traffic shaping
	fmt.Println("\n=== Integrated QoS ===")
	pq := qos.NewPriorityQueue(20)
	shaper := qos.NewTrafficShaper(5000.0, 10000)

	// Send packets through classifier -> shaper -> priority queue
	fmt.Println("Processing packets:")
	for i := 0; i < 4; i++ {
		// Alternate between connections
		source := "node-0"
		dest := "node-1"
		if i%2 == 1 {
			dest = "node-2"
		}

		priority, dscp := classifier.Classify(source, dest)
		packetSize := 1000

		if shaper.CanSend(packetSize) {
			packet := qos.Packet{
				ID:       fmt.Sprintf("pkt-%d", i),
				Priority: priority,
				DSCP:     dscp,
				Size:     packetSize,
				Source:   source,
				Dest:     dest,
			}
			pq.Enqueue(packet)
			fmt.Printf("  %s: %s->%s (priority=%s) - QUEUED\n",
				packet.ID, source, dest, priority)
		} else {
			fmt.Printf("  pkt-%d: %s->%s - SHAPED\n", i, source, dest)
		}
	}

	fmt.Printf("\nFinal queue depth: %d\n", pq.GetDepth())
	depths := pq.GetDepthByPriority()
	fmt.Printf("  High=%d, Medium=%d, Low=%d\n",
		depths[qos.PriorityHigh], depths[qos.PriorityMedium], depths[qos.PriorityLow])
}

// Helper function to build topology
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
		}
	}

	return topo
}
