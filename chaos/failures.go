package chaos

import (
	"fmt"
	"math/rand"

	"github.com/saintparish4/chaos/nodes"
	"github.com/saintparish4/chaos/simulator"
)

// FailureType represents different types of failures
type FailureType string

const (
	FailureNodeKill         FailureType = "NODE_KILL"
	FailureNodeDegrade      FailureType = "NODE_DEGRADE"
	FailureLinkSever        FailureType = "LINK_SEVER"
	FailureNetworkPartition FailureType = "NETWORK_PARTITION"
	FailureSlowLink         FailureType = "SLOW_LINK"
	FailureCongestion       FailureType = "CONGESTION"
	FailureByzantine        FailureType = "BYZANTINE"
)

// FailureInjector handles injecting failures into the network
type FailureInjector struct {
	sim *simulator.EventDrivenSimulator
}

// NewFailureInjector creates a new failure injector
func NewFailureInjector(sim *simulator.EventDrivenSimulator) *FailureInjector {
	return &FailureInjector{
		sim: sim,
	}
}

// InjectNodeFailure kills a node completely
func (fi *FailureInjector) InjectNodeFailure(nodeID string, duration float64) error {
	node, err := fi.sim.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to inject node failure: %w", err)
	}

	// Set node to failed state
	node.SetState(nodes.StateFailed)

	// Schedule recovery if duration > 0
	if duration > 0 {
		currentTime := fi.sim.GetClock().CurrentTime()
		recoveryEvent := &simulator.SimulationEvent{
			ID:        fmt.Sprintf("recover-node-%s", nodeID),
			Type:      simulator.EventNodeRecover,
			Timestamp: currentTime + duration,
			NodeID:    nodeID,
		}
		fi.sim.ScheduleEvent(recoveryEvent)
	}

	return nil
}

// InjectPartialNodeFailure degrades a node's capacity
func (fi *FailureInjector) InjectPartialNodeFailure(nodeID string, degradationPercent float64, duration float64) error {
	node, err := fi.sim.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to inject partial failure: %w", err)
	}

	// Set node to degraded state
	node.SetState(nodes.StateDegraded)

	// TODO: Could store original capacity and reduce it
	// For now, just mark as degraded

	// Schedule recovery if duration > 0
	if duration > 0 {
		currentTime := fi.sim.GetClock().CurrentTime()
		recoveryEvent := &simulator.SimulationEvent{
			ID:        fmt.Sprintf("recover-degraded-%s", nodeID),
			Type:      simulator.EventNodeRecover,
			Timestamp: currentTime + duration,
			NodeID:    nodeID,
		}
		fi.sim.ScheduleEvent(recoveryEvent)
	}

	return nil
}

// InjectLinkFailure severs a connection between two nodes
func (fi *FailureInjector) InjectLinkFailure(source, dest string, duration float64) error {
	link := fi.sim.GetLink(source, dest)
	if link == nil {
		return fmt.Errorf("link %s->%s not found", source, dest)
	}

	// Disable the link
	link.SetActive(false)

	// Schedule recovery if duration > 0
	if duration > 0 {
		currentTime := fi.sim.GetClock().CurrentTime()
		recoveryEvent := &simulator.SimulationEvent{
			ID:        fmt.Sprintf("recover-link-%s-%s", source, dest),
			Type:      simulator.EventLinkRecover,
			Timestamp: currentTime + duration,
			LinkID:    fmt.Sprintf("%s->%s", source, dest),
			Data: &simulator.LinkFailureEventData{
				Source:   source,
				Dest:     dest,
				Duration: 0, // Already recovered
			},
		}
		fi.sim.ScheduleEvent(recoveryEvent)
	}

	return nil
}

// InjectNetworkPartition splits the network into isolated groups
func (fi *FailureInjector) InjectNetworkPartition(partitions [][]string, duration float64) error {
	// Disable all links between partitions
	affectedLinks := make([]struct{ source, dest string }, 0)

	for i, partition1 := range partitions {
		for j, partition2 := range partitions {
			if i >= j {
				continue // Skip same partition and duplicates
			}

			// Disable all links between these two partitions
			for _, node1 := range partition1 {
				for _, node2 := range partition2 {
					// Try both directions
					link1 := fi.sim.GetLink(node1, node2)
					if link1 != nil {
						link1.SetActive(false)
						affectedLinks = append(affectedLinks, struct{ source, dest string }{node1, node2})
					}

					link2 := fi.sim.GetLink(node2, node1)
					if link2 != nil {
						link2.SetActive(false)
						affectedLinks = append(affectedLinks, struct{ source, dest string }{node2, node1})
					}
				}
			}
		}
	}

	// Schedule recovery if duration > 0
	if duration > 0 {
		currentTime := fi.sim.GetClock().CurrentTime()
		for _, linkInfo := range affectedLinks {
			recoveryEvent := &simulator.SimulationEvent{
				ID:        fmt.Sprintf("recover-partition-%s-%s", linkInfo.source, linkInfo.dest),
				Type:      simulator.EventLinkRecover,
				Timestamp: currentTime + duration,
				LinkID:    fmt.Sprintf("%s->%s", linkInfo.source, linkInfo.dest),
				Data: &simulator.LinkFailureEventData{
					Source:   linkInfo.source,
					Dest:     linkInfo.dest,
					Duration: 0,
				},
			}
			fi.sim.ScheduleEvent(recoveryEvent)
		}
	}

	return nil
}

