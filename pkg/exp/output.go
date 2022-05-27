package incarne

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"time"
)

type OutputItem struct {
	SentBytesCount int
	RespBytesCount int
	RespBytes      []byte
	Error          error
	Status         int
	StartTime      time.Time
	ConnectedTime  time.Time
	SentTime       time.Time
	FirstByteTime  time.Time
	ReadTime       time.Time
	EndTime        time.Time
}

func (i *OutputItem) EndWithError(err error) *OutputItem {
	i.Status = 999
	i.Error = err
	return i
}

func (i *OutputItem) ExtractValues(extractors map[string]*ExtractRegex, values map[string][]byte) {
	for name, outSpec := range extractors {
		all := outSpec.Re.FindAllSubmatch(i.RespBytes, -1)

		if len(all) <= 0 {
			log.Warningf("Nothing has matched the regex '%s': %v", name, outSpec.String())
		} else if outSpec.MatchNo >= 0 {
			values[name] = all[outSpec.MatchNo][outSpec.GroupNo]
		} else {
			values[name] = all[rand.Intn(len(all))][outSpec.GroupNo]
		}
	}
}

type RecordingReader struct {
	Limit     int
	R         io.Reader
	Buffer    bytes.Buffer
	Len       int
	FirstRead time.Time
	Err       error
}

func (r *RecordingReader) Read(p []byte) (n int, err error) {
	n, err = r.R.Read(p)

	if r.Len == 0 {
		r.FirstRead = time.Now()
	}

	r.Len += n
	log.Debugf("Read %d bytes, %d total: %v", n, r.Len, err)

	if (r.Limit <= 0 || r.Len <= r.Limit) && n > 0 {
		r.Buffer.Write(p[:n])
	}

	if err != nil {
		r.Err = err
	}

	return n, err
}
