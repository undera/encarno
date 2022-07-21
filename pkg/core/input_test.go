package core

import (
	"testing"
)

func TestReplaceValues(t *testing.T) {
	item := PayloadItem{
		Payload:  []byte("${noval} ${var} text ${var2}"),
		Replaces: []string{"var", "noval"},
	}
	vals := ValMap{
		"var":  []byte("val"),
		"var2": []byte("val2"),
	}
	item.ReplaceValues(vals)
	if string(item.Payload) != "NO_VALUE val text ${var2}" {
		t.Errorf("Wrong payload: %s", item.Payload)
	}

	item.ReplaceValues(vals) // to hit the cache branch
}

func TestReadPayloadRecord(t *testing.T) {
	ch := NewInput(InputConf{
		PayloadFile:    "../../examples/payload-strings.txt",
		StringsFile:    "",
		EnableRegexes:  true,
		Predefined:     nil,
		IterationLimit: 2,
	})

	items := make([]*PayloadItem, 0)
	for item := range ch {
		items = append(items, item)
	}
	
	if len(items) != 4 {
		t.Errorf("Wrong items len: %d", len(items))
	}
}
