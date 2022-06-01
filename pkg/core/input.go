package core

import (
	"regexp"
	"strconv"
	"time"
)

// reading input files
// input file can contain timestamps, or can rely on internal schedule calculator
// internal schedule calculator: warmup, ramp-up, steps, constant; for workers and for rps
// request has label

type InputChannel chan *InputItem

type Input interface {
	Start(input InputConf) InputChannel
}

type InputItem struct {
	TimeOffset time.Duration
	Label      string
	Hostname   string
	Payload    []byte
	RegexOut   map[string]*ExtractRegex
}

func (i *InputItem) ReplaceValues(values map[string][]byte) {
	// TODO: only do it for selected values
	for name, val := range values {
		re := regexp.MustCompile("\\$\\{" + name + "}")
		i.Payload = re.ReplaceAll(i.Payload, val)
	}
}

type ExtractRegex struct {
	Re      *regexp.Regexp
	GroupNo uint // group 0 means whole match that were found
	MatchNo int  // -1 means random
}

func (r *ExtractRegex) String() string {
	return r.Re.String() + " group " + strconv.Itoa(int(r.GroupNo)) + " match " + strconv.Itoa(r.MatchNo)
}
