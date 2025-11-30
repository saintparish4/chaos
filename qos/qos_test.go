package qos

import (
	"testing"
	"time"
)

func TestPriorityQueue(t *testing.T) {
	pq := NewPriorityQueue(10)

	// Test enqueue
	p1 := Packet{ID: "p1", Priority: PriorityHigh, Size: 100}
	p2 := Packet{ID: "p2", Priority: PriorityLow, Size: 200}
	p3 := Packet{ID: "p3", Priority: PriorityMedium, Size: 150}

	if !pq.Enqueue(p1) {
		t.Error("Failed to enqueue high priority packet")
	}

	if !pq.Enqueue(p2) {
		t.Error("Failed to enqueue low priority packet")
	}

	if !pq.Enqueue(p3) {
		t.Error("Failed to enqueue medium priority packet")
	}

	// Test depth
	if pq.GetDepth() != 3 {
		t.Errorf("Expected depth 3, got %d", pq.GetDepth())
	}

	depths := pq.GetDepthByPriority()
	if depths[PriorityHigh] != 1 || depths[PriorityMedium] != 1 || depths[PriorityLow] != 1 {
		t.Error("Incorrect depth distribution")
	}

	// Test strict priority dequeue
	packet, ok := pq.Dequeue()
	if !ok {
		t.Error("Dequeue failed")
	}

	if packet.ID != "p1" {
		t.Errorf("Expected highest priority packet first, got %s", packet.ID)
	}

	packet, _ = pq.Dequeue()
	if packet.ID != "p3" {
		t.Errorf("Expected medium priority next, got %s", packet.ID)
	}

	packet, _ = pq.Dequeue()
	if packet.ID != "p2" {
		t.Errorf("Expected low priority last, got %s", packet.ID)
	}

	// Queue should be empty
	if pq.GetDepth() != 0 {
		t.Errorf("Expected empty queue, got %d", pq.GetDepth())
	}
}

func TestPriorityQueueOverflow(t *testing.T) {
	pq := NewPriorityQueue(2) // Small queue

	// Fill queue
	pq.Enqueue(Packet{ID: "p1", Priority: PriorityHigh, Size: 100})
	pq.Enqueue(Packet{ID: "p2", Priority: PriorityHigh, Size: 100})

	// Try to overflow
	if pq.Enqueue(Packet{ID: "p3", Priority: PriorityHigh, Size: 100}) {
		t.Error("Should not accept packet when queue is full")
	}

	// Check drop stats
	drops := pq.GetDropStats()
	if drops[PriorityHigh] != 1 {
		t.Errorf("Expected 1 drop, got %d", drops[PriorityHigh])
	}
}

func TestTrafficShaper(t *testing.T) {
	shaper := NewTrafficShaper(1000.0, 2000) // 1000 bytes/sec, 2000 burst

	// Should allow first packet (have full burst)
	if !shaper.CanSend(1000) {
		t.Error("Should allow packet within burst")
	}

	// Should allow second packet (still have tokens)
	if !shaper.CanSend(500) {
		t.Error("Should allow packet with remaining tokens")
	}

	// Should drop packet exceeding tokens
	if shaper.CanSend(1000) {
		t.Error("Should not allow packet exceeding tokens")
	}

	stats := shaper.GetStats()
	if stats.PacketsShaped != 2 {
		t.Errorf("Expected 2 shaped packets, got %d", stats.PacketsShaped)
	}

	if stats.PacketsDropped != 1 {
		t.Errorf("Expected 1 dropped packet, got %d", stats.PacketsDropped)
	}

	// Wait for token refill
	time.Sleep(200 * time.Millisecond)

	// Should allow after refill
	if !shaper.CanSend(100) {
		t.Error("Should allow packet after token refill")
	}
}

func TestTokenRefill(t *testing.T) {
	shaper := NewTrafficShaper(1000.0, 500)

	// Drain tokens
	shaper.CanSend(500)

	// Should not allow immediately
	if shaper.CanSend(500) {
		t.Error("Should not allow without refill")
	}

	// Wait for refill (1000 bytes/sec = 100 bytes/100ms)
	time.Sleep(100 * time.Millisecond)

	// Should allow small packet after refill
	if !shaper.CanSend(50) {
		t.Error("Should allow small packet after partial refill")
	}
}