// InjectSlowLink increases latency on a link
func (fi *FailureInjector) InjectSlowLink(source, dest string, latencyMultiplier float64, duration float64) error {
	link := fi.sim.GetLink(source, dest)
	if link == nil {
		return fmt.Errorf("link %s->%s not found", source, dest)
	}

	// Store original latency and increase it
	originalLatency := link.Latency
	link.Latency = originalLatency * latencyMultiplier

	// Schedule recovery if duration > 0
	if duration > 0 {
		currentTime := fi.sim.GetClock().CurrentTime()
		// Create custom recovery event that restores original latency
		recoveryEvent := &simulator.SimulationEvent{
			ID:        fmt.Sprintf("recover-slow-link-%s-%s", source, dest),
			Type:      simulator.EventChaosAction,
			Timestamp: currentTime + duration,
			LinkID:    fmt.Sprintf("%s->%s", source, dest),
			Data: &simulator.ChaosActionEventData{
				Description: fmt.Sprintf("Recover slow link: %s->%s", source, dest),
				Execute: func() error {
					link := fi.sim.GetLink(source, dest)
					if link != nil {
						link.Latency = originalLatency
					}
					return nil
				},
			},
		}
		fi.sim.ScheduleEvent(recoveryEvent)
	}

	return nil
}

// InjectCongestion floods a link with traffic
func (fi *FailureInjector) InjectCongestion(source, dest string, messagesPerSecond float64, messageSize int, duration float64) error {
	// Generate heavy traffic on this link
	currentTime := fi.sim.GetClock().CurrentTime()

	gen := simulator.NewPoissonTrafficGenerator(source, dest, messagesPerSecond, messageSize)
	flow := simulator.NewTrafficFlow(
		fmt.Sprintf("congestion-%s-%s", source, dest),
		gen,
		currentTime,
		currentTime+duration,
	)

	fi.sim.AddTrafficFlow(flow)

	// Generate and schedule events
	events := gen.GenerateEvents(currentTime, currentTime+duration)
	for _, event := range events {
		fi.sim.ScheduleEvent(event)
	}

	return nil
}

// RandomNodeFailure kills 1-3 random nodes
func (fi *FailureInjector) RandomNodeFailure(minNodes, maxNodes int, duration float64) ([]string, error) {
	allNodes := fi.sim.GetAllNodes()
	if len(allNodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	// Random number of nodes to fail
	numToFail := minNodes + rand.Intn(maxNodes-minNodes+1)
	if numToFail > len(allNodes) {
		numToFail = len(allNodes)
	}

	// Shuffle and select random nodes
	failedNodes := make([]string, 0, numToFail)
	indices := rand.Perm(len(allNodes))

	for i := 0; i < numToFail; i++ {
		nodeID := allNodes[indices[i]].ID
		err := fi.InjectNodeFailure(nodeID, duration)
		if err != nil {
			continue
		}
		failedNodes = append(failedNodes, nodeID)
	}

	return failedNodes, nil
}

// CascadingFailure triggers failures that propagate through the network
func (fi *FailureInjector) CascadingFailure(initialNode string, cascadeProb float64, maxCascade int, duration float64) ([]string, error) {
	// Fail the initial node
	err := fi.InjectNodeFailure(initialNode, duration)
	if err != nil {
		return nil, err
	}

	failedNodes := []string{initialNode}
	toProcess := []string{initialNode}
	topo := fi.sim.GetTopology()

	// BFS-style cascade
	for len(toProcess) > 0 && len(failedNodes) < maxCascade {
		current := toProcess[0]
		toProcess = toProcess[1:]

		// Get neighbors
		neighbors := topo.GetNeighbors(current)
		for _, neighbor := range neighbors {
			// Check if already failed
			alreadyFailed := false
			for _, failed := range failedNodes {
				if failed == neighbor {
					alreadyFailed = true
					break
				}
			}
			if alreadyFailed {
				continue
			}

			// Cascade with probability
			if rand.Float64() < cascadeProb {
				err := fi.InjectNodeFailure(neighbor, duration)
				if err != nil {
					continue
				}
				failedNodes = append(failedNodes, neighbor)
				toProcess = append(toProcess, neighbor)

				if len(failedNodes) >= maxCascade {
					break
				}
			}
		}
	}

	return failedNodes, nil
}

// InjectByzantineFailure makes a node send corrupted messages
func (fi *FailureInjector) InjectByzantineFailure(nodeID string, corruptionRate float64, duration float64) error {
	node, err := fi.sim.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to inject byzantine failure: %w", err)
	}

	// Mark node as degraded (we'll use this to indicate Byzantine behavior)
	node.SetState(nodes.StateDegraded)

	// In a full implementation, we'd add corruption logic to message handling
	// For now, just mark the node as degraded

	// Schedule recovery if duration > 0
	if duration > 0 {
		currentTime := fi.sim.GetClock().CurrentTime()
		recoveryEvent := &simulator.SimulationEvent{
			ID:        fmt.Sprintf("recover-byzantine-%s", nodeID),
			Type:      simulator.EventNodeRecover,
			Timestamp: currentTime + duration,
			NodeID:    nodeID,
		}
		fi.sim.ScheduleEvent(recoveryEvent)
	}

	return nil
}

// FailureStats tracks statistics about injected failures
type FailureStats struct {
	NodesFailedCount    int
	LinksFailedCount    int
	PartitionsCreated   int
	CascadesTriggered   int
	ByzantineNodesCount int
}

// GetFailureImpact calculates the impact of failures
func (fi *FailureInjector) GetFailureImpact() FailureStats {
	stats := FailureStats{}

	// Count failed nodes
	for _, node := range fi.sim.GetAllNodes() {
		if node.GetState() == nodes.StateFailed {
			stats.NodesFailedCount++
		} else if node.GetState() == nodes.StateDegraded {
			stats.ByzantineNodesCount++
		}
	}

	// Count failed links
	// Would need to iterate through all links and check IsActive()
	// Simplified for now

	return stats
}
