package core

import (
	"bufio"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"time"
)

type OutputConf struct {
	LDJSONFile       string
	ReqRespFile      string
	ReqRespFileLevel int

	// TODO CSVFile string
	// TODO BinaryFile string
}

type Output interface {
	Start(output OutputConf)
	Push(res *OutputItem)
	Close()
}

type OutputItem struct {
	SentBytesCount int
	RespBytesCount int
	Label          string
	ReqBytes       []byte `json:"-"`
	RespBytes      []byte `json:"-"`
	Error          error  `json:"-"`
	ErrorStr       string // for JSON reader
	Status         int
	StartTime      time.Time `json:"-"`
	StartTS        int64     // for result readers, to avoid date parsing
	Concurrency    int64
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
	pipe chan *OutputItem
}

func (m *MultiFileOutput) Close() {
	log.Infof("Closing output")
	for _, out := range m.Outs {
		out.Close()
	}
}

func (m *MultiFileOutput) Start(OutputConf) {
	go m.Background()
}

func (m *MultiFileOutput) Push(res *OutputItem) {
	m.pipe <- res
}

func (m *MultiFileOutput) Background() {
	for {
		res := <-m.pipe
		for _, out := range m.Outs {
			out.Push(res)
		}
	}
}

func NewMultiOutput(conf OutputConf) Output {
	out := MultiFileOutput{
		Outs: make([]SingleOut, 0),
		pipe: make(chan *OutputItem),
	}

	if conf.LDJSONFile != "" {
		log.Infof("Opening result file for writing: %s", conf.LDJSONFile)
		file, err := os.OpenFile(conf.LDJSONFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}

		out.Outs = append(out.Outs, &LDJSONOut{
			fd:     file,
			writer: bufio.NewWriter(file),
		})
	}

	if conf.ReqRespFile != "" {
		log.Infof("Opening result file for writing: %s", conf.ReqRespFile)
		file, err := os.OpenFile(conf.ReqRespFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}

		out.Outs = append(out.Outs, &ReqRespOut{
			fd:     file,
			writer: bufio.NewWriter(file),
			Level:  conf.ReqRespFileLevel,
		})
	}

	return &out
}

type SingleOut interface {
	Push(*OutputItem)
	Close()
}

type LDJSONOut struct {
	writer *bufio.Writer
	fd     *os.File
}

func (L *LDJSONOut) Push(item *OutputItem) {
	data, err := json.Marshal(item)
	if err != nil {
		panic(err)
	}
	data = append(data, 13) // \r\n
	data = append(data, 10) // \n

	_, err = L.writer.Write(data)
	if err != nil {
		panic(err)
	}
}

func (L *LDJSONOut) Close() {
	_ = L.writer.Flush()
	_ = L.fd.Close()
}

type ReqRespOut struct {
	writer *bufio.Writer
	fd     *os.File
	Level  int // 0 would write all, 400 - all above 400, 600 - all non-http
}

func (d ReqRespOut) Push(item *OutputItem) {
	if item.Status >= d.Level {
		// meta
		data, err := json.Marshal(item)
		if err != nil {
			panic(err)
		}
		data = append(data, 13) // \r\n
		data = append(data, 10) // \n

		_, err = d.writer.Write(data)
		if err != nil {
			panic(err)
		}

		_, _ = d.writer.Write(item.ReqBytes)
		_, _ = d.writer.Write([]byte{13, 10})
		_, _ = d.writer.Write(item.RespBytes)
		_, _ = d.writer.Write([]byte{13, 10})
	}
}

func (d ReqRespOut) Close() {
	_ = d.writer.Flush()
	_ = d.fd.Close()
}
