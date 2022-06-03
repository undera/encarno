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
	maker := func() core.Nib {
		nib := DummyNib{}
		return &nib
	}

	scen := NewOpenWorkload(core.WorkerConf{}, core.InputConf{Predefined: &DummyInput{}}, maker, &dummyOutput{})

	scen.Run()
	log.Infof("Final sleep")
	time.Sleep(5 * time.Second)
}

type DummyInput struct {
}

func (d *DummyInput) Start(input core.InputConf) core.InputChannel {
	//TODO implement me
	panic("implement me")
}

func (d *DummyInput) Clone() core.Input {
	//TODO implement me
	panic("implement me")
}

func (d *DummyInput) Generator() core.InputChannel {
	ch := make(core.InputChannel)
	go func() {
		defer close(ch)
		for i := 0; i < 1000; i++ {
			log.Infof("Iteration %d", i)
			item := &core.PayloadItem{
				//TimeOffset: time.Duration(i) * 1000 * time.Millisecond,
				Label:   "label#" + strconv.Itoa(i%3),
				Payload: []byte("data"),
			}

			ch <- item
		}
	}()
	return ch
}

type DummyNib struct {
}

func (n *DummyNib) Punch(item *core.PayloadItem) *core.OutputItem {
	start := time.Now()
	log.Infof("Processed payload: %s", item.Payload)
	end := time.Now()
	return &core.OutputItem{
		StartTime: start,
		Elapsed:   end.Sub(start),
	}
}

type dummyOutput struct {
}

func (d dummyOutput) Start(output core.OutputConf) {
	//TODO implement me
	panic("implement me")
}

func (d dummyOutput) Push(res *core.OutputItem) {
	//TODO implement me
	panic("implement me")
}
