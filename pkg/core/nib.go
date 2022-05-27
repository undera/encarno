package core

type Nib interface {
	Punch(payload []byte) *OutputItem
}

// ipv4/ipv6
// http and https and dummy (udp? pluggable?)
// use less memory by direct send from file descriptor into network https://man7.org/linux/man-pages/man2/sendfile.2.html (does not work with SSL)
// multiple hosts allowed, working with connection pools and defaults to one
// handle HTTP-level errors and net-level errors separately
// allow bad SSL certs via option
// DNS - to cache or not to cache?
// track times breakdown - DNS/CONN/SSL/REQ/TTFB/RESP
// connect timeout, recv timeout, forced overall timeout
