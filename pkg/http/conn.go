package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type BufferedConn struct {
	net.Conn   // So that most methods are embedded
	Limit      int
	Buffer     bytes.Buffer
	ReadLen    int
	FirstRead  time.Time
	Err        error
	Canceled   bool
	readChunks chan []byte
	buf        []byte
}

func newBufferedConn(c net.Conn) *BufferedConn {
	conn := BufferedConn{
		Conn:       c,
		Limit:      1024 * 1024,
		buf:        make([]byte, 4096),
		readChunks: make(chan []byte),
	}
	go conn.readLoop()
	return &conn
}

func (r *BufferedConn) readLoop() {
	log.Debugf("Start reading loop")
	for !r.Canceled {
		n, err := r.Conn.Read(r.buf)

		if r.ReadLen == 0 {
			r.FirstRead = time.Now()
		}

		r.ReadLen += n
		log.Debugf("Read %d bytes, %d total: %v", n, r.ReadLen, err)

		if (r.Limit <= 0 || r.ReadLen <= r.Limit) && n > 0 {
			r.Buffer.Write(r.buf[:n])
		}

		if err != nil {
			r.Err = err
			r.Canceled = true
		}

		if n > 0 {
			r.readChunks <- r.buf[:n]
		}
	}
	log.Debugf("Done reading loop")

	err := r.Close()
	if err != nil {
		log.Warningf("Error while trying to close connection: %s", err)
	}
}

func (r *BufferedConn) Read(p []byte) (n int, err error) {
	select {
	case buf := <-r.readChunks:
		return copy(buf, p), nil
	default:
		return 0, r.Err
	}
}

func (r *BufferedConn) Close() error {
	r.Canceled = true
	return r.Conn.Close()
}

func (r *BufferedConn) Reset() {
	r.ReadLen = 0
	r.Buffer.Truncate(0)
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
		if conn.Err == io.EOF {
			log.Debugf("Cannot reuse idle connection: %v", conn.Err)
		} else if conn.Err != nil {
			log.Warningf("Cannot reuse idle connection: %v", conn.Err)
		} else {
			log.Debugf("Reusing idle connection to %s", hostname)
			conn.Reset()
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

func getTransport(maxConn int, timeout time.Duration) *http.Transport {
	r := NewResolver()

	plainDialer := net.Dialer{
		Timeout: timeout,
	}

	tlsDialer := tls.Dialer{
		NetDialer: &plainDialer,
		Config: &tls.Config{
			InsecureSkipVerify: true, // TODO: make configurable
		},
	}

	h := &http.Transport{
		DialContext: func(ctx context.Context, network string, addr string) (conn net.Conn, err error) {
			resolvedHost, err := r.ResolveHost(ctx, addr)
			if err != nil {
				return nil, err
			}
			return plainDialer.DialContext(ctx, network, resolvedHost)
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			resolvedHost, err := r.ResolveHost(ctx, addr)
			if err != nil {
				return nil, err
			}
			return tlsDialer.DialContext(ctx, network, resolvedHost)
		},
		MaxIdleConns:    maxConn,
		MaxConnsPerHost: maxConn,
		IdleConnTimeout: 10 * timeout, // TODO: is it right to do 10x?
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}
	return h
}

func NewResolver() *RRResolver {
	return &RRResolver{
		Resolver: net.Resolver{},
		Cache:    map[string][]string{},
		mx:       sync.Mutex{},
	}
}

type RRResolver struct {
	Resolver net.Resolver
	Cache    map[string][]string
	mx       sync.Mutex
}

func (r *RRResolver) ResolveHost(ctx context.Context, addr string) (string, error) {
	host, port, foundSep := strings.Cut(addr, ":")
	if !foundSep {
		panic("The address must constain port: " + addr)
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	ips, found := r.Cache[host]
	if !found {
		var err error
		log.Debugf("Looking up IP for: %s", host)
		ips, err = r.Resolver.LookupHost(ctx, host)
		if err != nil {
			return "", err
		}
		r.Cache[host] = ips
	}

	ip := ips[0]
	r.Cache[host] = append(ips[1:], ip)

	if strings.Contains(ip, ":") {
		ip = "[" + ip + "]"
	}

	return ip + ":" + port, nil
}
