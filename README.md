# 🌐 Distributed Network Simulator with Chaos Engineering

> A powerful telecom network topology simulator with built-in chaos engineering capabilities to test system resilience and fault tolerance.

## 🎯 Goal

Simulate complex telecom network topologies with chaos engineering to test and validate network resilience under adverse conditions.

## 🛠️ Tech Stack

- **Backend**: Go (network simulator core)
- **Communication**: gRPC (inter-node communication)
- **Visualization**: React + D3.js (interactive network graphs)
- **Infrastructure**: 100% local deployment with Docker for node isolation

## 🚀 Features

- Real-time network topology simulation
- Chaos engineering scenarios (node failures, network partitions, latency injection)
- Interactive network visualization
- Container-based node isolation
- Distributed system fault tolerance testing

## 📦 Quick Start

```bash
# Build the project
make build

# Run the simulator
make run

# Run with Docker
docker-compose up
```

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

Copyright (c) 2025 Bluesky Labs team
