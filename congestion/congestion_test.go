package congestion

import (
	"testing"
)

func TestCongestionController(t *testing.T) {
	cc := NewCongestionController("test-conn", "src", "dest")

	// Test initial state
	if cc.State != StateSlowStart {
		t.Errorf("Expected initial state SLOW_START, got %s", cc.State)
	}

	if cc.CongestionWindow != InitialWindow {
		t.Errorf("Expected initial window %.1f, got %.1f", InitialWindow, cc.CongestionWindow)
	}

	// Test slow start
	initialWindow := cc.CongestionWindow
	cc.OnAck(1)
	if cc.CongestionWindow <= initialWindow {
		t.Error("Window should increase in slow start")
	}

	// Test transition to congestion avoidance
	for i := 0; i < 100; i++ {
		cc.OnAck(1)
		if cc.State == StateCongestionAvoid {
			break
		}
	}

	if cc.State != StateCongestionAvoid {
		t.Error("Should transition to congestion avoidance")
	}

	// Test packet loss
	windowBeforeLoss := cc.CongestionWindow
	cc.OnLoss()

	if cc.CongestionWindow >= windowBeforeLoss {
		t.Error("Window should decrease on loss")
	}

	if cc.State != StateCongestionAvoid {
		t.Errorf("Expected CONGESTION_AVOIDANCE after loss, got %s", cc.State)
	}

	// Test timeout
	cc.OnTimeout()
	if cc.CongestionWindow != InitialWindow {
		t.Errorf("Window should reset to %f on timeout, got %.1f", InitialWindow, cc.CongestionWindow)
	}

	if cc.State != StateSlowStart {
		t.Errorf("Expected SLOW_START after timeout, got %s", cc.State)
	}
}

func TestCongestionWindow(t *testing.T) {
	cc := NewCongestionController("test", "a", "b")

	// Test window growth in slow start
	for i := 0; i < 5; i++ {
		before := cc.CongestionWindow
		cc.OnAck(1)
		after := cc.CongestionWindow

		// In slow start, window should grow by 1 per ACK
		if after <= before {
			t.Errorf("Window did not grow: before=%.1f, after=%.1f", before, after)
		}
	}

	// Test window cap
	cc.CongestionWindow = MaxWindow - 1
	cc.OnAck(10)
	if cc.CongestionWindow > MaxWindow {
		t.Errorf("Window exceeded maximum: %.1f > %.1f", cc.CongestionWindow, MaxWindow)
	}
}

func TestRTTMeasurement(t *testing.T) {
	cc := NewCongestionController("test", "a", "b")

	initialRTT := cc.RTT
	cc.UpdateRTT(0.2) // 200ms (different from initial 100ms)

	if cc.RTT == initialRTT {
		t.Error("RTT should be updated")
	}

	// Test EWMA
	cc.UpdateRTT(0.2) // 200ms
	if cc.RTT >= 0.2 {
		t.Error("RTT should be smoothed (EWMA)")
	}

	// Test RTO update
	if cc.RTO <= cc.RTT {
		t.Error("RTO should be larger than RTT")
	}
}

func TestCanSend(t *testing.T) {
	cc := NewCongestionController("test", "a", "b")
	cc.CongestionWindow = 5.0

	// Can send when outstanding < window
	if !cc.CanSend(4) {
		t.Error("Should be able to send when outstanding < window")
	}

	// Cannot send when outstanding >= window
	if cc.CanSend(5) {
		t.Error("Should not be able to send when outstanding >= window")
	}

	if cc.CanSend(6) {
		t.Error("Should not be able to send when outstanding > window")
	}
}

func TestFairQueuingScheduler(t *testing.T) {
	fq := NewFairQueuingScheduler()

	// Test enqueue
	fq.Enqueue("flow-1", "packet-1-1")
	fq.Enqueue("flow-1", "packet-1-2")
	fq.Enqueue("flow-2", "packet-2-1")
	fq.Enqueue("flow-3", "packet-3-1")

	if fq.GetQueueDepth() != 4 {
		t.Errorf("Expected queue depth 4, got %d", fq.GetQueueDepth())
	}

	// Test round-robin dequeue
	flows := make(map[string]int)
	for i := 0; i < 4; i++ {
		flowID, packet := fq.Dequeue()
		if packet == nil {
			t.Error("Dequeue returned nil packet")
		}
		flows[flowID]++
	}

	// Should have served all three flows
	if len(flows) != 3 {
		t.Errorf("Expected 3 flows served, got %d", len(flows))
	}

	// flow-1 should have been served twice (had 2 packets)
	if flows["flow-1"] != 2 {
		t.Errorf("Expected flow-1 to be served 2 times, got %d", flows["flow-1"])
	}

	// Queue should be empty
	if fq.GetQueueDepth() != 0 {
		t.Errorf("Expected empty queue, got depth %d", fq.GetQueueDepth())
	}
}

func TestFairQueuingStats(t *testing.T) {
	fq := NewFairQueuingScheduler()

	// Enqueue and dequeue packets
	fq.Enqueue("flow-1", "p1")
	fq.Enqueue("flow-1", "p2")
	fq.Enqueue("flow-2", "p3")

	fq.Dequeue()
	fq.Dequeue()
	fq.Dequeue()

	stats := fq.GetFlowStats()

	if stats["flow-1"] != 2 {
		t.Errorf("Expected flow-1 to have sent 2 packets, got %d", stats["flow-1"])
	}

	if stats["flow-2"] != 1 {
		t.Errorf("Expected flow-2 to have sent 1 packet, got %d", stats["flow-2"])
	}
}

func TestCongestionMetrics(t *testing.T) {
	cc := NewCongestionController("test", "a", "b")

	cc.PacketsSent = 100
	cc.PacketsAcked = 95
	cc.PacketsLost = 5

	metrics := cc.GetMetrics()

	if metrics.PacketsSent != 100 {
		t.Errorf("Expected 100 packets sent, got %d", metrics.PacketsSent)
	}

	if metrics.PacketsAcked != 95 {
		t.Errorf("Expected 95 packets acked, got %d", metrics.PacketsAcked)
	}

	if metrics.PacketsLost != 5 {
		t.Errorf("Expected 5 packets lost, got %d", metrics.PacketsLost)
	}

	// Test string output
	str := metrics.String()
	if str == "" {
		t.Error("Metrics string should not be empty")
	}
}

func TestReset(t *testing.T) {
	cc := NewCongestionController("test", "a", "b")

	// Modify state
	cc.CongestionWindow = 50.0
	cc.State = StateCongestionAvoid
	cc.PacketsSent = 100
	cc.PacketsLost = 10

	// Reset
	cc.Reset()

	// Check reset to initial values
	if cc.CongestionWindow != InitialWindow {
		t.Errorf("Window not reset, got %.1f", cc.CongestionWindow)
	}

	if cc.State != StateSlowStart {
		t.Errorf("State not reset, got %s", cc.State)
	}

	if cc.PacketsSent != 0 {
		t.Errorf("PacketsSent not reset, got %d", cc.PacketsSent)
	}
}