func TestPacketClassifier(t *testing.T) {
	classifier := NewPacketClassifier()

	// Add rules
	classifier.AddRule(ClassificationRule{
		Name:        "VoIP",
		SourceMatch: "voip-server",
		DestMatch:   "*",
		Priority:    PriorityHigh,
		DSCP:        46,
	})

	classifier.AddRule(ClassificationRule{
		Name:        "Default",
		SourceMatch: "*",
		DestMatch:   "*",
		Priority:    PriorityMedium,
		DSCP:        0,
	})

	// Test classification
	priority, dscp := classifier.Classify("voip-server", "client")
	if priority != PriorityHigh {
		t.Errorf("Expected HIGH priority, got %s", priority)
	}

	if dscp != 46 {
		t.Errorf("Expected DSCP 46, got %d", dscp)
	}

	// Test default rule
	priority, dscp = classifier.Classify("web-server", "client")
	if priority != PriorityMedium {
		t.Errorf("Expected MEDIUM priority, got %s", priority)
	}

	if dscp != 0 {
		t.Errorf("Expected DSCP 0, got %d", dscp)
	}
}

func TestQoSMetrics(t *testing.T) {
	metrics := NewQoSMetrics()

	// Record delays
	metrics.RecordDelay(PriorityHigh, 0.010)
	metrics.RecordDelay(PriorityHigh, 0.020)
	metrics.RecordDelay(PriorityMedium, 0.050)

	avgDelay := metrics.GetAverageDelay()
	expectedAvg := (0.010 + 0.020 + 0.050) / 3.0

	if avgDelay != expectedAvg {
		t.Errorf("Expected average delay %.3f, got %.3f", expectedAvg, avgDelay)
	}

	// Test jitter calculation
	jitter := metrics.GetAverageJitter()
	if jitter <= 0 {
		t.Error("Jitter should be positive")
	}

	// Record packet stats
	metrics.RecordPacket(PriorityHigh, false) // Sent
	metrics.RecordPacket(PriorityHigh, false) // Sent
	metrics.RecordPacket(PriorityHigh, true)  // Dropped

	lossRate := metrics.GetLossRate(PriorityHigh)
	expectedLoss := 1.0 / 3.0

	if lossRate != expectedLoss {
		t.Errorf("Expected loss rate %.3f, got %.3f", expectedLoss, lossRate)
	}
}

func TestQoSMetricsByPriority(t *testing.T) {
	metrics := NewQoSMetrics()

	// Record different delays for different priorities
	metrics.RecordDelay(PriorityHigh, 0.010)
	metrics.RecordDelay(PriorityMedium, 0.050)
	metrics.RecordDelay(PriorityLow, 0.100)

	// High priority should have lowest delay
	if metrics.HighPriorityDelay >= metrics.MediumPriorityDelay {
		t.Error("High priority delay should be less than medium")
	}

	if metrics.MediumPriorityDelay >= metrics.LowPriorityDelay {
		t.Error("Medium priority delay should be less than low")
	}
}

func TestPriorityLevelString(t *testing.T) {
	if PriorityHigh.String() != "HIGH" {
		t.Errorf("Expected 'HIGH', got '%s'", PriorityHigh.String())
	}

	if PriorityMedium.String() != "MEDIUM" {
		t.Errorf("Expected 'MEDIUM', got '%s'", PriorityMedium.String())
	}

	if PriorityLow.String() != "LOW" {
		t.Errorf("Expected 'LOW', got '%s'", PriorityLow.String())
	}
}

func TestQoSSummary(t *testing.T) {
	metrics := NewQoSMetrics()

	metrics.RecordDelay(PriorityHigh, 0.010)
	metrics.RecordDelay(PriorityHigh, 0.012)

	summary := metrics.GetSummary()
	if summary == "" {
		t.Error("Summary should not be empty")
	}
}

func TestShaperStats(t *testing.T) {
	shaper := NewTrafficShaper(1000.0, 2000)

	shaper.CanSend(500)
	shaper.CanSend(300)
	shaper.CanSend(1500) // Should be dropped

	stats := shaper.GetStats()

	if stats.Rate != 1000.0 {
		t.Errorf("Expected rate 1000.0, got %.1f", stats.Rate)
	}

	if stats.BurstSize != 2000 {
		t.Errorf("Expected burst size 2000, got %d", stats.BurstSize)
	}

	if stats.PacketsShaped != 2 {
		t.Errorf("Expected 2 shaped, got %d", stats.PacketsShaped)
	}

	if stats.PacketsDropped != 1 {
		t.Errorf("Expected 1 dropped, got %d", stats.PacketsDropped)
	}

	if stats.BytesShaped != 800 {
		t.Errorf("Expected 800 bytes shaped, got %d", stats.BytesShaped)
	}
}
