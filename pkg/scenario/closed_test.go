package scenario

import (
	"encarno/pkg/core"
	"reflect"
	"testing"
	"time"
)

func TestClosedGenerator(t *testing.T) {
	ichan := make(core.InputChannel)
	go func() {
		for {
			ichan <- &core.PayloadItem{}
		}
	}()
	inp := core.InputConf{
		Predefined: ichan,
	}

	maker := func() core.Nib {
		return core.DummyNib{}
	}

	out := &core.MultiFileOutput{}
	wconf := core.WorkerConf{
		WorkloadSchedule: []core.WorkloadLevel{
			{0, 10, 5 * time.Second},
			{10, 15, 2 * time.Second},
			{15, 15, 5 * time.Second},
		},
	}

	status := &core.Status{}

	base := core.NewBaseWorkload(maker, out, inp, wconf, status)
	scen := NewClosedWorkload(inp, base)

	vals := make([]time.Duration, 0)
	for offset := range scen.GenerateSchedule() {
		t.Logf("%v", offset)
		vals = append(vals, offset)
	}

	exp := []time.Duration{500 * time.Millisecond, 1 * time.Second, 1500 * time.Millisecond, 2 * time.Second, 2500 * time.Millisecond, 3 * time.Second, 3500 * time.Millisecond, 4 * time.Second, 4500 * time.Millisecond, 5 * time.Second, 5400 * time.Millisecond, 5800 * time.Millisecond, 6200 * time.Millisecond, 6600 * time.Millisecond, 7 * time.Second}

	if len(vals) != 15 {
		t.Errorf("Wrong len: %d", len(vals))
	}

	if !reflect.DeepEqual(vals, exp) {
		t.Errorf("%v!=%v", vals, exp)
	}
}
