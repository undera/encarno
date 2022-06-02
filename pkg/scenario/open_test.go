package scenario

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"strconv"
	"testing"
	"time"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestExternal(t *testing.T) {
	scen := OpenWorkload{
		BaseWorkload: BaseWorkload{
			Workers:   make([]core.Worker, 0),
			Input:     &DummyInput{},
			Output:    &DummyOutput{},
			StartTime: time.Now(),
			NibMaker: func() core.Nib {
				nib := DummyNib{}
				return &nib
			},
		},
		MinWorkers: 0,
		MaxWorkers: 0,
	}
	scen.Spawner = &scen

	scen.Run()
	log.Infof("Final sleep")
	time.Sleep(5 * time.Second)
}

type DummyInput struct {
}

func (d *DummyInput) Generator() core.InputChannel {
	ch := make(core.InputChannel)
	go func() {
		defer close(ch)
		for i := 0; i < 1000; i++ {
			log.Infof("Iteration %d", i)
			item := &core.InputItem{
				TimeOffset: time.Duration(i) * 1000 * time.Millisecond,
				Label:      "label#" + strconv.Itoa(i%3),
				Payload:    []byte("data"),
			}

			ch <- item
		}
	}()
	return ch
}

type DummyNib struct {
}

func (n *DummyNib) Punch(payload []byte) *core.OutputItem {
	start := time.Now()
	log.Infof("Processed payload: %s", payload)
	end := time.Now()
	return &core.OutputItem{
		StartTime: start,
		Elapsed:   end.Sub(start),
	}
}
