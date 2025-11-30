package routing

import (
	"fmt"
	"sync"
	"time"

	"github.com/saintparish4/chaos/simulator"
)

// RoutingProtocol defines the interface for routing protocols
type RoutingProtocol interface {
	GetName() string
	Initialize(nodeID string, topology *simulator.Topology)
	ProcessUpdate(update *RoutingUpdate)
	GetNextHop(dest string) (string, error)
	GetRoutingTable() map[string]RouteEntry
	SendUpdates(neighbors []string)
	GetMetrics() *ProtocolMetrics
}

// RouteEntry represents an entry in a routing table
type RouteEntry struct {
	Destination string
	NextHop     string
	Metric      int
	Path        []string
	Timestamp   time.Time
	Valid       bool
}

// RoutingUpdate represents a routing protocol message
type RoutingUpdate struct {
	SourceNode string
	Protocol   string
	Type       UpdateType
	Routes     []RouteEntry
	Timestamp  time.Time
}

// UpdateType represents the type of routing update
type UpdateType string

const (
	UpdateTypeFullTable   UpdateType = "FULL_TABLE"
	UpdateTypeIncremental UpdateType = "INCREMENTAL"
	UpdateTypeWithdrawal  UpdateType = "WITHDRAWAL"
	UpdateTypeLinkState   UpdateType = "LINK_STATE"
)

// ProtocolMetrics tracks protocol performance
type ProtocolMetrics struct {
	UpdatesSent       int
	UpdatesReceived   int
	RoutesLearned     int
	ConvergenceTime   float64
	RoutingTableSize  int
	AveragePathLength float64
	ProtocolOverhead  int
	LastUpdateTime    time.Time
	mu                sync.RWMutex
}

func (pm *ProtocolMetrics) RecordUpdateSent() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.UpdatesSent++
}

func (pm *ProtocolMetrics) RecordUpdateReceived() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.UpdatesReceived++
	pm.LastUpdateTime = time.Now()
}

func (pm *ProtocolMetrics) UpdateRoutingTableSize(size int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.RoutingTableSize = size
}

func (pm *ProtocolMetrics) GetSummary() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return fmt.Sprintf("Updates: sent=%d recv=%d | Routes: %d | Convergence: %.3fs | Overhead: %d bytes",
		pm.UpdatesSent, pm.UpdatesReceived, pm.RoutesLearned, pm.ConvergenceTime, pm.ProtocolOverhead)
}

// ProtocolManager manages multiple routing protocols
type ProtocolManager struct {
	protocols map[string]RoutingProtocol
	topology  *simulator.Topology
	mu        sync.RWMutex
}

func NewProtocolManager(topology *simulator.Topology) *ProtocolManager {
	return &ProtocolManager{
		protocols: make(map[string]RoutingProtocol),
		topology:  topology,
	}
}

func (pm *ProtocolManager) RegisterProtocol(name string, protocol RoutingProtocol) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.protocols[name] = protocol
}

func (pm *ProtocolManager) GetProtocol(name string) RoutingProtocol {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.protocols[name]
}

func (pm *ProtocolManager) InitializeAllProtocols(nodeID string) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, protocol := range pm.protocols {
		protocol.Initialize(nodeID, pm.topology)
	}
}

func (pm *ProtocolManager) CompareProtocols() map[string]*ProtocolMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	results := make(map[string]*ProtocolMetrics)
	for name, protocol := range pm.protocols {
		results[name] = protocol.GetMetrics()
	}
	return results
}

// RoutingProtocolBase provides common functionality for routing protocols
type RoutingProtocolBase struct {
	Name         string
	NodeID       string
	RoutingTable map[string]RouteEntry
	Neighbors    []string
	Topology     *simulator.Topology
	Metrics      *ProtocolMetrics
	mu           sync.RWMutex
}

func NewRoutingProtocolBase(name string) *RoutingProtocolBase {
	return &RoutingProtocolBase{
		Name:         name,
		RoutingTable: make(map[string]RouteEntry),
		Metrics:      &ProtocolMetrics{},
	}
}

func (rpb *RoutingProtocolBase) GetName() string {
	return rpb.Name
}

func (rpb *RoutingProtocolBase) GetRoutingTable() map[string]RouteEntry {
	rpb.mu.RLock()
	defer rpb.mu.RUnlock()

	result := make(map[string]RouteEntry)
	for k, v := range rpb.RoutingTable {
		result[k] = v
	}
	return result
}

func (rpb *RoutingProtocolBase) GetNextHop(dest string) (string, error) {
	rpb.mu.RLock()
	defer rpb.mu.RUnlock()

	if entry, exists := rpb.RoutingTable[dest]; exists && entry.Valid {
		return entry.NextHop, nil
	}
	return "", fmt.Errorf("no route to %s", dest)
}

func (rpb *RoutingProtocolBase) GetMetrics() *ProtocolMetrics {
	return rpb.Metrics
}

func (rpb *RoutingProtocolBase) UpdateRoute(dest, nextHop string, metric int, path []string) {
	rpb.mu.Lock()
	defer rpb.mu.Unlock()

	rpb.RoutingTable[dest] = RouteEntry{
		Destination: dest,
		NextHop:     nextHop,
		Metric:      metric,
		Path:        path,
		Timestamp:   time.Now(),
		Valid:       true,
	}

	rpb.Metrics.UpdateRoutingTableSize(len(rpb.RoutingTable))
}

func (rpb *RoutingProtocolBase) InvalidateRoute(dest string) {
	rpb.mu.Lock()
	defer rpb.mu.Unlock()

	if entry, exists := rpb.RoutingTable[dest]; exists {
		entry.Valid = false
		rpb.RoutingTable[dest] = entry
	}
}

func (rpb *RoutingProtocolBase) GetNeighbors() []string {
	rpb.mu.RLock()
	defer rpb.mu.RUnlock()

	neighbors := make([]string, len(rpb.Neighbors))
	copy(neighbors, rpb.Neighbors)
	return neighbors
}

func (rpb *RoutingProtocolBase) SetNeighbors(topology *simulator.Topology) {
	rpb.mu.Lock()
	defer rpb.mu.Unlock()

	if neighbors, exists := topology.AdjList[rpb.NodeID]; exists {
		rpb.Neighbors = make([]string, len(neighbors))
		copy(rpb.Neighbors, neighbors)
	}
}
