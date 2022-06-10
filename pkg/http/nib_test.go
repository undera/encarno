package http

import (
	"encarno/pkg/core"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"
)

var hostname = "localhost:8070"

func TestOne(t *testing.T) {
	//log.SetLevel(log.DebugLevel)

	nib := Nib{
		ConnPool: NewConnectionPool(100, 1*time.Second, core.ProtoConf{}),
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
				Address: "https://yandex.ru",
				Payload: []byte("GET / HTTP/1.1\r\nHost: yandex.ru\r\n\r\n"),
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
				RegexOut: map[string]*core.ExtractRegex{"test1": {Re: regexp.MustCompile("1+")}},
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

func TestConnClose(t *testing.T) {
	//log.SetLevel(log.DebugLevel)

	nib := Nib{
		ConnPool: NewConnectionPool(100, 1*time.Second, core.ProtoConf{}),
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
		ConnPool: NewConnectionPool(100, 5*time.Second, core.ProtoConf{
			TLSConf: core.TLSConf{
				TLSCipherSuites: []string{"TLS_AES_128_GCM_SHA256"},
			},
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

func TestLoop(t *testing.T) {
	//log.SetLevel(log.DebugLevel)

	nib := Nib{
		ConnPool: NewConnectionPool(100, 1*time.Second, core.ProtoConf{}),
	}

	item := core.PayloadItem{
		Address: hostname,
		Payload: []byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"),
	}

	start := time.Now()
	i := float64(0)
	for ; i < 100000; i++ {
		res := nib.Punch(&item)
		if res.Error != nil {
			t.Logf("Failed: %v", res.Error)
			break
		}
		//t.Logf("Status: %d", res.Status)
	}
	elapsed := time.Now().Sub(start)
	t.Logf("Iterations: %v", i)
	t.Logf("Elapsed: %v", elapsed)
	t.Logf("Avg: %v", elapsed.Seconds()/i)
	t.Logf("Rate: %v", i/elapsed.Seconds())
}

func TestLoopNative(t *testing.T) {
	return // was used to compare the performance
	start := time.Now()
	i := float64(0)
	for ; i < 100000; i++ {
		doreq(t)
	}
	elapsed := time.Now().Sub(start)
	t.Logf("Iterations: %v", i)
	t.Logf("Elapsed: %v", elapsed)
	t.Logf("Avg: %v", elapsed.Seconds()/i)
	t.Logf("Rate: %v", i/elapsed.Seconds())
}

var client = http.Client{
	Timeout: 1000 * time.Second,
	Transport: &http.Transport{
		IdleConnTimeout: 1000 * time.Second,
	},
}

func doreq(t *testing.T) {
	req, err := http.NewRequest("GET", "http://"+hostname+"/", nil)
	if err != nil {
		log.Fatalf("Error Occured. %+v", err)
	}
	//req.Header.Set("Connection", "close")
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("Failed: %v", err)
		return
	}
	io.ReadAll(res.Body)
	res.Body.Close()
	_ = res
	//t.Logf("Status: %d", res.StatusCode)
	//time.Sleep(10 * time.Second)
}
