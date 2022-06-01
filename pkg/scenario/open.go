package scenario

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"math"
)

// OpenWorkload imlements pre-calculated open workload scenario
type OpenWorkload struct {
	core.BaseWorkload
	Status     core.Status
	MinWorkers int
	MaxWorkers int
	Input      core.InputChannel

	interrupted bool
}

func (s *OpenWorkload) Interrupt() {
	s.interrupted = true
	// TODO: tell workers to stop
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
	log.Debugf("Starting open workload scenario")

	workerInputs := make(core.InputChannel)

	s.SpawnInitial(workerInputs)

	for x := range s.Input {
		select {
		case workerInputs <- x: // try putting if somebody is reading it
			continue
		default:
			working := s.Status.GetWorking()
			sleeping := s.Status.GetSleeping()
			log.Infof("Working: %d, sleeping: %d, busy: %d", working, sleeping, s.Status.GetBusy())
			if working >= int64(len(s.Workers)) && sleeping < 1 {
				s.SpawnWorker(workerInputs)
			}
			workerInputs <- x
		}
	}
}
