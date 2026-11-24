package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"sakukota-go/internal/pkg/httpx"
)

// WithRateLimit lets each client make at most limit requests per window (whole
// seconds), counted in Redis so every API instance shares the count, and
// answers the rest 429 TOO_MANY_REQUESTS. A client is its address behind
// proxyHops trusted proxies (see clientAddr). If Redis fails, the request is
// answered 500 rather than let through unlimited.
//
// shortcut: every route it wraps shares one count per client; add a key
// prefix parameter when a second, separate limit is needed.
func WithRateLimit(rdb *redis.Client, limit int64, window time.Duration, proxyHops int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			key := "ratelimit:" + clientAddr(r, proxyHops)

			var count *redis.IntCmd
			_, err := rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
				count = p.Incr(ctx, key)
				p.ExpireNX(ctx, key, window)
				return nil
			})
			if err != nil {
				httpx.RespondInternalError(w)
				return
			}
			if count.Val() > limit {
				httpx.RespondError(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientAddr returns the address of the client that sent r through proxyHops
// trusted proxies, each appending its peer to X-Forwarded-For: the entry
// proxyHops places left of r.RemoteAddr, so entries a client forges further
// left are ignored. A shorter chain did not come through the proxies, so its
// RemoteAddr is used. IPv6 addresses are cut to their /64, which one client
// usually holds whole.
func clientAddr(r *http.Request, proxyHops int) string {
	// net/http always sets RemoteAddr as host:port.
	addr, _, _ := net.SplitHostPort(r.RemoteAddr)

	var chain []string
	for _, v := range r.Header.Values("X-Forwarded-For") {
		for entry := range strings.SplitSeq(v, ",") {
			chain = append(chain, strings.TrimSpace(entry))
		}
	}
	if i := len(chain) - proxyHops; proxyHops > 0 && i >= 0 {
		addr = chain[i]
	}

	ip, err := netip.ParseAddr(addr)
	if err != nil {
		return addr
	}
	if ip = ip.Unmap().WithZone(""); ip.Is6() {
		return netip.PrefixFrom(ip, 64).Masked().String()
	}
	return ip.String()
}
