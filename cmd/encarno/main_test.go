package main

import (
	"encarno/pkg/core"
	"os"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	cfg := "/media/BIG/bzt-artifacts/some/encarno_cfg.yaml"
	if _, err := os.Stat(cfg); err == nil {
		config := LoadConfig(cfg)
		Run(config)
	}
}

func TestOpen(t *testing.T) {
	//log.SetLevel(log.DebugLevel)
	index := core.NewStringIndex("", false)

	ichan := make(core.InputChannel)
	go func() {
		for {
			ichan <- &core.PayloadItem{StrIndex: index, Address: "localhost:8070"}
		}
	}()
	inp := core.InputConf{
		Predefined: ichan,
	}

	c := core.Configuration{
		Input:  inp,
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
		Protocol: core.ProtoConf{Driver: "http"},
	}
	Run(c)
}

func TestClosed(t *testing.T) {
	//log.SetLevel(log.DebugLevel)
	resultFile, err := os.CreateTemp(os.TempDir(), "result_*.ldjson")
	if err != nil {
		panic(err)
	}
	_ = resultFile.Close()

	index := core.NewStringIndex("", false)

	ichan := make(core.InputChannel)
	go func() {
		for {
			ichan <- &core.PayloadItem{StrIndex: index}
		}
	}()
	inp := core.InputConf{
		Predefined: ichan,
	}

	c := core.Configuration{
		Input: inp,
		Output: core.OutputConf{
			LDJSONFile: resultFile.Name(),
		},
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
