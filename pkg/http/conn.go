package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"incarne/pkg/core"
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
}

func newBufferedConn(c net.Conn) *BufferedConn {
	conn := BufferedConn{
		Conn:            c,
		ReadRecordLimit: 1024 * 1024,
		buf:             make([]byte, 4096),
		readChunks:      make(chan []byte),
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

		if (r.ReadRecordLimit <= 0 || r.ReadLen <= r.ReadRecordLimit) && n > 0 {
			r.ReadRecorded.Write(r.buf[:n])
		}

		if err != nil {
			r.Err = err
			r.Canceled = true
		}

		r.readChunks <- r.buf[:n]
	}
	log.Debugf("Done reading loop")

	err := r.Close()
	if err != nil {
		log.Warningf("Error while trying to close connection: %s", err)
	}
}

func (r *BufferedConn) Read(p []byte) (n int, err error) {
	if r.Err != nil {
		return 0, r.Err
	}

	buf := <-r.readChunks
	n = copy(p, buf)
	return n, nil
}

func (r *BufferedConn) Close() error {
	log.Debugf("Closing connection")
	if !r.Canceled {
		r.Canceled = true
	}

	if r.Err != nil {
		return r.Conn.Close()
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
	var ch ConnChan
	if c, ok := p.Idle[hostname]; ok {
		ch = c
	} else {
		ch = make(ConnChan, p.MaxConnections)
		p.Idle[hostname] = ch
	}

	select {
	case conn := <-p.Idle[hostname]:
		if conn.Err == io.EOF {
			log.Debugf("Cannot reuse Idle connection: %v", conn.Err)
		} else if conn.Err != nil {
			log.Warningf("Cannot reuse Idle connection: %v", conn.Err)
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
	log.Debugf("Opening new connection to %s", hostname)

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
	if conn.Err == nil && !conn.Canceled {
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

type Conf struct {
	MaxConnections int
	Timeout        time.Duration
}

func ParseHTTPConf(conf core.ProtoConf) Conf {
	cfg := Conf{
		MaxConnections: 1,
		Timeout:        1 * time.Second,
	}
	err := yaml.Unmarshal(conf.FullText, &cfg)
	if err != nil {
		panic(err)
	}
	return cfg
}
