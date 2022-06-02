package core

import "gopkg.in/yaml.v3"

type Configuration struct {
	Input    InputConf
	Output   OutputConf
	Workers  WorkerConf
	Protocol ProtoConf
}

type ProtoConf struct {
	Driver   string
	FullText []byte
}

func (e *ProtoConf) UnmarshalYAML(value *yaml.Node) error {
	panic("TODO")
	return nil
}
