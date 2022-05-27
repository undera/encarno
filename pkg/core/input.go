package core

// reading input files
// input file can contain timestamps, or can rely on internal schedule calculator
// internal schedule calculator: warmup, ramp-up, steps, constant; for workers and for rps
// request has label

type InputChannel chan *InputItem

type Input interface {
	Generator() InputChannel
}

type InputItem struct {
	TimeOffset int64
	Label      string
	HostID     uint
	Payload    []byte
}
