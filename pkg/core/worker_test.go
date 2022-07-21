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
			return nil
		},
		InputPayload: func() InputChannel {
			return nil
		},
	}
	sc := make(ScheduleChannel)
	w := NewBasicWorker(0, abrt, wl, sc, vals)
	_ = w
}
