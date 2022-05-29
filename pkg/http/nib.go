package http

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Nib struct {
	transport *http.Transport
	values    map[string][]byte
}

func (n *Nib) Punch(item *core.InputItem) *core.OutputItem {
	outItem := core.OutputItem{
		StartTime: time.Now(),
	}

	conn := n.sendRequest(item, &outItem)
	if outItem.Error != nil {
		return &outItem
	}

	n.readResponse(item, conn, &outItem)
	outItem.Elapsed = time.Now().Sub(outItem.StartTime)
	return &outItem
}

func (n *Nib) sendRequest(item *core.InputItem, outItem *core.OutputItem) net.Conn {
	item.ReplaceValues(n.values)

	before := time.Now()
	conn, err := n.getConnection(item.Hostname)
	if err != nil {
		outItem.EndWithError(err)
		return nil
	}
	connected := time.Now()
	outItem.ConnectTime = connected.Sub(before)

	if write, err := conn.Write(item.Payload); err != nil {
		outItem.EndWithError(err)
		return nil
	} else {
		outItem.SentBytesCount = write
		outItem.SentTime = time.Now().Sub(connected)
	}
	return conn
}

func (n *Nib) getConnection(hostname string) (net.Conn, error) {
	log.Debugf("Opening new connection to %s", hostname)
	if !strings.Contains(hostname, "://") {
		hostname = "http://" + hostname
	}
	parsed, err := url.Parse(hostname)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse hostname '%s' as URL: %s", hostname, err))
	}

	host := parsed.Host // TODO: DNS round-robin here via own code
	if parsed.Scheme == "https" {
		if !strings.Contains(host, ":") {
			host = host + ":443"
		}

		return n.transport.DialTLSContext(context.Background(), "tcp", host)
	} else {
		if !strings.Contains(host, ":") {
			host = host + ":80"
		}

		return n.transport.DialContext(context.Background(), "tcp", host)
	}
}

func (n *Nib) readResponse(item *core.InputItem, conn net.Conn, result *core.OutputItem) {
	begin := time.Now()
	recorder := core.RecordingReader{
		Limit: 1024 * 1024, // TODO: make it configurable
		R:     conn,
	}
	reader := bufio.NewReader(&recorder)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		result.EndWithError(err)
		return
	}

	if len(item.RegexOut) > 0 {
		recorder.Limit = 0
	}

	buf := make([]byte, n.transport.ReadBufferSize)
	for {
		_, err := resp.Body.Read(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			result.EndWithError(err)
			return
		}
	}

	err = resp.Body.Close()
	if err != nil {
		log.Warningf("Failed to close response body")
	}
	result.ReadTime = time.Now().Sub(begin)

	result.RespBytesCount = recorder.Len
	result.RespBytes = recorder.Buffer.Bytes()

	result.ExtractValues(item.RegexOut, n.values)

	result.Status = resp.StatusCode
	result.FirstByteTime = recorder.FirstRead.Sub(begin)
}
