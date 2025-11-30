package qos

import (
	"fmt"
	"sync"
	"time"
)

// PriorityLevel represents the priority of traffic
type PriorityLevel int

const (
	PriorityHigh   PriorityLevel = 0
	PriorityMedium PriorityLevel = 1
	PriorityLow    PriorityLevel = 2
)

func (p PriorityLevel) String() string {
	switch p {
	case PriorityHigh:
		return "HIGH"
	case PriorityMedium:
		return "MEDIUM"
	case PriorityLow:
		return "LOW"
	default:
		return "UNKNOWN"
	}
}

// Packet represents a packet with QoS markings
type Packet struct {
	ID          string
	Payload     []byte
	Priority    PriorityLevel
	DSCP        int // Differentiated Services Code Point
	Size        int // In bytes
	ArrivalTime time.Time
	ServiceTime time.Time
	Source      string
	Dest        string
}

// PriorityQueue implements strict priority queuing
type PriorityQueue struct {
	Queues         [3][]Packet // One queue per priority level
	MaxDepth       int         // Maximum depth per queue
	DroppedPackets map[PriorityLevel]int
	mu             sync.RWMutex
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue(maxDepth int) *PriorityQueue {
	return &PriorityQueue{
		Queues:         [3][]Packet{},
		MaxDepth:       maxDepth,
		DroppedPackets: make(map[PriorityLevel]int),
	}
}

// Enqueue adds a packet to the appropriate priority queue
func (pq *PriorityQueue) Enqueue(packet Packet) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	priority := packet.Priority
	if priority < PriorityHigh || priority > PriorityLow {
		priority = PriorityMedium
	}

	// Check queue depth
	if len(pq.Queues[priority]) >= pq.MaxDepth {
		pq.DroppedPackets[priority]++
		return false
	}

	pq.Queues[priority] = append(pq.Queues[priority], packet)
	return true
}

// Dequeue removes and returns the highest priority packet
func (pq *PriorityQueue) Dequeue() (Packet, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	// Strict priority: always serve highest priority first
	for priority := PriorityHigh; priority <= PriorityLow; priority++ {
		if len(pq.Queues[priority]) > 0 {
			packet := pq.Queues[priority][0]
			pq.Queues[priority] = pq.Queues[priority][1:]
			packet.ServiceTime = time.Now()
			return packet, true
		}
	}

	return Packet{}, false
}

// GetDepth returns the total queue depth
func (pq *PriorityQueue) GetDepth() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	total := 0
	for i := 0; i < 3; i++ {
		total += len(pq.Queues[i])
	}
	return total
}

// GetDepthByPriority returns queue depth for each priority
func (pq *PriorityQueue) GetDepthByPriority() map[PriorityLevel]int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	depths := make(map[PriorityLevel]int)
	depths[PriorityHigh] = len(pq.Queues[PriorityHigh])
	depths[PriorityMedium] = len(pq.Queues[PriorityMedium])
	depths[PriorityLow] = len(pq.Queues[PriorityLow])
	return depths
}

// GetDropStats returns drop statistics
func (pq *PriorityQueue) GetDropStats() map[PriorityLevel]int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	stats := make(map[PriorityLevel]int)
	for priority, count := range pq.DroppedPackets {
		stats[priority] = count
	}
	return stats
}

// TrafficShaper implements token bucket rate limiting
type TrafficShaper struct {
	Rate           float64 // Tokens per second
	BurstSize      int     // Maximum burst size (tokens)
	Tokens         float64 // Current tokens
	LastUpdate     time.Time
	PacketsShaped  int
	BytesShaped    int
	PacketsDropped int
	mu             sync.RWMutex
}

// NewTrafficShaper creates a new traffic shaper
func NewTrafficShaper(rate float64, burstSize int) *TrafficShaper {
	return &TrafficShaper{
		Rate:       rate,
		BurstSize:  burstSize,
		Tokens:     float64(burstSize),
		LastUpdate: time.Now(),
	}
}

// CanSend checks if a packet can be sent based on token bucket
func (ts *TrafficShaper) CanSend(packetSize int) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Refill tokens
	ts.refillTokens()

	tokensNeeded := float64(packetSize)

	if ts.Tokens >= tokensNeeded {
		ts.Tokens -= tokensNeeded
		ts.PacketsShaped++
		ts.BytesShaped += packetSize
		return true
	}

	ts.PacketsDropped++
	return false
}

// refillTokens adds tokens based on elapsed time
func (ts *TrafficShaper) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(ts.LastUpdate).Seconds()
	ts.LastUpdate = now

	tokensToAdd := elapsed * ts.Rate
	ts.Tokens += tokensToAdd

	// Cap at burst size
	if ts.Tokens > float64(ts.BurstSize) {
		ts.Tokens = float64(ts.BurstSize)
	}
}

// GetStats returns shaper statistics
func (ts *TrafficShaper) GetStats() *ShaperStats {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return &ShaperStats{
		Rate:           ts.Rate,
		BurstSize:      ts.BurstSize,
		CurrentTokens:  ts.Tokens,
		PacketsShaped:  ts.PacketsShaped,
		BytesShaped:    ts.BytesShaped,
		PacketsDropped: ts.PacketsDropped,
	}
}

// ShaperStats holds traffic shaper statistics
type ShaperStats struct {
	Rate           float64
	BurstSize      int
	CurrentTokens  float64
	PacketsShaped  int
	BytesShaped    int
	PacketsDropped int
}

