package chaos

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/saintparish4/chaos/nodes"
	"github.com/saintparish4/chaos/simulator"
)

// ResilienceManager handles resilience mechanisms
type ResilienceManager struct {
	sim               *simulator.EventDrivenSimulator
	adaptiveRouting   *AdaptiveRoutingProtocol
	retryManager      *RetryManager
	circuitBreakers   map[string]*CircuitBreaker
	healthChecker     *HealthChecker
	mu                sync.RWMutex
}

// NewResilienceManager creates a new resilience manager
func NewResilienceManager(sim *simulator.EventDrivenSimulator) *ResilienceManager {
	rm := &ResilienceManager{
		sim:             sim,
		circuitBreakers: make(map[string]*CircuitBreaker),
	}

	rm.adaptiveRouting = NewAdaptiveRoutingProtocol(sim)
	rm.retryManager = NewRetryManager(sim)
	rm.healthChecker = NewHealthChecker(sim, 1.0) // Health check every 1 second

	return rm
}

// EnableAdaptiveRouting enables automatic route recalculation on failures
func (rm *ResilienceManager) EnableAdaptiveRouting() {
	rm.adaptiveRouting.Enable()
}

// EnableRetries enables message retry with exponential backoff
func (rm *ResilienceManager) EnableRetries(maxRetries int, initialBackoff float64) {
	rm.retryManager.Enable(maxRetries, initialBackoff)
}

// EnableHealthChecks starts periodic health checks
func (rm *ResilienceManager) EnableHealthChecks() {
	rm.healthChecker.Enable()
}

// GetCircuitBreaker gets or creates a circuit breaker for a node
func (rm *ResilienceManager) GetCircuitBreaker(nodeID string) *CircuitBreaker {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if cb, exists := rm.circuitBreakers[nodeID]; exists {
		return cb
	}

	cb := NewCircuitBreaker(nodeID, 5, 10.0) // 5 failures trigger open, 10s timeout
	rm.circuitBreakers[nodeID] = cb
	return cb
}

// AdaptiveRoutingProtocol recalculates routes when failures are detected
type AdaptiveRoutingProtocol struct {
	sim                 *simulator.EventDrivenSimulator
	enabled             bool
	lastUpdate          float64
	updateInterval      float64
	failureDetections   map[string]int
	mu                  sync.RWMutex
}

// NewAdaptiveRoutingProtocol creates a new adaptive routing protocol
func NewAdaptiveRoutingProtocol(sim *simulator.EventDrivenSimulator) *AdaptiveRoutingProtocol {
	return &AdaptiveRoutingProtocol{
		sim:               sim,
		enabled:           false,
		updateInterval:    0.5, // Update routes every 0.5 seconds
		failureDetections: make(map[string]int),
	}
}

// Enable enables adaptive routing
func (arp *AdaptiveRoutingProtocol) Enable() {
	arp.mu.Lock()
	defer arp.mu.Unlock()
	arp.enabled = true
	fmt.Println("✓ Adaptive routing enabled")
}

// DetectFailure records a failure detection and triggers route update if needed
func (arp *AdaptiveRoutingProtocol) DetectFailure(nodeOrLink string) {
	if !arp.enabled {
		return
	}

	arp.mu.Lock()
	defer arp.mu.Unlock()

	arp.failureDetections[nodeOrLink]++

	// Trigger route recalculation
	currentTime := arp.sim.GetClock().CurrentTime()
	if currentTime-arp.lastUpdate >= arp.updateInterval {
		arp.recalculateRoutes()
		arp.lastUpdate = currentTime
	}
}

// recalculateRoutes updates routing tables based on current topology
func (arp *AdaptiveRoutingProtocol) recalculateRoutes() {
	// Build active topology (excluding failed nodes and links)
	activeTopo := arp.buildActiveTopology()

	// Recompute routing tables
	allTables := simulator.ComputeAllRoutingTables(activeTopo)

	// Update all nodes
	for nodeID, table := range allTables {
		node, err := arp.sim.GetNode(nodeID)
		if err != nil {
			continue
		}
		if node.GetState() != nodes.StateFailed {
			node.SetRoutingTable(table)
		}
	}

	fmt.Printf("[%.2fs] Adaptive Routing: Routes recalculated (%d failures detected)\n",
		arp.sim.GetClock().CurrentTime(), len(arp.failureDetections))
}

