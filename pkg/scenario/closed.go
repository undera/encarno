package scenario

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
)

// ClosedWorkload implements closed workload scenario
type ClosedWorkload struct {
	core.BaseWorkload
	Scenario    []core.WorkloadLevel
	InputConfig core.InputConf
}

func (s *ClosedWorkload) Interrupt() {
	// TODO
}

func (s *ClosedWorkload) Run() {
	log.Debugf("Starting scenario 'external'")

	for _, milestone := range s.Scenario {
		_ = milestone
		input := core.NewInput(s.InputConfig)
		s.SpawnWorker(input)
	}
}
