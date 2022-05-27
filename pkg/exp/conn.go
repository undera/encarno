package incarne

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"net/url"
	"strings"
	"time"
)

type BufferedConn struct {
	r        *bufio.Reader
	net.Conn // So that most methods are embedded
}

func newBufferedConn(c net.Conn) *BufferedConn {
	return &BufferedConn{bufio.NewReader(c), c}
}

func (b BufferedConn) Peek(n int) ([]byte, error) {
	return b.r.Peek(n)
}

func (b BufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

type ConnChan = chan *BufferedConn

type ConnPool struct {
	idle           map[string]ConnChan
	MaxConnections int
	Timeout        time.Duration
}

func (p *ConnPool) Get(hostname string) (*BufferedConn, error) {
	var ch ConnChan
	if c, ok := p.idle[hostname]; ok {
		ch = c
	} else {
		ch = make(ConnChan, p.MaxConnections)
		p.idle[hostname] = ch
	}

	select {
	case conn := <-p.idle[hostname]:
		_, err := conn.Peek(1)
		if err != nil {
			log.Warningf("Cannot reuse idle connection: %v", err)
		} else {
			log.Debugf("Reusing idle connection to %s", hostname)
			return conn, nil
		}
	default:

	}
	c, err := p.openConnection(hostname)
	return newBufferedConn(c), err
}

func (p *ConnPool) openConnection(hostname string) (net.Conn, error) {
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

		ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
		defer cancel() // Ensure cancel is always called
		conf := &tls.Config{
			InsecureSkipVerify: true, // TODO: configurable

		}
		d := tls.Dialer{
			Config: conf,
		}
		return d.DialContext(ctx, "tcp", host)
	} else {
		if !strings.Contains(host, ":") {
			host = host + ":80"
		}

		return net.DialTimeout("tcp", host, p.Timeout)
	}
}

func (p *ConnPool) Return(hostname string, conn *BufferedConn) {
	p.idle[hostname] <- conn
}
