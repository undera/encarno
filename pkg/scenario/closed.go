package scenario

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"time"
)

// ClosedWorkload implements closed workload scenario
type ClosedWorkload struct {
	core.BaseWorkload
	InputConfig core.InputConf
}

func (s *ClosedWorkload) Interrupt() {
	// TODO
}

func (s *ClosedWorkload) Run() {
	log.Debugf("Starting closed workload")

	// dummy schedule to punch immediately
	sched := make(chan time.Duration)
	go func() {
		for {
			sched <- 0
		}
	}()

	for _, milestone := range s.Scenario {
		_ = milestone
		s.SpawnWorker(sched)
	}
	log.Infof("Closed workload scenario is complete")
}

func NewClosedWorkload(workers core.WorkerConf, inputConfig core.InputConf, maker core.NibMaker, output core.Output) core.WorkerSpawner {
	workload := ClosedWorkload{
		BaseWorkload: core.NewBaseWorkload(maker, output, inputConfig, core.WorkloadClosed),
		InputConfig:  inputConfig,
	}

	return &workload
}
