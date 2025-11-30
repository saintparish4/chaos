package simulator

import (
	"fmt"
	"sync"
)

// TimeSeriesMetric represents a single time-series data point
type TimeSeriesMetric struct {
	Timestamp float64
	Value     float64
}

// RingBuffer implements a circular buffer for time-series data
type RingBuffer struct {
	data     []TimeSeriesMetric
	capacity int
	head     int
	size     int
	mu       sync.RWMutex
}

// NewRingBuffer creates a new ring buffer
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		data:     make([]TimeSeriesMetric, capacity),
		capacity: capacity,
		head:     0,
		size:     0,
	}
}

// Add adds a new data point to the buffer
func (rb *RingBuffer) Add(timestamp, value float64) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	rb.data[rb.head] = TimeSeriesMetric{
		Timestamp: timestamp,
		Value:     value,
	}
	
	rb.head = (rb.head + 1) % rb.capacity
	if rb.size < rb.capacity {
		rb.size++
	}
}

// GetAll returns all data points in chronological order
func (rb *RingBuffer) GetAll() []TimeSeriesMetric {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if rb.size == 0 {
		return []TimeSeriesMetric{}
	}
	
	result := make([]TimeSeriesMetric, rb.size)
	
	// If buffer is not full, data is from 0 to size
	if rb.size < rb.capacity {
		copy(result, rb.data[:rb.size])
		return result
	}
	
	// If buffer is full, data wraps around
	idx := 0
	for i := rb.head; i < rb.capacity; i++ {
		result[idx] = rb.data[i]
		idx++
	}
	for i := 0; i < rb.head; i++ {
		result[idx] = rb.data[i]
		idx++
	}
	
	return result
}

// GetRecent returns the last N data points
func (rb *RingBuffer) GetRecent(n int) []TimeSeriesMetric {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if n > rb.size {
		n = rb.size
	}
	
	if n == 0 {
		return []TimeSeriesMetric{}
	}
	
	result := make([]TimeSeriesMetric, n)
	
	// Start from most recent
	for i := 0; i < n; i++ {
		idx := (rb.head - 1 - i + rb.capacity) % rb.capacity
		result[n-1-i] = rb.data[idx]
	}
	
	return result
}

// GetAverage returns the average value over all data points
func (rb *RingBuffer) GetAverage() float64 {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if rb.size == 0 {
		return 0.0
	}
	
	sum := 0.0
	for i := 0; i < rb.size; i++ {
		sum += rb.data[i].Value
	}
	
	return sum / float64(rb.size)
}

// GetMax returns the maximum value
func (rb *RingBuffer) GetMax() float64 {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if rb.size == 0 {
		return 0.0
	}
	
	max := rb.data[0].Value
	for i := 1; i < rb.size; i++ {
		if rb.data[i].Value > max {
			max = rb.data[i].Value
		}
	}
	
	return max
}

// GetMin returns the minimum value
func (rb *RingBuffer) GetMin() float64 {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if rb.size == 0 {
		return 0.0
	}
	
	min := rb.data[0].Value
	for i := 1; i < rb.size; i++ {
		if rb.data[i].Value < min {
			min = rb.data[i].Value
		}
	}
	
	return min
}

// Size returns the current number of data points
func (rb *RingBuffer) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// Clear clears all data
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.head = 0
	rb.size = 0
}

// SystemMetrics tracks system-wide metrics over time
type SystemMetrics struct {
	Throughput      *RingBuffer // messages per second
	DeliveryRate    *RingBuffer // percentage of messages delivered
	AverageLatency  *RingBuffer // average message latency
	TotalMessages   *RingBuffer // cumulative messages sent
	ActiveNodes     *RingBuffer // number of active nodes
	FailedNodes     *RingBuffer // number of failed nodes
	
	mu sync.RWMutex
}

// NewSystemMetrics creates new system metrics with ring buffers
func NewSystemMetrics(capacity int) *SystemMetrics {
	return &SystemMetrics{
		Throughput:     NewRingBuffer(capacity),
		DeliveryRate:   NewRingBuffer(capacity),
		AverageLatency: NewRingBuffer(capacity),
		TotalMessages:  NewRingBuffer(capacity),
		ActiveNodes:    NewRingBuffer(capacity),
		FailedNodes:    NewRingBuffer(capacity),
	}
}

// RecordSnapshot records a snapshot of system metrics
func (sm *SystemMetrics) RecordSnapshot(timestamp float64, throughput, deliveryRate, avgLatency float64, 
                                        totalMessages int64, activeNodes, failedNodes int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sm.Throughput.Add(timestamp, throughput)
	sm.DeliveryRate.Add(timestamp, deliveryRate)
	sm.AverageLatency.Add(timestamp, avgLatency)
	sm.TotalMessages.Add(timestamp, float64(totalMessages))
	sm.ActiveNodes.Add(timestamp, float64(activeNodes))
	sm.FailedNodes.Add(timestamp, float64(failedNodes))
}

// GetSummary returns a summary of current metrics
func (sm *SystemMetrics) GetSummary() MetricsSummary {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	return MetricsSummary{
		AvgThroughput:   sm.Throughput.GetAverage(),
		MaxThroughput:   sm.Throughput.GetMax(),
		AvgDeliveryRate: sm.DeliveryRate.GetAverage(),
		AvgLatency:      sm.AverageLatency.GetAverage(),
		MaxLatency:      sm.AverageLatency.GetMax(),
		MinLatency:      sm.AverageLatency.GetMin(),
		DataPoints:      sm.Throughput.Size(),
	}
}

