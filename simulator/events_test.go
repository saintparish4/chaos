package simulator

import (
	"testing"
)

func TestEventQueue(t *testing.T) {
	eq := NewEventQueue()

	if eq.Len() != 0 {
		t.Errorf("Expected empty queue, got length %d", eq.Len())
	}

	// Add events in random order
	events := []*SimulationEvent{
		{ID: "e3", Type: EventMessageSend, Timestamp: 3.0},
		{ID: "e1", Type: EventMessageSend, Timestamp: 1.0},
		{ID: "e2", Type: EventMessageSend, Timestamp: 2.0},
		{ID: "e5", Type: EventMessageSend, Timestamp: 5.0},
		{ID: "e4", Type: EventMessageSend, Timestamp: 4.0},
	}

	for _, event := range events {
		eq.Enqueue(event)
	}

	if eq.Len() != 5 {
		t.Errorf("Expected 5 events, got %d", eq.Len())
	}

	// Dequeue should return events in timestamp order
	expectedOrder := []string{"e1", "e2", "e3", "e4", "e5"}
	for i, expectedID := range expectedOrder {
		event := eq.Dequeue()
		if event == nil {
			t.Fatalf("Expected event %d, got nil", i)
		}
		if event.ID != expectedID {
			t.Errorf("Expected event %s at position %d, got %s", expectedID, i, event.ID)
		}
	}

	if eq.Len() != 0 {
		t.Errorf("Expected empty queue after dequeuing all, got length %d", eq.Len())
	}

	// Dequeue from empty queue should return nil
	event := eq.Dequeue()
	if event != nil {
		t.Error("Expected nil from empty queue")
	}
}

func TestEventQueuePeek(t *testing.T) {
	eq := NewEventQueue()

	// Peek empty queue
	event := eq.Peek()
	if event != nil {
		t.Error("Expected nil peek from empty queue")
	}

	// Add events
	eq.Enqueue(&SimulationEvent{ID: "e2", Timestamp: 2.0})
	eq.Enqueue(&SimulationEvent{ID: "e1", Timestamp: 1.0})

	// Peek should return earliest event without removing it
	event = eq.Peek()
	if event == nil {
		t.Fatal("Expected event from peek")
	}
	if event.ID != "e1" {
		t.Errorf("Expected e1, got %s", event.ID)
	}

	// Queue size should not change after peek
	if eq.Len() != 2 {
		t.Errorf("Expected size 2 after peek, got %d", eq.Len())
	}
}

func TestSimulationClock(t *testing.T) {
	clock := NewSimulationClock(1.0)

	if clock.CurrentTime() != 0.0 {
		t.Errorf("Expected initial time 0.0, got %.3f", clock.CurrentTime())
	}

	if clock.GetSpeed() != 1.0 {
		t.Errorf("Expected speed 1.0, got %.1f", clock.GetSpeed())
	}

	// Test time advancement
	clock.Advance(5.5)
	if clock.CurrentTime() != 5.5 {
		t.Errorf("Expected time 5.5, got %.3f", clock.CurrentTime())
	}

	// Test set time
	clock.SetTime(10.0)
	if clock.CurrentTime() != 10.0 {
		t.Errorf("Expected time 10.0, got %.3f", clock.CurrentTime())
	}

	// Test speed change
	clock.SetSpeed(10.0)
	if clock.GetSpeed() != 10.0 {
		t.Errorf("Expected speed 10.0, got %.1f", clock.GetSpeed())
	}

	// Test pause/resume
	if clock.IsPaused() {
		t.Error("Clock should not be paused initially")
	}

	clock.Pause()
	if !clock.IsPaused() {
		t.Error("Clock should be paused after Pause()")
	}

	clock.Resume()
	if clock.IsPaused() {
		t.Error("Clock should not be paused after Resume()")
	}

	// Test reset
	clock.Reset()
	if clock.CurrentTime() != 0.0 {
		t.Errorf("Expected time 0.0 after reset, got %.3f", clock.CurrentTime())
	}
}

func TestEventStatistics(t *testing.T) {
	stats := NewEventStatistics()

	if stats.TotalEvents != 0 {
		t.Errorf("Expected 0 total events, got %d", stats.TotalEvents)
	}

	// Record events
	stats.RecordEvent(EventMessageSend, 0.001)
	stats.RecordEvent(EventMessageSend, 0.002)
	stats.RecordEvent(EventMessageArrive, 0.001)

	if stats.TotalEvents != 3 {
		t.Errorf("Expected 3 total events, got %d", stats.TotalEvents)
	}

	if stats.EventsByType[EventMessageSend] != 2 {
		t.Errorf("Expected 2 MESSAGE_SEND events, got %d", stats.EventsByType[EventMessageSend])
	}

	if stats.EventsByType[EventMessageArrive] != 1 {
		t.Errorf("Expected 1 MESSAGE_ARRIVE event, got %d", stats.EventsByType[EventMessageArrive])
	}

	// Test average processing time
	avgTime := stats.GetAverageProcessingTime(EventMessageSend)
	expected := 0.0015 // (0.001 + 0.002) / 2
	if avgTime != expected {
		t.Errorf("Expected avg time %.6f, got %.6f", expected, avgTime)
	}

	// Test average for event type with no events
	avgTime = stats.GetAverageProcessingTime(EventNodeFailure)
	if avgTime != 0.0 {
		t.Errorf("Expected 0.0 for non-existent event type, got %.6f", avgTime)
	}
}

func TestEventPriority(t *testing.T) {
	eq := NewEventQueue()

	// Add events with same timestamp
	eq.Enqueue(&SimulationEvent{ID: "e1", Timestamp: 1.0})
	eq.Enqueue(&SimulationEvent{ID: "e2", Timestamp: 1.0})
	eq.Enqueue(&SimulationEvent{ID: "e3", Timestamp: 1.0})

	// Should maintain some consistent order
	e1 := eq.Dequeue()
	e2 := eq.Dequeue()
	e3 := eq.Dequeue()

	if e1 == nil || e2 == nil || e3 == nil {
		t.Fatal("Expected all events to be dequeued")
	}

	// All should have timestamp 1.0
	if e1.Timestamp != 1.0 || e2.Timestamp != 1.0 || e3.Timestamp != 1.0 {
		t.Error("All events should have timestamp 1.0")
	}
}

func TestEventTypes(t *testing.T) {
	types := []EventType{
		EventMessageSend,
		EventMessageArrive,
		EventNodeFailure,
		EventNodeRecover,
		EventLinkFailure,
		EventLinkRecover,
	}

	for _, et := range types {
		if string(et) == "" {
			t.Errorf("Event type %v should not be empty string", et)
		}
	}
}

func BenchmarkEventQueue(b *testing.B) {
	eq := NewEventQueue()

	// Pre-populate with events
	for i := 0; i < 1000; i++ {
		eq.Enqueue(&SimulationEvent{
			ID:        "event",
			Timestamp: float64(i),
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		eq.Enqueue(&SimulationEvent{
			ID:        "new",
			Timestamp: float64(i + 1000),
		})
		eq.Dequeue()
	}
}