// buildActiveTopology creates a topology excluding failed components
func (arp *AdaptiveRoutingProtocol) buildActiveTopology() *simulator.Topology {
	originalTopo := arp.sim.GetTopology()

	// Filter out failed nodes
	activeNodes := make([]simulator.NodeConfig, 0)
	for _, nodeCfg := range originalTopo.Config.Nodes {
		node, err := arp.sim.GetNode(nodeCfg.ID)
		if err != nil || node.GetState() == nodes.StateFailed {
			continue
		}
		activeNodes = append(activeNodes, nodeCfg)
	}

	// Filter out failed links
	activeLinks := make([]simulator.LinkConfig, 0)
	for _, linkCfg := range originalTopo.Config.Links {
		link := arp.sim.GetLink(linkCfg.Source, linkCfg.Dest)
		if link == nil || !link.IsActive() {
			continue
		}
		activeLinks = append(activeLinks, linkCfg)
	}

	// Build new topology
	newConfig := &simulator.TopologyConfig{
		Nodes: activeNodes,
		Links: activeLinks,
	}

	newTopo := &simulator.Topology{
		Config:  *newConfig,
		AdjList: make(map[string][]string),
		Links:   make(map[string]map[string]*simulator.Link),
		Nodes:   make(map[string]*simulator.NodeConfig),
	}

	// Build structures
	for i := range newConfig.Nodes {
		node := &newConfig.Nodes[i]
		newTopo.Nodes[node.ID] = node
	}

	for _, linkCfg := range newConfig.Links {
		newTopo.AdjList[linkCfg.Source] = append(newTopo.AdjList[linkCfg.Source], linkCfg.Dest)
		if newTopo.Links[linkCfg.Source] == nil {
			newTopo.Links[linkCfg.Source] = make(map[string]*simulator.Link)
		}
		newTopo.Links[linkCfg.Source][linkCfg.Dest] = &simulator.Link{
			Source:    linkCfg.Source,
			Dest:      linkCfg.Dest,
			Bandwidth: linkCfg.Bandwidth,
			Latency:   linkCfg.Latency,
		}
	}

	return newTopo
}

// RetryManager handles message retries with exponential backoff
type RetryManager struct {
	sim            *simulator.EventDrivenSimulator
	enabled        bool
	maxRetries     int
	initialBackoff float64
	retries        map[string]int
	mu             sync.RWMutex
}

// NewRetryManager creates a new retry manager
func NewRetryManager(sim *simulator.EventDrivenSimulator) *RetryManager {
	return &RetryManager{
		sim:            sim,
		enabled:        false,
		maxRetries:     3,
		initialBackoff: 0.1,
		retries:        make(map[string]int),
	}
}

// Enable enables retry mechanism
func (rm *RetryManager) Enable(maxRetries int, initialBackoff float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.enabled = true
	rm.maxRetries = maxRetries
	rm.initialBackoff = initialBackoff
	fmt.Printf("✓ Message retry enabled (max: %d, backoff: %.3fs)\n", maxRetries, initialBackoff)
}

// ShouldRetry checks if a message should be retried
func (rm *RetryManager) ShouldRetry(messageID string) bool {
	if !rm.enabled {
		return false
	}

	rm.mu.RLock()
	defer rm.mu.RUnlock()

	retryCount := rm.retries[messageID]
	return retryCount < rm.maxRetries
}

// GetBackoffDuration calculates exponential backoff duration
func (rm *RetryManager) GetBackoffDuration(messageID string) float64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	retryCount := rm.retries[messageID]
	return rm.initialBackoff * math.Pow(2, float64(retryCount))
}

