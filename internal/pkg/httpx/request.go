package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// DecodeJSON reads the request body as exactly one JSON object into a new T.
// Anything else gets a 400 INVALID_BODY response, and false is returned.
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (*T, bool) {
	var v *T
	decoder := json.NewDecoder(r.Body)

	// A body of null decodes to a nil pointer and is rejected with the others.
	err := decoder.Decode(&v)
	if err == nil && v != nil {
		if err = decoder.Decode(&json.RawMessage{}); errors.Is(err, io.EOF) {
			return v, true
		}
	}

	RespondError(w, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
	return nil, false
}
