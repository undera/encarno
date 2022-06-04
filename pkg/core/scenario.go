package core

import (
	log "github.com/sirupsen/logrus"
	"strconv"
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
}

type BaseWorkload struct {
	Workers      []*Worker
	NibMaker     NibMaker
	StartTime    time.Time
	Output       Output
	Status       Status
	InputPayload func() InputChannel
	Scenario     []WorkloadLevel
	cnt          int
}

func (s *BaseWorkload) SpawnWorker(scheduleChan ScheduleChannel) {
	s.cnt++
	name := "worker#" + strconv.Itoa(s.cnt)
	log.Infof("Spawning worker: %s", name)
	abort := make(chan struct{})
	worker := NewBasicWorker(name, abort, s, scheduleChan)
	s.Workers = append(s.Workers, worker)
	go worker.Run()
}

func NewBaseWorkload(maker NibMaker, output Output, inputConfig InputConf, wconf WorkerConf) BaseWorkload {
	var payloadGetter func() InputChannel
	if inputConfig.EnableRegexes || wconf.Mode == WorkloadClosed {
		inputChannel := NewInput(inputConfig)
		payloadGetter = func() InputChannel {
			return inputChannel
		}
	} else {
		payloadGetter = func() InputChannel {
			inputChannel := NewInput(inputConfig)
			return inputChannel
		}
	}

	status := output.GetStatusObj()
	return BaseWorkload{
		Workers:      make([]*Worker, 0),
		NibMaker:     maker,
		StartTime:    time.Now(),
		Output:       output,
		Status:       status,
		InputPayload: payloadGetter,
		Scenario:     wconf.WorkloadSchedule,
	}
}
