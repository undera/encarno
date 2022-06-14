package core

import "fmt"

type StrIndex struct {
	filename string
	index    map[uint16]string
}

func NewStringIndex(fname string) *StrIndex {
	ret := StrIndex{
		filename: fname,
		index:    map[uint16]string{},
	}
	ret.Load()
	return &ret
}

func (s *StrIndex) Add(str string) uint16 {
	panic("TODO")
	// add to map, add to file, flush file
}

func (s *StrIndex) Get(idx uint16) string {
	if res, ok := s.index[idx]; ok {
		return res
	}
	panic(fmt.Sprintf("String #%d not found in index: %s", idx, s.filename))
}

func (s *StrIndex) Load() {
	// TODO: if file exists, load initial values from it
}
