package http

import (
	"context"
	"testing"
)

func TestResolver(t *testing.T) {
	r := NewResolver()
	host, err := r.ResolveHost(context.Background(), "www.com")
	t.Logf("%s, %v", host, err)
	host2, err2 := r.ResolveHost(context.Background(), "www.com")
	t.Logf("%s, %v", host2, err2)
	if host2 == host {
		t.Error("Should round-robin")
	}

	host, err = r.ResolveHost(context.Background(), "ipv4.com")
	t.Logf("%s, %v", host, err)
	host2, err2 = r.ResolveHost(context.Background(), "ipv4.com")
	t.Logf("%s, %v", host, err)
	if host2 != host {
		t.Error("Should be just one")
	}
}
