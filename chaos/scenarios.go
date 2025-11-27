package chaos

import (
	"fmt"
	"math/rand"
)

// ChaosScenario represents a predefined chaos scenario
type ChaosScenario interface {
	Name() string
	Description() string
	Execute(injector *FailureInjector, startTime float64) error
}

// RandomNodeFailureScenario kills 1-3 random nodes
type RandomNodeFailureScenario struct {
	MinNodes int
	MaxNodes int
	Duration float64
}

func (s *RandomNodeFailureScenario) Name() string {
	return "Random Node Failure"
}

func (s *RandomNodeFailureScenario) Description() string {
	return fmt.Sprintf("Kill %d-%d random nodes for %.1fs", s.MinNodes, s.MaxNodes, s.Duration)
}

func (s *RandomNodeFailureScenario) Execute(injector *FailureInjector, startTime float64) error {
	failedNodes, err := injector.RandomNodeFailure(s.MinNodes, s.MaxNodes, s.Duration)
	if err != nil {
		return err
	}
	fmt.Printf("[%.2fs] Random Node Failure: Failed nodes: %v\n", startTime, failedNodes)
	return nil
}

// CascadingFailureScenario triggers cascading failures
type CascadingFailureScenario struct {
	InitialNode   string
	CascadeProb   float64
	MaxCascade    int
	Duration      float64
}

func (s *CascadingFailureScenario) Name() string {
	return "Cascading Failure"
}

func (s *CascadingFailureScenario) Description() string {
	return fmt.Sprintf("Cascade from %s (prob=%.2f, max=%d) for %.1fs", 
		s.InitialNode, s.CascadeProb, s.MaxCascade, s.Duration)
}

func (s *CascadingFailureScenario) Execute(injector *FailureInjector, startTime float64) error {
	failedNodes, err := injector.CascadingFailure(s.InitialNode, s.CascadeProb, s.MaxCascade, s.Duration)
	if err != nil {
		return err
	}
	fmt.Printf("[%.2fs] Cascading Failure: %d nodes failed: %v\n", startTime, len(failedNodes), failedNodes)
	return nil
}

// NetworkCongestionScenario floods links with traffic
type NetworkCongestionScenario struct {
	SourceNode        string
	DestNode          string
	MessagesPerSecond float64
	MessageSize       int
	Duration          float64
}

func (s *NetworkCongestionScenario) Name() string {
	return "Network Congestion"
}

func (s *NetworkCongestionScenario) Description() string {
	return fmt.Sprintf("Flood %s->%s with %.0f msg/s (%d bytes) for %.1fs",
		s.SourceNode, s.DestNode, s.MessagesPerSecond, s.MessageSize, s.Duration)
}

func (s *NetworkCongestionScenario) Execute(injector *FailureInjector, startTime float64) error {
	err := injector.InjectCongestion(s.SourceNode, s.DestNode, s.MessagesPerSecond, s.MessageSize, s.Duration)
	if err != nil {
		return err
	}
	fmt.Printf("[%.2fs] Network Congestion: Flooding %s->%s\n", startTime, s.SourceNode, s.DestNode)
	return nil
}

// ByzantineFailureScenario introduces Byzantine nodes
type ByzantineFailureScenario struct {
	NodeID          string
	CorruptionRate  float64
	Duration        float64
}

func (s *ByzantineFailureScenario) Name() string {
	return "Byzantine Failure"
}

func (s *ByzantineFailureScenario) Description() string {
	return fmt.Sprintf("Byzantine node %s (corruption=%.2f) for %.1fs",
		s.NodeID, s.CorruptionRate, s.Duration)
}

func (s *ByzantineFailureScenario) Execute(injector *FailureInjector, startTime float64) error {
	err := injector.InjectByzantineFailure(s.NodeID, s.CorruptionRate, s.Duration)
	if err != nil {
		return err
	}
	fmt.Printf("[%.2fs] Byzantine Failure: Node %s corrupting messages\n", startTime, s.NodeID)
	return nil
}

// SlowLinksScenario increases latency on random links
type SlowLinksScenario struct {
	NumLinks          int
	LatencyMultiplier float64
	Duration          float64
}

func (s *SlowLinksScenario) Name() string {
	return "Slow Links"
}

func (s *SlowLinksScenario) Description() string {
	return fmt.Sprintf("Slow down %d links by %.0fx for %.1fs",
		s.NumLinks, s.LatencyMultiplier, s.Duration)
}

