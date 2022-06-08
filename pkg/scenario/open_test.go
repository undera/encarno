package scenario

import (
	"encarno/pkg/core"
	log "github.com/sirupsen/logrus"
	"testing"
	"time"
)

func init() {
	log.SetLevel(log.DebugLevel)
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

	if len(vals) != 55 {
		t.Errorf("Wrong len: %d", len(vals))
	}
}
