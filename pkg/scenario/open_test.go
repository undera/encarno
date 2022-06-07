package scenario

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestExternal(t *testing.T) {
	maker := func() core.Nib {
		nib := core.DummyNib{}
		return &nib
	}

	scen := NewOpenWorkload(core.WorkerConf{}, core.InputConf{Predefined: DummyGenerator()}, maker, &dummyOutput{Status: new(core.Status)})

	scen.Run()
	log.Infof("Final sleep")
	time.Sleep(5 * time.Second)
}

func DummyGenerator() core.InputChannel {
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

type dummyOutput struct {
}

func (d dummyOutput) Close() {

}

func (d dummyOutput) Start(output core.OutputConf) {

}

func (d dummyOutput) Push(res *core.OutputItem) {

}

func TestOpenGenerator(t *testing.T) {
	scen := OpenWorkload{
		BaseWorkload: &core.BaseWorkload{
			Scenario: []core.WorkloadLevel{
				{0, 10, 5 * time.Second},
				{10, 10, 2 * time.Second},
				//{15, 15, 5 * time.Second},
			},
		},
	}

	vals := make([]time.Duration, 0)
	for offset := range scen.GenerateSchedule() {
		t.Logf("%v", offset)
		vals = append(vals, offset)
	}

	exp := []time.Duration{0, 0, 0, 0, 0}

	if len(vals) != 15 {
		t.Errorf("Wrong len: %d", len(vals))
	}

	if !reflect.DeepEqual(vals, exp) {
		t.Errorf("%v!=%v", vals, exp)
	}
}
