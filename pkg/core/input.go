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
	"strings"
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
type ValMap = map[string][]byte

type PayloadItem struct {
	LabelIdx uint16 `json:"l"`
	Label    string `json:"label"`

	AddressIdx uint16 `json:"a"`
	Address    string `json:"address"`

	PayloadLen int `json:"plen"`
	Payload    []byte

	ReplacesIdx []uint16 `json:"r"`
	Replaces    []string `json:"replaces"`

	RegexOutIdx []uint16 `json:"e"`
	RegexOut    map[string]*ExtractRegex

	StrIndex *StrIndex `json:"-"`
}

var regexCache = map[string]*regexp.Regexp{}

func (i *PayloadItem) ReplaceValues(values ValMap) {
	for _, name := range i.Replaces {
		i.ResolveStrings()

		val, ok := values[name]
		if !ok {
			val = []byte("NO_VALUE")
		}

		var re *regexp.Regexp
		if r, ok := regexCache[name]; ok {
			re = r
		} else {
			re = regexp.MustCompile(`(?m:\$\{` + name + "})")
			regexCache[name] = re
		}
		i.Payload = re.ReplaceAll(i.Payload, val)
		i.Label = re.ReplaceAllString(i.Label, string(val))
		i.Address = re.ReplaceAllString(i.Address, string(val))
	}
}

func (i *PayloadItem) ResolveStrings() {
	if i.Address == "" && i.AddressIdx > 0 {
		i.Address = i.StrIndex.Get(i.AddressIdx)
		i.AddressIdx = 0
	}

	if i.Label == "" && i.LabelIdx > 0 {
		i.Label = i.StrIndex.Get(i.LabelIdx)
		i.LabelIdx = 0
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

func (r *ExtractRegex) UnmarshalJSON([]byte) error {

	return nil
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

	var strIndex *StrIndex
	if config.StringsFile != "" {
		strIndex = NewStringIndex(config.StringsFile, true)
	}

	ch := make(InputChannel)
	go func() {
		cnt := 0
		buf := make([]byte, 4096)
		for {
			item, err := ReadPayloadRecord(file, buf, strIndex)
			if err == io.EOF {
				cnt += 1
				if config.IterationLimit > 0 && cnt >= config.IterationLimit {
					break
				}

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
		}
		log.Infof("Input exhausted")
		close(ch)
	}()
	return ch
}

func ReadPayloadRecord(file io.ReadSeeker, buf []byte, index *StrIndex) (*PayloadItem, error) {
	// read buf that hopefully contains meta info
	nread, err := file.Read(buf)
	if err != nil {
		return nil, err
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

	item := &PayloadItem{
		StrIndex: index,
		RegexOut: map[string]*ExtractRegex{},
		Replaces: []string{},
	}
	err = json.Unmarshal(meta, item)
	if err != nil {
		panic(err)
	}

	for _, idx := range item.ReplacesIdx {
		item.Replaces = append(item.Replaces, item.StrIndex.Get(idx))
	}
	item.ReplacesIdx = []uint16{}

	for _, idx := range item.RegexOutIdx {
		s := item.StrIndex.Get(idx)
		name, s, _ := strings.Cut(s, " ")
		match, s, _ := strings.Cut(s, " ")
		group, sre, _ := strings.Cut(s, " ")

		var re *regexp.Regexp
		if r, ok := regexCache[sre]; ok {
			re = r
		} else {
			re = regexp.MustCompile(sre)
			regexCache[sre] = re
		}

		g, _ := strconv.Atoi(group)
		m, _ := strconv.Atoi(match)

		item.RegexOut[name] = &ExtractRegex{
			Re:      re,
			GroupNo: uint(g),
			MatchNo: m,
		}
	}
	item.RegexOutIdx = []uint16{}

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
