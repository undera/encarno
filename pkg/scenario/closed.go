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
	log.Debugf("Starting closed workload")

	for _, milestone := range s.Scenario {
		_ = milestone
		input := core.NewInput(s.InputConfig)
		s.SpawnWorker(input)
	}
	log.Infof("Closed workload scenario is complete")
}

func NewClosedWorkload(workers core.WorkerConf, inputConfig core.InputConf, maker core.NibMaker, output core.Output) core.WorkerSpawner {
	workload := ClosedWorkload{
		BaseWorkload: core.NewBaseWorkload(maker, output),
		Scenario:     inputConfig.WorkloadSchedule,
		InputConfig:  inputConfig,
	}

	return &workload
}
