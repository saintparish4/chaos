package chaos

import (
	"testing"

	"github.com/saintparish4/chaos/simulator"
)

func TestFailureInjectorNodeFailure(t *testing.T) {
	// Create simple topology
	config := simulator.GenerateRingTopology(5)
	topo := buildTestTopology(config)
	sim := simulator.NewEventDrivenSimulatorWithSpeed(topo, 1.0)
	injector := NewFailureInjector(sim)

	// Inject node failure
	err := injector.InjectNodeFailure("node-0", 5.0)
	if err != nil {
		t.Errorf("Failed to inject node failure: %v", err)
	}

	// Verify node is failed
	node, _ := sim.GetNode("node-0")
	if node.GetState() != "FAILED" {
		t.Errorf("Expected node state FAILED, got %s", node.GetState())
	}
}

func TestFailureInjectorLinkFailure(t *testing.T) {
	config := simulator.GenerateRingTopology(4)
	topo := buildTestTopology(config)
	sim := simulator.NewEventDrivenSimulatorWithSpeed(topo, 1.0)
	injector := NewFailureInjector(sim)

	// Inject link failure
	err := injector.InjectLinkFailure("node-0", "node-1", 3.0)
	if err != nil {
		t.Errorf("Failed to inject link failure: %v", err)
	}

	// Verify link is down
	link := sim.GetLink("node-0", "node-1")
	if link == nil {
		t.Fatal("Link not found")
	}
	if link.IsActive() {
		t.Error("Expected link to be inactive")
	}
}

func TestNetworkPartition(t *testing.T) {
	config := simulator.GenerateMeshTopology(4)
	topo := buildTestTopology(config)
	sim := simulator.NewEventDrivenSimulatorWithSpeed(topo, 1.0)
	injector := NewFailureInjector(sim)

	// Create partition
	partitions := [][]string{
		{"node-0", "node-1"},
		{"node-2", "node-3"},
	}

	err := injector.InjectNetworkPartition(partitions, 5.0)
	if err != nil {
		t.Errorf("Failed to inject partition: %v", err)
	}

	// Verify links between partitions are down
	link := sim.GetLink("node-0", "node-2")
	if link == nil {
		t.Skip("Link not found (may not exist in this topology)")
	} else if link.IsActive() {
		t.Error("Expected link between partitions to be inactive")
	}
}

func TestRandomNodeFailure(t *testing.T) {
	config := simulator.GenerateRingTopology(10)
	topo := buildTestTopology(config)
	sim := simulator.NewEventDrivenSimulatorWithSpeed(topo, 1.0)
	injector := NewFailureInjector(sim)

	// Inject random failures
	failedNodes, err := injector.RandomNodeFailure(2, 4, 3.0)
	if err != nil {
		t.Errorf("Failed to inject random failures: %v", err)
	}

	if len(failedNodes) < 2 || len(failedNodes) > 4 {
		t.Errorf("Expected 2-4 failed nodes, got %d", len(failedNodes))
	}

	// Verify nodes are actually failed
	for _, nodeID := range failedNodes {
		node, _ := sim.GetNode(nodeID)
		if node.GetState() != "FAILED" {
			t.Errorf("Node %s should be FAILED", nodeID)
		}
	}
}

func TestCascadingFailure(t *testing.T) {
	config := simulator.GenerateMeshTopology(6)
	topo := buildTestTopology(config)
	sim := simulator.NewEventDrivenSimulatorWithSpeed(topo, 1.0)
	injector := NewFailureInjector(sim)

	// Trigger cascading failure
	failedNodes, err := injector.CascadingFailure("node-0", 0.5, 4, 5.0)
	if err != nil {
		t.Errorf("Failed to inject cascading failure: %v", err)
	}

	if len(failedNodes) < 1 {
		t.Error("Expected at least 1 failed node")
	}

	if len(failedNodes) > 4 {
		t.Errorf("Expected max 4 failed nodes, got %d", len(failedNodes))
	}

	// First node should be the initial failure
	if failedNodes[0] != "node-0" {
		t.Errorf("Expected first failed node to be node-0, got %s", failedNodes[0])
	}
}

