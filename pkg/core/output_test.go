package core

import (
	"io"
	"os"
	"regexp"
	"testing"
	"time"
)

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

	vals := ValMap{}
	extrs := map[string]*ExtractRegex{
		"var0": {
			Re: regexp.MustCompile("not found"),
		},
		"var1": {
			Re:      regexp.MustCompile("\\d+"),
			GroupNo: 0,
			MatchNo: -1,
		},
		"var2": {
			Re:      regexp.MustCompile("\\d+"),
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

	out.Push(&item)
	time.Sleep(1 * time.Second)
	out.Close()
}

func tmp() string {
	resultFile, err := os.CreateTemp(os.TempDir(), "encarno_*.tmp")
	if err != nil {
		panic(err)
	}
	_ = resultFile.Close()
	return resultFile.Name()
}
