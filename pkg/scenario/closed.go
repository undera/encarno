package scenario

import (
	"incarne/pkg/core"
)

// ClosedWorkload implements closed workload scenario
type ClosedWorkload struct {
	BaseWorkload
}

func (s *ClosedWorkload) SpawnForSample(inputs core.InputChannel, x *core.InputItem) {
	if x.TimeOffset > 0 {
		s.SpawnWorker(inputs)
	}
}
