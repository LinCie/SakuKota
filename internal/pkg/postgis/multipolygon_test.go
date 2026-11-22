package postgis_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/pkg/postgis"
)

const square = `[[106.8,-6.2],[106.9,-6.2],[106.9,-6.3],[106.8,-6.3],[106.8,-6.2]]`

func TestMultiPolygon_RoundTrips(t *testing.T) {
	want := `{"type":"MultiPolygon","coordinates":[[` + square + `]]}`

	for _, body := range []string{want, `{"type":"Polygon","coordinates":[` + square + `]}`} {
		var m postgis.MultiPolygon
		require.NoError(t, json.Unmarshal([]byte(body), &m), "Unmarshal(%s)", body)

		out, err := json.Marshal(m)
		require.NoError(t, err, "Marshal(%s)", body)
		// A Polygon is written as a MultiPolygon.
		assert.JSONEq(t, want, string(out), "Marshal(%s)", body)

		value, err := m.Value()
		require.NoError(t, err, "Value() of %s", body)

		var fromDB postgis.MultiPolygon
		require.NoError(t, fromDB.Scan(value), "Scan(%v)", value)
		assert.Equal(t, m, fromDB, "MultiPolygon from Scan of %s", body)
	}
}

func TestMultiPolygon_RejectsBadInput(t *testing.T) {
	for _, body := range []string{
		`{"type":"Point","coordinates":[106.8,-6.2]}`,
		`{"coordinates":[[` + square + `]]}`,
		`{"type":"MultiPolygon","coordinates":[]}`,
		`{"type":"MultiPolygon","coordinates":[[]]}`,
		`{"type":"MultiPolygon","coordinates":null}`,
		`{"type":"Polygon","coordinates":[[[106.8,-6.2],[106.9,-6.2],[106.8,-6.2]]]}`,
		`{"type":"Polygon","coordinates":[[[106.8,-6.2],[106.9,-6.2],[106.9,-6.3],[106.8,-6.3]]]}`,
		`{"type":"Polygon","coordinates":[[[106.8,-6.2,1],[106.9,-6.2,1],[106.9,-6.3,1],[106.8,-6.2,1]]]}`,
		`{"type":"Polygon","coordinates":[[[-6.2,106.8],[-6.2,106.9],[-6.3,106.9],[-6.2,106.8]]]}`,
		`{"type":"Polygon","coordinates":"square"}`,
	} {
		assert.Error(t, json.Unmarshal([]byte(body), &postgis.MultiPolygon{}), "Unmarshal(%s)", body)
	}

	assert.Error(t, (&postgis.MultiPolygon{}).Scan(nil)) // NULL
}

func TestMultiPolygonInIndonesia(t *testing.T) {
	var jakarta, outside postgis.MultiPolygon
	require.NoError(t, json.Unmarshal([]byte(`{"type":"Polygon","coordinates":[`+square+`]}`), &jakarta), "unmarshal jakarta")
	require.NoError(t, json.Unmarshal([]byte(`{"type":"Polygon","coordinates":[[[106.8,-6.2],[150,-6.2],[106.9,-6.3],[106.8,-6.2]]]}`), &outside), "unmarshal outside")

	assert.True(t, postgis.MultiPolygonInIndonesia(&jakarta))
	// One vertex is outside.
	assert.False(t, postgis.MultiPolygonInIndonesia(&outside))
	assert.False(t, postgis.MultiPolygonInIndonesia(nil))
}
