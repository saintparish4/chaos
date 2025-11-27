package chaos

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/saintparish4/chaos/simulator"
	"gopkg.in/yaml.v3"
)

// ChaosExperiment defines a chaos engineering experiment
type ChaosExperiment struct {
	Name        string             `yaml:"name"`
	Description string             `yaml:"description"`
	Duration    float64            `yaml:"duration"`
	Events      []ChaosEventConfig `yaml:"events"`
	Randomized  *RandomizedConfig  `yaml:"randomized,omitempty"`
}

// ChaosEventConfig defines a single chaos event
type ChaosEventConfig struct {
	Time     float64 `yaml:"time"`     // When to inject the failure
	Type     string  `yaml:"type"`     // Type of failure
	Target   string  `yaml:"target"`   // Target node/link (optional)
	Source   string  `yaml:"source"`   // Source for link failures
	Dest     string  `yaml:"dest"`     // Dest for link failures
	Duration float64 `yaml:"duration"` // How long the failure lasts

	// Additional parameters
	MinNodes          int        `yaml:"min_nodes,omitempty"`
	MaxNodes          int        `yaml:"max_nodes,omitempty"`
	CascadeProb       float64    `yaml:"cascade_prob,omitempty"`
	MaxCascade        int        `yaml:"max_cascade,omitempty"`
	LatencyMultiplier float64    `yaml:"latency_multiplier,omitempty"`
	MessagesPerSecond float64    `yaml:"messages_per_second,omitempty"`
	MessageSize       int        `yaml:"message_size,omitempty"`
	Partitions        [][]string `yaml:"partitions,omitempty"`
}

// RandomizedConfig for randomized chaos mode
type RandomizedConfig struct {
	Enabled      bool     `yaml:"enabled"`
	FailureRate  float64  `yaml:"failure_rate"`  // Failures per second
	FailureTypes []string `yaml:"failure_types"` // Which types to use
	MinDuration  float64  `yaml:"min_duration"`
	MaxDuration  float64  `yaml:"max_duration"`
}

// ChaosScheduler manages and executes chaos experiments
type ChaosScheduler struct {
	injector   *FailureInjector
	experiment *ChaosExperiment
}

// NewChaosScheduler creates a new chaos scheduler
func NewChaosScheduler(injector *FailureInjector) *ChaosScheduler {
	return &ChaosScheduler{
		injector: injector,
	}
}

// LoadExperiment loads a chaos experiment from a YAML file
func (cs *ChaosScheduler) LoadExperiment(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read experiment file: %w", err)
	}

	var experiment ChaosExperiment
	if err := yaml.Unmarshal(data, &experiment); err != nil {
		return fmt.Errorf("failed to parse experiment: %w", err)
	}

	cs.experiment = &experiment
	return nil
}

// SetExperiment sets the experiment programmatically
func (cs *ChaosScheduler) SetExperiment(experiment *ChaosExperiment) {
	cs.experiment = experiment
}

// ScheduleExperiment schedules all chaos events for the experiment
func (cs *ChaosScheduler) ScheduleExperiment() error {
	if cs.experiment == nil {
		return fmt.Errorf("no experiment loaded")
	}

	fmt.Printf("\n=== Chaos Experiment: %s ===\n", cs.experiment.Name)
	fmt.Printf("Description: %s\n", cs.experiment.Description)
	fmt.Printf("Duration: %.1fs\n", cs.experiment.Duration)
	fmt.Printf("Scheduled Events: %d\n\n", len(cs.experiment.Events))

	// Schedule each event
	for i, event := range cs.experiment.Events {
		err := cs.scheduleEvent(&event, i)
		if err != nil {
			fmt.Printf("Warning: Failed to schedule event %d: %v\n", i, err)
		}
	}

	// Handle randomized mode
	if cs.experiment.Randomized != nil && cs.experiment.Randomized.Enabled {
		cs.scheduleRandomizedChaos()
	}

	return nil
}

// scheduleEvent schedules a single chaos event
func (cs *ChaosScheduler) scheduleEvent(config *ChaosEventConfig, index int) error {
	currentTime := cs.injector.sim.GetClock().CurrentTime()
	eventTime := currentTime + config.Time

	fmt.Printf("Scheduling event %d: %s at t=%.2fs\n", index+1, config.Type, eventTime)

	switch config.Type {
	case "node_failure":
		return cs.scheduleNodeFailure(config, eventTime)
	case "node_degrade":
		return cs.scheduleNodeDegrade(config, eventTime)
	case "link_failure":
		return cs.scheduleLinkFailure(config, eventTime)
	case "network_partition":
		return cs.scheduleNetworkPartition(config, eventTime)
	case "slow_link":
		return cs.scheduleSlowLink(config, eventTime)
	case "congestion":
		return cs.scheduleCongestion(config, eventTime)
	case "random_node_failure":
		return cs.scheduleRandomNodeFailure(config, eventTime)
	case "cascading_failure":
		return cs.scheduleCascadingFailure(config, eventTime)
	default:
		return fmt.Errorf("unknown failure type: %s", config.Type)
	}
}

