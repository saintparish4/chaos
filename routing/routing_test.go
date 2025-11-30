package routing

import (
	"testing"

	"github.com/saintparish4/chaos/simulator"
)

func TestBGPProtocol(t *testing.T) {
	config := simulator.GenerateRingTopology(5)
	topo := buildTestTopo(config)

	bgp := NewBGPProtocol(65001)
	bgp.Initialize("node-0", topo)

	if bgp.NodeID != "node-0" {
		t.Errorf("Expected NodeID node-0, got %s", bgp.NodeID)
	}

	if bgp.AS != 65001 {
		t.Errorf("Expected AS 65001, got %d", bgp.AS)
	}

	bgp.AnnounceRoute("dest-1", 1)
	table := bgp.GetRoutingTable()
	if _, exists := table["dest-1"]; !exists {
		t.Error("Route announcement failed")
	}

	update := &RoutingUpdate{
		SourceNode: "node-1",
		Protocol:   "BGP",
		Routes: []RouteEntry{
			{Destination: "dest-2", Metric: 2, Path: []string{"node-1", "dest-2"}},
		},
	}

	bgp.ProcessUpdate(update)
	table = bgp.GetRoutingTable()
	if _, exists := table["dest-2"]; !exists {
		t.Error("Failed to process BGP update")
	}

	loopUpdate := &RoutingUpdate{
		SourceNode: "node-1",
		Protocol:   "BGP",
		Routes: []RouteEntry{
			{Destination: "dest-3", Metric: 3, Path: []string{"node-1", "node-0", "dest-3"}},
		},
	}

	beforeCount := len(bgp.GetRoutingTable())
	bgp.ProcessUpdate(loopUpdate)
	afterCount := len(bgp.GetRoutingTable())

	if afterCount > beforeCount {
		t.Error("Loop prevention failed - accepted route with loop")
	}
}

func TestOSPFProtocol(t *testing.T) {
	config := simulator.GenerateMeshTopology(4)
	topo := buildTestTopo(config)

	ospf := NewOSPFProtocol(0)
	ospf.Initialize("node-0", topo)

	if ospf.NodeID != "node-0" {
		t.Errorf("Expected NodeID node-0, got %s", ospf.NodeID)
	}

	if len(ospf.LinkStateDB) == 0 {
		t.Error("LSA not generated")
	}

	lsa, exists := ospf.LinkStateDB["node-0"]
	if !exists {
		t.Error("Own LSA not in database")
	}

	if lsa.OriginNode != "node-0" {
		t.Errorf("Expected origin node-0, got %s", lsa.OriginNode)
	}

	table := ospf.GetRoutingTable()
	if len(table) == 0 {
		t.Error("SPF calculation produced no routes")
	}

	for dest := range table {
		nextHop, err := ospf.GetNextHop(dest)
		if err != nil {
			t.Errorf("Failed to get next hop for %s: %v", dest, err)
		}
		if nextHop == "" {
			t.Errorf("Empty next hop for %s", dest)
		}
	}
}

