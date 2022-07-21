package core

import (
	"testing"
)

func TestBaseWorkload(t *testing.T) {
	var nm NibMaker = func() Nib {
		return nil
	}
	var out *Output
	iconf := InputConf{Predefined: make(InputChannel)}
	wconf := WorkerConf{
		Values: map[string]string{
			"var": "val",
		},
	}
	status := &Status{}
	wl := NewBaseWorkload(nm, out, iconf, wconf, status)
	wl.SpawnWorker(make(ScheduleChannel))
	if len(wl.Workers) != 1 {
		t.Errorf("No worker spawned")
	}
}
