package incarne

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"regexp"
	"testing"
	"time"
)

func TestOne(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	clients := &ConnPool{
		idle:           map[string]ConnChan{},
		MaxConnections: 10,
		Timeout:        1000 * time.Second,
	}
	values := map[string][]byte{"input": []byte("theinput")}
	nib := HTTPNib{
		connPool: clients,
		values:   values,
	}

	type Item struct {
		inp InputItem
		out string
	}

	items := []Item{
		{
			inp: InputItem{
				Hostname: "localhost:8000",
				Payload:  []byte("GET /scans.tgz HTTP/1.1\r\n\r\n"),
			},
		},
		{
			inp: InputItem{
				Hostname: "localhost:8000",
				Payload:  []byte("GET /pt.tgz HTTP/1.1\r\n\r\n"),
			},
		},
		{
			inp: InputItem{
				Hostname: "yandex.ru",
				Payload:  []byte("GET / HTTP/1.1\r\n\r\n"),
			},
		},
		{
			inp: InputItem{
				Hostname: "yandex.ru:80",
				Payload:  []byte("GET / HTTP/1.1\r\nHost: yandex.ru\r\n\r\n"),
			},
		},
		{
			inp: InputItem{
				Hostname: "httpbin.org",
				Payload:  []byte("GET /anything HTTP/1.1\r\nHost:httpbin.org\r\n\r\n"),
			},
		},
		{
			inp: InputItem{
				Hostname: "httpbin.org",
				Payload:  []byte("POST /anything HTTP/1.1\r\nHost:httpbin.org\r\nX-Hdr: ${input}\r\n\r\n"), // "test ${input} while producing 123"
				RegexOut: map[string]*ExtractRegex{"test1": {Re: regexp.MustCompile("1+")}},
			},
			out: "1",
		},
		{
			inp: InputItem{
				Hostname: "notexistent",
			},
		},
	}

	for _, item := range items {
		res := nib.Process(&item.inp)

		t.Logf("Status: %d", res.Status)

		if string(values["test1"]) != item.out {
			t.Errorf("No right value: %s", string(values["test1"]))
		}

		//<-nib.connPool.idle[item.inp.Hostname]
	}
}

func TestLoop(t *testing.T) {
	clients := &ConnPool{
		idle:           map[string]ConnChan{},
		MaxConnections: 100,
		Timeout:        1000 * time.Second,
	}
	values := map[string][]byte{"input": []byte("theinput")}
	nib := HTTPNib{
		connPool: clients,
		values:   values,
	}

	item := InputItem{
		Hostname: "localhost:8081",
		Payload:  []byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"),
	}

	start := time.Now()
	i := float64(0)
	for ; i < 10000; i++ {
		res := nib.Process(&item)
		if res.Error != nil {
			t.Errorf("Failed: %v", res.Error)
			break
		}
		//t.Logf("Status: %d", res.Status)
	}
	elapsed := time.Now().Sub(start)
	t.Logf("Elapsed: %v", elapsed)
	t.Logf("Avg: %v", elapsed.Seconds()/i)
	t.Logf("Rate: %v", i/elapsed.Seconds())
}

func TestLoopNative(t *testing.T) {
	start := time.Now()
	i := float64(0)
	for ; i < 10000; i++ {
		doreq(t)
	}
	elapsed := time.Now().Sub(start)
	t.Logf("Iterations: %v", i)
	t.Logf("Elapsed: %v", elapsed)
	t.Logf("Avg: %v", elapsed.Seconds()/i)
	t.Logf("Rate: %v", i/elapsed.Seconds())
}

func doreq(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost:8081/", nil)
	if err != nil {
		log.Fatalf("Error Occured. %+v", err)
	}
	req.Header.Set("Connection", "close")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("Failed: %v", err)
		return
	}
	res.Body.Close()
	_ = res
	t.Logf("Status: %d", res.StatusCode)
}
