package main

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"testing"
	"time"
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
					LevelStart: 0,
					LevelEnd:   10,
					Duration:   5 * time.Second,
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
					LevelStart: 0,
					LevelEnd:   10,
					Duration:   5 * time.Second,
				},
			},
		},
		Protocol: core.ProtoConf{Driver: "dummy"},
	}
	Run(c)
}
