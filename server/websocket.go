package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/saintparish4/chaos/chaos"
	"github.com/saintparish4/chaos/simulator"
)

// WebSocketServer manages WebSocket connections for real-time simulation updates
type WebSocketServer struct {
	Simulator      *simulator.EventDrivenSimulator
	ChaosInjector  *chaos.FailureInjector
	Resilience     *chaos.ResilienceManager
	clients        map[*websocket.Conn]bool
	broadcast      chan interface{}
	register       chan *websocket.Conn
	unregister     chan *websocket.Conn
	mu             sync.RWMutex
	updateInterval time.Duration
	running        bool
	speed          float64
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(sim *simulator.EventDrivenSimulator, injector *chaos.FailureInjector, resilience *chaos.ResilienceManager) *WebSocketServer {
	return &WebSocketServer{
		Simulator:      sim,
		ChaosInjector:  injector,
		Resilience:     resilience,
		clients:        make(map[*websocket.Conn]bool),
		broadcast:      make(chan interface{}, 100),
		register:       make(chan *websocket.Conn),
		unregister:     make(chan *websocket.Conn),
		updateInterval: 100 * time.Millisecond, // 10 updates per second
		running:        false,
		speed:          1.0,
	}
}

// Start begins the WebSocket server
func (ws *WebSocketServer) Start() {
	go ws.run()
	go ws.streamUpdates()
	go ws.runSimulation()
	ws.running = true
	log.Println("[WebSocket] Server started")
}

// runSimulation advances the simulation when running
func (ws *WebSocketServer) runSimulation() {
	ticker := time.NewTicker(10 * time.Millisecond) // Base tick rate
	defer ticker.Stop()

	for range ticker.C {
		if !ws.running {
			continue
		}

		// Step simulation based on speed
		steps := int(ws.speed)
		if steps < 1 {
			steps = 1
		}

		for i := 0; i < steps; i++ {
			ws.Simulator.Step()
		}
	}
}

// run handles client registration/unregistration and broadcasting
func (ws *WebSocketServer) run() {
	for {
		select {
		case client := <-ws.register:
			ws.mu.Lock()
			ws.clients[client] = true
			ws.mu.Unlock()
			log.Printf("[WebSocket] Client connected. Total clients: %d\n", len(ws.clients))

			// Send initial state to new client
			ws.sendInitialState(client)

		case client := <-ws.unregister:
			ws.mu.Lock()
			if _, ok := ws.clients[client]; ok {
				delete(ws.clients, client)
				client.Close()
			}
			ws.mu.Unlock()
			log.Printf("[WebSocket] Client disconnected. Total clients: %d\n", len(ws.clients))

		case message := <-ws.broadcast:
			ws.mu.RLock()
			for client := range ws.clients {
				err := client.WriteJSON(message)
				if err != nil {
					log.Printf("[WebSocket] Error sending to client: %v\n", err)
					client.Close()
					delete(ws.clients, client)
				}
			}
			ws.mu.RUnlock()
		}
	}
}

// streamUpdates periodically sends simulation state to all clients
func (ws *WebSocketServer) streamUpdates() {
	ticker := time.NewTicker(ws.updateInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !ws.running {
			continue
		}

		update := ws.buildStateUpdate()
		ws.broadcast <- update
	}
}

// HandleWebSocket handles WebSocket connections
func (ws *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v\n", err)
		return
	}

	ws.register <- conn

	// Read messages from client (for commands)
	go ws.handleClientMessages(conn)
}

// handleClientMessages processes messages from clients
func (ws *WebSocketServer) handleClientMessages(conn *websocket.Conn) {
	defer func() {
		ws.unregister <- conn
	}()

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Error reading message: %v\n", err)
			}
			break
		}

		// Handle commands
		ws.handleCommand(msg)
	}
}

// handleCommand processes commands from clients
func (ws *WebSocketServer) handleCommand(cmd map[string]interface{}) {
	cmdType, ok := cmd["type"].(string)
	if !ok {
		return
	}

	switch cmdType {
	case "START":
		ws.running = true
		ws.broadcast <- map[string]interface{}{
			"type":    "STATUS",
			"running": true,
		}

	case "PAUSE":
		ws.running = false
		ws.broadcast <- map[string]interface{}{
			"type":    "STATUS",
			"running": false,
		}

	case "RESET":
		// Reset simulation
		ws.Simulator.Reset()
		if ws.ChaosInjector != nil {
			ws.ChaosInjector = chaos.NewFailureInjector(ws.Simulator)
		}
		ws.broadcast <- map[string]interface{}{
			"type": "RESET",
		}

	case "SET_SPEED":
		if speed, ok := cmd["speed"].(float64); ok {
			ws.speed = speed
			ws.Simulator.SetSpeed(speed)
		}

	case "INJECT_FAILURE":
		ws.handleFailureInjection(cmd)

	case "REQUEST_STATE":
		// Send current state
		update := ws.buildStateUpdate()
		ws.broadcast <- update
	}
}

