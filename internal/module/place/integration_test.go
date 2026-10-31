//go:build integration

package place_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"rantau-go/internal/database"
	"rantau-go/internal/module/place"
)

// testDSN and testDB point at the PostGIS container started by TestMain.
var (
	testDSN string
	testDB  *sqlx.DB
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Same stack as docker/postgres: PostgreSQL 17 with PostGIS 3.
	container, err := postgres.Run(ctx, "postgis/postgis:17-3.5",
		postgres.WithDatabase("rantaugo"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("start postgres container: %v", err)
	}

	testDSN, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("connection string: %v", err)
	}

	testDB, err = database.NewDB(ctx, testDSN)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("goose dialect: %v", err)
	}
	if err := goose.UpContext(ctx, testDB.DB, "../../../migrations"); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	code := m.Run()

	testDB.Close()
	if err := container.Terminate(ctx); err != nil {
		log.Printf("terminate container: %v", err)
	}
	os.Exit(code)
}

// -----------------------------------------------------------------------------
// Fixtures and helpers
// -----------------------------------------------------------------------------

// resetDB empties the tables and restarts their identity sequences, so each
// test starts with IDs 1, 2, 3, ...
func resetDB(t *testing.T) {
	t.Helper()
	mustExec(t, `TRUNCATE PLACES, CATEGORY_FIELDS, CATEGORIES, SUB_DISTRICTS RESTART IDENTITY CASCADE`)
}

func mustExec(t *testing.T, query string) {
	t.Helper()
	if _, err := testDB.Exec(query); err != nil {
		t.Fatalf("exec failed: %v\nquery: %s", err, query)
	}
}

// seedCatalog creates sub-districts 1 (Menteng) and 2 (Kemang) and categories
// 1 (Warung) and 2 (Kopi).
func seedCatalog(t *testing.T) {
	t.Helper()
	mustExec(t, `INSERT INTO SUB_DISTRICTS (name, slug) VALUES ('Menteng', 'menteng'), ('Kemang', 'kemang')`)
	mustExec(t, `INSERT INTO CATEGORIES (name, slug, code, label, color)
		VALUES ('Warung', 'warung', 'W', 'Warung', 'chili'), ('Kopi', 'kopi', 'K', 'Kopi', 'plum')`)
}

// seedWarungFields gives Warung a required "contact" text field and an
// optional "seats" number field.
func seedWarungFields(t *testing.T) {
	t.Helper()
	mustExec(t, `INSERT INTO CATEGORY_FIELDS (category_id, key, label, data_type, required, sort_order)
		VALUES (1, 'contact', 'Contact', 'text', TRUE, 1), (1, 'seats', 'Seats', 'number', FALSE, 2)`)
}

// seedPlaces inserts five places with IDs 1 to 5, in this order:
//
//	1 Warung Bu Ani     Menteng  Warung
//	2 Kopi Senja        Menteng  Kopi
//	3 Warung Bu Ani     Kemang   Warung
//	4 Bakso 50% Enak    Kemang   Warung
//	5 Toko Buku         Kemang   Kopi   (attributes contain "hidden_gem")
func seedPlaces(t *testing.T) {
	t.Helper()
	mustExec(t, `INSERT INTO PLACES (sub_district_id, category_id, name, address, short_description, attributes) VALUES
		(1, 1, 'Warung Bu Ani', 'Jl. Merdeka 1', 'Nasi padang', '{}'),
		(1, 2, 'Kopi Senja', 'Jl. Sudirman 9', 'Kopi tubruk', '{}'),
		(2, 1, 'Warung Bu Ani', 'Jl. Kemang Raya 3', 'Nasi uduk', '{}'),
		(2, 1, 'Bakso 50% Enak', 'Jl. Kemang Utara 5', 'Promo akhir pekan', '{}'),
		(2, 2, 'Toko Buku', 'Jl. Kemang Selatan 2', 'Buku bekas', '{"note": "hidden_gem"}')`)
}

