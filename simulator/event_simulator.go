package simulator

import (
	"fmt"
	"sync"
	"time"

	"github.com/saintparish4/chaos/nodes"
)

// EventDrivenSimulator is the main simulation engine with discrete event processing
type EventDrivenSimulator struct {
	// Core components
	topology       *Topology
	nodes          map[string]*nodes.Node
	simulatedLinks map[string]*SimulatedLink // "source->dest" -> link

	// Event system
	eventQueue *EventQueue
	clock      *SimulationClock
	eventStats *EventStatistics

	// Traffic generation
	trafficMgr *TrafficManager

	// Metrics
	metrics        *MetricsCollector
	messageTracker map[string]*MessageTrack

	// Simulation state
	running   bool
	paused    bool
	mu        sync.RWMutex
	trackerMu sync.RWMutex

	// Configuration
	metricsInterval float64 // How often to collect metrics (seconds)
	lastMetricsTime float64

	// Statistics
	totalMessagesSent      int64
	totalMessagesDelivered int64
	totalLatency           float64
}

// NewEventDrivenSimulator creates a new event-driven simulator with default speed (1.0x)
func NewEventDrivenSimulator(topo *Topology) *EventDrivenSimulator {
	return NewEventDrivenSimulatorWithSpeed(topo, 1.0)
}

// NewEventDrivenSimulatorWithSpeed creates a new event-driven simulator with custom speed
func NewEventDrivenSimulatorWithSpeed(topo *Topology, speedMultiple float64) *EventDrivenSimulator {
	sim := &EventDrivenSimulator{
		topology:        topo,
		nodes:           make(map[string]*nodes.Node),
		simulatedLinks:  make(map[string]*SimulatedLink),
		eventQueue:      NewEventQueue(),
		clock:           NewSimulationClock(speedMultiple),
		eventStats:      NewEventStatistics(),
		trafficMgr:      NewTrafficManager(),
		metrics:         NewMetricsCollector(1000), // 1000 samples
		messageTracker:  make(map[string]*MessageTrack),
		metricsInterval: 0.1, // Collect metrics every 0.1 seconds
	}

	// Create nodes
	for _, nodeCfg := range topo.Config.Nodes {
		node := nodes.NewNode(nodeCfg.ID, nodes.NodeType(nodeCfg.Type), nodeCfg.Capacity)
		sim.nodes[nodeCfg.ID] = node
	}

	// Create simulated links
	for _, linkCfg := range topo.Config.Links {
		linkID := fmt.Sprintf("%s->%s", linkCfg.Source, linkCfg.Dest)
		link := NewSimulatedLink(
			linkCfg.Source,
			linkCfg.Dest,
			linkCfg.Bandwidth,
			float64(linkCfg.Latency)/1000.0, // Convert ms to seconds
		)
		sim.simulatedLinks[linkID] = link
	}

	// Compute routing tables
	sim.UpdateRoutingTables()

	return sim
}

// UpdateRoutingTables recomputes routing tables for all nodes
func (sim *EventDrivenSimulator) UpdateRoutingTables() {
	allTables := ComputeAllRoutingTables(sim.topology)

	sim.mu.Lock()
	defer sim.mu.Unlock()

	for nodeID, table := range allTables {
		if node, exists := sim.nodes[nodeID]; exists {
			node.SetRoutingTable(table)
		}
	}
}

// AddTrafficFlow adds a traffic flow to the simulation
func (sim *EventDrivenSimulator) AddTrafficFlow(flow *TrafficFlow) {
	sim.trafficMgr.AddFlow(flow)
}

// ScheduleEvent schedules a simulation event
func (sim *EventDrivenSimulator) ScheduleEvent(event *SimulationEvent) {
	sim.eventQueue.Enqueue(event)
}

