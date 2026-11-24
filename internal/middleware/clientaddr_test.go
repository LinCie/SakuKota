package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientAddr(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  []string
		hops       int
		want       string
	}{
		{name: "no-proxy", remoteAddr: "203.0.113.7:5000", want: "203.0.113.7"},
		{name: "no-proxy-ignores-forwarded", remoteAddr: "203.0.113.7:5000", forwarded: []string{"198.51.100.1"}, want: "203.0.113.7"},
		{name: "one-proxy", remoteAddr: "10.0.0.2:5000", forwarded: []string{"203.0.113.7"}, hops: 1, want: "203.0.113.7"},
		{name: "one-proxy-ignores-forged", remoteAddr: "10.0.0.2:5000", forwarded: []string{"198.51.100.1, 203.0.113.7"}, hops: 1, want: "203.0.113.7"},
		{name: "two-proxies", remoteAddr: "10.0.0.2:5000", forwarded: []string{"198.51.100.1, 203.0.113.7, 10.0.0.1"}, hops: 2, want: "203.0.113.7"},
		{name: "two-headers", remoteAddr: "10.0.0.2:5000", forwarded: []string{"198.51.100.1", "203.0.113.7"}, hops: 1, want: "203.0.113.7"},
		{name: "chain-too-short", remoteAddr: "203.0.113.7:5000", forwarded: []string{"198.51.100.1"}, hops: 2, want: "203.0.113.7"},
		{name: "ipv6-grouped-by-64", remoteAddr: "[2001:db8:1:2:3:4:5:6]:5000", want: "2001:db8:1:2::/64"},
		{name: "ipv4-mapped", remoteAddr: "[::ffff:203.0.113.7]:5000", want: "203.0.113.7"},
		{name: "not-an-address", remoteAddr: "10.0.0.2:5000", forwarded: []string{"unknown"}, hops: 1, want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", nil)
			r.RemoteAddr = tt.remoteAddr
			for _, v := range tt.forwarded {
				r.Header.Add("X-Forwarded-For", v)
			}

			assert.Equal(t, tt.want, clientAddr(r, tt.hops))
		})
	}
}
