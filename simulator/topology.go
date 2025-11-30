package simulator

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// NodeType represents the type of network node
type NodeType string

const (
	NodeTypeRouter NodeType = "router"
	NodeTypeSwitch NodeType = "switch"
	NodeTypeTower  NodeType = "tower"
)

// NodeConfig defines a node in the topology
type NodeConfig struct {
	ID       string   `yaml:"id"`
	Type     NodeType `yaml:"type"`
	Capacity int      `yaml:"capacity"` // messages per second
}

// LinkConfig defines a connection between two nodes
type LinkConfig struct {
	Source    string `yaml:"source"`
	Dest      string `yaml:"dest"`
	Bandwidth int    `yaml:"bandwidth"` // bytes per second
	Latency   int    `yaml:"latency"`   // milliseconds
}

// TopologyConfig is the root configuration struct
type TopologyConfig struct {
	Nodes []NodeConfig `yaml:"nodes"`
	Links []LinkConfig `yaml:"links"`
}

// Topology represnets the network graph
type Topology struct {
	Config  TopologyConfig
	AdjList map[string][]string         // adjacency list
	Links   map[string]map[string]*Link // source -> dest -> link
	Nodes   map[string]*NodeConfig
}

// Link represents a network connection
type Link struct {
	Source    string
	Dest      string
	Bandwidth int
	Latency   int
}

// LoadTopology loads and validates a topology from a YAML file
func LoadTopology(filename string) (*Topology, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read topology file: %w", err)
	}

	var config TopologyConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	topo := &Topology{
		Config:  config,
		AdjList: make(map[string][]string),
		Links:   make(map[string]map[string]*Link),
		Nodes:   make(map[string]*NodeConfig),
	}

	// Build node map
	for i := range config.Nodes {
		node := &config.Nodes[i]
		topo.Nodes[node.ID] = node
	}

	// Build adjacency list and link map
	for _, linkCfg := range config.Links {
		// Add to adjacency list
		topo.AdjList[linkCfg.Source] = append(topo.AdjList[linkCfg.Source], linkCfg.Dest)

		// Add to link map
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

	// Validate topology
	if err := topo.Validate(); err != nil {
		return nil, err
	}
	return topo, nil
}

// Validate checks for orphaned nodes and invalid links
func (t *Topology) Validate() error {
	// Check all links reference valid nodes
	for _, link := range t.Config.Links {
		if _, exists := t.Nodes[link.Source]; !exists {
			return fmt.Errorf("link references unknown source node: %s", link.Source)
		}
		if _, exists := t.Nodes[link.Dest]; !exists {
			return fmt.Errorf("link references unknown dest node: %s", link.Dest)
		}
	}

	// Check for orphaned nodes (nodes with no connections)
	connectedNodes := make(map[string]bool)
	for _, link := range t.Config.Links {
		connectedNodes[link.Source] = true
		connectedNodes[link.Dest] = true
	}

	for nodeID := range t.Nodes {
		if !connectedNodes[nodeID] {
			return fmt.Errorf("orphaned node detected: %s", nodeID)
		}
	}

	return nil
}

// GetNeighbors returns all neighbors of a node
func (t *Topology) GetNeighbors(nodeID string) []string {
	return t.AdjList[nodeID]
}

// GetLink returns the link between two nodes
func (t *Topology) GetLink(source, dest string) *Link {
	if links, ok := t.Links[source]; ok {
		return links[dest]
	}
	return nil
}
