package main

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"testing"
)

func TestOpen(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	c := core.Configuration{
		Input:  core.InputConf{},
		Output: core.OutputConf{},
		Workers: core.WorkerConf{
			Mode: core.WorkloadOpen,
			WorkloadSchedule: []core.WorkloadLevel{
				{
					Level:  100,
					RampUp: 3,
					Steady: 5,
				},
			},
		},
		Protocol: core.ProtoConf{Driver: "dummy"},
	}
	Run(c)
}

func TestClosed(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	c := core.Configuration{
		Input:  core.InputConf{},
		Output: core.OutputConf{},
		Workers: core.WorkerConf{
			Mode: core.WorkloadClosed,
			WorkloadSchedule: []core.WorkloadLevel{
				{
					Level:  10,
					RampUp: 3,
					Steady: 5,
				},
			},
		},
		Protocol: core.ProtoConf{Driver: "dummy"},
	}
	Run(c)
}
