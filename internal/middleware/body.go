package middleware

import (
	"net/http"

	"rantau-go/internal/pkg/httpx"
)

// maxBodyBytes caps a request body. Every JSON endpoint takes small payloads;
// images go straight to storage through presigned URLs.
const maxBodyBytes = 1 << 20

// WithMaxBody limits request bodies to maxBodyBytes. A handler reading more
// gets a read error, which it reports as an invalid body.
func WithMaxBody() httpx.Constructor {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
			next.ServeHTTP(w, r)
		})
	}
}