// handleFailureInjection processes failure injection commands
func (ws *WebSocketServer) handleFailureInjection(cmd map[string]interface{}) {
	if ws.ChaosInjector == nil {
		return
	}

	failureType, _ := cmd["failureType"].(string)
	target, _ := cmd["target"].(string)
	duration := 5.0 // Default duration

	if d, ok := cmd["duration"].(float64); ok {
		duration = d
	}

	switch failureType {
	case "NODE_FAILURE":
		ws.ChaosInjector.InjectNodeFailure(target, duration)
	case "LINK_FAILURE":
		if dest, ok := cmd["dest"].(string); ok {
			ws.ChaosInjector.InjectLinkFailure(target, dest, duration)
		}
	case "SLOW_LINK":
		if dest, ok := cmd["dest"].(string); ok {
			multiplier := 10.0
			if m, ok := cmd["multiplier"].(float64); ok {
				multiplier = m
			}
			ws.ChaosInjector.InjectSlowLink(target, dest, multiplier, duration)
		}
	}

	ws.broadcast <- map[string]interface{}{
		"type":    "FAILURE_INJECTED",
		"failure": failureType,
		"target":  target,
	}
}

// buildStateUpdate creates a complete state update
func (ws *WebSocketServer) buildStateUpdate() map[string]interface{} {
	nodes := ws.buildNodeStates()
	links := ws.buildLinkStates()
	messages := ws.buildActiveMessages()
	metrics := ws.buildMetrics()
	events := ws.buildRecentEvents()

	return map[string]interface{}{
		"type":      "STATE_UPDATE",
		"timestamp": time.Now().Unix(),
		"simTime":   ws.Simulator.GetCurrentTime(),
		"nodes":     nodes,
		"links":     links,
		"messages":  messages,
		"metrics":   metrics,
		"events":    events,
	}
}

// buildNodeStates builds node state information
func (ws *WebSocketServer) buildNodeStates() []map[string]interface{} {
	nodes := make([]map[string]interface{}, 0)

	allNodes := ws.Simulator.GetAllNodes()
	for _, node := range allNodes {
		nodeState := map[string]interface{}{
			"id":    node.ID,
			"type":  node.Type,
			"state": string(node.GetState()),
		}

		// Add metrics if available
		if metrics := ws.Simulator.GetNodeMetrics(node.ID); metrics != nil {
			nodeState["messagesSent"] = metrics.MessagesSent
			nodeState["messagesReceived"] = metrics.MessagesReceived
			nodeState["queueDepth"] = metrics.QueueDepth
		}

		nodes = append(nodes, nodeState)
	}

	return nodes
}

// buildLinkStates builds link state information
func (ws *WebSocketServer) buildLinkStates() []map[string]interface{} {
	links := make([]map[string]interface{}, 0)

	topology := ws.Simulator.GetTopology()
	for source, dests := range topology.Links {
		for dest, link := range dests {
			linkState := map[string]interface{}{
				"source": source,
				"dest":   dest,
				"active": link.Active,
			}

			// Add link metrics if available
			if metrics := ws.Simulator.GetLinkMetrics(source, dest); metrics != nil {
				linkState["utilization"] = metrics.Utilization
				linkState["packetLoss"] = metrics.PacketLossRate
				linkState["bytesSent"] = metrics.BytesSent
			}

			links = append(links, linkState)
		}
	}

	return links
}

// buildActiveMessages builds information about messages in flight
func (ws *WebSocketServer) buildActiveMessages() []map[string]interface{} {
	messages := make([]map[string]interface{}, 0)

	// Get active messages from simulator
	// This would need to be added to the simulator
	// For now, return empty array

	return messages
}

// buildMetrics builds system-wide metrics
func (ws *WebSocketServer) buildMetrics() map[string]interface{} {
	sysMetrics := ws.Simulator.GetSystemMetrics()

	return map[string]interface{}{
		"throughput":        sysMetrics.TotalThroughput,
		"deliveryRate":      sysMetrics.DeliveryRate,
		"averageLatency":    sysMetrics.AverageLatency,
		"activeNodes":       sysMetrics.ActiveNodes,
		"failedNodes":       sysMetrics.FailedNodes,
		"totalMessages":     sysMetrics.TotalMessagesSent,
		"deliveredMessages": sysMetrics.TotalMessagesDelivered,
	}
}