func (cs *ChaosScheduler) scheduleNodeFailure(config *ChaosEventConfig, eventTime float64) error {
	// Schedule the failure to happen at the specified time
	cs.injector.sim.ScheduleEvent(&simulator.SimulationEvent{
		ID:        fmt.Sprintf("chaos-node-failure-%s-%.2f", config.Target, eventTime),
		Type:      simulator.EventChaosAction,
		Timestamp: eventTime,
		NodeID:    config.Target,
		Data: &simulator.ChaosActionEventData{
			Description: fmt.Sprintf("Node failure: %s", config.Target),
			Execute: func() error {
				return cs.injector.InjectNodeFailure(config.Target, config.Duration)
			},
		},
	})
	return nil
}

func (cs *ChaosScheduler) scheduleNodeDegrade(config *ChaosEventConfig, eventTime float64) error {
	cs.injector.sim.ScheduleEvent(&simulator.SimulationEvent{
		ID:        fmt.Sprintf("chaos-node-degrade-%s-%.2f", config.Target, eventTime),
		Type:      simulator.EventChaosAction,
		Timestamp: eventTime,
		NodeID:    config.Target,
		Data: &simulator.ChaosActionEventData{
			Description: fmt.Sprintf("Node degradation: %s", config.Target),
			Execute: func() error {
				return cs.injector.InjectPartialNodeFailure(config.Target, 0.5, config.Duration)
			},
		},
	})
	return nil
}

func (cs *ChaosScheduler) scheduleLinkFailure(config *ChaosEventConfig, eventTime float64) error {
	cs.injector.sim.ScheduleEvent(&simulator.SimulationEvent{
		ID:        fmt.Sprintf("chaos-link-failure-%s-%s-%.2f", config.Source, config.Dest, eventTime),
		Type:      simulator.EventChaosAction,
		Timestamp: eventTime,
		LinkID:    fmt.Sprintf("%s->%s", config.Source, config.Dest),
		Data: &simulator.ChaosActionEventData{
			Description: fmt.Sprintf("Link failure: %s->%s", config.Source, config.Dest),
			Execute: func() error {
				return cs.injector.InjectLinkFailure(config.Source, config.Dest, config.Duration)
			},
		},
	})
	return nil
}

func (cs *ChaosScheduler) scheduleNetworkPartition(config *ChaosEventConfig, eventTime float64) error {
	cs.injector.sim.ScheduleEvent(&simulator.SimulationEvent{
		ID:        fmt.Sprintf("chaos-partition-%.2f", eventTime),
		Type:      simulator.EventChaosAction,
		Timestamp: eventTime,
		Data: &simulator.ChaosActionEventData{
			Description: "Network partition",
			Execute: func() error {
				return cs.injector.InjectNetworkPartition(config.Partitions, config.Duration)
			},
		},
	})
	return nil
}

func (cs *ChaosScheduler) scheduleSlowLink(config *ChaosEventConfig, eventTime float64) error {
	cs.injector.sim.ScheduleEvent(&simulator.SimulationEvent{
		ID:        fmt.Sprintf("chaos-slow-link-%s-%s-%.2f", config.Source, config.Dest, eventTime),
		Type:      simulator.EventChaosAction,
		Timestamp: eventTime,
		LinkID:    fmt.Sprintf("%s->%s", config.Source, config.Dest),
		Data: &simulator.ChaosActionEventData{
			Description: fmt.Sprintf("Slow link: %s->%s", config.Source, config.Dest),
			Execute: func() error {
				multiplier := config.LatencyMultiplier
				if multiplier == 0 {
					multiplier = 10.0
				}
				return cs.injector.InjectSlowLink(config.Source, config.Dest, multiplier, config.Duration)
			},
		},
	})
	return nil
}

