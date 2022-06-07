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

	subInitial := time.Duration(-1)
	for offset := range s.GenerateSchedule() {
		// eliminate initial delay, if any
		if subInitial < 0 {
			subInitial = offset
		}
		offset -= subInitial

		delay := s.StartTime.Add(offset).Sub(time.Now())
		if delay > 0 {
			log.Debugf("Sleeping %v before starting new worker", delay)
			time.Sleep(delay) // todo: make it cancelable
		}
		s.SpawnWorker(sched)
	}

	duration := time.Duration(0)
	for _, item := range s.Scenario {
		duration += item.Duration
	}

	delay := s.StartTime.Add(duration).Sub(time.Now())
	if delay > 0 {
		log.Debugf("Sleeping %v to wait for end", delay)
		time.Sleep(delay) // todo: make it cancelable
	}

	log.Infof("Closed workload scenario is complete")
}

func (s *ClosedWorkload) GenerateSchedule() core.ScheduleChannel {
	ch := make(core.ScheduleChannel)
	go func() {
		curLevel := float64(0)
		curOffset := time.Duration(0)
		for _, step := range s.Scenario {
			// reach starting level of scenario step
			for i := curLevel; i < step.LevelStart; i++ {
				// TODO: can we support decreasing the level?
				ch <- curOffset
			}
			curLevel = step.LevelStart

			// progress through step
			if step.LevelStart < step.LevelEnd {
				durStep := float64(step.Duration.Nanoseconds()) / (step.LevelEnd - step.LevelStart)
				for i := 1.0; i <= (step.LevelEnd - step.LevelStart); i++ { // starting from 1 because 0 is covered above
					ch <- curOffset + time.Duration(durStep*i)
				}
			} else if step.LevelStart > step.LevelEnd {
				// TODO: can we support decreasing the level?
				log.Warningf("Decreasing worker count is not supported at the moment. The step %v is invalid", step)
			}

			curLevel = step.LevelEnd
			curOffset += step.Duration
		}
		close(ch)
	}()
	return ch
}

func NewClosedWorkload(workers core.WorkerConf, inputConfig core.InputConf, maker core.NibMaker, output core.Output) core.WorkerSpawner {
	workload := ClosedWorkload{
		BaseWorkload: core.NewBaseWorkload(maker, output, inputConfig, workers),
		InputConfig:  inputConfig,
	}

	return &workload
}
