package incarne

import (
	"bufio"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"time"
)

type HTTPNib struct {
	connPool *ConnPool
	values   map[string][]byte
}

func (n *HTTPNib) Process(item *InputItem) *OutputItem {
	outItem := OutputItem{
		StartTime: time.Now(),
	}

	conn := n.sendRequest(item, &outItem)
	if outItem.Error != nil {
		return &outItem
	}

	n.readResponse(item, conn, &outItem)
	outItem.EndTime = time.Now()
	return &outItem
}

func (n *HTTPNib) sendRequest(item *InputItem, outItem *OutputItem) *BufferedConn {
	item.ReplaceValues(n.values)
	conn, err := n.connPool.Get(item.Hostname)
	if err != nil {
		outItem.EndWithError(err)
		return nil
	}
	outItem.ConnectedTime = time.Now()

	if err := conn.SetDeadline(time.Now().Add(n.connPool.Timeout)); err != nil {
		outItem.EndWithError(err)
		return nil
	}

	if write, err := conn.Write(item.Payload); err != nil {
		outItem.EndWithError(err)
		return nil
	} else {
		outItem.SentBytesCount = write
		outItem.SentTime = time.Now()
	}
	return conn
}

func (n *HTTPNib) readResponse(item *InputItem, conn *BufferedConn, result *OutputItem) {
	recorder := RecordingReader{
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

	buf := make([]byte, 4096)
	hadEOF := false
	for {
		_, err := resp.Body.Read(buf)
		if err == io.EOF {
			hadEOF = recorder.Err == io.EOF
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

	result.RespBytesCount = recorder.Len
	result.RespBytes = recorder.Buffer.Bytes()

	result.ExtractValues(item.RegexOut, n.values)

	result.Status = resp.StatusCode
	result.FirstByteTime = recorder.FirstRead

	if !hadEOF {
		n.connPool.Return(item.Hostname, conn)
	}
}
