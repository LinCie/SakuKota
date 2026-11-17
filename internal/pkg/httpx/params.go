package httpx

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// ParseID parses s as a positive int64 identifier.
func ParseID(s string) (int64, bool) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}

// PathID reads a positive integer URL parameter, or answers 400 with code and message.
func PathID(w http.ResponseWriter, r *http.Request, name, code, message string) (int64, bool) {
	id, ok := ParseID(chi.URLParam(r, name))
	if !ok {
		RespondError(w, http.StatusBadRequest, code, message)
	}

	return id, ok
}