// buildRecentEvents builds recent simulation events
func (ws *WebSocketServer) buildRecentEvents() []map[string]interface{} {
	events := make([]map[string]interface{}, 0)

	// Get recent events from event queue
	stats := ws.Simulator.GetEventStats()

	events = append(events, map[string]interface{}{
		"type":    "EVENT_STATS",
		"total":   stats.TotalEvents,
		"sent":    stats.MessageSentEvents,
		"arrived": stats.MessageArriveEvents,
	})

	return events
}

// sendInitialState sends the complete initial state to a new client
func (ws *WebSocketServer) sendInitialState(client *websocket.Conn) {
	topology := ws.Simulator.GetTopology()

	initialState := map[string]interface{}{
		"type":     "INITIAL_STATE",
		"topology": ws.buildTopologyInfo(topology),
		"state":    ws.buildStateUpdate(),
	}

	client.WriteJSON(initialState)
}

// buildTopologyInfo builds topology configuration
func (ws *WebSocketServer) buildTopologyInfo(topology *simulator.Topology) map[string]interface{} {
	nodes := make([]map[string]interface{}, 0)
	for _, node := range topology.Config.Nodes {
		nodes = append(nodes, map[string]interface{}{
			"id":   node.ID,
			"type": node.Type,
		})
	}

	links := make([]map[string]interface{}, 0)
	for _, link := range topology.Config.Links {
		links = append(links, map[string]interface{}{
			"source":    link.Source,
			"dest":      link.Dest,
			"bandwidth": link.Bandwidth,
			"latency":   link.Latency,
		})
	}

	return map[string]interface{}{
		"nodes": nodes,
		"links": links,
	}
}

// GetMetricsHistory returns historical metrics for charting
func (ws *WebSocketServer) GetMetricsHistory() []map[string]interface{} {
	history := make([]map[string]interface{}, 0)

	// Get time series data from simulator
	if tsData := ws.Simulator.GetTimeSeriesData(); tsData != nil {
		samples := tsData.GetRecentSamples(100) // Last 100 samples

		for _, sample := range samples {
			history = append(history, map[string]interface{}{
				"time":       sample.Timestamp,
				"throughput": sample.Throughput,
				"latency":    sample.AverageLatency,
				"delivery":   sample.DeliveryRate,
			})
		}
	}

	return history
}

// HTTPHandler provides HTTP endpoints
type HTTPHandler struct {
	Server *WebSocketServer
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(server *WebSocketServer) *HTTPHandler {
	return &HTTPHandler{Server: server}
}

// ServeHTTP handles HTTP requests
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.URL.Path {
	case "/ws":
		h.Server.HandleWebSocket(w, r)

	case "/api/state":
		h.handleGetState(w, r)

	case "/api/metrics":
		h.handleGetMetrics(w, r)

	case "/api/topology":
		h.handleGetTopology(w, r)

	case "/api/control":
		h.handleControl(w, r)

	case "/api/export":
		h.handleExport(w, r)

	default:
		http.NotFound(w, r)
	}
}

func (h *HTTPHandler) handleGetState(w http.ResponseWriter, r *http.Request) {
	state := h.Server.buildStateUpdate()
	json.NewEncoder(w).Encode(state)
}

func (h *HTTPHandler) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.Server.GetMetricsHistory()
	json.NewEncoder(w).Encode(metrics)
}

func (h *HTTPHandler) handleGetTopology(w http.ResponseWriter, r *http.Request) {
	topology := h.Server.Simulator.GetTopology()
	topoInfo := h.Server.buildTopologyInfo(topology)
	json.NewEncoder(w).Encode(topoInfo)
}

func (h *HTTPHandler) handleControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cmd map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.Server.handleCommand(cmd)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
}

func (h *HTTPHandler) handleExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")

	if format == "csv" {
		h.exportCSV(w)
	} else {
		h.exportJSON(w)
	}
}

func (h *HTTPHandler) exportCSV(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=simulation_results.csv")

	metrics := h.Server.GetMetricsHistory()

	fmt.Fprintf(w, "Time,Throughput,Latency,DeliveryRate\n")
	for _, m := range metrics {
		fmt.Fprintf(w, "%v,%v,%v,%v\n",
			m["time"], m["throughput"], m["latency"], m["delivery"])
	}
}

func (h *HTTPHandler) exportJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=simulation_results.json")

	data := map[string]interface{}{
		"metrics":  h.Server.GetMetricsHistory(),
		"topology": h.Server.buildTopologyInfo(h.Server.Simulator.GetTopology()),
		"state":    h.Server.buildStateUpdate(),
	}

	json.NewEncoder(w).Encode(data)
}
