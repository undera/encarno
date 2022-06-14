package core

import (
	"errors"
	"math/rand"
	"strconv"
	"time"
)

type Nib interface {
	Punch(item *PayloadItem) *OutputItem
}

type NibMaker = func() Nib

// ipv4/ipv6
// http and https and dummy (udp? pluggable?)
// use less memory by direct send from file descriptor into network https://man7.org/linux/man-pages/man2/sendfile.2.html (does not work with SSL)
// multiple hosts allowed, working with connection pools and defaults to one
// handle HTTP-level errors and net-level errors separately
// allow bad SSL certs via option
// DNS - to cache or not to cache?
// track times breakdown - DNS/CONN/SSL/REQ/TTFB/RESP
// connect timeout, recv timeout, forced overall timeout

type DummyNib struct {
}

func (d DummyNib) Punch(item *PayloadItem) *OutputItem {
	o := &OutputItem{
		StartTime: time.Now(),
		Status:    uint16(rand.Intn(5)*100 + 100),
		Label:     "label#" + strconv.Itoa(rand.Intn(3)),
	}

	duration := time.Duration(rand.Intn(1000)) * time.Microsecond
	time.Sleep(duration)

	o.Elapsed = time.Now().Sub(o.StartTime)
	if o.Elapsed > duration*10 {
		o.Error = errors.New("timeout occured")
	}
	return o
}
