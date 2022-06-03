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

	for offset := range s.GenerateSchedule() {
		select {
		case scheduleChan <- offset: // try putting if somebody is reading it
			continue
		default:
			working := s.Status.GetWorking()
			sleeping := s.Status.GetSleeping()
			log.Infof("Working: %d, sleeping: %d, busy: %d", working, sleeping, s.Status.GetBusy())
			workerCnt := len(s.Workers)
			if working >= int64(workerCnt) && sleeping < 1 && workerCnt < s.MaxWorkers {
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
	//TODO implement me
	panic("implement me")
}

func NewOpenWorkload(workers core.WorkerConf, inputConfig core.InputConf, maker core.NibMaker, output core.Output) core.WorkerSpawner {
	workload := OpenWorkload{
		BaseWorkload: core.NewBaseWorkload(maker, output, inputConfig, core.WorkloadOpen),
		MinWorkers:   workers.StartingWorkers,
		MaxWorkers:   workers.MaxWorkers,
	}
	return &workload
}