// Run runs the simulation for a specified duration
func (sim *EventDrivenSimulator) Run(duration float64) error {
	sim.mu.Lock()
	sim.running = true
	sim.mu.Unlock()

	endTime := sim.clock.CurrentTime() + duration

	// Generate traffic events
	trafficEvents := sim.trafficMgr.GenerateAllEvents()
	for _, event := range trafficEvents {
		if event.Timestamp <= endTime {
			sim.ScheduleEvent(event)
		}
	}

	fmt.Printf("Starting simulation: %.2fs -> %.2fs (%.1fx speed)\n",
		sim.clock.CurrentTime(), endTime, sim.clock.GetSpeed())
	fmt.Printf("Event queue: %d events\n", sim.eventQueue.Len())

	eventsProcessed := 0

	// Main event loop
	for {
		// Check if we should stop
		sim.mu.RLock()
		if !sim.running {
			sim.mu.RUnlock()
			break
		}
		sim.mu.RUnlock()

		// Get next event
		nextEvent := sim.eventQueue.Peek()
		if nextEvent == nil {
			break // No more events
		}

		// Check if event is beyond simulation end time
		if nextEvent.Timestamp > endTime {
			break
		}

		// Process event
		event := sim.eventQueue.Dequeue()

		// Advance clock to event time
		sim.clock.SetTime(event.Timestamp)

		// Check if we should collect metrics
		if sim.clock.CurrentTime() >= sim.lastMetricsTime+sim.metricsInterval {
			sim.collectMetrics()
			sim.lastMetricsTime = sim.clock.CurrentTime()
		}

		// Process the event
		startTime := time.Now()
		sim.processEvent(event)
		processingTime := time.Since(startTime).Seconds()

		sim.eventStats.RecordEvent(event.Type, processingTime)
		eventsProcessed++

		// Optional: limit event processing rate for visualization
		if eventsProcessed%1000 == 0 {
			time.Sleep(1 * time.Millisecond)
		}
	}

	// Final metrics collection
	sim.collectMetrics()

	sim.mu.Lock()
	sim.running = false
	sim.mu.Unlock()

	fmt.Printf("Simulation complete: processed %d events\n", eventsProcessed)

	return nil
}

// processEvent processes a single simulation event
func (sim *EventDrivenSimulator) processEvent(event *SimulationEvent) {
	switch event.Type {
	case EventMessageSend:
		sim.handleMessageSend(event)
	case EventMessageArrive:
		sim.handleMessageArrive(event)
	case EventNodeFailure:
		sim.handleNodeFailure(event)
	case EventNodeRecover:
		sim.handleNodeRecover(event)
	case EventLinkFailure:
		sim.handleLinkFailure(event)
	case EventLinkRecover:
		sim.handleLinkRecover(event)
	case EventChaosAction:
		sim.handleChaosAction(event)
	}
}

// handleMessageSend handles MESSAGE_SEND event
func (sim *EventDrivenSimulator) handleMessageSend(event *SimulationEvent) {
	data := event.Data.(*MessageSendEventData)

	// Create message
	msgID := fmt.Sprintf("msg-%s-%.6f", data.Source, event.Timestamp)
	msg := &nodes.Message{
		ID:          msgID,
		Source:      data.Source,
		Destination: data.Destination,
		Payload:     data.Payload,
		Timestamp:   time.Now(),
		HopCount:    0,
		Path:        []string{data.Source},
	}

	// Track message
	sim.trackerMu.Lock()
	sim.messageTracker[msgID] = &MessageTrack{
		Message:   msg,
		StartTime: time.Unix(0, int64(event.Timestamp*1e9)),
		Delivered: false,
		Path:      []string{data.Source},
	}
	sim.totalMessagesSent++
	sim.trackerMu.Unlock()

	// If source == destination, deliver immediately
	if data.Source == data.Destination {
		sim.trackerMu.Lock()
		if track, exists := sim.messageTracker[msgID]; exists {
			track.Delivered = true
			track.EndTime = time.Unix(0, int64(event.Timestamp*1e9))
		}
		sim.totalMessagesDelivered++
		sim.trackerMu.Unlock()
		return
	}

	// Get source node
	sourceNode, exists := sim.nodes[data.Source]
	if !exists || sourceNode.GetState() == nodes.StateFailed {
		return // Source doesn't exist or is failed
	}

	// Get next hop
	nextHop, exists := sourceNode.GetNextHop(data.Destination)
	if !exists {
		return // No route
	}

	// Try to transmit on link
	linkID := fmt.Sprintf("%s->%s", data.Source, nextHop)
	link, exists := sim.simulatedLinks[linkID]
	if !exists || !link.IsActive() {
		return // Link doesn't exist or is down
	}

	// Transmit message
	inTransit, err := link.TransmitMessage(msg, data.Size, event.Timestamp)
	if err != nil {
		// Message dropped
		return
	}

	// Update message path
	msg.HopCount++
	msg.Path = append(msg.Path, nextHop)

	// Schedule arrival event
	arrivalEvent := &SimulationEvent{
		ID:        fmt.Sprintf("arrive-%s-%s", msgID, nextHop),
		Type:      EventMessageArrive,
		Timestamp: inTransit.ArrivalTime,
		NodeID:    nextHop,
		MessageID: msgID,
		Data: &MessageArriveEventData{
			MessageID: msgID,
			NodeID:    nextHop,
			Size:      data.Size,
		},
	}
	sim.ScheduleEvent(arrivalEvent)

	sourceNode.IncrementMessagesSent()
}

