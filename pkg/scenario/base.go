package scenario

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"strconv"
	"time"
)

type BaseWorkload struct {
	Workers   []core.Worker
	Input     core.Input
	Output    core.Output
	StartTime time.Time
	NibMaker  func() core.Nib
	Spawner   core.WorkerSpawner
}

func (s *BaseWorkload) SpawnWorker(inputs core.InputChannel) {
	name := "worker#" + strconv.Itoa(len(s.Workers)+1)
	log.Infof("Spawning worker: %s", name)
	abort := make(chan struct{})
	worker := core.NewBasicWorker(name, abort, inputs, s.Output, s.StartTime, s.NibMaker())
	s.Workers = append(s.Workers, worker)
	go worker.Run()
}

func (s *BaseWorkload) Run() {
	// read from input, spawn workers if needed
	log.Debugf("Starting scenario 'external'")

	inputs := make(core.InputChannel)

	s.Spawner.SpawnInitial(inputs)

	for x := range s.Input.Generator() {
		select {
		case inputs <- x: // try putting if somebody is reading it
			continue
		default:
			s.Spawner.SpawnForSample(inputs, x)
			inputs <- x
		}
	}
}
