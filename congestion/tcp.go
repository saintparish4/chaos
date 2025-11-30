package congestion

import (
	"fmt"
	"math"
	"sync"
)

// CongestionController implements TCP-like congestion control
type CongestionController struct {
	// Connection state
	ConnectionID string
	SourceNode   string
	DestNode     string

	// Congestion window
	CongestionWindow float64 // In packets
	SlowStartThresh  float64 // Slow start threshold

	// State
	State CongestionState
	RTT   float64 // Round-trip time
	RTO   float64 // Retransmission timeout

	// Packet tracking
	PacketsSent     int
	PacketsAcked    int
	PacketsLost     int
	Retransmissions int

	// Metrics
	MaxWindow       float64
	MinWindow       float64
	TimeInSlowStart float64
	TimeInCongAvoid float64

	mu sync.RWMutex
}

// CongestionState represents the state of the congestion controller
type CongestionState string

const (
	StateSlowStart       CongestionState = "SLOW_START"
	StateCongestionAvoid CongestionState = "CONGESTION_AVOIDANCE"
	StateFastRecovery    CongestionState = "FAST_RECOVERY"
)

const (
	InitialWindow    = 1.0    // Initial congestion window (1 packet)
	InitialThreshold = 64.0   // Initial slow start threshold
	MaxWindow        = 1000.0 // Maximum congestion window
	MinWindow        = 1.0    // Minimum congestion window
)

// NewCongestionController creates a new congestion controller
func NewCongestionController(connID, source, dest string) *CongestionController {
	return &CongestionController{
		ConnectionID:     connID,
		SourceNode:       source,
		DestNode:         dest,
		CongestionWindow: InitialWindow,
		SlowStartThresh:  InitialThreshold,
		State:            StateSlowStart,
		RTT:              0.1, // Default 100ms
		RTO:              0.3, // Default 300ms
		MaxWindow:        InitialWindow,
		MinWindow:        InitialWindow,
	}
}

// OnAck processes an acknowledgment
func (cc *CongestionController) OnAck(packetsAcked int) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.PacketsAcked += packetsAcked

	switch cc.State {
	case StateSlowStart:
		// Exponential growth: increase by number of ACKed packets
		cc.CongestionWindow += float64(packetsAcked)

		// Check if we've reached the threshold
		if cc.CongestionWindow >= cc.SlowStartThresh {
			cc.State = StateCongestionAvoid
			fmt.Printf("[TCP] %s: Entering Congestion Avoidance (cwnd=%.1f)\n",
				cc.ConnectionID, cc.CongestionWindow)
		}

	case StateCongestionAvoid:
		// Linear growth: increase by (acked / cwnd)
		// This approximates increasing by 1 per RTT
		increment := float64(packetsAcked) / cc.CongestionWindow
		cc.CongestionWindow += increment

	case StateFastRecovery:
		// Inflate window for each duplicate ACK
		cc.CongestionWindow += float64(packetsAcked)
	}

	// Cap at maximum window
	if cc.CongestionWindow > MaxWindow {
		cc.CongestionWindow = MaxWindow
	}

	// Track max window
	if cc.CongestionWindow > cc.MaxWindow {
		cc.MaxWindow = cc.CongestionWindow
	}
}

// OnLoss processes a packet loss event
func (cc *CongestionController) OnLoss() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.PacketsLost++

	fmt.Printf("[TCP] %s: Packet loss detected (cwnd=%.1f -> ",
		cc.ConnectionID, cc.CongestionWindow)

	// Multiplicative decrease
	cc.SlowStartThresh = math.Max(cc.CongestionWindow/2.0, 2.0)
	cc.CongestionWindow = cc.SlowStartThresh

	// Enter congestion avoidance (or fast recovery in more advanced TCP)
	cc.State = StateCongestionAvoid

	fmt.Printf("%.1f, ssthresh=%.1f)\n", cc.CongestionWindow, cc.SlowStartThresh)

	// Track min window
	if cc.CongestionWindow < cc.MinWindow {
		cc.MinWindow = cc.CongestionWindow
	}
}

// OnTimeout processes a timeout event
func (cc *CongestionController) OnTimeout() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.Retransmissions++

	fmt.Printf("[TCP] %s: Timeout (cwnd=%.1f -> ", cc.ConnectionID, cc.CongestionWindow)

	// More aggressive: reset to initial window
	cc.SlowStartThresh = math.Max(cc.CongestionWindow/2.0, 2.0)
	cc.CongestionWindow = InitialWindow
	cc.State = StateSlowStart

	// Double RTO (exponential backoff)
	cc.RTO *= 2.0
	if cc.RTO > 60.0 {
		cc.RTO = 60.0
	}

	fmt.Printf("%.1f, ssthresh=%.1f, RTO=%.3fs)\n",
		cc.CongestionWindow, cc.SlowStartThresh, cc.RTO)

	cc.MinWindow = InitialWindow
}

// GetWindow returns the current congestion window size
func (cc *CongestionController) GetWindow() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return int(math.Ceil(cc.CongestionWindow))
}

// CanSend checks if we can send more packets
func (cc *CongestionController) CanSend(outstandingPackets int) bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return float64(outstandingPackets) < cc.CongestionWindow
}

// UpdateRTT updates the RTT estimate
func (cc *CongestionController) UpdateRTT(measuredRTT float64) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Exponential weighted moving average
	alpha := 0.125
	cc.RTT = (1-alpha)*cc.RTT + alpha*measuredRTT

	// Update RTO (simplified)
	cc.RTO = cc.RTT * 2.0
	if cc.RTO < 0.2 {
		cc.RTO = 0.2
	}
}

