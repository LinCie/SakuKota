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
	"sync"
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
	mustExec(t, `INSERT INTO CATEGORIES (name, slug) VALUES ('Warung', 'warung'), ('Kopi', 'kopi')`)
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

// listPlaces calls GET target and returns the places in the order the API lists them.
func listPlaces(t *testing.T, router http.Handler, target string) []placeRow {
	t.Helper()
	rec := call(t, router, http.MethodGet, target, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d; body: %s", target, rec.Code, rec.Body)
	}

	var rows []placeRow
	if err := json.Unmarshal(responseData(t, rec), &rows); err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	return rows
}

// fieldJSON builds a text field body. extra is appended to the object, for
// example `,"sort_order":1`.
func fieldJSON(key, extra string) string {
	return fmt.Sprintf(`{"key":%q,"label":"Label","data_type":"text"%s}`, key, extra)
}

type fieldStep struct {
	category   string // "warung", "kopi", or "missing"
	body       string
	wantStatus int
	wantCode   string
}

// added is a step that must create the field.
func added(category, key, extra string) fieldStep {
	return fieldStep{category: category, body: fieldJSON(key, extra), wantStatus: http.StatusCreated}
}

// refused is a step that must fail with the given status and code.
func refused(category, key, extra string, status int, code string) fieldStep {
	return fieldStep{category: category, body: fieldJSON(key, extra), wantStatus: status, wantCode: code}
}

type fieldRow struct {
	Key       string `json:"key"`
	SortOrder int    `json:"sort_order"`
}

