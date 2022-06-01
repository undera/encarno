package core

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	abort := make(chan struct{})
	inputs := make(InputChannel)
	output := DummyOutput{
		queue: make(chan *OutputItem),
	}
	go output.bg()
	worker := NewBasicWorker("test", abort, inputs, &output, time.Now(), &DummyNib{}, status)
	go worker.Run()

	for i := time.Duration(0); i < 1000; i++ {
		inputs <- &InputItem{TimeOffset: i * 1 * time.Millisecond}
	}
	inputs <- nil
}

type DummyOutput struct {
	queue chan *OutputItem
}

func (d DummyOutput) DecBusy() {
}

func (d DummyOutput) IncBusy() {
}

func (d *DummyOutput) Push(res *OutputItem) {
	//d.queue <- res
	d.log(res)
}

func (d DummyOutput) IncSleeping() {
}

func (d DummyOutput) DecSleeping() {
}

func (d DummyOutput) IncWorking() {
}

func (d DummyOutput) DecWorking() {
}

func (d DummyOutput) GetWorking() int64 {
	return 0
}

func (d DummyOutput) GetSleeping() int64 {
	return 0
}

func (d DummyOutput) GetBusy() int64 {
	return 0
}

func (d *DummyOutput) bg() {
	for {
		res := <-d.queue
		d.log(res)
	}
}

func (d *DummyOutput) log(res *OutputItem) {
	data, err := json.Marshal(res)
	log.Infof("Status result: %s %v", data, err)
}