// GetMetrics returns congestion control metrics
func (cc *CongestionController) GetMetrics() *CongestionMetrics {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return &CongestionMetrics{
		ConnectionID:    cc.ConnectionID,
		CurrentWindow:   cc.CongestionWindow,
		SlowStartThresh: cc.SlowStartThresh,
		State:           cc.State,
		PacketsSent:     cc.PacketsSent,
		PacketsAcked:    cc.PacketsAcked,
		PacketsLost:     cc.PacketsLost,
		Retransmissions: cc.Retransmissions,
		MaxWindow:       cc.MaxWindow,
		MinWindow:       cc.MinWindow,
		RTT:             cc.RTT,
		RTO:             cc.RTO,
	}
}

// Reset resets the congestion controller
func (cc *CongestionController) Reset() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.CongestionWindow = InitialWindow
	cc.SlowStartThresh = InitialThreshold
	cc.State = StateSlowStart
	cc.PacketsSent = 0
	cc.PacketsAcked = 0
	cc.PacketsLost = 0
	cc.Retransmissions = 0
	cc.MaxWindow = InitialWindow
	cc.MinWindow = InitialWindow
}

// CongestionMetrics holds metrics for a congestion controller
type CongestionMetrics struct {
	ConnectionID    string
	CurrentWindow   float64
	SlowStartThresh float64
	State           CongestionState
	PacketsSent     int
	PacketsAcked    int
	PacketsLost     int
	Retransmissions int
	MaxWindow       float64
	MinWindow       float64
	RTT             float64
	RTO             float64
}

// String returns a string representation of metrics
func (cm *CongestionMetrics) String() string {
	lossRate := 0.0
	if cm.PacketsSent > 0 {
		lossRate = float64(cm.PacketsLost) / float64(cm.PacketsSent) * 100
	}

	return fmt.Sprintf("cwnd=%.1f ssthresh=%.1f state=%s sent=%d acked=%d lost=%d (%.1f%%) rtt=%.3fs",
		cm.CurrentWindow, cm.SlowStartThresh, cm.State,
		cm.PacketsSent, cm.PacketsAcked, cm.PacketsLost, lossRate, cm.RTT)
}

// FairQueuingScheduler implements fair queuing for nodes
type FairQueuingScheduler struct {
	Queues       map[string]*FlowQueue // Flow ID -> Queue
	ActiveFlows  []string
	CurrentRound int
	mu           sync.RWMutex
}

// FlowQueue represents a per-flow queue
type FlowQueue struct {
	FlowID      string
	Packets     []interface{}
	BytesSent   int
	PacketsSent int
	VirtualTime float64
}

// NewFairQueuingScheduler creates a new fair queuing scheduler
func NewFairQueuingScheduler() *FairQueuingScheduler {
	return &FairQueuingScheduler{
		Queues:       make(map[string]*FlowQueue),
		ActiveFlows:  make([]string, 0),
		CurrentRound: 0,
	}
}

// Enqueue adds a packet to a flow's queue
func (fqs *FairQueuingScheduler) Enqueue(flowID string, packet interface{}) {
	fqs.mu.Lock()
	defer fqs.mu.Unlock()

	queue, exists := fqs.Queues[flowID]
	if !exists {
		queue = &FlowQueue{
			FlowID:  flowID,
			Packets: make([]interface{}, 0),
		}
		fqs.Queues[flowID] = queue
		fqs.ActiveFlows = append(fqs.ActiveFlows, flowID)
	}

	queue.Packets = append(queue.Packets, packet)
}

// Dequeue removes and returns the next packet using round-robin
func (fqs *FairQueuingScheduler) Dequeue() (string, interface{}) {
	fqs.mu.Lock()
	defer fqs.mu.Unlock()

	if len(fqs.ActiveFlows) == 0 {
		return "", nil
	}

	// Round-robin through active flows
	for i := 0; i < len(fqs.ActiveFlows); i++ {
		idx := (fqs.CurrentRound + i) % len(fqs.ActiveFlows)
		flowID := fqs.ActiveFlows[idx]
		queue := fqs.Queues[flowID]

		if len(queue.Packets) > 0 {
			packet := queue.Packets[0]
			queue.Packets = queue.Packets[1:]
			queue.PacketsSent++

			fqs.CurrentRound = (idx + 1) % len(fqs.ActiveFlows)

			// Remove flow if queue is empty
			if len(queue.Packets) == 0 {
				fqs.removeFlow(flowID)
			}

			return flowID, packet
		}
	}

	return "", nil
}

// removeFlow removes a flow from active flows
func (fqs *FairQueuingScheduler) removeFlow(flowID string) {
	for i, id := range fqs.ActiveFlows {
		if id == flowID {
			fqs.ActiveFlows = append(fqs.ActiveFlows[:i], fqs.ActiveFlows[i+1:]...)
			break
		}
	}
}

// GetQueueDepth returns the total queue depth
func (fqs *FairQueuingScheduler) GetQueueDepth() int {
	fqs.mu.RLock()
	defer fqs.mu.RUnlock()

	total := 0
	for _, queue := range fqs.Queues {
		total += len(queue.Packets)
	}
	return total
}

// GetFlowStats returns statistics for all flows
func (fqs *FairQueuingScheduler) GetFlowStats() map[string]int {
	fqs.mu.RLock()
	defer fqs.mu.RUnlock()

	stats := make(map[string]int)
	for flowID, queue := range fqs.Queues {
		stats[flowID] = queue.PacketsSent
	}
	return stats
}
