package simulator

import (
	"testing"
)

func TestPoissonTrafficGenerator(t *testing.T) {
	gen := NewPoissonTrafficGenerator("A", "B", 10.0, 1000)

	if gen.Source != "A" {
		t.Errorf("Expected source A, got %s", gen.Source)
	}
	if gen.Destination != "B" {
		t.Errorf("Expected destination B, got %s", gen.Destination)
	}
	if gen.Rate != 10.0 {
		t.Errorf("Expected rate 10.0, got %.2f", gen.Rate)
	}
	if gen.PacketSize != 1000 {
		t.Errorf("Expected packet size 1000, got %d", gen.PacketSize)
	}

	// Generate events
	events := gen.GenerateEvents(0.0, 10.0)

	// Should generate roughly 10 * 10 = 100 messages
	if len(events) < 50 || len(events) > 150 {
		t.Errorf("Expected ~100 events, got %d", len(events))
	}

	// All events should be MESSAGE_SEND
	for _, event := range events {
		if event.Type != EventMessageSend {
			t.Errorf("Expected MESSAGE_SEND, got %v", event.Type)
		}
		if event.Timestamp < 0.0 || event.Timestamp > 10.0 {
			t.Errorf("Event timestamp %.3f out of range [0, 10]", event.Timestamp)
		}
	}

	// Events should be in chronological order
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp < events[i-1].Timestamp {
			t.Error("Events not in chronological order")
			break
		}
	}
}

func TestBurstyTrafficGenerator(t *testing.T) {
	gen := NewBurstyTrafficGenerator(
		"A", "B",
		20.0, // burst rate
		10,   // messages per burst
		2.0,  // burst every 2 seconds
		500,  // packet size
	)

	events := gen.GenerateEvents(0.0, 10.0)

	// 10 seconds / 2 second period = 5 bursts
	// 5 bursts * 10 messages = 50 messages expected
	if len(events) < 40 || len(events) > 60 {
		t.Errorf("Expected ~50 events, got %d", len(events))
	}

	// Check event data
	for _, event := range events {
		if event.Type != EventMessageSend {
			t.Errorf("Expected MESSAGE_SEND, got %v", event.Type)
		}
		data := event.Data.(*MessageSendEventData)
		if data.Size != 500 {
			t.Errorf("Expected size 500, got %d", data.Size)
		}
	}
}

func TestConstantTrafficGenerator(t *testing.T) {
	gen := NewConstantTrafficGenerator("A", "B", 5.0, 200)

	events := gen.GenerateEvents(0.0, 10.0)

	// 5 messages per second * 10 seconds = 50 messages
	expectedCount := 50
	if len(events) != expectedCount {
		t.Errorf("Expected %d events, got %d", expectedCount, len(events))
	}

	// Check interval between messages
	if len(events) > 1 {
		expectedInterval := 0.2 // 1 / 5.0
		actualInterval := events[1].Timestamp - events[0].Timestamp
		if actualInterval != expectedInterval {
			t.Errorf("Expected interval %.3f, got %.3f", expectedInterval, actualInterval)
		}
	}
}

