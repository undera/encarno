package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
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
	Predefined     InputChannel `yaml:"-"`
	IterationLimit int
}

type InputChannel chan *PayloadItem
type ScheduleChannel chan time.Duration

type PayloadItem struct {
	Label      string
	Hostname   string
	Payload    []byte
	PayloadLen int
	RegexOut   map[string]*ExtractRegex
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
		return config.Predefined
	}

	log.Infof("Opening payload input file: %s", config.PayloadFile)
	file, err := os.Open(config.PayloadFile)
	if err != nil {
		panic(err)
	}

	ch := make(InputChannel)
	go func() {
		cnt := 0
		buf := make([]byte, 4096)
		for {
			item, err := ReadPayloadRecord(file, buf)
			if err == io.EOF {
				log.Debugf("Rewind payload file")
				_, err = file.Seek(0, 0)
				if err != nil {
					panic(err)
				}
				continue
			} else if err != nil {
				panic(err)
			}

			ch <- item
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

func ReadPayloadRecord(file *os.File, buf []byte) (*PayloadItem, error) {
	// read buf that hopefully contains meta info
	nread, err := file.Read(buf)
	if err != nil {
		panic(err)
	}

	// skip inter-record separators
	offset := 0
	for offset < nread && (buf[offset] == 10 || buf[offset] == 13) {
		offset++
	}

	if offset == nread {
		return nil, io.EOF
	}

	// read meta
	meta, rest, found := bytes.Cut(buf[offset:nread], []byte{10})
	if !found {
		return nil, errors.New(fmt.Sprintf("Meta information line did not contain the newline within %d bytes buffer: %s", nread, buf[:nread]))
	}

	item := new(PayloadItem)
	err = json.Unmarshal(meta, item)
	if err != nil {
		panic(err)
	}

	// seek payload start
	o, err := file.Seek(-int64(len(rest)), 1)
	_ = o
	if err != nil {
		panic(err)
	}

	// read payload
	item.Payload = make([]byte, item.PayloadLen)
	n, err := io.ReadFull(file, item.Payload)
	_ = n
	if err != nil {
		panic(err)
	}

	log.Debugf("Produced item: %s", meta)
	return item, nil
}
