package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// DecodeJSON decodes the request body, which must be exactly one JSON object,
// into a new T. An empty body, malformed JSON, a value that is not an object
// (including null), wrong field types, and data after the object are answered
// with 400 INVALID_BODY, and false is returned.
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
