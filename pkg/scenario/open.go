package scenario

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"math"
)

// OpenWorkload imlements pre-calculated open workload scenario
type OpenWorkload struct {
	core.BaseWorkload
	MinWorkers int
	MaxWorkers int
	Input      core.InputChannel

	interrupted bool
}

func (s *OpenWorkload) Interrupt() {
	s.interrupted = true // TODO: use that flag
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

	if s.Input == nil {
		panic("Cannot have nil as input channel for open workload")
	}

	workerInputs := make(core.InputChannel)
	// TODO: separate channels to read payload and schedule

	s.SpawnInitial(workerInputs)

	for x := range s.Input {
		select {
		case workerInputs <- x: // try putting if somebody is reading it
			continue
		default:
			working := s.Status.GetWorking()
			sleeping := s.Status.GetSleeping()
			log.Infof("Working: %d, sleeping: %d, busy: %d", working, sleeping, s.Status.GetBusy())
			workerCnt := len(s.Workers)
			if working >= int64(workerCnt) && sleeping < 1 && workerCnt < s.MaxWorkers {
				s.SpawnWorker(workerInputs)
			}
			workerInputs <- x
		}
	}
}

func NewOpenWorkload(workers core.WorkerConf, inputConfig core.InputConf, maker core.NibMaker, output core.Output) core.WorkerSpawner {
	inputChannel := core.NewInput(inputConfig)
	workload := OpenWorkload{
		BaseWorkload: core.NewBaseWorkload(maker, output),
		MinWorkers:   workers.StartingWorkers,
		MaxWorkers:   workers.MaxWorkers,
		Input:        inputChannel,
	}
	return &workload
}