// RecordRetry records a retry attempt
func (rm *RetryManager) RecordRetry(messageID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.retries[messageID]++
}

// CircuitBreaker prevents routing through failed nodes
type CircuitBreaker struct {
	NodeID          string
	State           CircuitBreakerState
	FailureCount    int
	FailureThreshold int
	Timeout         float64
	LastFailureTime time.Time
	mu              sync.RWMutex
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitClosed    CircuitBreakerState = "CLOSED"     // Normal operation
	CircuitOpen      CircuitBreakerState = "OPEN"       // Blocking traffic
	CircuitHalfOpen  CircuitBreakerState = "HALF_OPEN"  // Testing if recovered
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(nodeID string, threshold int, timeout float64) *CircuitBreaker {
	return &CircuitBreaker{
		NodeID:           nodeID,
		State:            CircuitClosed,
		FailureThreshold: threshold,
		Timeout:          timeout,
	}
}

// RecordFailure records a failure and updates state
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.FailureCount++
	cb.LastFailureTime = time.Now()

	if cb.FailureCount >= cb.FailureThreshold && cb.State == CircuitClosed {
		cb.State = CircuitOpen
		fmt.Printf("[Circuit Breaker] %s opened (failures: %d)\n", cb.NodeID, cb.FailureCount)
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.State == CircuitHalfOpen {
		cb.State = CircuitClosed
		cb.FailureCount = 0
		fmt.Printf("[Circuit Breaker] %s closed (recovered)\n", cb.NodeID)
	}
}

// ShouldAllow checks if traffic should be allowed
func (cb *CircuitBreaker) ShouldAllow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.State == CircuitClosed {
		return true
	}

	if cb.State == CircuitOpen {
		// Check if timeout has elapsed
		if time.Since(cb.LastFailureTime).Seconds() >= cb.Timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.State = CircuitHalfOpen
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	}

	// Half-open: allow limited traffic to test
	return true
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.State
}

// HealthChecker performs periodic health checks between nodes
type HealthChecker struct {
	sim              *simulator.EventDrivenSimulator
	enabled          bool
	checkInterval    float64
	lastCheck        float64
	unhealthyNodes   map[string]bool
	mu               sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(sim *simulator.EventDrivenSimulator, interval float64) *HealthChecker {
	return &HealthChecker{
		sim:            sim,
		checkInterval:  interval,
		unhealthyNodes: make(map[string]bool),
	}
}

// Enable enables health checking
func (hc *HealthChecker) Enable() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.enabled = true
	fmt.Printf("✓ Health checks enabled (interval: %.1fs)\n", hc.checkInterval)
}

// PerformHealthChecks checks all nodes
func (hc *HealthChecker) PerformHealthChecks() {
	if !hc.enabled {
		return
	}

	currentTime := hc.sim.GetClock().CurrentTime()
	if currentTime-hc.lastCheck < hc.checkInterval {
		return
	}

	hc.mu.Lock()
	defer hc.mu.Unlock()

	allNodes := hc.sim.GetAllNodes()
	unhealthy := 0

	for _, node := range allNodes {
		if node.GetState() == nodes.StateFailed {
			hc.unhealthyNodes[node.ID] = true
			unhealthy++
		} else {
			delete(hc.unhealthyNodes, node.ID)
		}
	}

	if unhealthy > 0 {
		fmt.Printf("[%.2fs] Health Check: %d unhealthy nodes detected\n", currentTime, unhealthy)
	}

	hc.lastCheck = currentTime
}

// IsHealthy checks if a node is healthy
func (hc *HealthChecker) IsHealthy(nodeID string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return !hc.unhealthyNodes[nodeID]
}

// GetUnhealthyNodes returns list of unhealthy nodes
func (hc *HealthChecker) GetUnhealthyNodes() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	nodes := make([]string, 0, len(hc.unhealthyNodes))
	for nodeID := range hc.unhealthyNodes {
		nodes = append(nodes, nodeID)
	}
	return nodes
}