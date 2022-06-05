package http

import (
	"bufio"
	log "github.com/sirupsen/logrus"
	"incarne/pkg/core"
	"io"
	"net/http"
	"time"
)

type Nib struct {
	ConnPool *ConnPool
}

func (n *Nib) Punch(item *core.PayloadItem) *core.OutputItem {
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

func (n *Nib) sendRequest(item *core.PayloadItem, outItem *core.OutputItem) *BufferedConn {
	before := time.Now()
	conn, err := n.ConnPool.Get(item.Hostname)
	if err != nil {
		outItem.EndWithError(err)
		return nil
	}
	connected := time.Now()
	outItem.ConnectTime = connected.Sub(before)

	if err := conn.SetDeadline(time.Now().Add(n.ConnPool.Timeout)); err != nil {
		outItem.EndWithError(err)
		return nil
	}

	log.Debugf("Writing %d bytes into connection", len(item.Payload))
	if write, err := conn.Write(item.Payload); err != nil {
		outItem.EndWithError(err)
		return nil
	} else {
		outItem.SentBytesCount = write
		outItem.SentTime = time.Now().Sub(connected)
	}
	return conn
}

func (n *Nib) readResponse(item *core.PayloadItem, conn *BufferedConn, result *core.OutputItem) {
	begin := time.Now()
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		result.EndWithError(err)
		return
	}

	if len(item.RegexOut) > 0 {
		conn.ReadRecordLimit = 0
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

	if resp.Close {
		err := conn.Close()
		if err != nil {
			log.Warningf("Failed to close connection: %s", err)
		}
	} else {
		n.ConnPool.Return(item.Hostname, conn)
	}
}

func (n *Nib) readerLoop() {

}
