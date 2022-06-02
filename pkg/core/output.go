package core

import (
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

type OutputConf struct {
	LDJSONFile string
	// TODO CSVFile string
	// TODO BinaryFile string
	// TODO ReqRespFile string
	// TODO ReqRespLevel string - all, status>=400, errors only
}

type Output interface {
	Start(output OutputConf)
	Push(res *OutputItem)
}

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

type MultiFileOutput struct {
	Outs []SingleOut

	// get result from worker via channel
	// write small binary results
	// write full request/response for debugging
	// write only failures request/response
}

func (m *MultiFileOutput) Start(output OutputConf) {
	// TODO: do we need it?
}

func (m *MultiFileOutput) Push(res *OutputItem) {
	for _, out := range m.Outs {
		out.Push(res)
	}
}

func NewMultiOutput(conf OutputConf) Output {
	out := MultiFileOutput{
		Outs: make([]SingleOut, 0),
	}

	if conf.LDJSONFile != "" {
		out.Outs = append(out.Outs, &LDJSONOut{})
	}

	return &out
}

type SingleOut interface {
	Push(*OutputItem)
	Close()
}

type LDJSONOut struct {
}

func (L *LDJSONOut) Push(item *OutputItem) {
	//TODO implement me
	panic("implement me")
}

func (L *LDJSONOut) Close() {
	//TODO implement me
	panic("implement me")
}