func TestTrafficManager(t *testing.T) {
	tm := NewTrafficManager()

	gen1 := NewPoissonTrafficGenerator("A", "B", 10.0, 1000)
	flow1 := NewTrafficFlow("flow1", gen1, 0.0, 5.0)

	gen2 := NewConstantTrafficGenerator("C", "D", 5.0, 500)
	flow2 := NewTrafficFlow("flow2", gen2, 0.0, 5.0)

	tm.AddFlow(flow1)
	tm.AddFlow(flow2)

	// Test GetFlow
	retrieved := tm.GetFlow("flow1")
	if retrieved == nil {
		t.Fatal("Expected to retrieve flow1")
	}
	if retrieved.ID != "flow1" {
		t.Errorf("Expected ID flow1, got %s", retrieved.ID)
	}

	// Test GetActiveFlows
	activeFlows := tm.GetActiveFlows()
	if len(activeFlows) != 2 {
		t.Errorf("Expected 2 active flows, got %d", len(activeFlows))
	}

	// Deactivate a flow
	flow1.Active = false
	activeFlows = tm.GetActiveFlows()
	if len(activeFlows) != 1 {
		t.Errorf("Expected 1 active flow after deactivation, got %d", len(activeFlows))
	}

	// Test RemoveFlow
	tm.RemoveFlow("flow2")
	retrieved = tm.GetFlow("flow2")
	if retrieved != nil {
		t.Error("flow2 should have been removed")
	}

	// Test GenerateAllEvents
	flow1.Active = true // Reactivate
	events := tm.GenerateAllEvents()
	if len(events) == 0 {
		t.Error("Expected events to be generated")
	}
}

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(5)

	if rb.Size() != 0 {
		t.Errorf("Expected size 0, got %d", rb.Size())
	}

	// Add data
	rb.Add(1.0, 10.0)
	rb.Add(2.0, 20.0)
	rb.Add(3.0, 30.0)

	if rb.Size() != 3 {
		t.Errorf("Expected size 3, got %d", rb.Size())
	}

	// Test GetAverage
	avg := rb.GetAverage()
	expected := 20.0
	if avg != expected {
		t.Errorf("Expected average %.1f, got %.1f", expected, avg)
	}

	// Test GetMax
	max := rb.GetMax()
	if max != 30.0 {
		t.Errorf("Expected max 30.0, got %.1f", max)
	}

	// Test GetMin
	min := rb.GetMin()
	if min != 10.0 {
		t.Errorf("Expected min 10.0, got %.1f", min)
	}

	// Test GetAll
	all := rb.GetAll()
	if len(all) != 3 {
		t.Errorf("Expected 3 data points, got %d", len(all))
	}

	// Fill buffer beyond capacity
	rb.Add(4.0, 40.0)
	rb.Add(5.0, 50.0)
	rb.Add(6.0, 60.0) // This should overwrite oldest

	if rb.Size() != 5 {
		t.Errorf("Expected size 5 (capacity), got %d", rb.Size())
	}

	// Oldest value (10.0) should be gone
	all = rb.GetAll()
	values := make([]float64, len(all))
	for i, point := range all {
		values[i] = point.Value
	}

	// Should contain 20, 30, 40, 50, 60 (10 was overwritten)
	for _, v := range values {
		if v == 10.0 {
			t.Error("Buffer should not contain oldest value 10.0 after overflow")
			break
		}
	}
}

func TestRingBufferGetRecent(t *testing.T) {
	rb := NewRingBuffer(10)

	// Add 5 values
	for i := 1; i <= 5; i++ {
		rb.Add(float64(i), float64(i*10))
	}

	// Get recent 3
	recent := rb.GetRecent(3)
	if len(recent) != 3 {
		t.Errorf("Expected 3 recent values, got %d", len(recent))
	}

	// Should be most recent: 30, 40, 50
	expected := []float64{30.0, 40.0, 50.0}
	for i, point := range recent {
		if point.Value != expected[i] {
			t.Errorf("Expected value %.1f at position %d, got %.1f", expected[i], i, point.Value)
		}
	}

	// Get more than available
	recent = rb.GetRecent(10)
	if len(recent) != 5 {
		t.Errorf("Expected 5 values (all available), got %d", len(recent))
	}
}

func TestSystemMetrics(t *testing.T) {
	sm := NewSystemMetrics(100)

	// Record some snapshots
	sm.RecordSnapshot(1.0, 100.0, 0.95, 0.01, 1000, 5, 0)
	sm.RecordSnapshot(2.0, 150.0, 0.90, 0.02, 2000, 4, 1)

	// Test summary
	summary := sm.GetSummary()

	if summary.AvgThroughput != 125.0 {
		t.Errorf("Expected avg throughput 125.0, got %.1f", summary.AvgThroughput)
	}

	if summary.MaxThroughput != 150.0 {
		t.Errorf("Expected max throughput 150.0, got %.1f", summary.MaxThroughput)
	}

	if summary.DataPoints != 2 {
		t.Errorf("Expected 2 data points, got %d", summary.DataPoints)
	}
}

func TestMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector(100)

	// Test GetOrCreate for nodes
	nodeMetrics1 := mc.GetOrCreateNodeMetrics("node-1")
	if nodeMetrics1 == nil {
		t.Fatal("Expected node metrics to be created")
	}
	if nodeMetrics1.NodeID != "node-1" {
		t.Errorf("Expected NodeID node-1, got %s", nodeMetrics1.NodeID)
	}

	// Getting same node should return same instance
	nodeMetrics2 := mc.GetOrCreateNodeMetrics("node-1")
	if nodeMetrics1 != nodeMetrics2 {
		t.Error("Expected same instance for same node ID")
	}

	// Test GetOrCreate for links
	linkMetrics := mc.GetOrCreateLinkMetrics("link-1")
	if linkMetrics == nil {
		t.Error("Expected link metrics to be created")
	}

	// Test Clear
	mc.Clear()
	retrievedNode := mc.GetNodeMetrics("node-1")
	if retrievedNode != nil {
		t.Error("Metrics should be cleared")
	}
}

func BenchmarkRingBuffer(b *testing.B) {
	rb := NewRingBuffer(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Add(float64(i), float64(i))
	}
}

func BenchmarkPoissonGeneration(b *testing.B) {
	gen := NewPoissonTrafficGenerator("A", "B", 100.0, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.GenerateEvents(0.0, 1.0)
	}
}