func (cs *ChaosScheduler) scheduleCongestion(config *ChaosEventConfig, eventTime float64) error {
	cs.injector.sim.ScheduleEvent(&simulator.SimulationEvent{
		ID:        fmt.Sprintf("chaos-congestion-%s-%s-%.2f", config.Source, config.Dest, eventTime),
		Type:      simulator.EventChaosAction,
		Timestamp: eventTime,
		LinkID:    fmt.Sprintf("%s->%s", config.Source, config.Dest),
		Data: &simulator.ChaosActionEventData{
			Description: fmt.Sprintf("Congestion: %s->%s", config.Source, config.Dest),
			Execute: func() error {
				return cs.injector.InjectCongestion(
					config.Source,
					config.Dest,
					config.MessagesPerSecond,
					config.MessageSize,
					config.Duration,
				)
			},
		},
	})
	return nil
}

func (cs *ChaosScheduler) scheduleRandomNodeFailure(config *ChaosEventConfig, eventTime float64) error {
	cs.injector.sim.ScheduleEvent(&simulator.SimulationEvent{
		ID:        fmt.Sprintf("chaos-random-node-%.2f", eventTime),
		Type:      simulator.EventChaosAction,
		Timestamp: eventTime,
		Data: &simulator.ChaosActionEventData{
			Description: "Random node failure",
			Execute: func() error {
				_, err := cs.injector.RandomNodeFailure(config.MinNodes, config.MaxNodes, config.Duration)
				return err
			},
		},
	})
	return nil
}

func (cs *ChaosScheduler) scheduleCascadingFailure(config *ChaosEventConfig, eventTime float64) error {
	cs.injector.sim.ScheduleEvent(&simulator.SimulationEvent{
		ID:        fmt.Sprintf("chaos-cascade-%s-%.2f", config.Target, eventTime),
		Type:      simulator.EventChaosAction,
		Timestamp: eventTime,
		NodeID:    config.Target,
		Data: &simulator.ChaosActionEventData{
			Description: fmt.Sprintf("Cascading failure from %s", config.Target),
			Execute: func() error {
				_, err := cs.injector.CascadingFailure(
					config.Target,
					config.CascadeProb,
					config.MaxCascade,
					config.Duration,
				)
				return err
			},
		},
	})
	return nil
}

// scheduleRandomizedChaos schedules random failures throughout the experiment
func (cs *ChaosScheduler) scheduleRandomizedChaos() {
	if cs.experiment.Randomized == nil {
		return
	}

	config := cs.experiment.Randomized
	currentTime := cs.injector.sim.GetClock().CurrentTime()
	endTime := currentTime + cs.experiment.Duration

	fmt.Printf("\nRandomized Chaos Mode: %.2f failures/second\n", config.FailureRate)
	fmt.Printf("Failure types: %v\n\n", config.FailureTypes)

	// Generate random failure times using Poisson process
	t := currentTime
	for t < endTime {
		// Exponential inter-arrival
		interval := -1.0 / config.FailureRate * rand.Float64()
		t += interval

		if t >= endTime {
			break
		}

		// Random failure type
		failureType := config.FailureTypes[rand.Intn(len(config.FailureTypes))]

		// Random duration
		duration := config.MinDuration + rand.Float64()*(config.MaxDuration-config.MinDuration)

		// Create event based on type
		event := &ChaosEventConfig{
			Time:     t - currentTime,
			Type:     failureType,
			Duration: duration,
		}

		// Fill in type-specific parameters
		cs.fillRandomEventParameters(event)

		cs.scheduleEvent(event, -1) // -1 index for random events
	}
}

// fillRandomEventParameters fills in random parameters for an event
func (cs *ChaosScheduler) fillRandomEventParameters(event *ChaosEventConfig) {
	allNodes := cs.injector.sim.GetAllNodes()
	if len(allNodes) == 0 {
		return
	}

	topo := cs.injector.sim.GetTopology()
	allLinks := topo.Config.Links

	switch event.Type {
	case "node_failure", "node_degrade":
		event.Target = allNodes[rand.Intn(len(allNodes))].ID

	case "link_failure", "slow_link":
		if len(allLinks) > 0 {
			link := allLinks[rand.Intn(len(allLinks))]
			event.Source = link.Source
			event.Dest = link.Dest
			if event.Type == "slow_link" {
				event.LatencyMultiplier = 5.0 + rand.Float64()*10.0
			}
		}

	case "congestion":
		if len(allLinks) > 0 {
			link := allLinks[rand.Intn(len(allLinks))]
			event.Source = link.Source
			event.Dest = link.Dest
			event.MessagesPerSecond = 50.0 + rand.Float64()*100.0
			event.MessageSize = 500 + rand.Intn(2000)
		}

	case "random_node_failure":
		event.MinNodes = 1
		event.MaxNodes = 3
	}
}