// MetricsSummary contains summary statistics
type MetricsSummary struct {
	AvgThroughput   float64
	MaxThroughput   float64
	AvgDeliveryRate float64
	AvgLatency      float64
	MaxLatency      float64
	MinLatency      float64
	DataPoints      int
}

// String returns a string representation
func (ms MetricsSummary) String() string {
	return fmt.Sprintf("Throughput: %.2f msg/s (max: %.2f), Delivery: %.1f%%, Latency: %.3fs (%.3f-%.3fs), Samples: %d",
		ms.AvgThroughput, ms.MaxThroughput, ms.AvgDeliveryRate*100, 
		ms.AvgLatency, ms.MinLatency, ms.MaxLatency, ms.DataPoints)
}

// NodeMetricsTimeSeries tracks per-node metrics over time
type NodeMetricsTimeSeries struct {
	NodeID         string
	MessagesSent   *RingBuffer
	MessagesRecv   *RingBuffer
	QueueDepth     *RingBuffer
	Utilization    *RingBuffer
	mu             sync.RWMutex
}

// NewNodeMetricsTimeSeries creates new node metrics time series
func NewNodeMetricsTimeSeries(nodeID string, capacity int) *NodeMetricsTimeSeries {
	return &NodeMetricsTimeSeries{
		NodeID:       nodeID,
		MessagesSent: NewRingBuffer(capacity),
		MessagesRecv: NewRingBuffer(capacity),
		QueueDepth:   NewRingBuffer(capacity),
		Utilization:  NewRingBuffer(capacity),
	}
}

// RecordSnapshot records a snapshot of node metrics
func (nmt *NodeMetricsTimeSeries) RecordSnapshot(timestamp float64, sent, recv int64, queueDepth int, utilization float64) {
	nmt.mu.Lock()
	defer nmt.mu.Unlock()
	
	nmt.MessagesSent.Add(timestamp, float64(sent))
	nmt.MessagesRecv.Add(timestamp, float64(recv))
	nmt.QueueDepth.Add(timestamp, float64(queueDepth))
	nmt.Utilization.Add(timestamp, utilization)
}

// LinkMetricsTimeSeries tracks per-link metrics over time
type LinkMetricsTimeSeries struct {
	LinkID       string
	Utilization  *RingBuffer
	PacketLoss   *RingBuffer
	Latency      *RingBuffer
	BytesSent    *RingBuffer
	mu           sync.RWMutex
}

// NewLinkMetricsTimeSeries creates new link metrics time series
func NewLinkMetricsTimeSeries(linkID string, capacity int) *LinkMetricsTimeSeries {
	return &LinkMetricsTimeSeries{
		LinkID:      linkID,
		Utilization: NewRingBuffer(capacity),
		PacketLoss:  NewRingBuffer(capacity),
		Latency:     NewRingBuffer(capacity),
		BytesSent:   NewRingBuffer(capacity),
	}
}

// RecordSnapshot records a snapshot of link metrics
func (lmt *LinkMetricsTimeSeries) RecordSnapshot(timestamp, utilization, packetLoss, latency float64, bytesSent int64) {
	lmt.mu.Lock()
	defer lmt.mu.Unlock()
	
	lmt.Utilization.Add(timestamp, utilization)
	lmt.PacketLoss.Add(timestamp, packetLoss)
	lmt.Latency.Add(timestamp, latency)
	lmt.BytesSent.Add(timestamp, float64(bytesSent))
}

// MetricsCollector aggregates all metrics
type MetricsCollector struct {
	System      *SystemMetrics
	Nodes       map[string]*NodeMetricsTimeSeries
	Links       map[string]*LinkMetricsTimeSeries
	capacity    int
	mu          sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(capacity int) *MetricsCollector {
	return &MetricsCollector{
		System:   NewSystemMetrics(capacity),
		Nodes:    make(map[string]*NodeMetricsTimeSeries),
		Links:    make(map[string]*LinkMetricsTimeSeries),
		capacity: capacity,
	}
}

// GetOrCreateNodeMetrics gets or creates node metrics
func (mc *MetricsCollector) GetOrCreateNodeMetrics(nodeID string) *NodeMetricsTimeSeries {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if metrics, exists := mc.Nodes[nodeID]; exists {
		return metrics
	}
	
	metrics := NewNodeMetricsTimeSeries(nodeID, mc.capacity)
	mc.Nodes[nodeID] = metrics
	return metrics
}

// GetOrCreateLinkMetrics gets or creates link metrics
func (mc *MetricsCollector) GetOrCreateLinkMetrics(linkID string) *LinkMetricsTimeSeries {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if metrics, exists := mc.Links[linkID]; exists {
		return metrics
	}
	
	metrics := NewLinkMetricsTimeSeries(linkID, mc.capacity)
	mc.Links[linkID] = metrics
	return metrics
}

// GetNodeMetrics returns node metrics
func (mc *MetricsCollector) GetNodeMetrics(nodeID string) *NodeMetricsTimeSeries {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.Nodes[nodeID]
}

// GetLinkMetrics returns link metrics
func (mc *MetricsCollector) GetLinkMetrics(linkID string) *LinkMetricsTimeSeries {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.Links[linkID]
}

// Clear clears all metrics
func (mc *MetricsCollector) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.System = NewSystemMetrics(mc.capacity)
	mc.Nodes = make(map[string]*NodeMetricsTimeSeries)
	mc.Links = make(map[string]*LinkMetricsTimeSeries)
}