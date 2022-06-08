package scenario

import (
	"encarno/pkg/core"
	"reflect"
	"testing"
	"time"
)

func TestClosedGenerator(t *testing.T) {
	scen := ClosedWorkload{
		BaseWorkload: core.BaseWorkload{
			Scenario: []core.WorkloadLevel{
				{0, 10, 5 * time.Second},
				{10, 15, 2 * time.Second},
				{15, 15, 5 * time.Second},
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
