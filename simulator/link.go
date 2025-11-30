package simulator

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/saintparish4/chaos/nodes"
)

// SimulatedLink represents a network link with realistic constraints
type SimulatedLink struct {
	Source    string
	Dest      string
	Bandwidth int     // bytes per second
	Latency   float64 // seconds (propagation delay)
	
	// Link state
	Active       bool
	mu           sync.RWMutex
	
	// Queue for messages in transit
	messageQueue []*InTransitMessage
	queueMu      sync.Mutex
	
	// Current utilization
	currentLoad  int     // bytes currently being transmitted
	lastUpdate   float64 // last time utilization was updated
	
	// Packet drop configuration
	dropRate     float64 // probability of random packet drop (0.0 - 1.0)
	dropOnCongestion bool // drop packets when link is saturated
	
	// Metrics
	metrics *LinkMetrics
}

// InTransitMessage represents a message traveling through a link
type InTransitMessage struct {
	Message      *nodes.Message
	Size         int     // bytes
	ArrivalTime  float64 // when message will arrive at destination
	Dropped      bool
}

// LinkMetrics tracks link performance metrics
type LinkMetrics struct {
	mu                sync.RWMutex
	BytesSent         int64
	BytesDropped      int64
	MessagesSent      int64
	MessagesDropped   int64
	TotalLatency      float64
	UtilizationSamples []float64
	maxSamples        int
}

// NewLinkMetrics creates new link metrics
func NewLinkMetrics() *LinkMetrics {
	return &LinkMetrics{
		UtilizationSamples: make([]float64, 0),
		maxSamples:         1000, // Keep last 1000 samples
	}
}

// RecordSend records a message being sent
func (lm *LinkMetrics) RecordSend(size int, latency float64) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.BytesSent += int64(size)
	lm.MessagesSent++
	lm.TotalLatency += latency
}

// RecordDrop records a message being dropped
func (lm *LinkMetrics) RecordDrop(size int) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.BytesDropped += int64(size)
	lm.MessagesDropped++
}

// RecordUtilization records link utilization sample
func (lm *LinkMetrics) RecordUtilization(utilization float64) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	if len(lm.UtilizationSamples) >= lm.maxSamples {
		// Remove oldest sample
		lm.UtilizationSamples = lm.UtilizationSamples[1:]
	}
	lm.UtilizationSamples = append(lm.UtilizationSamples, utilization)
}

// GetUtilization returns current average utilization
func (lm *LinkMetrics) GetUtilization() float64 {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	if len(lm.UtilizationSamples) == 0 {
		return 0.0
	}
	
	sum := 0.0
	for _, u := range lm.UtilizationSamples {
		sum += u
	}
	return sum / float64(len(lm.UtilizationSamples))
}

// GetPacketLossRate returns packet loss rate
func (lm *LinkMetrics) GetPacketLossRate() float64 {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	total := lm.MessagesSent + lm.MessagesDropped
	if total == 0 {
		return 0.0
	}
	return float64(lm.MessagesDropped) / float64(total)
}

// GetAverageLatency returns average message latency
func (lm *LinkMetrics) GetAverageLatency() float64 {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	if lm.MessagesSent == 0 {
		return 0.0
	}
	return lm.TotalLatency / float64(lm.MessagesSent)
}

// GetStats returns all metrics
func (lm *LinkMetrics) GetStats() (sent int64, dropped int64, lossRate float64, avgLatency float64, utilization float64) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	total := lm.MessagesSent + lm.MessagesDropped
	if total > 0 {
		lossRate = float64(lm.MessagesDropped) / float64(total)
	}
	
	if lm.MessagesSent > 0 {
		avgLatency = lm.TotalLatency / float64(lm.MessagesSent)
	}
	
	if len(lm.UtilizationSamples) > 0 {
		sum := 0.0
		for _, u := range lm.UtilizationSamples {
			sum += u
		}
		utilization = sum / float64(len(lm.UtilizationSamples))
	}
	
	return lm.MessagesSent, lm.MessagesDropped, lossRate, avgLatency, utilization
}

// NewSimulatedLink creates a new simulated link
func NewSimulatedLink(source, dest string, bandwidth int, latency float64) *SimulatedLink {
	return &SimulatedLink{
		Source:           source,
		Dest:             dest,
		Bandwidth:        bandwidth,
		Latency:          latency,
		Active:           true,
		messageQueue:     make([]*InTransitMessage, 0),
		currentLoad:      0,
		lastUpdate:       0.0,
		dropRate:         0.0,
		dropOnCongestion: false,
		metrics:          NewLinkMetrics(),
	}
}

