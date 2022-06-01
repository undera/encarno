package core

import (
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

type Output interface {
	Start(output OutputConf)
	Push(res *OutputItem)
}

type Status interface {
	DecBusy()
	IncBusy()
	IncSleeping()
	DecSleeping()
	IncWorking()
	DecWorking()
	GetWorking() int64
	GetSleeping() int64
	GetBusy() int64
}

// get result from worker via channel
// write small binary results
// write full request/response for debugging
// write only failures request/response

type OutputItem struct {
	SentBytesCount int
	RespBytesCount int
	RespBytes      []byte
	Error          error
	Status         int
	StartTime      time.Time
	ConnectTime    time.Duration
	SentTime       time.Duration
	FirstByteTime  time.Duration
	ReadTime       time.Duration
	Elapsed        time.Duration
	StartMissed    time.Duration
}

func (i *OutputItem) EndWithError(err error) *OutputItem {
	i.Status = 999
	i.Error = err
	return i
}

func (i *OutputItem) ExtractValues(extractors map[string]*ExtractRegex, values map[string][]byte) {
	for name, outSpec := range extractors {
		all := outSpec.Re.FindAllSubmatch(i.RespBytes, -1)

		if len(all) <= 0 {
			log.Warningf("Nothing has matched the regex '%s': %v", name, outSpec.String())
		} else if outSpec.MatchNo >= 0 {
			values[name] = all[outSpec.MatchNo][outSpec.GroupNo]
		} else {
			values[name] = all[rand.Intn(len(all))][outSpec.GroupNo]
		}
	}
}
