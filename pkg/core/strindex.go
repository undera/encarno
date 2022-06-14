package core

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

type StrIndex struct {
	filename string
	index    []string
	mapping  map[string]uint16
	mx       *sync.Mutex
	fd       *os.File
}

func NewStringIndex(fname string) *StrIndex {
	ret := StrIndex{
		filename: fname,
		index:    []string{""},             // placeholder for zero index
		mapping:  map[string]uint16{"": 0}, // reverse index
		mx:       new(sync.Mutex),
	}

	if fname != "" {
		ret.Load()
	}
	return &ret
}

func (s *StrIndex) Load() {
	s.mx.Lock()
	defer s.mx.Unlock()
	file, err := os.Open(s.filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		text := scanner.Text()
		s.index = append(s.index, text)
		s.mapping[text] = uint16(len(s.index) - 1)
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func (s *StrIndex) Get(idx uint16) string {
	if int(idx) >= len(s.index) {
		panic(fmt.Sprintf("String #%d not found in index: %s", idx, s.filename))
	}

	return s.index[idx]
}

func (s *StrIndex) Idx(label string) uint16 {
	// optimistic attempt with no mutex
	if idx, ok := s.mapping[label]; ok {
		return idx
	} else {
		s.mx.Lock()
		defer s.mx.Unlock()
		if idx, ok := s.mapping[label]; ok { // repeat the attempt under mutex
			return idx
		}
		s.index = append(s.index, label)
		idx := uint16(len(s.index) - 1)
		s.mapping[label] = idx

		s.appendFile(label)

		return idx
	}
}

func (s *StrIndex) appendFile(label string) {
	if s.filename != "" {
		if s.fd == nil { // lazy open file
			f, err := os.OpenFile(s.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // TODO: need to close it?
			if err != nil {
				panic(err)
			}
			s.fd = f
		}

		if _, err := s.fd.WriteString(label + "\n"); err != nil {
			panic(err)
		}
	}
}