// listFields returns the fields of the named category in the order the API lists them.
func listFields(t *testing.T, router http.Handler, category string) []fieldRow {
	t.Helper()
	rec := call(t, router, http.MethodGet, "/categories", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /categories: status %d; body: %s", rec.Code, rec.Body)
	}

	var cats []struct {
		Name   string     `json:"name"`
		Fields []fieldRow `json:"fields"`
	}
	if err := json.Unmarshal(responseData(t, rec), &cats); err != nil {
		t.Fatalf("GET /categories: %v", err)
	}
	for _, c := range cats {
		if c.Name == category {
			return c.Fields
		}
	}
	t.Fatalf("category %q is not listed", category)
	return nil
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
		{name: "unknown sub-district", target: "/places?sub_district_id=99", wantCode: "sub_district_not_found"},
		{name: "unknown category in the list", target: "/places?category_id=1,99", wantCode: "category_not_found"},
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

// -----------------------------------------------------------------------------
// Categories
// -----------------------------------------------------------------------------

func TestIntegration_CategoryNamesAreUniqueIgnoringCase(t *testing.T) {
	tests := []struct {
		name       string
		newName    string
		newSlug    string
		wantStatus int
		wantCode   string
	}{
		{name: "same name in lowercase", newName: "warung", newSlug: "warung-2", wantStatus: http.StatusConflict, wantCode: "category_exists"},
		{name: "same name in uppercase", newName: "WARUNG", newSlug: "warung-3", wantStatus: http.StatusConflict, wantCode: "category_exists"},
		{name: "same slug", newName: "Kedai", newSlug: "warung", wantStatus: http.StatusConflict, wantCode: "category_exists"},
		{name: "different name and slug", newName: "Kedai Warung", newSlug: "kedai-warung", wantStatus: http.StatusCreated, wantCode: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			router := integrationRouter(testDB)

			if rec := call(t, router, http.MethodPost, "/categories", `{"name":"Warung","slug":"warung"}`); rec.Code != http.StatusCreated {
				t.Fatalf("setup: status %d; body: %s", rec.Code, rec.Body)
			}

			body := fmt.Sprintf(`{"name":%q,"slug":%q}`, tc.newName, tc.newSlug)
			rec := call(t, router, http.MethodPost, "/categories", body)
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			} else if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Adding fields
// -----------------------------------------------------------------------------

func TestIntegration_FieldPositions(t *testing.T) {
	categoryIDs := map[string]int{"warung": 1, "kopi": 2, "missing": 99}

	tests := []struct {
		name       string
		steps      []fieldStep
		wantWarung []string // keys in list order after the steps
		wantKopi   []string
	}{
		{
			name:       "without sort_order each field is appended last",
			steps:      []fieldStep{added("warung", "a", ""), added("warung", "b", ""), added("warung", "c", "")},
			wantWarung: []string{"a", "b", "c"},
		},
		{
			name:       "sort_order 1 puts the field first",
			steps:      []fieldStep{added("warung", "a", ""), added("warung", "b", ""), added("warung", "c", `,"sort_order":1`)},
			wantWarung: []string{"c", "a", "b"},
		},
		{
			name:       "a middle insert keeps the other fields in order",
			steps:      []fieldStep{added("warung", "a", ""), added("warung", "b", ""), added("warung", "c", ""), added("warung", "d", `,"sort_order":2`)},
			wantWarung: []string{"a", "d", "b", "c"},
		},
		{
			name:       "one past the last position is accepted and goes last",
			steps:      []fieldStep{added("warung", "a", ""), added("warung", "b", ""), added("warung", "c", `,"sort_order":3`)},
			wantWarung: []string{"a", "b", "c"},
		},
		{
			name:       "beyond one past the last is rejected",
			steps:      []fieldStep{added("warung", "a", ""), added("warung", "b", ""), refused("warung", "c", `,"sort_order":4`, http.StatusBadRequest, "validation_error")},
			wantWarung: []string{"a", "b"},
		},
		{
			name:       "sort_order 0 is rejected and stores nothing",
			steps:      []fieldStep{added("warung", "a", ""), refused("warung", "b", `,"sort_order":0`, http.StatusBadRequest, "validation_error")},
			wantWarung: []string{"a"},
		},
		{
			name:       "negative sort_order is rejected and stores nothing",
			steps:      []fieldStep{added("warung", "a", ""), refused("warung", "b", `,"sort_order":-1`, http.StatusBadRequest, "validation_error")},
			wantWarung: []string{"a"},
		},
		{
			name: "a duplicate key is rejected without shifting other fields",
			steps: []fieldStep{
				added("warung", "a", ""), added("warung", "b", ""), added("warung", "c", ""),
				refused("warung", "a", `,"sort_order":1`, http.StatusConflict, "category_field_exists"),
			},
			wantWarung: []string{"a", "b", "c"},
		},
		{
			name: "a key with surrounding spaces conflicts with its trimmed form",
			steps: []fieldStep{
				added("warung", "halal", ""),
				refused("warung", " halal ", "", http.StatusConflict, "category_field_exists"),
			},
			wantWarung: []string{"halal"},
		},
		{
			name:       "the same key in another category is accepted",
			steps:      []fieldStep{added("warung", "a", ""), added("kopi", "a", "")},
			wantWarung: []string{"a"},
			wantKopi:   []string{"a"},
		},
		{
			name:       "unknown category returns 404 and stores nothing",
			steps:      []fieldStep{refused("missing", "a", "", http.StatusNotFound, "category_not_found")},
			wantWarung: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			router := integrationRouter(testDB)

			for i, s := range tc.steps {
				rec := call(t, router, http.MethodPost, fmt.Sprintf("/categories/%d/fields", categoryIDs[s.category]), s.body)
				if s.wantCode != "" {
					assertError(t, rec, s.wantStatus, s.wantCode)
				} else if rec.Code != s.wantStatus {
					t.Fatalf("step %d: status = %d, want %d; body: %s", i, rec.Code, s.wantStatus, rec.Body)
				}
			}

			assertKeys(t, listFields(t, router, "Warung"), tc.wantWarung)
			assertKeys(t, listFields(t, router, "Kopi"), tc.wantKopi)
		})
	}
}

func TestIntegration_ConcurrentFieldsGetDistinctPositions(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	const n = 10
	statuses := make([]int, n)

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := fmt.Sprintf(`{"key":"field_%d","label":"Field %d","data_type":"text"}`, i, i)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/categories/1/fields", body))
			statuses[i] = rec.Code
		}()
	}
	wg.Wait()

	for i, s := range statuses {
		if s != http.StatusCreated {
			t.Errorf("concurrent add %d: status %d, want 201", i, s)
		}
	}
	fields := listFields(t, router, "Warung")
	if len(fields) != n {
		t.Fatalf("listed %d fields, want %d", len(fields), n)
	}
	for i, f := range fields {
		if f.SortOrder != i+1 {
			t.Errorf("position %d holds %q with sort_order %d", i+1, f.Key, f.SortOrder)
		}
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
		{name: "blank name", body: `{"name":"  ","sub_district_id":1,"category_id":2}`, wantStatus: http.StatusBadRequest, wantCode: "validation_error", wantStored: 0},
		{name: "unknown sub-district", body: `{"name":"Kopi Senja","sub_district_id":99,"category_id":2}`, wantStatus: http.StatusBadRequest, wantCode: "validation_error", wantStored: 0},
		{name: "unknown category", body: `{"name":"Kopi Senja","sub_district_id":1,"category_id":99}`, wantStatus: http.StatusBadRequest, wantCode: "validation_error", wantStored: 0},
		{name: "location in Jakarta", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[106.8,-6.2]}}`, wantStatus: http.StatusCreated, wantCode: "", wantStored: 1},
		{name: "location in Tokyo", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[139.7,35.7]}}`, wantStatus: http.StatusBadRequest, wantCode: "validation_error", wantStored: 0},
		{name: "location at null island", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[0,0]}}`, wantStatus: http.StatusBadRequest, wantCode: "validation_error", wantStored: 0},
		{name: "swapped Jakarta coordinates", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[-6.2,106.8]}}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_body", wantStored: 0},
		{name: "latitude above 90", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[106.8,91]}}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_body", wantStored: 0},
		{name: "missing required attribute", body: `{` + warung + `,"attributes":{"seats":2}}`, wantStatus: http.StatusBadRequest, wantCode: "validation_error", wantStored: 0},
		{name: "unknown attribute key", body: `{` + warung + `,"attributes":{"contact":"0812","parking":true}}`, wantStatus: http.StatusBadRequest, wantCode: "validation_error", wantStored: 0},
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

func TestIntegration_AddingRequiredFieldLeavesExistingPlacesAlone(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	// Created while Warung has no fields.
	if rec := call(t, router, http.MethodPost, "/places", `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":1}`); rec.Code != http.StatusCreated {
		t.Fatalf("setup place: status %d; body: %s", rec.Code, rec.Body)
	}

	// Warung then gets a required field.
	field := `{"key":"contact","label":"Contact","data_type":"text","required":true}`
	if rec := call(t, router, http.MethodPost, "/categories/1/fields", field); rec.Code != http.StatusCreated {
		t.Fatalf("add field: status %d; body: %s", rec.Code, rec.Body)
	}

	rows := listPlaces(t, router, "/places")
	if len(rows) != 1 || rows[0].Name != "Warung Bu Ani" {
		t.Fatalf("existing place is no longer listed as before: %+v", rows)
	}
	assertJSON(t, rows[0].Attributes, `{}`)

	rec := call(t, router, http.MethodPost, "/places", `{"name":"Warung Pagi","sub_district_id":1,"category_id":1}`)
	assertError(t, rec, http.StatusBadRequest, "validation_error")
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
		{name: "create category", method: http.MethodPost, target: "/categories", body: `{"name":"Kedai","slug":"kedai"}`},
		{name: "create place", method: http.MethodPost, target: "/places", body: `{"name":"Kopi","sub_district_id":1,"category_id":2}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := call(t, router, tc.method, tc.target, tc.body)
			assertError(t, rec, http.StatusInternalServerError, "internal_error")
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
		{name: "unknown place", target: "/places/99", body: `{"name":"Warung Baru"}`, wantStatus: http.StatusNotFound, wantCode: "place_not_found", check: nil},
		{name: "invalid body", target: "/places/1", body: `{"name":5}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_body", check: nil},
		{name: "name longer than 150 characters", target: "/places/1",
			body: `{"name":"` + strings.Repeat("a", 151) + `"}`, wantStatus: http.StatusBadRequest, wantCode: "validation_error", check: nil},
		{name: "unknown sub-district", target: "/places/1", body: `{"sub_district_id":99}`, wantStatus: http.StatusBadRequest, wantCode: "validation_error", check: nil},
		{name: "unknown category", target: "/places/1", body: `{"category_id":99}`, wantStatus: http.StatusBadRequest, wantCode: "validation_error", check: nil},
		{name: "attributes miss the required field of the category", target: "/places/1",
			body: `{"attributes":{"seats":3}}`, wantStatus: http.StatusBadRequest, wantCode: "validation_error", check: nil},
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
		{name: "unknown place leaves the places", target: "/places/99", wantStatus: http.StatusNotFound, wantCode: "place_not_found", wantIDs: all},
		{name: "invalid id leaves the places", target: "/places/abc", wantStatus: http.StatusBadRequest, wantCode: "invalid_place_id", wantIDs: all},
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

	assertError(t, call(t, router, http.MethodDelete, "/places/2", ""), http.StatusNotFound, "place_not_found")
}
