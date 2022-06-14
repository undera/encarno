package core

import (
	"bufio"
	"fmt"
	"os"
)

type StrIndex struct {
	filename string
	index    []string
}

func NewStringIndex(fname string) *StrIndex {
	ret := StrIndex{
		filename: fname,
		index:    []string{""}, // placeholder for zero index
	}
	ret.Load()
	return &ret
}

func (s *StrIndex) Add(str string) uint16 {
	panic("TODO")
	// add to map, add to file, flush file
}

func (s *StrIndex) Get(idx uint16) string {
	if int(idx) >= len(s.index) {
		panic(fmt.Sprintf("String #%d not found in index: %s", idx, s.filename))
	}

	return s.index[idx]
}

func (s *StrIndex) Load() {
	file, err := os.Open(s.filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		s.index = append(s.index, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
