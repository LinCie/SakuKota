package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sakukota-go/internal/pkg/httpx"
)

type payload struct {
	Name string `json:"name"`
}

func decodeBody(body string) (*payload, bool, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	got, ok := httpx.DecodeJSON[payload](rec, req)
	return got, ok, rec
}

func TestDecodeJSON_Valid(t *testing.T) {
	for _, body := range []string{`{"name":"Ana"}`, " {\"name\":\"Ana\"} \n", `{"name":"Ana","unknown":1}`} {
		got, ok, rec := decodeBody(body)
		require.True(t, ok, "DecodeJSON(%q) response: %s", body, rec.Body.String())
		assert.Equal(t, "Ana", got.Name, "DecodeJSON(%q)", body)
	}
}

func TestDecodeJSON_Invalid(t *testing.T) {
	tests := []struct{ name, body string }{
		{name: "empty", body: ``},
		{name: "malformed", body: `{"name":`},
		{name: "null", body: `null`},
		{name: "array", body: `[]`},
		{name: "wrongType", body: `{"name":5}`},
		{name: "dataAfterObject", body: `{"name":"Ana"} trailing`},
		{name: "twoObjects", body: `{"name":"Ana"}{"name":"Budi"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok, rec := decodeBody(tc.body)
			assert.False(t, ok)
			assert.Nil(t, got)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var resp struct{ Code string }
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp), "response %q", rec.Body.String())
			assert.Equal(t, httpx.CodeInvalidBody, resp.Code)
		})
	}
}
