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
		curOffset := time.Duration(0)
		for _, step := range s.Scenario {
			k := (step.LevelEnd - step.LevelStart) / float64(step.Duration/time.Second)
			x := 0.0
			accum := time.Duration(0)
			for {
				// antiderivative from linear function
				//1/X to turn it into intervals
				rate := k*x*x/2.0 + step.LevelStart*x
				offset := time.Duration(0)
				if rate != 0 {
					offset = time.Duration(float64(time.Second.Nanoseconds()) / rate)
				}
				if accum > step.Duration {
					break
				}
				accum += offset
				log.Infof("%s", accum)
				ch <- curOffset + accum
				x++
			}

			curOffset += step.Duration
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
