package http

import (
	"bytes"
	"encarno/pkg/core"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
	"time"
)

type Nib struct {
	ConnPool *ConnPool
}

func (n *Nib) Punch(item *core.PayloadItem) *core.OutputItem {
	outItem := core.OutputItem{
		StartTime: time.Now(),
	}

	conn, connClose := n.sendRequest(item, &outItem)
	if outItem.Error != nil {
		return &outItem
	}

	n.readResponse(item, conn, &outItem, connClose)
	outItem.Elapsed = time.Now().Sub(outItem.StartTime)
	return &outItem
}

func (n *Nib) sendRequest(item *core.PayloadItem, outItem *core.OutputItem) (*BufferedConn, bool) {
	before := time.Now()
	hostHint, connClose := getHostAndConnHeaderValues(item.Payload)

	conn, err := n.ConnPool.Get(item.Address, hostHint)
	if err != nil {
		outItem.EndWithError(err)
		return nil, connClose
	}
	connected := time.Now()
	outItem.ConnectTime = connected.Sub(before)

	if err := conn.SetDeadline(time.Now().Add(n.ConnPool.Timeout)); err != nil {
		outItem.EndWithError(err)
		return nil, connClose
	}

	log.Debugf("Writing %d bytes into connection", len(item.Payload))
	if write, err := conn.Write(item.Payload); err != nil {
		outItem.EndWithError(err)
		return nil, connClose
	} else {
		outItem.SentBytesCount = write
		outItem.SentTime = time.Now().Sub(connected)
	}
	return conn, connClose
}

func getHostAndConnHeaderValues(payload []byte) (host string, close bool) {
	nlSep := []byte{10}
	colonSep := []byte{':'}
	_, _, _ = bytes.Cut(payload, nlSep) // swallow req line
	for {                               // read headers
		before, after, found := bytes.Cut(payload, nlSep)
		if !found || len(after) < 2 { // minimal possible header is "x:"
			break
		}

		hname, hval, found := bytes.Cut(before, colonSep)

		if string(hname) == "Host" || string(hname) == "host" {
			host = strings.TrimSpace(string(hval))
		}

		if string(hname) == "Connection" || string(hname) == "connection" {
			close = strings.TrimSpace(string(hval)) == "close"
		}

		payload = after
	}
	return
}

func (n *Nib) readResponse(item *core.PayloadItem, conn *BufferedConn, result *core.OutputItem, connClose bool) {
	begin := time.Now()
	resp, err := http.ReadResponse(conn.BufReader, nil)
	if err != nil {
		result.EndWithError(err)
		return
	}

	if len(item.RegexOut) > 0 {
		conn.ReadRecordLimit = 0 // FIXME: this affects reused connections
	}

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		if err != nil {
			result.EndWithError(err)
			return
		}
	}

	err = resp.Body.Close()
	if err != nil {
		log.Warningf("Failed to close response body")
	}
	result.ReadTime = time.Now().Sub(begin)

	result.RespBytesCount = conn.ReadLen
	result.RespBytes = conn.ReadRecorded.Bytes()

	result.Status = resp.StatusCode
	result.FirstByteTime = conn.FirstRead.Sub(begin)

	if resp.Close || connClose {
		err := conn.Close()
		if err != nil {
			log.Warningf("Failed to close connection: %s", err)
		}
	} else {
		n.ConnPool.Return(item.Address, conn)
	}
}

func (n *Nib) readerLoop() {

}
