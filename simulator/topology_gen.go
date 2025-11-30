package simulator

import (
	"fmt"
)

// GenerateRingTopology creates a ring network topology
func GenerateRingTopology(numNodes int) *TopologyConfig {
	config := &TopologyConfig{
		Nodes: make([]NodeConfig, numNodes),
		Links: make([]LinkConfig, numNodes),
	}

	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		config.Nodes[i] = NodeConfig{
			ID:       nodeID,
			Type:     NodeTypeRouter,
			Capacity: 1000,
		}

		nextNode := fmt.Sprintf("node-%d", (i+1)%numNodes)
		config.Links[i] = LinkConfig{
			Source:    nodeID,
			Dest:      nextNode,
			Bandwidth: 1000000, // 1 Mbps
			Latency:   10,      // 10ms
		}
	}
	return config
}

// GenerateMeshTopology creates a fully connected mesh network
func GenerateMeshTopology(numNodes int) *TopologyConfig {
	config := &TopologyConfig{
		Nodes: make([]NodeConfig, numNodes),
		Links: make([]LinkConfig, 0),
	}

	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		config.Nodes[i] = NodeConfig{
			ID:       nodeID,
			Type:     NodeTypeRouter,
			Capacity: 1000,
		}
	}

	// Create links between all pairs
	for i := 0; i < numNodes; i++ {
		for j := 0; j < numNodes; j++ {
			if i != j {
				config.Links = append(config.Links, LinkConfig{
					Source:    fmt.Sprintf("node-%d", i),
					Dest:      fmt.Sprintf("node-%d", j),
					Bandwidth: 1000000,
					Latency:   10,
				})
			}
		}
	}

	return config
}

// GenerateTreeTopology creates a binary tree network topology
func GenerateTreeTopology(depth int) *TopologyConfig {
	numNodes := (1 << depth) - 1 // 2^depth - 1 nodes
	config := &TopologyConfig{
		Nodes: make([]NodeConfig, numNodes),
		Links: make([]LinkConfig, 0),
	}

	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		config.Nodes[i] = NodeConfig{
			ID:       nodeID,
			Type:     NodeTypeRouter,
			Capacity: 1000,
		}

		// Add links to children
		leftChild := 2*i + 1
		rightChild := 2*i + 2

		if leftChild < numNodes {
			config.Links = append(config.Links, LinkConfig{
				Source:    nodeID,
				Dest:      fmt.Sprintf("node-%d", leftChild),
				Bandwidth: 1000000,
				Latency:   10,
			})
		}

		if rightChild < numNodes {
			config.Links = append(config.Links, LinkConfig{
				Source:    nodeID,
				Dest:      fmt.Sprintf("node-%d", rightChild),
				Bandwidth: 1000000,
				Latency:   10,
			})
		}
	}

	return config
}

// SaveTopology saves a topology configuration to a YAML file
func SaveTopology(config *TopologyConfig, filename string) error {
	// This would use yaml.Marshal and os.WriteFile
	// Implementation left as exercise or can be added
	return nil
}