func TestRIPProtocol(t *testing.T) {
	config := simulator.GenerateRingTopology(6)
	topo := buildTestTopo(config)

	rip := NewRIPProtocol()
	rip.Initialize("node-0", topo)

	if rip.MaxHops != RIPMaxHops {
		t.Errorf("Expected MaxHops %d, got %d", RIPMaxHops, rip.MaxHops)
	}

	table := rip.GetRoutingTable()
	for _, neighbor := range rip.Neighbors {
		if entry, exists := table[neighbor]; !exists || entry.Metric != 1 {
			t.Errorf("Direct neighbor %s not in table with metric 1", neighbor)
		}
	}

	update := &RoutingUpdate{
		SourceNode: "node-1",
		Protocol:   "RIP",
		Routes: []RouteEntry{
			{Destination: "node-3", Metric: 2, Path: []string{"node-1", "node-2", "node-3"}},
		},
	}

	rip.ProcessUpdate(update)
	table = rip.GetRoutingTable()

	if entry, exists := table["node-3"]; !exists {
		t.Error("Failed to learn route from RIP update")
	} else if entry.Metric != 3 {
		t.Errorf("Expected metric 3, got %d", entry.Metric)
	}

	infinityUpdate := &RoutingUpdate{
		SourceNode: "node-1",
		Protocol:   "RIP",
		Routes: []RouteEntry{
			{Destination: "node-4", Metric: RIPInfinity, Path: []string{}},
		},
	}

	rip.ProcessUpdate(infinityUpdate)
	table = rip.GetRoutingTable()

	if entry, exists := table["node-4"]; exists && entry.Metric < RIPInfinity {
		t.Error("Should not add route with infinite metric")
	}

	routes := rip.prepareUpdateForNeighbor("node-1")
	for _, route := range routes {
		if route.NextHop == "node-1" && route.Metric != RIPInfinity {
			t.Errorf("Split horizon violated for route %s", route.Destination)
		}
	}
}

func TestProtocolComparison(t *testing.T) {
	config := simulator.GenerateMeshTopology(5)
	topo := buildTestTopo(config)

	mgr := NewProtocolManager(topo)

	bgp := NewBGPProtocol(65001)
	ospf := NewOSPFProtocol(0)
	rip := NewRIPProtocol()

	mgr.RegisterProtocol("BGP", bgp)
	mgr.RegisterProtocol("OSPF", ospf)
	mgr.RegisterProtocol("RIP", rip)

	mgr.InitializeAllProtocols("node-0")

	results := mgr.CompareProtocols()

	if len(results) != 3 {
		t.Errorf("Expected 3 protocols, got %d", len(results))
	}

	for name, metrics := range results {
		if metrics == nil {
			t.Errorf("Nil metrics for protocol %s", name)
		}
		if metrics.RoutingTableSize == 0 {
			t.Errorf("Empty routing table for protocol %s", name)
		}
	}
}

func TestRoutingProtocolBase(t *testing.T) {
	base := NewRoutingProtocolBase("TEST")

	if base.GetName() != "TEST" {
		t.Errorf("Expected name TEST, got %s", base.GetName())
	}

	base.UpdateRoute("dest-1", "next-1", 5, []string{"node-0", "next-1", "dest-1"})

	table := base.GetRoutingTable()
	if len(table) != 1 {
		t.Errorf("Expected 1 route, got %d", len(table))
	}

	entry, exists := table["dest-1"]
	if !exists {
		t.Fatal("Route not added")
	}

	if entry.NextHop != "next-1" {
		t.Errorf("Expected next hop next-1, got %s", entry.NextHop)
	}

	if entry.Metric != 5 {
		t.Errorf("Expected metric 5, got %d", entry.Metric)
	}

	base.InvalidateRoute("dest-1")
	table = base.GetRoutingTable()
	entry = table["dest-1"]
	if entry.Valid {
		t.Error("Route should be invalid")
	}
}

func buildTestTopo(config *simulator.TopologyConfig) *simulator.Topology {
	topo := &simulator.Topology{
		Config:  *config,
		AdjList: make(map[string][]string),
		Links:   make(map[string]map[string]*simulator.Link),
		Nodes:   make(map[string]*simulator.NodeConfig),
	}

	for i := range config.Nodes {
		node := &config.Nodes[i]
		topo.Nodes[node.ID] = node
	}

	for _, linkCfg := range config.Links {
		topo.AdjList[linkCfg.Source] = append(topo.AdjList[linkCfg.Source], linkCfg.Dest)
		if topo.Links[linkCfg.Source] == nil {
			topo.Links[linkCfg.Source] = make(map[string]*simulator.Link)
		}
		topo.Links[linkCfg.Source][linkCfg.Dest] = &simulator.Link{
			Source:    linkCfg.Source,
			Dest:      linkCfg.Dest,
			Bandwidth: 1000000,
			Latency:   1,
		}
	}

	return topo
}
