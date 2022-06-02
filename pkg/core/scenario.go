package core

import (
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

// WorkloadLevel arrays may be used to specify workload scenario (warmup, ramp-up, steps, steady)
type WorkloadLevel struct {
	Level  int
	RampUp time.Duration
	Steady time.Duration
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
	Workers   []*Worker
	NibMaker  NibMaker
	StartTime time.Time
	Output    Output
	Status    Status
	cnt       int
}

func (s *BaseWorkload) SpawnWorker(inputs InputChannel) {
	s.cnt++
	name := "worker#" + strconv.Itoa(s.cnt)
	log.Infof("Spawning worker: %s", name)
	abort := make(chan struct{})
	worker := NewBasicWorker(name, abort, inputs, s.Output, s.StartTime, s.NibMaker(), s.Status)
	s.Workers = append(s.Workers, worker)
	go worker.Run()
}

func NewBaseWorkload(maker NibMaker, output Output) BaseWorkload {
	return BaseWorkload{
		Workers:   make([]*Worker, 0),
		NibMaker:  maker,
		StartTime: time.Now(),
		Output:    output,
		Status:    &StatusImpl{},
	}
}
