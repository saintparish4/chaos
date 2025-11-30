package simulator

import (
	"fmt"
	"math"
	"math/rand"
)

// TrafficGenerator generates traffic patterns for simulation
type TrafficGenerator interface {
	GenerateEvents(startTime, endTime float64) []*SimulationEvent
	GetDescription() string
}

// PoissonTrafficGenerator generates traffic following a Poisson process
type PoissonTrafficGenerator struct {
	Source      string
	Destination string
	Rate        float64 // messages per second (lambda)
	PacketSize  int     // bytes per message
	Payload     []byte
}

// NewPoissonTrafficGenerator creates a new Poisson traffic generator
func NewPoissonTrafficGenerator(source, dest string, rate float64, packetSize int) *PoissonTrafficGenerator {
	return &PoissonTrafficGenerator{
		Source:      source,
		Destination: dest,
		Rate:        rate,
		PacketSize:  packetSize,
		Payload:     make([]byte, packetSize),
	}
}

// GenerateEvents generates MESSAGE_SEND events following Poisson distribution
func (ptg *PoissonTrafficGenerator) GenerateEvents(startTime, endTime float64) []*SimulationEvent {
	events := make([]*SimulationEvent, 0)

	currentTime := startTime
	for currentTime < endTime {
		// Inter-arrival time follows exponential distribution
		// T = -ln(U) / lambda, where U is uniform(0,1)
		interArrival := -math.Log(rand.Float64()) / ptg.Rate
		currentTime += interArrival

		if currentTime >= endTime {
			break
		}

		eventID := fmt.Sprintf("poisson-%s-%s-%.6f", ptg.Source, ptg.Destination, currentTime)
		event := &SimulationEvent{
			ID:        eventID,
			Type:      EventMessageSend,
			Timestamp: currentTime,
			NodeID:    ptg.Source,
			Data: &MessageSendEventData{
				Source:      ptg.Source,
				Destination: ptg.Destination,
				Payload:     ptg.Payload,
				Size:        ptg.PacketSize,
			},
		}
		events = append(events, event)
	}

	return events
}

// GetDescription returns a description of this generator
func (ptg *PoissonTrafficGenerator) GetDescription() string {
	return fmt.Sprintf("Poisson(%s->%s, λ=%.2f msg/s, %d bytes)",
		ptg.Source, ptg.Destination, ptg.Rate, ptg.PacketSize)
}

// BurstyTrafficGenerator generates bursty traffic patterns
type BurstyTrafficGenerator struct {
	Source      string
	Destination string
	BurstRate   float64 // messages per second during burst
	BurstSize   int     // number of messages per burst
	BurstPeriod float64 // time between bursts (seconds)
	PacketSize  int     // bytes per message
	Payload     []byte
}

// NewBurstyTrafficGenerator creates a new bursty traffic generator
func NewBurstyTrafficGenerator(source, dest string, burstRate float64, burstSize int, burstPeriod float64, packetSize int) *BurstyTrafficGenerator {
	return &BurstyTrafficGenerator{
		Source:      source,
		Destination: dest,
		BurstRate:   burstRate,
		BurstSize:   burstSize,
		BurstPeriod: burstPeriod,
		PacketSize:  packetSize,
		Payload:     make([]byte, packetSize),
	}
}

// GenerateEvents generates bursty MESSAGE_SEND events
func (btg *BurstyTrafficGenerator) GenerateEvents(startTime, endTime float64) []*SimulationEvent {
	events := make([]*SimulationEvent, 0)

	currentBurstTime := startTime
	for currentBurstTime < endTime {
		// Generate a burst
		for i := 0; i < btg.BurstSize; i++ {
			// Messages within burst follow exponential inter-arrival
			interArrival := -math.Log(rand.Float64()) / btg.BurstRate
			messageTime := currentBurstTime + interArrival

			if messageTime >= endTime {
				break
			}

			eventID := fmt.Sprintf("burst-%s-%s-%.6f", btg.Source, btg.Destination, messageTime)
			event := &SimulationEvent{
				ID:        eventID,
				Type:      EventMessageSend,
				Timestamp: messageTime,
				NodeID:    btg.Source,
				Data: &MessageSendEventData{
					Source:      btg.Source,
					Destination: btg.Destination,
					Payload:     btg.Payload,
					Size:        btg.PacketSize,
				},
			}
			events = append(events, event)
		}

		// Move to next burst
		currentBurstTime += btg.BurstPeriod
	}

	return events
}

// GetDescription returns a description of this generator
func (btg *BurstyTrafficGenerator) GetDescription() string {
	return fmt.Sprintf("Bursty(%s->%s, %d msgs/burst, every %.2fs, %d bytes)",
		btg.Source, btg.Destination, btg.BurstSize, btg.BurstPeriod, btg.PacketSize)
}

