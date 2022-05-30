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
	output := DummyOutput{}
	worker := NewBasicWorker("test", abort, inputs, output, time.Now(), &DummyNib{})
	go worker.Run()

	for i := time.Duration(0); i < 1000; i++ {
		inputs <- &InputItem{TimeOffset: i * 10 * time.Millisecond}
	}
	inputs <- nil
}

type DummyNib struct {
}

func (n *DummyNib) Punch(payload []byte) *OutputItem {
	start := time.Now()
	log.Infof("Processed payload: %s", payload)
	end := time.Now()
	return &OutputItem{
		StartTime: start,
		Elapsed:   end.Sub(start),
	}
}

type DummyOutput struct {
}

func (d DummyOutput) DecBusy() {
}

func (d DummyOutput) IncBusy() {
}

func (d DummyOutput) Push(res *OutputItem) {
	data, err := json.Marshal(res)
	log.Infof("Output result: %s %v", data, err)
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