// PacketClassifier classifies packets based on rules
type PacketClassifier struct {
	Rules []ClassificationRule
	mu    sync.RWMutex
}

// ClassificationRule defines a rule for classifying packets
type ClassificationRule struct {
	Name        string
	SourceMatch string // Match source (can be prefix or exact)
	DestMatch   string // Match destination
	Priority    PriorityLevel
	DSCP        int
}

// NewPacketClassifier creates a new packet classifier
func NewPacketClassifier() *PacketClassifier {
	return &PacketClassifier{
		Rules: make([]ClassificationRule, 0),
	}
}

// AddRule adds a classification rule
func (pc *PacketClassifier) AddRule(rule ClassificationRule) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.Rules = append(pc.Rules, rule)
}

// Classify classifies a packet and returns its priority and DSCP
func (pc *PacketClassifier) Classify(source, dest string) (PriorityLevel, int) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	// Check rules in order (first match wins)
	for _, rule := range pc.Rules {
		if pc.matches(source, rule.SourceMatch) && pc.matches(dest, rule.DestMatch) {
			return rule.Priority, rule.DSCP
		}
	}

	// Default classification
	return PriorityMedium, 0
}

// matches checks if a value matches a pattern
func (pc *PacketClassifier) matches(value, pattern string) bool {
	if pattern == "*" || pattern == "" {
		return true
	}
	return value == pattern
}

// QoSMetrics tracks QoS performance metrics
type QoSMetrics struct {
	// Delay metrics
	TotalDelay          float64
	HighPriorityDelay   float64
	MediumPriorityDelay float64
	LowPriorityDelay    float64
	DelayMeasurements   int

	// Jitter metrics
	TotalJitter        float64
	JitterMeasurements int
	LastDelay          float64

	// Loss metrics
	PacketsSent    map[PriorityLevel]int
	PacketsDropped map[PriorityLevel]int

	mu sync.RWMutex
}

// NewQoSMetrics creates a new QoS metrics tracker
func NewQoSMetrics() *QoSMetrics {
	return &QoSMetrics{
		PacketsSent:    make(map[PriorityLevel]int),
		PacketsDropped: make(map[PriorityLevel]int),
	}
}

// RecordDelay records a delay measurement
func (qm *QoSMetrics) RecordDelay(priority PriorityLevel, delay float64) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	qm.TotalDelay += delay
	qm.DelayMeasurements++

	switch priority {
	case PriorityHigh:
		qm.HighPriorityDelay += delay
	case PriorityMedium:
		qm.MediumPriorityDelay += delay
	case PriorityLow:
		qm.LowPriorityDelay += delay
	}

	// Calculate jitter (variation in delay)
	if qm.LastDelay > 0 {
		jitter := delay - qm.LastDelay
		if jitter < 0 {
			jitter = -jitter
		}
		qm.TotalJitter += jitter
		qm.JitterMeasurements++
	}
	qm.LastDelay = delay
}

// RecordPacket records a sent or dropped packet
func (qm *QoSMetrics) RecordPacket(priority PriorityLevel, dropped bool) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if dropped {
		qm.PacketsDropped[priority]++
	} else {
		qm.PacketsSent[priority]++
	}
}

// GetAverageDelay returns the average delay
func (qm *QoSMetrics) GetAverageDelay() float64 {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	if qm.DelayMeasurements == 0 {
		return 0
	}
	return qm.TotalDelay / float64(qm.DelayMeasurements)
}

// GetAverageJitter returns the average jitter
func (qm *QoSMetrics) GetAverageJitter() float64 {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	if qm.JitterMeasurements == 0 {
		return 0
	}
	return qm.TotalJitter / float64(qm.JitterMeasurements)
}

// GetLossRate returns the packet loss rate by priority
func (qm *QoSMetrics) GetLossRate(priority PriorityLevel) float64 {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	sent := qm.PacketsSent[priority]
	dropped := qm.PacketsDropped[priority]
	total := sent + dropped

	if total == 0 {
		return 0
	}
	return float64(dropped) / float64(total)
}

// GetSummary returns a summary string
func (qm *QoSMetrics) GetSummary() string {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	avgDelay := qm.GetAverageDelay()
	avgJitter := qm.GetAverageJitter()

	return fmt.Sprintf("Avg Delay: %.3fs, Jitter: %.3fs, Measurements: %d",
		avgDelay, avgJitter, qm.DelayMeasurements)
}

// PrintDetailedStats prints detailed QoS statistics
func (qm *QoSMetrics) PrintDetailedStats() {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	fmt.Println("\n=== QoS Metrics ===")
	fmt.Printf("Average Delay: %.3fs\n", qm.GetAverageDelay())
	fmt.Printf("Average Jitter: %.3fs\n", qm.GetAverageJitter())

	fmt.Println("\nPer-Priority Statistics:")
	for _, priority := range []PriorityLevel{PriorityHigh, PriorityMedium, PriorityLow} {
		sent := qm.PacketsSent[priority]
		dropped := qm.PacketsDropped[priority]
		total := sent + dropped
		lossRate := 0.0
		if total > 0 {
			lossRate = float64(dropped) / float64(total) * 100
		}

		fmt.Printf("  %s: sent=%d dropped=%d (%.1f%% loss)\n",
			priority, sent, dropped, lossRate)
	}
}
