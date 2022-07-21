package core

import (
	"testing"
)

func TestBaseWorkload(t *testing.T) {
	var nm NibMaker
	var out *Output
	iconf := InputConf{Predefined: make(InputChannel)}
	wconf := WorkerConf{
		Values: map[string]string{
			"var": "val",
		},
	}
	status := &Status{}
	wl := NewBaseWorkload(nm, out, iconf, wconf, status)
	_ = wl
}
