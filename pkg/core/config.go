package core

import "gopkg.in/yaml.v3"

type Configuration struct {
	Input    InputConf
	Output   OutputConf
	Workers  WorkerConf
	Protocol ProtoConf
}

type InputConf struct {
}

type OutputConf struct {
	// detailed log path
	// error log path
	// binary log path
	// csv log path
	// logging print options
}

type WorkloadMode = string

const (
	WorkloadOpen   WorkloadMode = "open"
	WorkloadClosed WorkloadMode = "closed"
)

type WorkerConf struct {
	Mode WorkloadMode
}

type ProtoConf struct {
	Driver   string
	FullText []byte
}

func (e *ProtoConf) UnmarshalYAML(value *yaml.Node) error {
	panic("TODO")
	return nil
}
