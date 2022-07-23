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

	RegexOutIdx []uint16                 `json:"e"`
	RegexOut    map[string]*ExtractRegex `json:"extracts"`

	AssertsIdx []uint16      `json:"c"`
	Asserts    []*AssertItem `json:"asserts"`

	StrIndex *StrIndex `json:"-"`
}

var regexCache = map[string]*regexp.Regexp{}

func (i *PayloadItem) ReplaceValues(values ValMap) {
	for _, name := range i.Replaces {
		i.ResolveStrings()

		val, ok := values[name]
		if !ok {
			val = []byte("NO_VALUE") // TODO: document that it works like that
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

type RegexpProxy struct {
	*regexp.Regexp
}

func (r *RegexpProxy) UnmarshalText(b []byte) error {
	regex, err := regexp.Compile(string(b))
	if err != nil {
		return err
	}

	r.Regexp = regex

	return nil
}

func (r *RegexpProxy) MarshalText() ([]byte, error) {
	if r.Regexp != nil {
		return []byte(r.Regexp.String()), nil
	}

	return nil, nil
}

type ExtractRegex struct {
	Re      *RegexpProxy
	GroupNo uint // group 0 means whole match that were found
	MatchNo int  // -1 means random
}

func (r *ExtractRegex) String() string {
	return r.Re.String() + " group " + strconv.Itoa(int(r.GroupNo)) + " match " + strconv.Itoa(r.MatchNo)
}

type AssertItem struct {
	Re     *RegexpProxy
	Invert bool
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
		iterations := 0
		good := 0
		bad := 0
		buf := make([]byte, 4096)
		for {
			offset, _ := file.Seek(0, io.SeekCurrent)
			item, err := readPayloadRecord(file, buf, strIndex)
			if err == io.EOF {
				iterations++
				if config.IterationLimit > 0 && iterations >= config.IterationLimit {
					break
				}

				ratio := float64(good) / float64(good+bad)
				if ratio < 0.5 {
					panic(fmt.Sprintf("Payload input file is problematic: %d good records and %d bad records read", good, bad))
				}

				log.Debugf("Rewind payload file")
				_, err = file.Seek(0, 0)
				if err != nil {
					log.Errorf("Failed to rewind the input file")
					panic(err)
				}
				continue
			} else if err != nil {
				log.Errorf("Failed to read payload record at offset %d: %s", offset, err)
				bad++
				continue
			}

			good++
			ch <- item
		}
		log.Infof("Input exhausted")
		close(ch)
	}()
	return ch
}

func readPayloadRecord(file io.ReadSeeker, buf []byte, index *StrIndex) (*PayloadItem, error) {
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
		Asserts:  []*AssertItem{},
	}
	err = json.Unmarshal(meta, item)
	if err != nil {
		log.Warningf("Problematic metadata: %s", meta)
		return nil, errors.New(fmt.Sprintf("Failed to decode metadata: %s", err))
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

		var re = &RegexpProxy{}
		if r, ok := regexCache[sre]; ok {
			re.Regexp = r
		} else {
			r, err := regexp.Compile(sre)
			if err != nil {
				return nil, err
			}
			re.Regexp = r
			regexCache[sre] = re.Regexp
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

	for _, idx := range item.AssertsIdx {
		s := item.StrIndex.Get(idx)
		invert, sre, _ := strings.Cut(s, " ")

		var re = &RegexpProxy{}
		if r, ok := regexCache[sre]; ok {
			re.Regexp = r
		} else {
			r, err := regexp.Compile(sre)
			if err != nil {
				return nil, err
			}
			re.Regexp = r
			regexCache[sre] = re.Regexp
		}

		item.Asserts = append(item.Asserts, &AssertItem{Invert: invert != "0", Re: re})
	}
	item.AssertsIdx = []uint16{}

	// seek payload start
	o, err := file.Seek(-int64(len(rest)), io.SeekCurrent)
	_ = o
	if err != nil {
		log.Warningf("Failed to seek payload start %d", -len(rest))
		panic(err)
	}

	// read payload
	item.Payload = make([]byte, item.PayloadLen)
	n, err := io.ReadFull(file, item.Payload)
	_ = n
	if err != nil {
		log.Warningf("Failed to read payload of len %d", item.PayloadLen)
		panic(err)
	}

	log.Debugf("Produced item: %s", meta)
	return item, nil
}
