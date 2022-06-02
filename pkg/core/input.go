package core

import (
	log "github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"time"
)

// input file can contain timestamps, or can rely on internal schedule calculator
// internal schedule calculator: warmup, ramp-up, steps, constant; for workers and for rps

type InputConf struct {
	PayloadFile  string
	ScheduleFile string
	StringsFile  string
}

type InputChannel chan *InputItem

type Input interface {
	Start(input InputConf) InputChannel
	Clone() Input
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

func NewInput(config InputConf) InputChannel {
	ch := make(InputChannel)
	go func() {
		for i := 0; i < 1000; i++ {
			ch <- &InputItem{
				TimeOffset: time.Duration(i) * time.Millisecond,
			}
		}
		log.Infof("Input exhausted")
		close(ch)
	}()
	return ch
}
