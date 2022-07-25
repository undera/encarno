package http

import (
	"encarno/pkg/core"
	log "github.com/sirupsen/logrus"
	"net/http"
	"regexp"
	"testing"
	"time"
)

var hostname = "localhost:8070"

func TestOne(t *testing.T) {
	//log.SetLevel(log.DebugLevel)

	nib := Nib{
		ConnPool: NewConnectionPool(100, 1*time.Second, core.TLSConf{}),
	}

	type Item struct {
		inp core.PayloadItem
		out string
	}

	items := []Item{
		{
			inp: core.PayloadItem{
				Address: hostname,
				Payload: []byte("GET /scans.tgz HTTP/1.1\r\n\r\n"),
			},
		},
		{
			inp: core.PayloadItem{
				Address: hostname,
				Payload: []byte("GET /pt.tgz HTTP/1.1\r\n\r\n"),
			},
		},
		{
			inp: core.PayloadItem{
				Address: "yandex.ru",
				Payload: []byte("GET / HTTP/1.1\r\n\r\n"),
			},
		},
		{
			inp: core.PayloadItem{
				Address: "https://www.olx.pt",
				Payload: []byte("GET / HTTP/1.1\r\nHost: www.olx.pt\r\n\r\n"),
			},
		},
		{
			inp: core.PayloadItem{
				Address: "httpbin.org",
				Payload: []byte("GET /anything HTTP/1.1\r\nHost:httpbin.org\r\n\r\n"),
			},
		},
		{
			inp: core.PayloadItem{
				Address:  "httpbin.org",
				Payload:  []byte("POST /anything HTTP/1.1\r\nHost:httpbin.org\r\nX-Hdr: ${input}\r\n\r\n"), // "test ${input} while producing 123"
				RegexOut: map[string]*core.ExtractRegex{"test1": {Re: &core.RegexpProxy{Regexp: regexp.MustCompile("1+")}}},
			},
			out: "1",
		},
		{
			inp: core.PayloadItem{
				Address: "notexistent",
			},
		},
	}

	for _, item := range items {
		res := nib.Punch(&item.inp)

		t.Logf("Status: %d %v", res.Status, res.Error)

		//<-nib.transport.Idle[item.inp.Address]
	}
}

func TestDynamicLenVariable(t *testing.T) {
	nib := Nib{
		ConnPool: NewConnectionPool(100, 1*time.Second, core.TLSConf{}),
	}

	inp := core.PayloadItem{
		Address:  hostname,
		Payload:  []byte("POST /?${:content-length:} HTTP/1.1\r\n\r\nbody"),
		Replaces: []string{"var"},
	}
	res := nib.Punch(&inp)
	_ = res
	if string(inp.Payload) != "POST /?4 HTTP/1.1\r\n\r\nbody" {
		t.Errorf("Wrong payload: %s", inp.Payload)
	}
}

func TestConnClose(t *testing.T) {
	//log.SetLevel(log.DebugLevel)

	nib := Nib{
		ConnPool: NewConnectionPool(100, 1*time.Second, core.TLSConf{}),
	}

	type Item struct {
		inp core.PayloadItem
		out string
	}

	item := Item{
		inp: core.PayloadItem{
			Address: hostname,
			// Payload:  []byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"),
			Payload: []byte("GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"),
		},
	}

	items := []Item{
		item,
		item,
		item,
	}

	for _, item := range items {
		res := nib.Punch(&item.inp)

		t.Logf("Status: %d %v", res.Status, res.Error)
		t.Logf("Response:\n%s", res.RespBytes)

		if res.Error != nil {
			t.Errorf("Should not fail: %s", res.Error)
			t.FailNow()
		}
	}
}

func TestTLSIssues(t *testing.T) {
	//log.SetLevel(log.DebugLevel)
	res, err := http.Get("https://statics.otomoto.pl/optimus-storage/s/_next/static/chunks/80565.4e2f86f692555637.js")
	log.Debugf("%s, %v", res.Status, err)

	nib := Nib{
		ConnPool: NewConnectionPool(100, 5*time.Second, core.TLSConf{
			TLSCipherSuites: []string{"TLS_AES_128_GCM_SHA256"},
		}),
	}

	type Item struct {
		inp core.PayloadItem
		out string
	}

	item := Item{
		inp: core.PayloadItem{
			Address: "https://13.225.244.117",
			Payload: []byte("GET /optimus-storage/s/_next/static/chunks/80565.4e2f86f692555637.js HTTP/1.1\r\nHost: statics.otomoto.pl\r\nConnection: close\r\n\r\n"),
		},
	}

	items := []Item{
		item,
		item,
	}

	for _, item := range items {
		res := nib.Punch(&item.inp)

		t.Logf("Status: %d %v", res.Status, res.Error)
		t.Logf("Response:\n%s", res.RespBytes)

		if res.Error != nil {
			t.Errorf("Should not fail: %s", res.Error)
			t.FailNow()
		}
	}
}
