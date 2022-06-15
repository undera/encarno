package http

import (
	"encarno/pkg/core"
	"testing"
	"time"
)

func TestConnPool(t *testing.T) {
	pool := NewConnectionPool(1, 1*time.Second, core.TLSConf{})
	conn, err := pool.Get("http://localhost:8070", "")
	if err != nil {
		t.Error(err)
	}

	pool.Return("http://localhost:8070", conn)
	// TODO: assert something
}