// handleMessageArrive handles MESSAGE_ARRIVE event
func (sim *EventDrivenSimulator) handleMessageArrive(event *SimulationEvent) {
	data := event.Data.(*MessageArriveEventData)

	// Get message from tracker
	sim.trackerMu.RLock()
	track, exists := sim.messageTracker[data.MessageID]
	sim.trackerMu.RUnlock()

	if !exists {
		return // Message not found
	}

	msg := track.Message

	// Check if message reached destination
	if data.NodeID == msg.Destination {
		// Message delivered!
		sim.trackerMu.Lock()
		track.Delivered = true
		track.EndTime = time.Unix(0, int64(event.Timestamp*1e9))
		track.HopCount = msg.HopCount
		track.Path = msg.Path
		sim.totalMessagesDelivered++
		latency := track.EndTime.Sub(track.StartTime).Seconds()
		sim.totalLatency += latency
		sim.trackerMu.Unlock()
		return
	}

	// Get current node
	currentNode, exists := sim.nodes[data.NodeID]
	if !exists || currentNode.GetState() == nodes.StateFailed {
		return // Node doesn't exist or failed
	}

	// Get next hop
	nextHop, exists := currentNode.GetNextHop(msg.Destination)
	if !exists {
		return // No route
	}

	// Try to transmit on next link
	linkID := fmt.Sprintf("%s->%s", data.NodeID, nextHop)
	link, exists := sim.simulatedLinks[linkID]
	if !exists || !link.IsActive() {
		return // Link doesn't exist or is down
	}

	// Transmit message
	inTransit, err := link.TransmitMessage(msg, data.Size, event.Timestamp)
	if err != nil {
		// Message dropped
		return
	}

	// Update message
	msg.HopCount++
	msg.Path = append(msg.Path, nextHop)

	// Update tracker
	sim.trackerMu.Lock()
	track.Path = msg.Path
	track.HopCount = msg.HopCount
	sim.trackerMu.Unlock()

	// Schedule next arrival
	arrivalEvent := &SimulationEvent{
		ID:        fmt.Sprintf("arrive-%s-%s", data.MessageID, nextHop),
		Type:      EventMessageArrive,
		Timestamp: inTransit.ArrivalTime,
		NodeID:    nextHop,
		MessageID: data.MessageID,
		Data: &MessageArriveEventData{
			MessageID: data.MessageID,
			NodeID:    nextHop,
			Size:      data.Size,
		},
	}
	sim.ScheduleEvent(arrivalEvent)

	currentNode.IncrementMessagesSent()
}

// handleNodeFailure handles NODE_FAILURE event
func (sim *EventDrivenSimulator) handleNodeFailure(event *SimulationEvent) {
	data := event.Data.(*NodeFailureEventData)

	node, exists := sim.nodes[data.NodeID]
	if !exists {
		return
	}

	node.SetState(nodes.StateFailed)

	// Schedule recovery if duration > 0
	if data.Duration > 0 {
		recoveryEvent := &SimulationEvent{
			ID:        fmt.Sprintf("recover-%s", data.NodeID),
			Type:      EventNodeRecover,
			Timestamp: event.Timestamp + data.Duration,
			NodeID:    data.NodeID,
		}
		sim.ScheduleEvent(recoveryEvent)
	}
}

// handleNodeRecover handles NODE_RECOVER event
func (sim *EventDrivenSimulator) handleNodeRecover(event *SimulationEvent) {
	node, exists := sim.nodes[event.NodeID]
	if !exists {
		return
	}

	node.SetState(nodes.StateActive)
}

// handleLinkFailure handles LINK_FAILURE event
func (sim *EventDrivenSimulator) handleLinkFailure(event *SimulationEvent) {
	data := event.Data.(*LinkFailureEventData)

	linkID := fmt.Sprintf("%s->%s", data.Source, data.Dest)
	link, exists := sim.simulatedLinks[linkID]
	if !exists {
		return
	}

	link.SetActive(false)

	// Schedule recovery if duration > 0
	if data.Duration > 0 {
		recoveryEvent := &SimulationEvent{
			ID:        fmt.Sprintf("link-recover-%s", linkID),
			Type:      EventLinkRecover,
			Timestamp: event.Timestamp + data.Duration,
			LinkID:    linkID,
			Data:      data,
		}
		sim.ScheduleEvent(recoveryEvent)
	}
}

