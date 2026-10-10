package httpx

import "net/http"

type Constructor func(http.Handler) http.Handler
