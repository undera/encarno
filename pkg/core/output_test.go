package core

import (
	"io"
	"os"
	"regexp"
	"testing"
)

func tmp() string {
	resultFile, err := os.CreateTemp(os.TempDir(), "encarno_*.tmp")
	if err != nil {
		panic(err)
	}
	_ = resultFile.Close()
	return resultFile.Name()
}

func TestOutput(t *testing.T) {
	cfg := OutputConf{
		LDJSONFile:       tmp(),
		ReqRespFile:      tmp(),
		ReqRespFileLevel: 0,
		BinaryFile:       tmp(),
		StringsFile:      tmp(),
	}
	out := NewOutput(cfg)
	out.Start(cfg)
	item := OutputItem{Label: "newlabel", RespBytes: []byte("test 123")}
	item.EndWithError(io.EOF)

	out.Push(&item)
	out.Close()
}

func TestExtract(t *testing.T) {
	item := OutputItem{Label: "newlabel", RespBytes: []byte("test 123")}

	vals := ValMap{}
	extrs := map[string]*ExtractRegex{
		"var0": {
			Re: &RegexpProxy{Regexp: regexp.MustCompile("not found")},
		},
		"var1": {
			Re:      &RegexpProxy{Regexp: regexp.MustCompile("\\d+")},
			GroupNo: 0,
			MatchNo: -1,
		},
		"var2": {
			Re:      &RegexpProxy{Regexp: regexp.MustCompile("\\d+")},
			GroupNo: 0,
			MatchNo: 0,
		},
	}
	item.ExtractValues(extrs, vals)
	if string(vals["var0"]) != "NOT_FOUND" {
		t.Errorf("No var0")
	}
	if string(vals["var1"]) != "123" {
		t.Errorf("No var1")
	}
	if string(vals["var2"]) != "123" {
		t.Errorf("No var2")
	}
}

func TestAssert(t *testing.T) {
	item := OutputItem{Label: "newlabel", RespBytes: []byte("test 123")}
	asserts := []*AssertItem{
		{Invert: false, Re: &RegexpProxy{Regexp: regexp.MustCompile("\\d+")}},
		{Invert: false, Re: &RegexpProxy{Regexp: regexp.MustCompile("notpresent")}},
		{Invert: true, Re: &RegexpProxy{Regexp: regexp.MustCompile("notpresent")}},
		{Invert: true, Re: &RegexpProxy{Regexp: regexp.MustCompile("\\d+")}},
	}
	item.Assert(asserts)
	if item.Error.Error() != "Assert failed on regexp: notpresent\nAssert failed on inverted regexp: \\d+" {
		t.Errorf("Should not be errors, got: %s", item.Error)
	}
}
