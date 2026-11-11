package httpx

import (
	"encoding/json"
	"net/http"
)

type dataResponse struct {
	Data any `json:"data"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RespondJSON writes data as a JSON response with the given status code,
// wrapped in the { "data": any } convention.
func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(dataResponse{Data: data})
}

// RespondError writes a JSON error response with the given status code,
// wrapped in the { "code": string, "message": string } convention.
func RespondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Code: code, Message: message})
}

// RespondInternalError writes a generic 500 response, so storage details stay
// out of the body.
func RespondInternalError(w http.ResponseWriter) {
	RespondError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
}
