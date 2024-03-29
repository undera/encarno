package core

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

type OutputConf struct {
	LDJSONFile       string
	ReqRespFile      string
	ReqRespFileLevel uint16
	BinaryFile       string
	StringsFile      string

	// TODO CSVFile string
}

type OutputItem struct { // all fields should have fixed types
	StartTime time.Time `json:"-"`
	StartTS   uint32    // for result readers, to avoid date parsing

	Status      uint16
	Error       error  `json:"-"`
	ErrorStr    string // for JSON reader
	ErrorStrIdx uint16 `json:"-"`

	Concurrency uint32

	Elapsed       time.Duration
	ConnectTime   time.Duration
	SentTime      time.Duration
	FirstByteTime time.Duration
	ReadTime      time.Duration

	Worker   uint32
	Label    string
	LabelIdx uint16 `json:"-"`

	SentBytesCount uint64
	RespBytesCount uint64
	ReqBytes       []byte `json:"-"`
	RespBytes      []byte `json:"-"`

	strIndex *StrIndex
}

func (i *OutputItem) EndWithError(err error) *OutputItem {
	i.Status = 999
	i.Error = err
	return i
}

func (i *OutputItem) ExtractValues(extractors map[string]*ExtractRegex, values ValMap) {
	placeholder := []byte("NOT_FOUND") // TODO: parameterize it
	for name, outSpec := range extractors {
		limit := outSpec.MatchNo + 1
		if outSpec.MatchNo < 0 {
			limit = outSpec.MatchNo
		}

		all := outSpec.Re.FindAllSubmatch(i.RespBytes, limit)

		var val []byte
		if len(all) <= 0 {
			log.Debugf("Nothing has matched the regex '%s': %v", name, outSpec.String())
			val = placeholder
		} else if outSpec.MatchNo >= 0 {
			val = all[outSpec.MatchNo][outSpec.GroupNo]
		} else {
			val = all[rand.Intn(len(all))][outSpec.GroupNo]
		}

		values[name] = make([]byte, len(val))
		copy(values[name], val)
	}
}

func (i *OutputItem) Assert(asserts []*AssertItem) {
	problems := ""
	if i.Error != nil {
		problems = i.Error.Error()
	}

	for _, a := range asserts {
		found := a.Re.Find(i.RespBytes) == nil
		if found != a.Invert {
			i := ""
			if a.Invert {
				i = "inverted "
			}
			problems += fmt.Sprintf("\nAssert failed on %sregexp: %s", i, a.Re)
		}
	}

	if problems != "" {
		i.Error = errors.New(strings.TrimSpace(problems))
	}
}

func (i *OutputItem) WriteBinary(fd io.Writer) {
	endian := binary.LittleEndian
	err := binary.Write(fd, endian, i.StartTS) // TODO: nano?
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.Status)
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.ErrorStrIdx)
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.Concurrency)
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.Elapsed.Seconds())
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.ConnectTime.Seconds())
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.SentTime.Seconds())
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.FirstByteTime.Seconds())
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.ReadTime.Seconds())
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.Worker)
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.LabelIdx)
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.SentBytesCount)
	if err != nil {
		panic(err)
	}

	err = binary.Write(fd, endian, i.RespBytesCount)
	if err != nil {
		panic(err)
	}
}

func (i *OutputItem) StringFriendly() {
	if i.Error != nil {
		i.ErrorStr = i.Error.Error()
	}

	if i.Label == "" && i.LabelIdx > 0 {
		i.Label = i.strIndex.Get(i.LabelIdx)
	}
}

type Output struct {
	Outs []SingleOut

	pipe     chan *OutputItem
	strIndex *StrIndex
}

func (m *Output) Close() {
	log.Infof("Closing output")
	for _, out := range m.Outs {
		out.Close()
	}
}

func (m *Output) Start(OutputConf) {
	go m.background()
}

func (m *Output) Push(res *OutputItem) {
	res.strIndex = m.strIndex
	m.pipe <- res
}

func (m *Output) background() {
	for {
		res := <-m.pipe
		for _, out := range m.Outs {
			out.Push(res)
		}
	}
}

func NewOutput(conf OutputConf) *Output {
	out := Output{
		Outs: make([]SingleOut, 0),
		pipe: make(chan *OutputItem),
	}

	if conf.StringsFile != "" {
		out.strIndex = NewStringIndex(conf.StringsFile, false)
	}

	if conf.LDJSONFile != "" {
		log.Infof("Opening LDJSON file for writing: %s", conf.LDJSONFile)
		file, err := os.OpenFile(conf.LDJSONFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}

		out.Outs = append(out.Outs, &LDJSONOut{
			fd:     file,
			writer: bufio.NewWriter(file),
			mx:     new(sync.Mutex),
		})
	}

	if conf.BinaryFile != "" {
		log.Infof("Opening binary file for writing: %s", conf.BinaryFile)
		file, err := os.OpenFile(conf.BinaryFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}

		out.Outs = append(out.Outs, &BinaryOut{
			fd:     file,
			writer: bufio.NewWriter(file),
			mx:     new(sync.Mutex),
		})
	}

	if conf.ReqRespFile != "" {
		log.Infof("Opening trace file for writing: %s", conf.ReqRespFile)
		file, err := os.OpenFile(conf.ReqRespFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}

		out.Outs = append(out.Outs, &ReqRespOut{
			fd:     file,
			writer: bufio.NewWriter(file),
			Level:  conf.ReqRespFileLevel,
		})
	}

	out.Start(conf)
	return &out
}

type SingleOut interface {
	Push(*OutputItem)
	Close()
}

type LDJSONOut struct {
	writer *bufio.Writer
	fd     *os.File
	mx     *sync.Mutex
}

func (L *LDJSONOut) Push(item *OutputItem) {
	item.StringFriendly()
	data, err := json.Marshal(item)
	if err != nil {
		panic(err)
	}
	data = append(data, 13) // \r\n
	data = append(data, 10) // \n

	L.mx.Lock()
	defer L.mx.Unlock()
	_, err = L.writer.Write(data)
	if err != nil {
		panic(err)
	}
}

func (L *LDJSONOut) Close() {
	L.mx.Lock()
	defer L.mx.Unlock()
	_ = L.writer.Flush()
	_ = L.fd.Close()
}

type ReqRespOut struct {
	writer *bufio.Writer
	fd     *os.File
	Level  uint16 // 0 would write all, 400 - all above 400, 600 - all non-http
}

func (d ReqRespOut) Push(item *OutputItem) {
	if item.Status >= d.Level {
		item.StringFriendly()
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

type BinaryOut struct {
	fd     *os.File
	writer *bufio.Writer
	lastTS uint32
	mx     *sync.Mutex
}

func (o *BinaryOut) Close() {
	_ = o.writer.Flush()
	_ = o.fd.Close()
}

func (o *BinaryOut) Push(item *OutputItem) {
	if item.Error != nil && item.ErrorStrIdx == 0 {
		item.ErrorStrIdx = item.strIndex.Idx(item.Error.Error())
	}

	if item.Label != "" && item.LabelIdx == 0 {
		item.LabelIdx = item.strIndex.Idx(item.Label)
	}

	item.WriteBinary(o.writer)

	if item.StartTS > o.lastTS {
		_ = o.writer.Flush()
		o.lastTS = item.StartTS
	}
}