// ConstantTrafficGenerator generates constant-rate traffic
type ConstantTrafficGenerator struct {
	Source      string
	Destination string
	Rate        float64 // messages per second
	PacketSize  int     // bytes per message
	Payload     []byte
}

// NewConstantTrafficGenerator creates a new constant-rate traffic generator
func NewConstantTrafficGenerator(source, dest string, rate float64, packetSize int) *ConstantTrafficGenerator {
	return &ConstantTrafficGenerator{
		Source:      source,
		Destination: dest,
		Rate:        rate,
		PacketSize:  packetSize,
		Payload:     make([]byte, packetSize),
	}
}

// GenerateEvents generates MESSAGE_SEND events at constant rate
func (ctg *ConstantTrafficGenerator) GenerateEvents(startTime, endTime float64) []*SimulationEvent {
	events := make([]*SimulationEvent, 0)

	interval := 1.0 / ctg.Rate

	for i := 0; ; i++ {
		currentTime := startTime + float64(i)*interval

		if currentTime >= endTime {
			break
		}

		eventID := fmt.Sprintf("constant-%s-%s-%.6f", ctg.Source, ctg.Destination, currentTime)
		event := &SimulationEvent{
			ID:        eventID,
			Type:      EventMessageSend,
			Timestamp: currentTime,
			NodeID:    ctg.Source,
			Data: &MessageSendEventData{
				Source:      ctg.Source,
				Destination: ctg.Destination,
				Payload:     ctg.Payload,
				Size:        ctg.PacketSize,
			},
		}
		events = append(events, event)
	}

	return events
}

// GetDescription returns a description of this generator
func (ctg *ConstantTrafficGenerator) GetDescription() string {
	return fmt.Sprintf("Constant(%s->%s, %.2f msg/s, %d bytes)",
		ctg.Source, ctg.Destination, ctg.Rate, ctg.PacketSize)
}

// TrafficFlow represents a configured traffic flow
type TrafficFlow struct {
	ID        string
	Generator TrafficGenerator
	StartTime float64
	EndTime   float64
	Active    bool
}

// NewTrafficFlow creates a new traffic flow
func NewTrafficFlow(id string, generator TrafficGenerator, startTime, endTime float64) *TrafficFlow {
	return &TrafficFlow{
		ID:        id,
		Generator: generator,
		StartTime: startTime,
		EndTime:   endTime,
		Active:    true,
	}
}

// GenerateEvents generates all events for this flow
func (tf *TrafficFlow) GenerateEvents() []*SimulationEvent {
	if !tf.Active {
		return []*SimulationEvent{}
	}
	return tf.Generator.GenerateEvents(tf.StartTime, tf.EndTime)
}

// TrafficManager manages multiple traffic flows
type TrafficManager struct {
	flows map[string]*TrafficFlow
}

// NewTrafficManager creates a new traffic manager
func NewTrafficManager() *TrafficManager {
	return &TrafficManager{
		flows: make(map[string]*TrafficFlow),
	}
}

// AddFlow adds a traffic flow
func (tm *TrafficManager) AddFlow(flow *TrafficFlow) {
	tm.flows[flow.ID] = flow
}

// RemoveFlow removes a traffic flow
func (tm *TrafficManager) RemoveFlow(flowID string) {
	delete(tm.flows, flowID)
}

// GetFlow returns a traffic flow by ID
func (tm *TrafficManager) GetFlow(flowID string) *TrafficFlow {
	return tm.flows[flowID]
}

// GenerateAllEvents generates events for all active flows
func (tm *TrafficManager) GenerateAllEvents() []*SimulationEvent {
	allEvents := make([]*SimulationEvent, 0)

	for _, flow := range tm.flows {
		if flow.Active {
			events := flow.GenerateEvents()
			allEvents = append(allEvents, events...)
		}
	}

	return allEvents
}

// GetActiveFlows returns all active flows
func (tm *TrafficManager) GetActiveFlows() []*TrafficFlow {
	active := make([]*TrafficFlow, 0)
	for _, flow := range tm.flows {
		if flow.Active {
			active = append(active, flow)
		}
	}
	return active
}

// ListFlows returns descriptions of all flows
func (tm *TrafficManager) ListFlows() []string {
	descriptions := make([]string, 0)
	for _, flow := range tm.flows {
		status := "ACTIVE"
		if !flow.Active {
			status = "INACTIVE"
		}
		desc := fmt.Sprintf("[%s] %s (%.2f-%.2fs) %s",
			status, flow.ID, flow.StartTime, flow.EndTime, flow.Generator.GetDescription())
		descriptions = append(descriptions, desc)
	}
	return descriptions
}
