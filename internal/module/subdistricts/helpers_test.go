package subdistricts_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"rantau-go/internal/pkg/postgis"
)

// squareJSON is a small square in Jakarta as a GeoJSON MultiPolygon.
const squareJSON = `{"type":"MultiPolygon","coordinates":[[[[106.8,-6.2],[106.9,-6.2],[106.9,-6.3],[106.8,-6.3],[106.8,-6.2]]]]}`

// mustBoundary reads a GeoJSON Polygon or MultiPolygon.
func mustBoundary(t *testing.T, geoJSON string) *postgis.MultiPolygon {
	t.Helper()

	var m postgis.MultiPolygon
	require.NoError(t, json.Unmarshal([]byte(geoJSON), &m), "unmarshal boundary %s", geoJSON)

	return &m
}
