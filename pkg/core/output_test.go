package core

import (
	"io"
	"os"
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
	item := OutputItem{Label: "newlabel"}
	item.EndWithError(io.EOF)
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
