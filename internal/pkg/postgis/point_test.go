package postgis_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/pkg/postgis"
)

func TestPoint_RoundTrips(t *testing.T) {
	want := postgis.MustPoint(106.8, -6.2)

	body, err := json.Marshal(want)
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"Point","coordinates":[106.8,-6.2]}`, string(body))

	var fromJSON postgis.Point
	require.NoError(t, json.Unmarshal(body, &fromJSON))
	assert.Equal(t, *want, fromJSON)

	value, err := want.Value()
	require.NoError(t, err)

	var fromDB postgis.Point
	require.NoError(t, fromDB.Scan(value))
	assert.Equal(t, *want, fromDB)
}

func TestPoint_RejectsBadInput(t *testing.T) {
	for _, body := range []string{
		`{"type":"LineString","coordinates":[[106.8,-6.2],[106.9,-6.3]]}`,
		`{"type":"Point","coordinates":[106.8]}`,
		`{"type":"Point","coordinates":[106.8,-6.2,5]}`,
		`{"type":"Point","coordinates":[-6.2,106.8]}`,
		`{"coordinates":[106.8,-6.2]}`,
	} {
		assert.Error(t, json.Unmarshal([]byte(body), &postgis.Point{}), "Unmarshal(%s)", body)
	}

	assert.Error(t, (&postgis.Point{}).Scan(nil)) // NULL
}
