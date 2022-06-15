package http

import (
	"encarno/pkg/core"
	"testing"
	"time"
)

func TestConnPool(t *testing.T) {
	addrs := []string{
		"localhost:8070",
		"https://www.google.com",
	}

	for _, addr := range addrs {
		pool := NewConnectionPool(1, 1*time.Second, core.TLSConf{})
		conn, err := pool.Get(addr, "")
		if err != nil {
			t.Error(err)
		} else {
			pool.Return(addr, conn)
			// TODO: assert something
		}
	}
}
