package core

import (
	log "github.com/sirupsen/logrus"
	"time"
)

// WorkloadLevel arrays may be used to specify workload scenario (warmup, ramp-up, steps, steady)
type WorkloadLevel struct {
	LevelStart float64
	LevelEnd   float64
	Duration   time.Duration
}

type WorkloadMode = string

const (
	WorkloadOpen   WorkloadMode = "open"
	WorkloadClosed WorkloadMode = "closed"
)

type WorkerConf struct {
	Mode             WorkloadMode
	WorkloadSchedule []WorkloadLevel
	StartingWorkers  int
	MaxWorkers       int
	Values           map[string]string
}

type BaseWorkload struct {
	Workers      []*Worker
	NibMaker     NibMaker
	StartTime    time.Time
	Output       *Output
	InputPayload func() InputChannel
	Scenario     []WorkloadLevel
	cnt          int
	Status       *Status
	Values       ValMap
}

func (s *BaseWorkload) SpawnWorker(scheduleChan ScheduleChannel) {
	s.cnt++
	log.Infof("Spawning worker: #%d", s.cnt)
	abort := make(chan struct{})
	worker := NewBasicWorker(s.cnt, abort, s, scheduleChan, s.Values)
	s.Workers = append(s.Workers, worker)
	go worker.Run()
}

func (s *BaseWorkload) Stop() {
	log.Infof("Telling workers to not continue...")
	for _, worker := range s.Workers {
		worker.Stop()
	}
}

func NewBaseWorkload(maker NibMaker, output *Output, inputConfig InputConf, wconf WorkerConf, status *Status) *BaseWorkload {
	var payloadGetter func() InputChannel
	if inputConfig.EnableRegexes {
		payloadGetter = func() InputChannel {
			inputChannel := NewInput(inputConfig)
			return inputChannel
		}
	} else {
		inputChannel := NewInput(inputConfig)
		payloadGetter = func() InputChannel {
			return inputChannel
		}
	}

	values := make(ValMap)
	for k, v := range wconf.Values {
		values[k] = []byte(v)
	}

	return &BaseWorkload{
		Workers:      make([]*Worker, 0),
		NibMaker:     maker,
		StartTime:    time.Now(),
		Output:       output,
		InputPayload: payloadGetter,
		Scenario:     wconf.WorkloadSchedule,
		Status:       status,
		Values:       values,
	}
}
