# 🌐 Distributed Network Simulator with Chaos Engineering

> A realistic distributed network simulator demonstrating chaos engineering principles and self-healing systems. Watch how production-grade networks handle failures, reroute traffic, and recover automatically.

```
╔════════════════════════════════════════════════════════════════╗
║    DISTRIBUTED NETWORK SIMULATOR WITH CHAOS ENGINEERING       ║
║              HYPOTHETICAL DEMONSTRATION SCENARIO               ║
╚════════════════════════════════════════════════════════════════╝
```

## 🎯 What This Demonstrates

This simulator shows **what WOULD happen** in real production systems when failures occur:

- ✅ **20-node datacenter** with redundant network paths
- ✅ **Realistic traffic patterns**: API requests, database replication, microservices
- ✅ **Chaos injection**: Simulate node crashes, network failures
- ✅ **Self-healing**: Automatic rerouting, retries, and recovery
- ✅ **Real metrics**: Track delivery rates, latency, and system health

## 🎬 Live Demo Output

### Phase 1: Normal Operations
```
📨 Running simulation with production traffic load...

═══ Metrics: Baseline State (Normal Operations) ═══

Messages Sent:       268
Messages Delivered:  267
Delivery Rate:      ███████████████████░  99%
Average Latency:    0.012s
```

### Phase 2: Node Failure
```
💥 Injecting failure: node-10 going offline (simulated crash)...

Network Status: Immediately After Failure Injection

  ● node-8 [ACTIVE]
  ● node-9 [ACTIVE]
  ○ node-10 [FAILED]  ← Critical database node offline!
  ● node-11 [ACTIVE]

📊 EXPECTED IMPACT: Without resilience mechanisms, we WOULD see:
   • Degraded delivery rates as messages are dropped
   • Increased latency for affected traffic paths
   • Service disruption for clients using the failed node
```

### Phase 3: Self-Healing Activated
```
🔄 Activating Resilience & Self-Healing Mechanisms

✓ Adaptive routing ENABLED: Network automatically reroutes
✓ Message retry logic ENABLED: Failed messages retransmitted
✓ Circuit breaker patterns ENABLED: Prevents cascade failures

═══ Metrics: With Resilience Active (Self-Healing) ═══

Messages Sent:       663
Messages Delivered:  663
Delivery Rate:      ████████████████████ 100%  ← Recovered!
Average Latency:    0.012s
```

### Phase 4: Full Recovery
```
♻️  Node-10 has completed recovery and rejoined the cluster

PHASE 4: Recovery & Stabilization
  ✓ Failed node automatically restarted
  ✓ Rejoined cluster after health checks
  ✓ System returned to full capacity
  ✓ Metrics normalized to baseline levels

Final Metrics:
  Total Messages:       1156
  Successfully Delivered: 1156 (100%)
  Average Latency:      0.012 seconds
  Active Nodes:         20/20 ✓
```

## 🏢 Real-World Applications

This simulation demonstrates patterns used by:

- **Cloud Providers**: AWS, Azure, GCP datacenter failure handling
- **Microservices**: Service mesh (Istio, Linkerd) resilience
- **Databases**: Distributed DB failover (MongoDB, Cassandra, PostgreSQL)
- **Message Queues**: Kafka, RabbitMQ cluster recovery
- **Container Orchestration**: Kubernetes pod rescheduling

## 🚀 Quick Start

### Run the Main Simulation (Hypothetical Scenario)
```bash
# Run the full interactive demonstration
go run main.go

# Shows what WOULD happen in production:
# - 20-node network with realistic traffic
# - Chaos injection and failure scenarios  
# - Self-healing and automatic recovery
# - Real-world implications and metrics
```

### Run the Technical Demo (See Real Data)
```bash
# Run the verbose demo showing actual data generation
go run demo/main-demo.go

# Proves the simulation generates REAL data:
# - Shows actual routing tables
# - Displays event processing
# - Explains physics calculations
# - Demonstrates state changes
```

## 🔍 Key Takeaways

| Without Resilience | With Resilience |
|-------------------|-----------------|
| ❌ Minutes-hours of downtime | ✅ Seconds of degradation |
| ❌ Manual intervention required | ✅ Automatic recovery |
| ❌ Cascading failures | ✅ Isolated failures |
| ❌ Lost revenue | ✅ Business continuity |

**This is why companies invest in distributed systems engineering, SRE practices, and chaos engineering.**

## 🛠️ Tech Stack

- **Backend**: Go (discrete event simulation engine)
- **Routing**: Dijkstra's shortest path with adaptive rerouting
- **Chaos Engineering**: Failure injection, latency, network partitions
- **Metrics**: Real-time delivery rates, latency tracking, node health
- **Visualization**: Terminal-based real-time monitoring

## 📊 Technical Features

- ✅ **Discrete Event Simulation**: Chronological event processing
- ✅ **Real Network Physics**: Transmission delay = size / bandwidth
- ✅ **Dijkstra Routing**: Shortest path with dynamic updates
- ✅ **Traffic Generators**: Poisson, bursty, constant patterns
- ✅ **Failure Injection**: Node crashes, link failures, latency
- ✅ **Self-Healing**: Adaptive routing, retries, circuit breakers
- ✅ **Observability**: Real-time metrics and node activity tracking

## 📦 Build & Run

```bash
# Install dependencies
go mod download

# Run main simulation (interactive, with pauses)
go run main.go

# Run technical demo (verbose, shows internals)
go run demo/main-demo.go

# Build executables
go build -o simulator main.go
go build -o demo-simulator demo/main-demo.go
```

## 🤝 Contributing

This is a demonstration project showing expected behavior in distributed systems. Contributions welcome!

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

Copyright (c) 2025 Bluesky Labs team

---

**⚠️ Note**: This is a simulation demonstrating what WOULD happen in production systems. The scenarios show expected behavior patterns based on standard distributed systems practices.
