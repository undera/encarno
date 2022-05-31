package core

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
}

type WorkerConf struct {
	// workload open/closed
}

type ProtoConf struct {
}
