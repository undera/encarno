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

	for offset := range s.GenerateSchedule() {
		delay := s.StartTime.Add(offset).Sub(time.Now())
		if delay > 0 {
			log.Debugf("Sleeping %v before starting new worker", delay)
			time.Sleep(delay) // todo: make it cancelable
		}
		s.SpawnWorker(sched)
	}

	duration := time.Duration(0)
	for _, item := range s.Scenario {
		duration += item.RampUp + item.Steady
	}

	delay := s.StartTime.Add(duration).Sub(time.Now())
	if delay > 0 {
		log.Debugf("Sleeping %v to wait for end", delay)
		time.Sleep(delay) // todo: make it cancelable
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
