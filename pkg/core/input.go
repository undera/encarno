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
	PayloadFile    string
	StringsFile    string
	EnableRegexes  bool
	Predefined     Input
	IterationLimit int
}

type InputChannel chan *PayloadItem
type ScheduleChannel chan time.Duration

type Input interface {
	Start(input InputConf) InputChannel
	Clone() Input
}

type PayloadItem struct {
	Label    string
	Hostname string
	Payload  []byte
	RegexOut map[string]*ExtractRegex
}

func (i *PayloadItem) ReplaceValues(values map[string][]byte) {
	// TODO: only do it for selected Values
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
	if config.Predefined != nil {
		return config.Predefined.Start(config)
	}

	ch := make(InputChannel)
	go func() {
		cnt := 0
		for {
			ch <- &PayloadItem{
				//TimeOffset: time.Duration(i) * time.Millisecond,
			}
			cnt += 1
			if config.IterationLimit > 0 && cnt >= config.IterationLimit {
				break
			}
		}
		log.Infof("Input exhausted")
		close(ch)
	}()
	return ch
}
