package core

import (
	"time"
)

type Configuration struct {
	Input    InputConf
	Output   OutputConf
	Workers  WorkerConf
	Protocol ProtoConf
}

type TLSConf struct {
	InsecureSkipVerify bool
	TLSCipherSuites    []string
	MinVersion         uint16
	MaxVersion         uint16
}

type ProtoConf struct {
	Driver         string
	MaxConnections int
	Timeout        time.Duration
	TLSConf        TLSConf
}
