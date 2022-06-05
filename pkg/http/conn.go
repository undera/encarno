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
	"net/url"
	"strings"
	"sync"
	"time"
)

type BufferedConn struct {
	net.Conn        // So that most methods are embedded
	ReadRecordLimit int
	ReadRecorded    bytes.Buffer
	ReadLen         int
	FirstRead       time.Time
	Err             error
	Canceled        bool
	readChunks      chan []byte
	buf             []byte
	loopDone        bool
	mxErr           *sync.Mutex
}

func newBufferedConn(c net.Conn) *BufferedConn {
	conn := &BufferedConn{
		Conn:            c,
		ReadRecordLimit: 1024 * 1024,
		buf:             make([]byte, 4096),
		readChunks:      make(chan []byte),
		mxErr:           new(sync.Mutex),
	}
	go conn.readLoop()
	return conn
}

func (r *BufferedConn) readLoop() {
	log.Debugf("Start reading loop")
	defer func() {
		log.Debugf("Closing connection")
		err := r.Conn.Close()
		if err != nil {
			log.Warningf("Failed to close connection: %s", err)
		}
	}()

	for !r.Canceled {
		n, err := r.Conn.Read(r.buf)
		if err != nil {
			r.setErr(err)
			r.Canceled = true // but we still should finish the iteration
		}

		if r.ReadLen == 0 {
			r.FirstRead = time.Now()
		}

		r.ReadLen += n
		log.Debugf("Read %d/%d bytes, err: %v", n, r.ReadLen, err)

		if (r.ReadRecordLimit <= 0 || r.ReadLen <= r.ReadRecordLimit) && n > 0 {
			_, err := r.ReadRecorded.Write(r.buf[:n])
			if err != nil {
				panic(err)
			}
		}

		r.readChunks <- r.buf[:n]
	}
	log.Debugf("Done reading loop")

	err := r.Close()
	if err != nil {
		log.Warningf("Error while trying to close connection: %s", err)
	}
	r.loopDone = true
}

func (r *BufferedConn) setErr(err error) {
	r.mxErr.Lock()
	defer r.mxErr.Unlock()
	r.Err = err
}

func (r *BufferedConn) GetErr() error {
	r.mxErr.Lock()
	defer r.mxErr.Unlock()
	return r.Err
}

func (r *BufferedConn) Read(p []byte) (int, error) {
	err := r.GetErr()
	if err != nil {
		return 0, err
	}

	buf := <-r.readChunks
	n := copy(p, buf)
	return n, nil
}

func (r *BufferedConn) Close() error {
	if !r.Canceled {
		r.Canceled = true
	}
	return nil
}

func (r *BufferedConn) Reset() {
	r.ReadLen = 0
	r.ReadRecorded.Truncate(0)
}

type ConnChan = chan *BufferedConn

type ConnPool struct {
	Idle           map[string]ConnChan
	MaxConnections int
	Timeout        time.Duration
	resolver       *RRResolver
	plainDialer    net.Dialer
	tlsDialer      tls.Dialer
}

func NewConnectionPool(maxConnections int, timeout time.Duration) *ConnPool {
	plainDialer := net.Dialer{
		Timeout: timeout,
	}

	pool := &ConnPool{
		resolver:    NewResolver(),
		plainDialer: plainDialer,
		tlsDialer: tls.Dialer{
			NetDialer: &plainDialer,
			Config: &tls.Config{
				InsecureSkipVerify: true, // TODO: make configurable
			},
		},
		Idle:           make(map[string]ConnChan),
		MaxConnections: maxConnections,
		Timeout:        timeout,
	}
	return pool
}

func (p *ConnPool) Get(hostname string) (*BufferedConn, error) {
	// lazy initialize per-host pool
	var ch ConnChan
	if c, ok := p.Idle[hostname]; ok {
		ch = c
	} else {
		ch = make(ConnChan, p.MaxConnections)
		p.Idle[hostname] = ch
	}

	select {
	case conn := <-p.Idle[hostname]:
		err := conn.GetErr()
		if err == io.EOF {
			log.Debugf("Cannot reuse Idle connection: %v", err)
		} else if err != nil {
			log.Warningf("Cannot reuse Idle connection: %v", err)
		} else {
			log.Debugf("Reusing Idle connection to %s", hostname)
			conn.Reset()
			return conn, nil
		}
	default:

	}
	c, err := p.openConnection(hostname)
	if err == nil {
		return newBufferedConn(c), nil
	} else {
		return nil, err
	}
}

func (p *ConnPool) openConnection(hostname string) (net.Conn, error) {
	log.Infof("Opening new connection to %s", hostname)

	if !strings.Contains(hostname, "://") {
		hostname = "http://" + hostname
	}
	parsed, err := url.Parse(hostname)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse hostname '%s' as URL: %s", hostname, err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()

	host, port, foundSep := strings.Cut(parsed.Host, ":") // FIXME: what if it's already an ipv6?

	host, err = p.resolver.ResolveHost(ctx, host)
	if err != nil {
		return nil, err
	}

	if parsed.Scheme == "https" {
		if foundSep { //FIXME: ipv6 won't work well
			host = host + ":" + port
		} else {
			host = host + ":443"
		}

		return p.tlsDialer.DialContext(ctx, "tcp", host)
	} else {
		if foundSep { //FIXME: ipv6 won't work well
			host = host + ":" + port
		} else {
			host = host + ":80"
		}

		return p.plainDialer.DialContext(ctx, "tcp", host)
	}
}

func (p *ConnPool) Return(hostname string, conn *BufferedConn) {
	if conn.GetErr() == nil && !conn.Canceled {
		p.Idle[hostname] <- conn
	}
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

func (r *RRResolver) ResolveHost(ctx context.Context, host string) (string, error) {
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

	return ip, nil
}
