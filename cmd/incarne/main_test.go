package main

import (
	"incarne/pkg/core"
	"testing"
)

func TestOpen(t *testing.T) {
	c := core.Configuration{
		Input:    core.InputConf{},
		Output:   core.OutputConf{},
		Workers:  core.WorkerConf{Mode: core.WorkloadOpen},
		Protocol: core.ProtoConf{Driver: "dummy"},
	}
	Run(c)
}
