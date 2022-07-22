package core

import (
	"testing"
)

func TestNewBasicWorker(t *testing.T) {
	vals := ValMap{
		"var": []byte("val"),
	}
	abrt := make(chan struct{})
	wl := &BaseWorkload{
		NibMaker: func() Nib {
			return DummyNib{}
		},
		InputPayload: func() InputChannel {
			return nil
		},
		Status: NewStatus(),
		Output: NewOutput(OutputConf{}),
	}
	sc := make(ScheduleChannel)
	w := NewBasicWorker(0, abrt, wl, sc, vals)
	_ = w.DoBusy(&PayloadItem{StrIndex: &StrIndex{}})
}
