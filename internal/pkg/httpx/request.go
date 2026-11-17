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
	var v *T // a pointer, so a body of null leaves it nil
	decoder := json.NewDecoder(r.Body)

	invalid := decoder.Decode(&v) != nil || v == nil
	// A second read must hit the end of the body, or there is data after the object.
	if invalid || !errors.Is(decoder.Decode(&json.RawMessage{}), io.EOF) {
		RespondError(w, http.StatusBadRequest, CodeInvalidBody, "invalid request body")
		return nil, false
	}

	return v, true
}
