package main

import (
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"testing"
)

func TestOpen(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	c := core.Configuration{
		Input:    core.InputConf{},
		Output:   core.OutputConf{},
		Workers:  core.WorkerConf{Mode: core.WorkloadOpen},
		Protocol: core.ProtoConf{Driver: "dummy"},
	}
	Run(c)
}