func TestChaosScenarios(t *testing.T) {
	config := simulator.GenerateRingTopology(7)
	topo := buildTestTopology(config)
	sim := simulator.NewEventDrivenSimulatorWithSpeed(topo, 1.0)
	injector := NewFailureInjector(sim)

	// Test RandomNodeFailureScenario
	scenario1 := &RandomNodeFailureScenario{
		MinNodes: 1,
		MaxNodes: 2,
		Duration: 3.0,
	}

	err := scenario1.Execute(injector, 0.0)
	if err != nil {
		t.Errorf("Failed to execute random node failure scenario: %v", err)
	}

	if scenario1.Name() == "" {
		t.Error("Scenario name should not be empty")
	}

	if scenario1.Description() == "" {
		t.Error("Scenario description should not be empty")
	}
}

func TestChaosScheduler(t *testing.T) {
	config := simulator.GenerateRingTopology(5)
	topo := buildTestTopology(config)
	sim := simulator.NewEventDrivenSimulatorWithSpeed(topo, 1.0)
	injector := NewFailureInjector(sim)
	scheduler := NewChaosScheduler(injector)

	// Create experiment programmatically
	experiment := &ChaosExperiment{
		Name:        "Test Experiment",
		Description: "Testing chaos scheduler",
		Duration:    10.0,
		Events: []ChaosEventConfig{
			{
				Time:     2.0,
				Type:     "node_failure",
				Target:   "node-0",
				Duration: 3.0,
			},
			{
				Time:     5.0,
				Type:     "link_failure",
				Source:   "node-1",
				Dest:     "node-2",
				Duration: 2.0,
			},
		},
	}

	scheduler.SetExperiment(experiment)

	// Schedule should not error
	err := scheduler.ScheduleExperiment()
	if err != nil {
		t.Errorf("Failed to schedule experiment: %v", err)
	}
}

func TestResilienceManager(t *testing.T) {
	config := simulator.GenerateRingTopology(6)
	topo := buildTestTopology(config)
	sim := simulator.NewEventDrivenSimulatorWithSpeed(topo, 1.0)

	rm := NewResilienceManager(sim)

	// Test enabling mechanisms
	rm.EnableAdaptiveRouting()
	rm.EnableRetries(3, 0.1)
	rm.EnableHealthChecks()

	// Should not crash
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker("test-node", 3, 5.0)

	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected initial state CLOSED, got %s", cb.GetState())
	}

	// Should allow traffic initially
	if !cb.ShouldAllow() {
		t.Error("Circuit breaker should allow traffic when closed")
	}

	// Record failures
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Should open after threshold
	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected state OPEN after threshold, got %s", cb.GetState())
	}

	// Should block traffic when open
	if cb.ShouldAllow() {
		t.Error("Circuit breaker should block traffic when open")
	}

	// Record success should close if half-open
	cb.State = CircuitHalfOpen
	cb.RecordSuccess()
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected state CLOSED after success, got %s", cb.GetState())
	}
}

func TestRetryManager(t *testing.T) {
	config := simulator.GenerateRingTopology(3)
	topo := buildTestTopology(config)
	sim := simulator.NewEventDrivenSimulatorWithSpeed(topo, 1.0)

	rm := NewRetryManager(sim)
	rm.Enable(3, 0.1)

	msgID := "test-message-1"

	// Should retry initially
	if !rm.ShouldRetry(msgID) {
		t.Error("Should allow retry for new message")
	}

	// Record retries
	rm.RecordRetry(msgID)
	rm.RecordRetry(msgID)
	rm.RecordRetry(msgID)

	// Should not retry after max attempts
	if rm.ShouldRetry(msgID) {
		t.Error("Should not retry after max attempts")
	}

	// Test backoff duration
	backoff := rm.GetBackoffDuration(msgID)
	if backoff <= 0 {
		t.Error("Backoff duration should be positive")
	}
}

func TestHealthChecker(t *testing.T) {
	config := simulator.GenerateRingTopology(4)
	topo := buildTestTopology(config)
	sim := simulator.NewEventDrivenSimulatorWithSpeed(topo, 1.0)

	hc := NewHealthChecker(sim, 1.0)
	hc.Enable()

	// All nodes should be healthy initially
	if !hc.IsHealthy("node-0") {
		t.Error("Node should be healthy initially")
	}

	// Perform health checks (no failures yet)
	hc.PerformHealthChecks()

	unhealthy := hc.GetUnhealthyNodes()
	if len(unhealthy) != 0 {
		t.Errorf("Expected no unhealthy nodes, got %d", len(unhealthy))
	}
}

// Helper function to build test topology
func buildTestTopology(config *simulator.TopologyConfig) *simulator.Topology {
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
