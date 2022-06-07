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

	interrupted  bool
	sumDurations time.Duration
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

	stopCutoff := s.StartTime.Add(s.sumDurations + s.sumDurations/10)
	log.Infof("Duration cutoff is at %s", stopCutoff)

	scheduleChan := make(chan time.Duration)

	s.SpawnInitial(scheduleChan)

	last := time.Duration(0)
	for offset := range s.GenerateSchedule() {
		if offset < last {
			panic("Schedule offsets have to be ever-increasing")
		} else {
			last = offset
		}

		if time.Now().After(stopCutoff) {
			log.Warningf("The test exceeds expected duration of %v, interrupting...", s.sumDurations)
			break
		}

		select {
		case scheduleChan <- offset: // try putting if somebody is reading it
			continue
		default:
			workerCnt := len(s.Workers)
			working := s.Status.GetWorking()
			sleeping := s.Status.GetSleeping()
			busy := s.Status.GetBusy()
			waiting := s.Status.GetWaiting()
			log.Debugf("len: %d, waiting: %d, working: %d, sleeping: %d, busy: %d", workerCnt, waiting, working, sleeping, busy)

			notMaxed := s.MaxWorkers <= 0 || workerCnt < s.MaxWorkers
			if notMaxed && sleeping <= 0 {
				s.SpawnWorker(scheduleChan)
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

			var rate float64
			if step.LevelStart == step.LevelEnd {
				rate = step.LevelEnd
			} else {
				durSec := float64(step.Duration) / float64(time.Second)
				k := (step.LevelEnd - step.LevelStart) / durSec
				rate = k * math.Sqrt(2*float64(cnt)/k)
			}

			var interval float64
			if rate > 0 {
				interval = 1 / rate
			} else {
				interval = 0
			}

			if accum != 0 && interval == 0 {
				panic("schedule calculations stuck")
			}

			accum += time.Duration(int64(interval * float64(time.Second)))
			ch <- accum
			cnt += 1
			if accum > finishedSteps+step.Duration {
				curStep += 1
				finishedSteps = accum
			}
		}

		close(ch)
	}()
	return ch
}

func NewOpenWorkload(workers core.WorkerConf, inputConfig core.InputConf, maker core.NibMaker, output core.Output) core.WorkerSpawner {

	sumDurations := time.Duration(0)
	for _, step := range workers.WorkloadSchedule {
		sumDurations += step.Duration
	}

	workload := OpenWorkload{
		BaseWorkload: core.NewBaseWorkload(maker, output, inputConfig, workers),
		MinWorkers:   workers.StartingWorkers,
		MaxWorkers:   workers.MaxWorkers,
		sumDurations: sumDurations,
	}
	return &workload
}
