package scenario

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestExternal(t *testing.T) {
	scen := External{
		Workers:    make([]core.Worker, 0),
		Input:      &DummyInput{},
		Output:     &DummyOutput{},
		MinWorkers: 0,
		MaxWorkers: 0,
		StartTime:  time.Now().UnixNano(),
		NibMaker: func() core.Nib {
			nib := DummyNib{}
			return &nib
		},
	}

	scen.Run()
	time.Sleep(5 * time.Second)
}

type DummyInput struct {
}

func (d *DummyInput) Generator() core.InputChannel {
	ch := make(core.InputChannel)
	go func() {
		defer close(ch)
		for i := 0; i < 100; i++ {
			log.Infof("Iteration %d", i)
			inc := 1000 * int64(time.Millisecond)
			item := &core.InputItem{
				TimeOffset: int64(i) * inc,
				Label:      "label#" + strconv.Itoa(i%3),
				Payload:    []byte("data"),
			}

			ch <- item
		}
	}()
	return ch
}

type DummyOutput struct {
	sleeping int64
	busy     int64
	working  int64
}

func (o *DummyOutput) GetWorking() int64 {
	return o.working
}

func (o *DummyOutput) GetSleeping() int64 {
	return o.sleeping
}

func (o *DummyOutput) GetBusy() int64 {
	return o.busy
}

func (o *DummyOutput) IncWorking() {
	atomic.AddInt64(&o.working, 1)
}

func (o *DummyOutput) DecWorking() {
	atomic.AddInt64(&o.working, -1)
	if o.working < 0 {
		panic("Counter cannot be negative")
	}
}

func (o *DummyOutput) IncSleeping() {
	atomic.AddInt64(&o.sleeping, 1)
	if o.sleeping < 0 {
		panic("Counter cannot be negative")
	}
}

func (o *DummyOutput) DecSleeping() {
	atomic.AddInt64(&o.sleeping, -1)
}

func (o *DummyOutput) Push(res *core.OutputItem) {
	data, err := json.Marshal(res)
	log.Infof("Output result: %s %v", data, err)
}

func (o *DummyOutput) DecBusy() {
	atomic.AddInt64(&o.busy, -1)
	if o.busy < 0 {
		panic("Counter cannot be negative")
	}
}

func (o *DummyOutput) IncBusy() {
	atomic.AddInt64(&o.busy, 1)
}

type DummyNib struct {
}

func (n *DummyNib) Punch(payload []byte) *core.OutputItem {
	start := time.Now().UnixNano()
	log.Infof("Processed payload: %s", payload)
	end := time.Now().UnixNano()
	return &core.OutputItem{
		Start: start,
		End:   end,
	}
}