func (s *SlowLinksScenario) Execute(injector *FailureInjector, startTime float64) error {
	// Get all links from topology
	topo := injector.sim.GetTopology()
	allLinks := topo.Config.Links

	if len(allLinks) == 0 {
		return fmt.Errorf("no links available")
	}

	// Select random links
	numToSlow := s.NumLinks
	if numToSlow > len(allLinks) {
		numToSlow = len(allLinks)
	}

	indices := rand.Perm(len(allLinks))
	slowedLinks := make([]string, 0)

	for i := 0; i < numToSlow; i++ {
		link := allLinks[indices[i]]
		err := injector.InjectSlowLink(link.Source, link.Dest, s.LatencyMultiplier, s.Duration)
		if err != nil {
			continue
		}
		slowedLinks = append(slowedLinks, fmt.Sprintf("%s->%s", link.Source, link.Dest))
	}

	fmt.Printf("[%.2fs] Slow Links: Slowed %d links: %v\n", startTime, len(slowedLinks), slowedLinks)
	return nil
}

// NetworkPartitionScenario splits the network
type NetworkPartitionScenario struct {
	Partitions [][]string
	Duration   float64
}

func (s *NetworkPartitionScenario) Name() string {
	return "Network Partition"
}

func (s *NetworkPartitionScenario) Description() string {
	return fmt.Sprintf("Split network into %d partitions for %.1fs", len(s.Partitions), s.Duration)
}

func (s *NetworkPartitionScenario) Execute(injector *FailureInjector, startTime float64) error {
	err := injector.InjectNetworkPartition(s.Partitions, s.Duration)
	if err != nil {
		return err
	}
	fmt.Printf("[%.2fs] Network Partition: Split into %d groups\n", startTime, len(s.Partitions))
	for i, partition := range s.Partitions {
		fmt.Printf("  Partition %d: %v\n", i+1, partition)
	}
	return nil
}

// MultipleFailuresScenario combines multiple failure types
type MultipleFailuresScenario struct {
	NodeFailures []string
	LinkFailures []struct{ Source, Dest string }
	Duration     float64
}

func (s *MultipleFailuresScenario) Name() string {
	return "Multiple Simultaneous Failures"
}

func (s *MultipleFailuresScenario) Description() string {
	return fmt.Sprintf("Fail %d nodes and %d links for %.1fs",
		len(s.NodeFailures), len(s.LinkFailures), s.Duration)
}

func (s *MultipleFailuresScenario) Execute(injector *FailureInjector, startTime float64) error {
	// Fail nodes
	for _, nodeID := range s.NodeFailures {
		err := injector.InjectNodeFailure(nodeID, s.Duration)
		if err != nil {
			fmt.Printf("Warning: Failed to kill node %s: %v\n", nodeID, err)
		}
	}

	// Fail links
	for _, link := range s.LinkFailures {
		err := injector.InjectLinkFailure(link.Source, link.Dest, s.Duration)
		if err != nil {
			fmt.Printf("Warning: Failed to sever link %s->%s: %v\n", link.Source, link.Dest, err)
		}
	}

	fmt.Printf("[%.2fs] Multiple Failures: %d nodes, %d links failed\n",
		startTime, len(s.NodeFailures), len(s.LinkFailures))
	return nil
}

// ScenarioLibrary provides predefined scenarios
type ScenarioLibrary struct {
	scenarios map[string]ChaosScenario
}

// NewScenarioLibrary creates a library with predefined scenarios
func NewScenarioLibrary() *ScenarioLibrary {
	lib := &ScenarioLibrary{
		scenarios: make(map[string]ChaosScenario),
	}

	// Add default scenarios
	lib.scenarios["random-node-failure"] = &RandomNodeFailureScenario{
		MinNodes: 1,
		MaxNodes: 3,
		Duration: 5.0,
	}

	lib.scenarios["network-congestion"] = &NetworkCongestionScenario{
		SourceNode:        "router-1",
		DestNode:          "router-2",
		MessagesPerSecond: 100.0,
		MessageSize:       2000,
		Duration:          3.0,
	}

	lib.scenarios["slow-links"] = &SlowLinksScenario{
		NumLinks:          2,
		LatencyMultiplier: 10.0,
		Duration:          5.0,
	}

	return lib
}

// AddScenario adds a scenario to the library
func (sl *ScenarioLibrary) AddScenario(name string, scenario ChaosScenario) {
	sl.scenarios[name] = scenario
}

// GetScenario retrieves a scenario by name
func (sl *ScenarioLibrary) GetScenario(name string) ChaosScenario {
	return sl.scenarios[name]
}

// ListScenarios returns all scenario names
func (sl *ScenarioLibrary) ListScenarios() []string {
	names := make([]string, 0, len(sl.scenarios))
	for name := range sl.scenarios {
		names = append(names, name)
	}
	return names
}