// handleLinkRecover handles LINK_RECOVER event
func (sim *EventDrivenSimulator) handleLinkRecover(event *SimulationEvent) {
	data := event.Data.(*LinkFailureEventData)

	linkID := fmt.Sprintf("%s->%s", data.Source, data.Dest)
	link, exists := sim.simulatedLinks[linkID]
	if !exists {
		return
	}

	link.SetActive(true)
}

// handleChaosAction handles CHAOS_ACTION event
func (sim *EventDrivenSimulator) handleChaosAction(event *SimulationEvent) {
	data := event.Data.(*ChaosActionEventData)
	if data.Execute != nil {
		if err := data.Execute(); err != nil {
			fmt.Printf("Chaos action error: %v\n", err)
		}
	}
}

// collectMetrics collects current metrics snapshot
func (sim *EventDrivenSimulator) collectMetrics() {
	currentTime := sim.clock.CurrentTime()

	// System metrics
	sim.trackerMu.RLock()
	deliveryRate := 0.0
	if sim.totalMessagesSent > 0 {
		deliveryRate = float64(sim.totalMessagesDelivered) / float64(sim.totalMessagesSent)
	}
	avgLatency := 0.0
	if sim.totalMessagesDelivered > 0 {
		avgLatency = sim.totalLatency / float64(sim.totalMessagesDelivered)
	}
	sim.trackerMu.RUnlock()

	// Count active/failed nodes
	activeNodes := 0
	failedNodes := 0
	for _, node := range sim.nodes {
		if node.GetState() == nodes.StateActive {
			activeNodes++
		} else if node.GetState() == nodes.StateFailed {
			failedNodes++
		}
	}

	throughput := 0.0
	if sim.metricsInterval > 0 {
		throughput = float64(sim.totalMessagesSent) / currentTime
	}

	sim.metrics.System.RecordSnapshot(currentTime, throughput, deliveryRate, avgLatency,
		sim.totalMessagesSent, activeNodes, failedNodes)

	// Node metrics
	for nodeID, node := range sim.nodes {
		sent, recv, _, queueDepth := node.GetMetrics()
		utilization := float64(queueDepth) / 100.0 // Assuming queue size of 100
		nodeMetrics := sim.metrics.GetOrCreateNodeMetrics(nodeID)
		nodeMetrics.RecordSnapshot(currentTime, sent, recv, queueDepth, utilization)
	}

	// Link metrics
	for linkID, link := range sim.simulatedLinks {
		link.UpdateMetrics(currentTime)
		linkMetrics := link.GetMetrics()
		sent, dropped, lossRate, avgLat, util := linkMetrics.GetStats()

		linkTS := sim.metrics.GetOrCreateLinkMetrics(linkID)
		linkTS.RecordSnapshot(currentTime, util, lossRate, avgLat, sent)
		_ = dropped // Avoid unused variable warning
	}
}

// GetClock returns the simulation clock
func (sim *EventDrivenSimulator) GetClock() *SimulationClock {
	return sim.clock
}

// GetMetrics returns the metrics collector
func (sim *EventDrivenSimulator) GetMetrics() *MetricsCollector {
	return sim.metrics
}

// GetLink returns a simulated link
func (sim *EventDrivenSimulator) GetLink(source, dest string) *SimulatedLink {
	linkID := fmt.Sprintf("%s->%s", source, dest)
	return sim.simulatedLinks[linkID]
}

// GetNode returns a node by ID
func (sim *EventDrivenSimulator) GetNode(nodeID string) (*nodes.Node, error) {
	sim.mu.RLock()
	defer sim.mu.RUnlock()

	node, exists := sim.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}
	return node, nil
}

// GetAllNodes returns all nodes in the network
func (sim *EventDrivenSimulator) GetAllNodes() []*nodes.Node {
	sim.mu.RLock()
	defer sim.mu.RUnlock()

	result := make([]*nodes.Node, 0, len(sim.nodes))
	for _, node := range sim.nodes {
		result = append(result, node)
	}
	return result
}

// GetTopology returns the network topology
func (sim *EventDrivenSimulator) GetTopology() *Topology {
	return sim.topology
}

