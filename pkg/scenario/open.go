package scenario

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"math"
	"time"
)

// OpenWorkload imlements pre-calculated open workload scenario
type OpenWorkload struct {
	core.BaseWorkload
	MinWorkers int
	MaxWorkers int

	interrupted bool
}

func (s *OpenWorkload) Interrupt() {
	s.interrupted = true // TODO: use that flag
	// TODO: tell workers to stop
}

func (s *OpenWorkload) SpawnInitial(scheduleChan core.ScheduleChannel) {
	// spawn initial workers
	initialWorkers := s.MinWorkers
	initialWorkers = int(math.Min(float64(s.MaxWorkers), float64(initialWorkers)))
	initialWorkers = int(math.Max(1, float64(initialWorkers)))
	for x := 0; x < initialWorkers; x++ {
		s.SpawnWorker(scheduleChan)
	}
}

func (s *OpenWorkload) Run() {
	log.Debugf("Starting open workload scenario")

	scheduleChan := make(chan time.Duration)

	s.SpawnInitial(scheduleChan)

	last := time.Duration(0)
	for offset := range s.GenerateSchedule() {
		if offset < last {
			panic("can't be")
		} else {
			last = offset
		}

		select {
		case scheduleChan <- offset: // try putting if somebody is reading it
			continue
		default:
			working := s.Status.GetWorking()
			sleeping := s.Status.GetSleeping()
			workerCnt := len(s.Workers)
			if working >= int64(workerCnt) && sleeping < 1 && workerCnt < s.MaxWorkers {
				log.Infof("Working: %d, sleeping: %d, busy: %d", working, sleeping, s.Status.GetBusy())
				s.SpawnWorker(nil)
			}
			scheduleChan <- offset
		}
	}

	// TODO: make sure workers have finished before exiting

	close(scheduleChan)
	log.Infof("Open workload scenario is complete")
}

func (s *OpenWorkload) GenerateSchedule() core.ScheduleChannel {
	ch := make(core.ScheduleChannel)
	go func() {
		curStep := 0
		cnt := 0
		accum := time.Duration(0)
		finishedSteps := time.Duration(0)
		for curStep < len(s.Scenario) {
			step := s.Scenario[curStep]
			durSec := float64(step.Duration) / float64(time.Second)
			k := (step.LevelEnd - step.LevelStart) / durSec

			var offset float64
			if k != 0 && cnt != 0 {
				offset = 1 / (k * math.Sqrt(2*float64(cnt)/k))
			} else {
				offset = 0
			}
			accum += time.Duration(int64(offset * float64(time.Second)))
			ch <- accum
			cnt += 1
			if accum > finishedSteps+step.Duration {
				curStep += 1
				finishedSteps += accum
			}
		}

		close(ch)
	}()
	return ch
}

func NewOpenWorkload(workers core.WorkerConf, inputConfig core.InputConf, maker core.NibMaker, output core.Output) core.WorkerSpawner {
	workload := OpenWorkload{
		BaseWorkload: core.NewBaseWorkload(maker, output, inputConfig, workers),
		MinWorkers:   workers.StartingWorkers,
		MaxWorkers:   workers.MaxWorkers,
	}
	return &workload
}
