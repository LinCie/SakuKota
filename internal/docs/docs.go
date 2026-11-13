package docs

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var scalarHTML []byte

//go:embed openapi.json
var openAPIJSON []byte

// Scalar serves the interactive API docs page.
func Scalar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(scalarHTML)
}

// OpenAPI serves the OpenAPI spec as JSON.
func OpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(openAPIJSON)
}