// PrintStatus prints current simulation status
func (sim *EventDrivenSimulator) PrintStatus() {
	fmt.Printf("\n=== Simulation Status ===\n")
	fmt.Printf("%s\n", sim.clock.String())
	fmt.Printf("Events in queue: %d\n", sim.eventQueue.Len())
	fmt.Printf("Messages: sent=%d, delivered=%d (%.1f%%)\n",
		sim.totalMessagesSent, sim.totalMessagesDelivered,
		float64(sim.totalMessagesDelivered)/float64(sim.totalMessagesSent)*100)

	summary := sim.metrics.System.GetSummary()
	fmt.Printf("Metrics: %s\n", summary.String())
}

// EnableMetrics enables metrics collection at the specified interval
func (sim *EventDrivenSimulator) EnableMetrics(interval float64) {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	sim.metricsInterval = interval
}

// Reset resets the simulator to its initial state
func (sim *EventDrivenSimulator) Reset() {
	sim.mu.Lock()
	defer sim.mu.Unlock()

	// Reset clock
	sim.clock.Reset()

	// Clear event queue
	sim.eventQueue = NewEventQueue()

	// Reset all nodes
	for _, node := range sim.nodes {
		node.SetState(nodes.StateActive)
	}

	// Reactivate all links
	for _, link := range sim.simulatedLinks {
		link.SetActive(true)
	}

	// Clear traffic manager
	sim.trafficMgr = NewTrafficManager()

	// Clear metrics
	sim.metrics.Clear()

	// Clear message tracking
	sim.trackerMu.Lock()
	sim.messageTracker = make(map[string]*MessageTrack)
	sim.totalMessagesSent = 0
	sim.totalMessagesDelivered = 0
	sim.totalLatency = 0
	sim.trackerMu.Unlock()

	sim.running = false
	sim.paused = false
}

// SetSpeed sets the simulation speed multiplier
func (sim *EventDrivenSimulator) SetSpeed(speed float64) {
	sim.clock.SetSpeed(speed)
}

// GetCurrentTime returns the current simulation time
func (sim *EventDrivenSimulator) GetCurrentTime() float64 {
	return sim.clock.CurrentTime()
}

// AddTrafficFlowConfig adds a traffic flow from a config struct
func (sim *EventDrivenSimulator) AddTrafficFlowConfig(cfg TrafficFlowConfig) {
	flowID := fmt.Sprintf("flow-%s-%s-%.2f", cfg.Source, cfg.Dest, cfg.StartTime)
	flow := CreateTrafficFlow(flowID, cfg)
	sim.trafficMgr.AddFlow(flow)
}

// SystemMetricsSnapshot represents a snapshot of system-wide metrics
type SystemMetricsSnapshot struct {
	TotalThroughput        float64
	DeliveryRate           float64
	AverageLatency         float64
	ActiveNodes            int
	FailedNodes            int
	TotalMessagesSent      int64
	TotalMessagesDelivered int64
}

// GetSystemMetrics returns current system-wide metrics
func (sim *EventDrivenSimulator) GetSystemMetrics() SystemMetricsSnapshot {
	sim.trackerMu.RLock()
	deliveryRate := 0.0
	if sim.totalMessagesSent > 0 {
		deliveryRate = float64(sim.totalMessagesDelivered) / float64(sim.totalMessagesSent)
	}
	avgLatency := 0.0
	if sim.totalMessagesDelivered > 0 {
		avgLatency = sim.totalLatency / float64(sim.totalMessagesDelivered)
	}
	totalSent := sim.totalMessagesSent
	totalDelivered := sim.totalMessagesDelivered
	sim.trackerMu.RUnlock()

	// Count active/failed nodes
	activeNodes := 0
	failedNodes := 0
	for _, node := range sim.nodes {
		if node.GetState() == nodes.StateActive {
			activeNodes++
		} else if node.GetState() == nodes.StateFailed {
			failedNodes++
		}
	}

	throughput := 0.0
	currentTime := sim.clock.CurrentTime()
	if currentTime > 0 {
		throughput = float64(totalSent) / currentTime
	}

	return SystemMetricsSnapshot{
		TotalThroughput:        throughput,
		DeliveryRate:           deliveryRate,
		AverageLatency:         avgLatency,
		ActiveNodes:            activeNodes,
		FailedNodes:            failedNodes,
		TotalMessagesSent:      totalSent,
		TotalMessagesDelivered: totalDelivered,
	}
}

// NodeMetricsSnapshot represents a snapshot of node metrics
type NodeMetricsSnapshot struct {
	MessagesSent     int64
	MessagesReceived int64
	QueueDepth       int
}