// SetDropRate sets the random packet drop rate
func (sl *SimulatedLink) SetDropRate(rate float64) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.dropRate = rate
}

// SetDropOnCongestion enables/disables dropping packets when congested
func (sl *SimulatedLink) SetDropOnCongestion(enabled bool) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.dropOnCongestion = enabled
}

// IsActive returns whether the link is active
func (sl *SimulatedLink) IsActive() bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.Active
}

// SetActive sets the link active state
func (sl *SimulatedLink) SetActive(active bool) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.Active = active
}

// CanTransmit checks if the link can accept a message
func (sl *SimulatedLink) CanTransmit(size int, currentTime float64) bool {
	sl.queueMu.Lock()
	defer sl.queueMu.Unlock()
	
	if !sl.Active {
		return false
	}
	
	// Update current load based on elapsed time
	sl.updateLoad(currentTime)
	
	// Check if adding this message would exceed bandwidth
	return sl.currentLoad+size <= sl.Bandwidth
}

// GetCurrentUtilization returns current link utilization (0.0 - 1.0)
func (sl *SimulatedLink) GetCurrentUtilization(currentTime float64) float64 {
	sl.queueMu.Lock()
	defer sl.queueMu.Unlock()
	
	sl.updateLoad(currentTime)
	
	if sl.Bandwidth == 0 {
		return 0.0
	}
	return float64(sl.currentLoad) / float64(sl.Bandwidth)
}

// updateLoad updates the current load based on elapsed time
func (sl *SimulatedLink) updateLoad(currentTime float64) {
	elapsed := currentTime - sl.lastUpdate
	if elapsed > 0 {
		// Reduce load based on elapsed time and bandwidth
		bytesTransmitted := int(float64(sl.Bandwidth) * elapsed)
		sl.currentLoad -= bytesTransmitted
		if sl.currentLoad < 0 {
			sl.currentLoad = 0
		}
		sl.lastUpdate = currentTime
	}
}

// TransmitMessage attempts to transmit a message on the link
func (sl *SimulatedLink) TransmitMessage(msg *nodes.Message, size int, currentTime float64) (*InTransitMessage, error) {
	sl.queueMu.Lock()
	defer sl.queueMu.Unlock()
	
	if !sl.Active {
		sl.metrics.RecordDrop(size)
		return nil, fmt.Errorf("link is not active")
	}
	
	// Update load
	sl.updateLoad(currentTime)
	
	// Check for random drop
	if sl.dropRate > 0 && rand.Float64() < sl.dropRate {
		sl.metrics.RecordDrop(size)
		return nil, fmt.Errorf("packet randomly dropped")
	}
	
	// Check for congestion drop
	if sl.dropOnCongestion && sl.currentLoad+size > sl.Bandwidth {
		sl.metrics.RecordDrop(size)
		return nil, fmt.Errorf("link congested, packet dropped")
	}
	
	// Calculate arrival time = current time + propagation delay + transmission delay
	transmissionDelay := float64(size) / float64(sl.Bandwidth)
	arrivalTime := currentTime + sl.Latency + transmissionDelay
	
	// Create in-transit message
	inTransit := &InTransitMessage{
		Message:     msg,
		Size:        size,
		ArrivalTime: arrivalTime,
		Dropped:     false,
	}
	
	// Add to load
	sl.currentLoad += size
	
	// Record metrics
	sl.metrics.RecordSend(size, sl.Latency+transmissionDelay)
	
	return inTransit, nil
}

// UpdateMetrics updates link metrics at the current time
func (sl *SimulatedLink) UpdateMetrics(currentTime float64) {
	utilization := sl.GetCurrentUtilization(currentTime)
	sl.metrics.RecordUtilization(utilization)
}

// GetMetrics returns link metrics
func (sl *SimulatedLink) GetMetrics() *LinkMetrics {
	return sl.metrics
}

// String returns a string representation of the link
func (sl *SimulatedLink) String() string {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	
	status := "UP"
	if !sl.Active {
		status = "DOWN"
	}
	
	return fmt.Sprintf("Link{%s->%s, BW:%d B/s, Latency:%.3fs, Status:%s}",
		sl.Source, sl.Dest, sl.Bandwidth, sl.Latency, status)
}