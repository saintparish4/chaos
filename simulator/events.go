package simulator

import (
	"container/heap"
	"fmt"
	"time"
)

// EventType represents different types of simulation events
type EventType string

const (
	EventMessageSend   EventType = "MESSAGE_SEND"
	EventMessageArrive EventType = "MESSAGE_ARRIVE"
	EventNodeFailure   EventType = "NODE_FAILURE"
	EventNodeRecover   EventType = "NODE_RECOVER"
	EventLinkFailure   EventType = "LINK_FAILURE"
	EventLinkRecover   EventType = "LINK_RECOVER"
	EventChaosAction   EventType = "CHAOS_ACTION"
)

// SimulationEvent represents an event in the discrete event simulation
type SimulationEvent struct {
	ID        string
	Type      EventType
	Timestamp float64 // Simulation time in seconds
	NodeID    string  // Associated node (if applicable)
	MessageID string  // Associated message (if applicable)
	LinkID    string  // Associated link (if applicable)
	Data      interface{}
	index     int // Index in the priority queue
}

// EventQueue implements a priority queue for simulation events
type EventQueue struct {
	events []*SimulationEvent
}

func NewEventQueue() *EventQueue {
	eq := &EventQueue{
		events: make([]*SimulationEvent, 0),
	}
	heap.Init(eq)
	return eq
}

// Len returns the number of events in the queue
func (eq EventQueue) Len() int {
	return len(eq.events)
}

// Less compares two events by timestamp (earlier events have higher priority)
func (eq EventQueue) Less(i, j int) bool {
	return eq.events[i].Timestamp < eq.events[j].Timestamp
}

// Swap swaps two events in the queue
func (eq EventQueue) Swap(i, j int) {
	eq.events[i], eq.events[j] = eq.events[j], eq.events[i]
	eq.events[i].index = i
	eq.events[j].index = j
}

// Push adds an event to the queue
func (eq *EventQueue) Push(x interface{}) {
	n := len(eq.events)
	event := x.(*SimulationEvent)
	event.index = n
	eq.events = append(eq.events, event)
}

// Pop removes and returns the highest priority event
func (eq *EventQueue) Pop() interface{} {
	old := eq.events
	n := len(old)
	event := old[n-1]
	old[n-1] = nil
	event.index = -1
	eq.events = old[0 : n-1]
	return event
}

// Enqueue adds an event to the queue
func (eq *EventQueue) Enqueue(event *SimulationEvent) {
	heap.Push(eq, event)
}

// Dequeue removes and returns the next event
func (eq *EventQueue) Dequeue() *SimulationEvent {
	if eq.Len() == 0 {
		return nil
	}
	return heap.Pop(eq).(*SimulationEvent)
}

// Peek returns the next event without removing it
func (eq *EventQueue) Peek() *SimulationEvent {
	if eq.Len() == 0 {
		return nil
	}
	return eq.events[0]
}

// SimulationClock manages simulation time
type SimulationClock struct {
	currentTime   float64 // Current simulation time in seconds
	speedMultiple float64 // Simulation speed (1.0 = real-time, 10.0 = 10x)
	realStartTime time.Time
	simStartTime  float64
	paused        bool
}

// NewSimulationClock creates a new simulation clock
func NewSimulationClock(speedMultiple float64) *SimulationClock {
	return &SimulationClock{
		currentTime:   0.0,
		speedMultiple: speedMultiple,
		realStartTime: time.Now(),
		simStartTime:  0.0,
		paused:        false,
	}
}

// CurrentTime returns the current simulation time
func (sc *SimulationClock) CurrentTime() float64 {
	return sc.currentTime
}

// SetTime sets the simulation time
func (sc *SimulationClock) SetTime(t float64) {
	sc.currentTime = t
}

// Advance advances the simulation clock by the given duration
func (sc *SimulationClock) Advance(duration float64) {
	sc.currentTime += duration
}

// SetSpeed sets the simulation speed multiplier
func (sc *SimulationClock) SetSpeed(multiple float64) {
	sc.speedMultiple = multiple
}

// GetSpeed returns the current simulation speed
func (sc *SimulationClock) GetSpeed() float64 {
	return sc.speedMultiple
}

// Pause pauses the simulation clock
func (sc *SimulationClock) Pause() {
	sc.paused = true
}

// Resume resumes the simulation clock
func (sc *SimulationClock) Resume() {
	sc.paused = false
}

// IsPaused returns whether the simulation is paused
func (sc *SimulationClock) IsPaused() bool {
	return sc.paused
}

// Reset resets the simulation clock
func (sc *SimulationClock) Reset() {
	sc.currentTime = 0.0
	sc.realStartTime = time.Now()
	sc.simStartTime = 0.0
	sc.paused = false
}

// String returns a string representation of the clock
func (sc *SimulationClock) String() string {
	return fmt.Sprintf("SimTime: %.3fs (%.1fx speed)", sc.currentTime, sc.speedMultiple)
}

// MessageSendEventData contains data for MESSAGE_SEND events
type MessageSendEventData struct {
	Source      string
	Destination string
	Payload     []byte
	Size        int // bytes
}

// MessageArriveEventData contains data for MESSAGE_ARRIVE events
type MessageArriveEventData struct {
	MessageID string
	NodeID    string
	Size      int
}

// NodeFailureEventData contains data for NODE_FAILURE events
type NodeFailureEventData struct {
	NodeID   string
	Duration float64 // How long the failure lasts (0 = permanent)
}

// LinkFailureEventData contains data for LINK_FAILURE events
type LinkFailureEventData struct {
	Source   string
	Dest     string
	Duration float64
}

// ChaosActionEventData contains data for CHAOS_ACTION events
type ChaosActionEventData struct {
	Description string
	Execute     func() error
}

// EventStatistics tracks event processing statistics
type EventStatistics struct {
	TotalEvents     int64
	EventsByType    map[EventType]int64
	ProcessingTimes map[EventType]float64
}

// NewEventStatistics creates new event statistics
func NewEventStatistics() *EventStatistics {
	return &EventStatistics{
		EventsByType:    make(map[EventType]int64),
		ProcessingTimes: make(map[EventType]float64),
	}
}

// RecordEvent records an event being processed
func (es *EventStatistics) RecordEvent(eventType EventType, processingTime float64) {
	es.TotalEvents++
	es.EventsByType[eventType]++
	es.ProcessingTimes[eventType] += processingTime
}

// GetAverageProcessingTime returns average processing time for an event type
func (es *EventStatistics) GetAverageProcessingTime(eventType EventType) float64 {
	count := es.EventsByType[eventType]
	if count == 0 {
		return 0.0
	}
	return es.ProcessingTimes[eventType] / float64(count)
}

// String returns a string representation of the statistics
func (es *EventStatistics) String() string {
	return fmt.Sprintf("Events: %d, Types: %d", es.TotalEvents, len(es.EventsByType))
}