// GetNodeMetrics returns metrics for a specific node
func (sim *EventDrivenSimulator) GetNodeMetrics(nodeID string) *NodeMetricsSnapshot {
	node, exists := sim.nodes[nodeID]
	if !exists {
		return nil
	}

	sent, recv, _, queueDepth := node.GetMetrics()
	return &NodeMetricsSnapshot{
		MessagesSent:     sent,
		MessagesReceived: recv,
		QueueDepth:       queueDepth,
	}
}

// LinkMetricsSnapshot represents a snapshot of link metrics
type LinkMetricsSnapshot struct {
	Utilization    float64
	PacketLossRate float64
	BytesSent      int64
}

// GetLinkMetrics returns metrics for a specific link
func (sim *EventDrivenSimulator) GetLinkMetrics(source, dest string) *LinkMetricsSnapshot {
	linkID := fmt.Sprintf("%s->%s", source, dest)
	link, exists := sim.simulatedLinks[linkID]
	if !exists {
		return nil
	}

	metrics := link.GetMetrics()
	sent, _, lossRate, _, util := metrics.GetStats()

	return &LinkMetricsSnapshot{
		Utilization:    util,
		PacketLossRate: lossRate,
		BytesSent:      sent,
	}
}

// EventStatsSnapshot represents a snapshot of event statistics
type EventStatsSnapshot struct {
	TotalEvents         int64
	MessageSentEvents   int64
	MessageArriveEvents int64
}

// GetEventStats returns event statistics
func (sim *EventDrivenSimulator) GetEventStats() EventStatsSnapshot {
	return EventStatsSnapshot{
		TotalEvents:         sim.eventStats.TotalEvents,
		MessageSentEvents:   sim.eventStats.EventsByType[EventMessageSend],
		MessageArriveEvents: sim.eventStats.EventsByType[EventMessageArrive],
	}
}

// TimeSeriesSample represents a single time series data point
type TimeSeriesSample struct {
	Timestamp      float64
	Throughput     float64
	AverageLatency float64
	DeliveryRate   float64
}

// TimeSeriesData wraps time series metrics data
type TimeSeriesData struct {
	metrics *MetricsCollector
}

// GetRecentSamples returns recent time series samples
func (tsd *TimeSeriesData) GetRecentSamples(n int) []TimeSeriesSample {
	if tsd.metrics == nil {
		return nil
	}

	throughput := tsd.metrics.System.Throughput.GetRecent(n)
	latency := tsd.metrics.System.AverageLatency.GetRecent(n)
	delivery := tsd.metrics.System.DeliveryRate.GetRecent(n)

	// Combine into samples
	samples := make([]TimeSeriesSample, 0, len(throughput))
	for i := 0; i < len(throughput); i++ {
		sample := TimeSeriesSample{
			Timestamp:  throughput[i].Timestamp,
			Throughput: throughput[i].Value,
		}
		if i < len(latency) {
			sample.AverageLatency = latency[i].Value
		}
		if i < len(delivery) {
			sample.DeliveryRate = delivery[i].Value
		}
		samples = append(samples, sample)
	}

	return samples
}

// GetTimeSeriesData returns time series data wrapper
func (sim *EventDrivenSimulator) GetTimeSeriesData() *TimeSeriesData {
	return &TimeSeriesData{metrics: sim.metrics}
}

// Step advances the simulation by processing one event or a small time step
func (sim *EventDrivenSimulator) Step() {
	sim.mu.Lock()
	if !sim.running {
		sim.running = true
		// Generate traffic events if this is the first step
		trafficEvents := sim.trafficMgr.GenerateAllEvents()
		for _, event := range trafficEvents {
			sim.eventQueue.Enqueue(event)
		}
	}
	sim.mu.Unlock()

	// Process next event if available
	nextEvent := sim.eventQueue.Peek()
	if nextEvent == nil {
		// No events, just advance time slightly
		sim.clock.SetTime(sim.clock.CurrentTime() + 0.1)
		return
	}

	// Process the event
	event := sim.eventQueue.Dequeue()
	sim.clock.SetTime(event.Timestamp)

	// Check if we should collect metrics
	if sim.clock.CurrentTime() >= sim.lastMetricsTime+sim.metricsInterval {
		sim.collectMetrics()
		sim.lastMetricsTime = sim.clock.CurrentTime()
	}

	// Process the event
	sim.processEvent(event)
	sim.eventStats.RecordEvent(event.Type, 0)
}
