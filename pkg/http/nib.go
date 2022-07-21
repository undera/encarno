package http

import (
	"bytes"
	"encarno/pkg/core"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"regexp"
	"strconv"
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
	return &outItem
}

var contentLengthRe = regexp.MustCompile(`(?m:\$\{:content-length:})`)

func (n *Nib) sendRequest(item *core.PayloadItem, outItem *core.OutputItem) (*BufferedConn, bool) {
	hostHint, connClose, bodyLen := getHostAndConnHeaderValues(item.Payload)
	if len(item.Replaces) > 0 {
		item.Payload = contentLengthRe.ReplaceAll(item.Payload, []byte(strconv.Itoa(bodyLen)))
	}

	before := time.Now()
	conn, err := n.ConnPool.Get(item.Address, hostHint)
	connected := time.Now()

	outItem.ConnectTime = connected.Sub(before)
	if err != nil {
		outItem.EndWithError(err)
		return nil, connClose
	}

	if len(item.RegexOut) > 0 {
		conn.ReadRecordLimit = 0
	} else {
		conn.ReadRecordLimit = 1024 * 1024
	}

	if err := conn.SetDeadline(time.Now().Add(n.ConnPool.Timeout)); err != nil {
		outItem.EndWithError(err)
		return nil, connClose
	}

	log.Debugf("Writing %d bytes into connection", len(item.Payload))
	write, err := conn.Write(item.Payload)
	outItem.SentBytesCount = uint64(write)
	outItem.SentTime = time.Now().Sub(connected)
	if err != nil {
		outItem.EndWithError(err)
		return nil, connClose
	}
	return conn, connClose
}

func getHostAndConnHeaderValues(payload []byte) (host string, close bool, bodyLen int) {
	nlSep := []byte{10}
	colonSep := []byte{':'}
	_, payload, _ = bytes.Cut(payload, nlSep) // swallow req line
	for {                                     // read headers
		before, after, found := bytes.Cut(payload, nlSep)
		payload = after
		if !found || len(before) < 2 { // minimal possible header is "x:"
			break
		}

		hname, hval, found := bytes.Cut(before, colonSep)

		if string(hname) == "Host" || string(hname) == "host" {
			host = strings.TrimSpace(string(hval))
		}

		if string(hname) == "Connection" || string(hname) == "connection" {
			close = strings.TrimSpace(string(hval)) == "close"
		}
	}

	bodyLen = len(payload)

	return
}

func (n *Nib) readResponse(item *core.PayloadItem, conn *BufferedConn, result *core.OutputItem, connClose bool) {
	begin := time.Now()
	resp, err := http.ReadResponse(conn.BufReader, nil)
	result.ReadTime = time.Now().Sub(begin) // in case there will be an error
	if err != nil {
		result.EndWithError(err)
		return
	}
	result.Status = uint16(resp.StatusCode)

	if !conn.FirstRead.IsZero() {
		result.FirstByteTime = conn.FirstRead.Sub(begin)
	}

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		result.EndWithError(err) // TODO: unclosed connection leak?
		return
	}

	finish := time.Now()
	result.ReadTime = finish.Sub(conn.FirstRead) // now it's final read time
	result.Elapsed = finish.Sub(result.StartTime)

	err = resp.Body.Close()
	if err != nil {
		log.Warningf("Failed to close response body")
	}

	result.RespBytesCount = uint64(conn.ReadLen)
	result.RespBytes = conn.ReadRecorded.Bytes()

	// close or reuse
	if resp.Close || connClose {
		go conn.Close()
	} else {
		n.ConnPool.Return(item.Address, conn)
	}
}
