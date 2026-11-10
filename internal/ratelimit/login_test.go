package ratelimit

import (
	"net"
	"testing"
)

func TestIPSlice(t *testing.T) {
	ip := net.ParseIP("203.0.113.1")
	b := ipSlice(ip)
	if len(b) != 4 {
		t.Fatalf("ipv4 len %d", len(b))
	}
}
