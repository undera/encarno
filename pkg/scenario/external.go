package scenario

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"math"
	"strconv"
	"time"
)

// External imlements pre-calculated open workload scenario
type External struct {
	Workers    []core.Worker
	Input      core.Input
	Output     core.Output
	MinWorkers int
	MaxWorkers int
	StartTime  time.Time
	NibMaker   func() core.Nib
}

func (s *External) Run() {
	// read from input, spawn workers if needed
	log.Debugf("Starting scenario 'external'")

	inputs := make(core.InputChannel)

	// spawn initial workers
	initialWorkers := s.MinWorkers
	initialWorkers = int(math.Min(float64(s.MaxWorkers), float64(initialWorkers)))
	initialWorkers = int(math.Max(1, float64(initialWorkers)))
	for x := 0; x < initialWorkers; x++ {
		s.SpawnWorker(inputs)
	}

	for x := range s.Input.Generator() {
		select {
		case inputs <- x: // try putting if somebody is reading it
			continue
		default:
			working := s.Output.GetWorking()
			sleeping := s.Output.GetSleeping()
			log.Infof("Working: %d, sleeping: %d, busy: %d", working, sleeping, s.Output.GetBusy())
			if working >= int64(len(s.Workers)) && sleeping < 1 {
				s.SpawnWorker(inputs)
			}
			inputs <- x
		}
	}
}

func (s *External) SpawnWorker(inputs core.InputChannel) {
	name := "worker#" + strconv.Itoa(len(s.Workers)+1)
	log.Infof("Spawning worker: %s", name)
	abort := make(chan struct{})
	worker := core.NewBasicWorker(name, abort, inputs, s.Output, s.StartTime, s.NibMaker())
	s.Workers = append(s.Workers, worker)
	go worker.Run()
}