// seedLocations places 1 to 4 in Jakarta; place 5 stays without a location:
//
//	1  106.83, -6.19  (Menteng)
//	2  106.84, -6.20  (Menteng)
//	3  106.81, -6.26  (Kemang)
//	4  106.82, -6.27  (Kemang)
func seedLocations(t *testing.T) {
	t.Helper()
	mustExec(t, `UPDATE PLACES p SET location = ST_SetSRID(ST_MakePoint(v.lng, v.lat), 4326)
		FROM (VALUES (1, 106.83, -6.19), (2, 106.84, -6.20), (3, 106.81, -6.26), (4, 106.82, -6.27))
			AS v(id, lng, lat)
		WHERE p.id = v.id`)
}

// integrationRouter wires the real service and repository behind the HTTP handlers.
func integrationRouter(db *sqlx.DB) http.Handler {
	return newRouter(place.NewService(place.NewPostgresRepository(db)))
}

// call sends one request through router.
func call(t *testing.T, router http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serve(t, router, jsonRequest(method, target, body))
}

func countPlaces(t *testing.T) int {
	t.Helper()
	var n int
	if err := testDB.Get(&n, `SELECT COUNT(*) FROM PLACES`); err != nil {
		t.Fatalf("count places: %v", err)
	}
	return n
}

// createdID returns the id in a successful create response.
func createdID(t *testing.T, rec *httptest.ResponseRecorder) int64 {
	t.Helper()
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(responseData(t, rec), &created); err != nil {
		t.Fatalf("created resource has no id: %v", err)
	}
	return created.ID
}

