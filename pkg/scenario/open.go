package scenario

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"math"
)

// OpenWorkload imlements pre-calculated open workload scenario
type OpenWorkload struct {
	BaseWorkload
	MinWorkers int
	MaxWorkers int
}

func (s *OpenWorkload) SpawnForSample(inputs core.InputChannel, x *core.InputItem) {
	working := s.Output.GetWorking()
	sleeping := s.Output.GetSleeping()
	log.Infof("Working: %d, sleeping: %d, busy: %d", working, sleeping, s.Output.GetBusy())
	if working >= int64(len(s.Workers)) && sleeping < 1 {
		s.SpawnWorker(inputs)
	}
}

func (s *OpenWorkload) SpawnInitial(inputs core.InputChannel) {
	// spawn initial workers
	initialWorkers := s.MinWorkers
	initialWorkers = int(math.Min(float64(s.MaxWorkers), float64(initialWorkers)))
	initialWorkers = int(math.Max(1, float64(initialWorkers)))
	for x := 0; x < initialWorkers; x++ {
		s.SpawnWorker(inputs)
	}
}

func (s *OpenWorkload) Run() {
	s.BaseWorkload.Run()
}
