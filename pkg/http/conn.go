package http

import (
	"context"
	"crypto/tls"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

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