type placeRow struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Address    *string         `json:"address"`
	Location   json.RawMessage `json:"location"`
	Attributes json.RawMessage `json:"attributes"`
	Category   struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"category"`
}

type placePage struct {
	Places []placeRow `json:"places"`
	Total  int        `json:"total"`
}

// listPage calls GET target and returns the listed places and the total match count.
func listPage(t *testing.T, router http.Handler, target string) placePage {
	t.Helper()
	rec := call(t, router, http.MethodGet, target, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d; body: %s", target, rec.Code, rec.Body)
	}

	var page placePage
	if err := json.Unmarshal(responseData(t, rec), &page); err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	return page
}

// listPlaces calls GET target and returns the places in the order the API lists them.
func listPlaces(t *testing.T, router http.Handler, target string) []placeRow {
	t.Helper()
	return listPage(t, router, target).Places
}

// placeIDs returns the ids of rows in order.
func placeIDs(rows []placeRow) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

type fieldRow struct {
	Key       string `json:"key"`
	SortOrder int    `json:"sort_order"`
}

// assertKeys checks the keys in list order, and that positions run 1..n with
// no gaps or repeats.
func assertKeys(t *testing.T, fields []fieldRow, want []string) {
	t.Helper()
	got := make([]string, 0, len(fields))
	for i, f := range fields {
		got = append(got, f.Key)
		if f.SortOrder != i+1 {
			t.Errorf("field %q has sort_order %d, want %d", f.Key, f.SortOrder, i+1)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
}

// -----------------------------------------------------------------------------
// Browsing places
// -----------------------------------------------------------------------------

func TestIntegration_ListPlacesFilters(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   []int64
	}{
		{name: "no filters returns everything in id order", target: "/places", want: []int64{1, 2, 3, 4, 5}},
		{name: "sub-district", target: "/places?sub_district_id=1", want: []int64{1, 2}},
		{name: "one category", target: "/places?category_id=1", want: []int64{1, 3, 4}},
		{name: "two categories match either", target: "/places?category_id=1,2", want: []int64{1, 2, 3, 4, 5}},
		{name: "overlapping categories do not duplicate rows", target: "/places?category_id=1,1", want: []int64{1, 3, 4}},
		{name: "sub-district and category combine", target: "/places?sub_district_id=2&category_id=2", want: []int64{5}},
		{name: "query matches the name, case-insensitively", target: "/places?q=KOPI%20SENJA", want: []int64{2}},
		{name: "query matches the address", target: "/places?q=jl.%20merdeka", want: []int64{1}},
		{name: "query matches the description", target: "/places?q=nasi", want: []int64{1, 3}},
		{name: "query matches the category name", target: "/places?q=warung", want: []int64{1, 3, 4}},
		{name: "query is literal for percent signs", target: "/places?q=%25", want: []int64{4}},
		{name: "query is literal for underscores", target: "/places?q=_", want: nil},
		{name: "attributes are not searched", target: "/places?q=hidden_gem", want: nil},
		{name: "whitespace-only query is no search", target: "/places?q=%20%20", want: []int64{1, 2, 3, 4, 5}},
		{name: "query combines with filters", target: "/places?sub_district_id=2&category_id=1&q=bakso", want: []int64{4}},
		{name: "no matches is an empty list", target: "/places?q=zzz", want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			seedPlaces(t)

			rows := listPlaces(t, integrationRouter(testDB), tc.target)
			got := make([]int64, 0, len(rows))
			for _, r := range rows {
				got = append(got, r.ID)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("place IDs = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIntegration_ListPlacesUnknownFilters(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		wantCode string
	}{
		{name: "unknown sub-district", target: "/places?sub_district_id=99", wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "unknown category in the list", target: "/places?category_id=1,99", wantCode: "CATEGORY_NOT_FOUND"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			seedPlaces(t)

			rec := call(t, integrationRouter(testDB), http.MethodGet, tc.target, "")
			assertError(t, rec, http.StatusNotFound, tc.wantCode)
		})
	}
}

func TestIntegration_ListPlacesLimit(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   []int64
	}{
		{name: "limit keeps the lowest ids", target: "/places?limit=2", want: []int64{1, 2}},
		{name: "limit above the count returns everything", target: "/places?limit=50", want: []int64{1, 2, 3, 4, 5}},
		{name: "limit applies after the sub-district filter", target: "/places?sub_district_id=2&limit=2", want: []int64{3, 4}},
		{name: "limit applies after the category filter", target: "/places?category_id=1&limit=2", want: []int64{1, 3}},
		{name: "limit applies after the query", target: "/places?q=nasi&limit=1", want: []int64{1}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			seedPlaces(t)

			rows := listPlaces(t, integrationRouter(testDB), tc.target)
			got := make([]int64, 0, len(rows))
			for _, r := range rows {
				got = append(got, r.ID)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("place IDs = %v, want %v", got, tc.want)
			}
		})
	}
}

// everywhere is a bbox around all seeded locations.
const everywhere = "bbox=106.0,-7.0,107.0,-6.0"

func TestIntegration_ListPlacesBBox(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   []int64
	}{
		{name: "bbox keeps the places inside it", target: "/places?bbox=106.80,-6.28,106.83,-6.25", want: []int64{3, 4}},
		{name: "bbox drops places without a location", target: "/places?" + everywhere, want: []int64{1, 2, 3, 4}},
		{name: "a place on the edge is inside", target: "/places?bbox=106.83,-6.19,106.90,-6.10", want: []int64{1}},
		{name: "bbox with no places is an empty list", target: "/places?bbox=100,0,101,1", want: nil},
		{name: "bbox combines with the sub-district", target: "/places?sub_district_id=1&" + everywhere, want: []int64{1, 2}},
		{name: "bbox combines with the category", target: "/places?category_id=1&" + everywhere, want: []int64{1, 3, 4}},
		{name: "bbox combines with the query", target: "/places?q=nasi&" + everywhere, want: []int64{1, 3}},
		{name: "limit applies inside the bbox, lowest ids first", target: "/places?limit=2&bbox=106.80,-6.28,106.85,-6.18", want: []int64{1, 2}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			seedPlaces(t)
			seedLocations(t)

			got := placeIDs(listPlaces(t, integrationRouter(testDB), tc.target))
			if !slices.Equal(got, tc.want) {
				t.Errorf("place IDs = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIntegration_ListPlacesTotal(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		wantIDs   []int64
		wantTotal int
	}{
		{name: "no filters counts everything", target: "/places", wantIDs: []int64{1, 2, 3, 4, 5}, wantTotal: 5},
		{name: "limit does not shrink the total", target: "/places?limit=2", wantIDs: []int64{1, 2}, wantTotal: 5},
		{name: "total counts after the filters", target: "/places?sub_district_id=2&limit=2", wantIDs: []int64{3, 4}, wantTotal: 3},
		{name: "total counts inside the bbox", target: "/places?limit=1&" + everywhere, wantIDs: []int64{1}, wantTotal: 4},
		{name: "no matches has a zero total", target: "/places?q=zzz", wantIDs: nil, wantTotal: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			seedPlaces(t)
			seedLocations(t)

			page := listPage(t, integrationRouter(testDB), tc.target)
			if got := placeIDs(page.Places); !slices.Equal(got, tc.wantIDs) {
				t.Errorf("place IDs = %v, want %v", got, tc.wantIDs)
			}
			if page.Total != tc.wantTotal {
				t.Errorf("total = %d, want %d", page.Total, tc.wantTotal)
			}
		})
	}
}

func TestIntegration_PlaceLocationsHaveSpatialIndex(t *testing.T) {
	var exists bool
	err := testDB.Get(&exists, `SELECT EXISTS (
		SELECT 1 FROM pg_indexes
		WHERE tablename = 'places' AND indexdef ILIKE '%USING gist (location)%'
	)`)
	if err != nil {
		t.Fatalf("query indexes: %v", err)
	}
	if !exists {
		t.Error("PLACES.location has no GIST index")
	}
}

// -----------------------------------------------------------------------------
// Creating places
// -----------------------------------------------------------------------------

func TestIntegration_CreatePlaceRules(t *testing.T) {
	const kopi = `"name":"Kopi Senja","sub_district_id":1,"category_id":2`
	const warung = `"name":"Warung Bu Ani","sub_district_id":1,"category_id":1`

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string // empty on success
		wantStored int
	}{
		{name: "minimal place", body: `{` + kopi + `}`, wantStatus: http.StatusCreated, wantCode: "", wantStored: 1},
		{name: "blank name", body: `{"name":"  ","sub_district_id":1,"category_id":2}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "unknown sub-district", body: `{"name":"Kopi Senja","sub_district_id":99,"category_id":2}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "unknown category", body: `{"name":"Kopi Senja","sub_district_id":1,"category_id":99}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "location in Jakarta", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[106.8,-6.2]}}`, wantStatus: http.StatusCreated, wantCode: "", wantStored: 1},
		{name: "location in Tokyo", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[139.7,35.7]}}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "location at null island", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[0,0]}}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "swapped Jakarta coordinates", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[-6.2,106.8]}}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY", wantStored: 0},
		{name: "latitude above 90", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[106.8,91]}}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY", wantStored: 0},
		{name: "missing required attribute", body: `{` + warung + `,"attributes":{"seats":2}}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "unknown attribute key", body: `{` + warung + `,"attributes":{"contact":"0812","parking":true}}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "null optional attribute is accepted", body: `{` + warung + `,"attributes":{"contact":"0812","seats":null}}`, wantStatus: http.StatusCreated, wantCode: "", wantStored: 1},
		{name: "unknown top-level keys are ignored", body: `{` + kopi + `,"id":999,"created_at":"yesterday"}`, wantStatus: http.StatusCreated, wantCode: "", wantStored: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			seedWarungFields(t)

			rec := call(t, integrationRouter(testDB), http.MethodPost, "/places", tc.body)
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			} else if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}

			if got := countPlaces(t); got != tc.wantStored {
				t.Errorf("stored %d places, want %d", got, tc.wantStored)
			}
		})
	}
}

func TestIntegration_PlaceNamesMayRepeatInDistrict(t *testing.T) {
	tests := []struct {
		name   string
		first  string
		second string
	}{
		{name: "same name", first: "Warung Bu Ani", second: "Warung Bu Ani"},
		{name: "same name in another case", first: "Warung Bu Ani", second: "warung bu ani"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			router := integrationRouter(testDB)

			var ids []int64
			for _, name := range []string{tc.first, tc.second} {
				body := fmt.Sprintf(`{"name":%q,"sub_district_id":1,"category_id":2}`, name)
				rec := call(t, router, http.MethodPost, "/places", body)
				if rec.Code != http.StatusCreated {
					t.Fatalf("create %q: status %d; body: %s", name, rec.Code, rec.Body)
				}
				ids = append(ids, createdID(t, rec))
			}

			if ids[0] == ids[1] {
				t.Errorf("both places got id %d", ids[0])
			}
			if rows := listPlaces(t, router, "/places?sub_district_id=1"); len(rows) != 2 {
				t.Errorf("listed %d places in the district, want 2", len(rows))
			}
		})
	}
}

func TestIntegration_StoredAttributesComeBackExactly(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantAttributes string
	}{
		{
			name:           "large integer keeps every digit",
			body:           `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":1,"attributes":{"contact":"0812","seats":12345678901234567890}}`,
			wantAttributes: `{"contact":"0812","seats":12345678901234567890}`,
		},
		{
			name:           "decimal keeps its trailing zero",
			body:           `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":1,"attributes":{"contact":"0812","seats":1.50}}`,
			wantAttributes: `{"contact":"0812","seats":1.50}`,
		},
		{
			name:           "text keeps quotes, emoji, and SQL-like content",
			body:           `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":1,"attributes":{"contact":"🍜 \"Bu Ani\" '; DROP TABLE places; --"}}`,
			wantAttributes: `{"contact":"🍜 \"Bu Ani\" '; DROP TABLE places; --"}`,
		},
		{
			name:           "optional null is dropped",
			body:           `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":1,"attributes":{"contact":"0812","seats":null}}`,
			wantAttributes: `{"contact":"0812"}`,
		},
		{
			name:           "no attributes is an empty object",
			body:           `{"name":"Kopi Senja","sub_district_id":1,"category_id":2}`,
			wantAttributes: `{}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			seedWarungFields(t)
			router := integrationRouter(testDB)

			if rec := call(t, router, http.MethodPost, "/places", tc.body); rec.Code != http.StatusCreated {
				t.Fatalf("status %d; body: %s", rec.Code, rec.Body)
			}

			rows := listPlaces(t, router, "/places")
			if len(rows) != 1 {
				t.Fatalf("listed %d places, want 1", len(rows))
			}
			assertJSON(t, rows[0].Attributes, tc.wantAttributes)
		})
	}
}

func TestIntegration_CreatedPlaceIsListedAsSent(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	seedWarungFields(t)
	router := integrationRouter(testDB)

	body := `{"name":"Kopi Senja","sub_district_id":1,"category_id":2,` +
		`"location":{"type":"Point","coordinates":[106.8,-6.2]}}`
	if rec := call(t, router, http.MethodPost, "/places", body); rec.Code != http.StatusCreated {
		t.Fatalf("status %d; body: %s", rec.Code, rec.Body)
	}

	rows := listPlaces(t, router, "/places?sub_district_id=1")
	if len(rows) != 1 {
		t.Fatalf("listed %d places, want 1", len(rows))
	}
	p := rows[0]
	if p.Name != "Kopi Senja" {
		t.Errorf("name = %q, want Kopi Senja", p.Name)
	}
	if p.Address != nil {
		t.Errorf("absent address came back as %q, want null", *p.Address)
	}
	if p.Category.ID != 2 || p.Category.Name != "Kopi" || p.Category.Slug != "kopi" {
		t.Errorf("category = %+v, want Kopi (id 2)", p.Category)
	}
	assertJSON(t, p.Location, `{"type":"Point","coordinates":[106.8,-6.2]}`)
}

// -----------------------------------------------------------------------------
// Failures
// -----------------------------------------------------------------------------

func TestIntegration_StorageFailureHidesDetails(t *testing.T) {
	resetDB(t)
	seedCatalog(t)

	db, err := database.NewDB(context.Background(), testDSN)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	router := integrationRouter(db)
	db.Close() // every query from here on fails

	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{name: "list places", method: http.MethodGet, target: "/places", body: ""},
		{name: "list categories", method: http.MethodGet, target: "/categories", body: ""},
		{name: "create place", method: http.MethodPost, target: "/places", body: `{"name":"Kopi","sub_district_id":1,"category_id":2}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := call(t, router, tc.method, tc.target, tc.body)
			assertError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
			if strings.Contains(rec.Body.String(), "database is closed") {
				t.Errorf("500 body exposes driver detail: %s", rec.Body)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Updating and deleting places
// -----------------------------------------------------------------------------

// findPlace returns the listed place with the given id.
func findPlace(t *testing.T, rows []placeRow, id int64) placeRow {
	t.Helper()
	for _, r := range rows {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("place %d is not listed", id)
	return placeRow{}
}

func TestIntegration_PatchPlace(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		body       string
		wantStatus int
		wantCode   string
		check      func(t *testing.T, rows []placeRow) // nil means the places must not change
	}{
		{name: "renames a place and keeps its other fields", target: "/places/1", body: `{"name":"Warung Baru"}`, wantStatus: http.StatusOK, wantCode: "",
			check: func(t *testing.T, rows []placeRow) {
				p := findPlace(t, rows, 1)
				if p.Name != "Warung Baru" || p.Address == nil || *p.Address != "Jl. Merdeka 1" {
					t.Errorf("place 1 = name %q, address %v; want Warung Baru and Jl. Merdeka 1", p.Name, p.Address)
				}
			}},
		{name: "null clears an optional field", target: "/places/1", body: `{"address":null}`, wantStatus: http.StatusOK, wantCode: "",
			check: func(t *testing.T, rows []placeRow) {
				p := findPlace(t, rows, 1)
				if p.Address != nil {
					t.Errorf("address = %q, want null", *p.Address)
				}
				if p.Name != "Warung Bu Ani" {
					t.Errorf("name = %q, want Warung Bu Ani", p.Name)
				}
			}},
		{name: "location is stored and listed", target: "/places/1", body: `{"location":{"type":"Point","coordinates":[106.8,-6.2]}}`, wantStatus: http.StatusOK, wantCode: "",
			check: func(t *testing.T, rows []placeRow) {
				assertJSON(t, findPlace(t, rows, 1).Location, `{"type":"Point","coordinates":[106.8,-6.2]}`)
			}},
		{name: "moves a place to a category with the attributes it needs", target: "/places/2",
			body: `{"category_id":1,"attributes":{"contact":"0812"}}`, wantStatus: http.StatusOK, wantCode: "",
			check: func(t *testing.T, rows []placeRow) {
				p := findPlace(t, rows, 2)
				if p.Category.ID != 1 {
					t.Errorf("category = %d, want 1", p.Category.ID)
				}
				assertJSON(t, p.Attributes, `{"contact":"0812"}`)
			}},
		{name: "an empty body changes nothing", target: "/places/1", body: `{}`, wantStatus: http.StatusOK, wantCode: "", check: nil},
		{name: "unknown place", target: "/places/99", body: `{"name":"Warung Baru"}`, wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND", check: nil},
		{name: "invalid body", target: "/places/1", body: `{"name":5}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY", check: nil},
		{name: "name longer than 150 characters", target: "/places/1",
			body: `{"name":"` + strings.Repeat("a", 151) + `"}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", check: nil},
		{name: "unknown sub-district", target: "/places/1", body: `{"sub_district_id":99}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", check: nil},
		{name: "unknown category", target: "/places/1", body: `{"category_id":99}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", check: nil},
		{name: "attributes miss the required field of the category", target: "/places/1",
			body: `{"attributes":{"seats":3}}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", check: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			seedWarungFields(t)
			seedPlaces(t)

			router := integrationRouter(testDB)
			before := listPlaces(t, router, "/places")

			rec := call(t, router, http.MethodPatch, tc.target, tc.body)
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			} else if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}

			after := listPlaces(t, router, "/places")
			if tc.check == nil {
				if !reflect.DeepEqual(after, before) {
					t.Errorf("places changed: got %+v, want %+v", after, before)
				}
				return
			}
			tc.check(t, after)
		})
	}
}

func TestIntegration_DeletePlace(t *testing.T) {
	all := []int64{1, 2, 3, 4, 5}

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantCode   string
		wantIDs    []int64
	}{
		{name: "deletes the place", target: "/places/2", wantStatus: http.StatusNoContent, wantCode: "", wantIDs: []int64{1, 3, 4, 5}},
		{name: "unknown place leaves the places", target: "/places/99", wantStatus: http.StatusNotFound, wantCode: "PLACE_NOT_FOUND", wantIDs: all},
		{name: "invalid id leaves the places", target: "/places/abc", wantStatus: http.StatusBadRequest, wantCode: "INVALID_PLACE_ID", wantIDs: all},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			seedWarungFields(t)
			seedPlaces(t)

			router := integrationRouter(testDB)
			rec := call(t, router, http.MethodDelete, tc.target, "")
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			} else if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}

			rows := listPlaces(t, router, "/places")
			got := make([]int64, 0, len(rows))
			for _, r := range rows {
				got = append(got, r.ID)
			}
			if !slices.Equal(got, tc.wantIDs) {
				t.Errorf("place IDs = %v, want %v", got, tc.wantIDs)
			}
		})
	}
}

func TestIntegration_DeletePlaceTwice(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	seedWarungFields(t)
	seedPlaces(t)

	router := integrationRouter(testDB)
	first := call(t, router, http.MethodDelete, "/places/2", "")
	if first.Code != http.StatusNoContent {
		t.Fatalf("first delete: status %d; body: %s", first.Code, first.Body)
	}

	assertError(t, call(t, router, http.MethodDelete, "/places/2", ""), http.StatusNotFound, "PLACE_NOT_FOUND")
}

// -----------------------------------------------------------------------------
// Reading one place
// -----------------------------------------------------------------------------

func TestIntegration_GetPlace(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	seedWarungFields(t)
	seedPlaces(t)
	router := integrationRouter(testDB)

	t.Run("returns the place with its category", func(t *testing.T) {
		rec := call(t, router, http.MethodGet, "/places/5", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
		}

		var got placeRow
		if err := json.Unmarshal(responseData(t, rec), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.ID != 5 || got.Name != "Toko Buku" {
			t.Errorf("got place %d %q, want 5 %q", got.ID, got.Name, "Toko Buku")
		}
		if got.Address == nil || *got.Address != "Jl. Kemang Selatan 2" {
			t.Errorf("address = %v, want Jl. Kemang Selatan 2", got.Address)
		}
		if got.Category.ID != 2 || got.Category.Name != "Kopi" || got.Category.Slug != "kopi" {
			t.Errorf("category = %+v, want Kopi (2)", got.Category)
		}
		assertJSON(t, got.Attributes, `{"note":"hidden_gem"}`)
	})

	t.Run("unknown place", func(t *testing.T) {
		assertError(t, call(t, router, http.MethodGet, "/places/99", ""), http.StatusNotFound, "PLACE_NOT_FOUND")
	})

	t.Run("invalid id", func(t *testing.T) {
		assertError(t, call(t, router, http.MethodGet, "/places/abc", ""), http.StatusBadRequest, "INVALID_PLACE_ID")
	})
}

type ratingRow struct {
	ID            int64    `json:"id"`
	RatingAverage *float64 `json:"rating_average"`
	RatingCount   int      `json:"rating_count"`
}

// ratingOf decodes the rating of the place in a successful response.
func ratingOf(t *testing.T, rec *httptest.ResponseRecorder) ratingRow {
	t.Helper()
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body)
	}
	var row ratingRow
	if err := json.Unmarshal(responseData(t, rec), &row); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return row
}

func TestIntegration_PlaceRatings(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	seedWarungFields(t)
	seedPlaces(t)
	// Place 2 has three ratings adding up to 11; the others have none.
	mustExec(t, `UPDATE PLACES SET rating_count = 3, rating_sum = 11 WHERE id = 2`)
	router := integrationRouter(testDB)

	rated := ratingRow{ID: 2, RatingAverage: ptr(11.0 / 3.0), RatingCount: 3}

	t.Run("listed places carry their ratings", func(t *testing.T) {
		rec := call(t, router, http.MethodGet, "/places?sub_district_id=1", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
		}
		var page struct {
			Places []ratingRow `json:"places"`
		}
		if err := json.Unmarshal(responseData(t, rec), &page); err != nil {
			t.Fatalf("decode: %v", err)
		}
		want := []ratingRow{{ID: 1, RatingAverage: nil, RatingCount: 0}, rated}
		if !reflect.DeepEqual(page.Places, want) {
			t.Errorf("ratings = %+v, want %+v", page.Places, want)
		}
	})

	t.Run("one place carries its rating", func(t *testing.T) {
		if got := ratingOf(t, call(t, router, http.MethodGet, "/places/2", "")); !reflect.DeepEqual(got, rated) {
			t.Errorf("rating = %+v, want %+v", got, rated)
		}
	})

	t.Run("a new place is not rated", func(t *testing.T) {
		rec := call(t, router, http.MethodPost, "/places", `{"name":"Kopi Baru","sub_district_id":1,"category_id":2}`)
		got := ratingOf(t, rec)
		if got.RatingAverage != nil || got.RatingCount != 0 {
			t.Errorf("new place rating = %+v, want none", got)
		}
	})

	t.Run("editing a place keeps its rating", func(t *testing.T) {
		rec := call(t, router, http.MethodPatch, "/places/2", `{"name":"Kopi Senja Baru"}`)
		if got := ratingOf(t, rec); !reflect.DeepEqual(got, rated) {
			t.Errorf("edited place rating = %+v, want %+v", got, rated)
		}
		if got := ratingOf(t, call(t, router, http.MethodGet, "/places/2", "")); !reflect.DeepEqual(got, rated) {
			t.Errorf("stored rating after the edit = %+v, want %+v", got, rated)
		}
	})
}

// -----------------------------------------------------------------------------
// Reading one category
// -----------------------------------------------------------------------------

func TestIntegration_GetCategory(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	seedWarungFields(t)
	router := integrationRouter(testDB)

	type categoryBody struct {
		ID     int64      `json:"id"`
		Name   string     `json:"name"`
		Slug   string     `json:"slug"`
		Code   string     `json:"code"`
		Label  string     `json:"label"`
		Color  string     `json:"color"`
		Fields []fieldRow `json:"fields"`
	}

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantCode   string
		wantName   string
		wantStyle  string // code, label and color, joined by spaces
		wantKeys   []string
	}{
		{name: "category with fields in order", target: "/categories/1", wantStatus: http.StatusOK, wantName: "Warung", wantStyle: "W Warung chili", wantKeys: []string{"contact", "seats"}},
		{name: "category without fields", target: "/categories/2", wantStatus: http.StatusOK, wantName: "Kopi", wantStyle: "K Kopi plum", wantKeys: []string{}},
		{name: "unknown category", target: "/categories/99", wantStatus: http.StatusNotFound, wantCode: "CATEGORY_NOT_FOUND"},
		{name: "invalid id", target: "/categories/abc", wantStatus: http.StatusBadRequest, wantCode: "INVALID_CATEGORY_ID"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := call(t, router, http.MethodGet, tc.target, "")
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
				return
			}
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}

			var got categoryBody
			if err := json.Unmarshal(responseData(t, rec), &got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got.Name != tc.wantName {
				t.Errorf("name = %q, want %q", got.Name, tc.wantName)
			}
			if style := got.Code + " " + got.Label + " " + got.Color; style != tc.wantStyle {
				t.Errorf("code, label and color = %q, want %q", style, tc.wantStyle)
			}
			if got.Fields == nil {
				t.Error("fields is null, want an array")
			}
			assertKeys(t, got.Fields, tc.wantKeys)
		})
	}
}
