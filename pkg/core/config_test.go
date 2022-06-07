package core

import (
	"gopkg.in/yaml.v3"
	"testing"
	"time"
)

func TestConfDump(t *testing.T) {
	data, err := yaml.Marshal(Configuration{
		Workers: WorkerConf{
			WorkloadSchedule: []WorkloadLevel{
				{
					LevelStart: 0,
					LevelEnd:   10,
					Duration:   5 * time.Second,
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	t.Logf("\n:%s", data)
}

func TestConfLoad(t *testing.T) {
	txt := `{}`
	res := new(Configuration)
	err := yaml.Unmarshal([]byte(txt), res)
	if err != nil {
		panic(err)
	}
	t.Logf("\n:%v", res)
}
