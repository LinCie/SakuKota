package middleware

import (
	"net/http"
	"slices"

	"rantau-go/internal/pkg/httpx"
)

// Guard allows the request only if the role set by WithAuth is one of the
// given roles. It must be applied after WithAuth.
func Guard(roles []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := RoleFromContext(r.Context())
			if !ok {
				httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing role")
				return
			}

			if !slices.Contains(roles, role) {
				httpx.RespondError(w, http.StatusForbidden, httpx.CodeForbidden, "insufficient role")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
