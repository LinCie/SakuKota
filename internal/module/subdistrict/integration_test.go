//go:build integration

package subdistrict_test

import (
	"context"
	"encoding/json"
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
	"rantau-go/internal/module/subdistrict"
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
	mustExec(t, `TRUNCATE SUB_DISTRICTS, PLACES, CATEGORIES RESTART IDENTITY CASCADE`)
}

func mustExec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := testDB.Exec(query, args...); err != nil {
		t.Fatalf("exec failed: %v\nquery: %s", err, query)
	}
}

// seedSubDistricts inserts three sub-districts with IDs 1 to 3:
//
//	1 Menteng           every optional field set
//	2 Kemang            every optional field null
//	3 Kebayoran Baru    every optional field null
func seedSubDistricts(t *testing.T) {
	t.Helper()
	mustExec(t, `INSERT INTO SUB_DISTRICTS (name, slug, flood_history,
			safety_crime_information, estimated_daily_spending, static_map_image_url)
		VALUES ('Menteng', 'menteng', 'Banjir 2020', 'Aman di siang hari', 1500000.50,
			'https://example.com/menteng.png'),
			('Kemang', 'kemang', NULL, NULL, NULL, NULL),
			('Kebayoran Baru', 'kebayoran-baru', NULL, NULL, NULL, NULL)`)
}

// seedPlaceIn adds a category and a place in the sub-district, so that the
// sub-district is referenced by a place.
func seedPlaceIn(t *testing.T, subDistrictID int64) {
	t.Helper()
	mustExec(t, `INSERT INTO CATEGORIES (name, slug, code, label, color) VALUES ('Warung', 'warung', 'W', 'Warung', 'chili')`)
	mustExec(t, `INSERT INTO PLACES (sub_district_id, category_id, name) VALUES ($1, 1, 'Warung Bu Ani')`,
		subDistrictID)
}

// integrationRouter wires the real service and repository behind the HTTP handlers.
func integrationRouter(db *sqlx.DB) http.Handler {
	return newRouter(subdistrict.NewService(subdistrict.NewPostgresRepository(db)))
}

// call sends one request through router.
func call(t *testing.T, router http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serve(t, router, jsonRequest(method, target, body))
}

func countSubDistricts(t *testing.T) int {
	t.Helper()
	var n int
	if err := testDB.Get(&n, `SELECT COUNT(*) FROM SUB_DISTRICTS`); err != nil {
		t.Fatalf("count sub-districts: %v", err)
	}
	return n
}

type subDistrictRow struct {
	ID                     int64    `json:"id"`
	Name                   string   `json:"name"`
	Slug                   string   `json:"slug"`
	FloodHistory           *string  `json:"flood_history"`
	SafetyCrimeInformation *string  `json:"safety_crime_information"`
	EstimatedDailySpending *float64 `json:"estimated_daily_spending"`
	StaticMapImageURL      *string  `json:"static_map_image_url"`
}

// listSubDistricts calls GET /subdistricts and returns the sub-districts in the
// order the API lists them.
func listSubDistricts(t *testing.T, router http.Handler) []subDistrictRow {
	t.Helper()
	rec := call(t, router, http.MethodGet, "/subdistricts", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /subdistricts: status %d; body: %s", rec.Code, rec.Body)
	}

	var rows []subDistrictRow
	if err := json.Unmarshal(responseData(t, rec), &rows); err != nil {
		t.Fatalf("GET /subdistricts: %v", err)
	}
	return rows
}

// listedIDs returns the IDs of the rows in list order.
func listedIDs(rows []subDistrictRow) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

// findSubDistrict returns the listed sub-district with the given id.
func findSubDistrict(t *testing.T, rows []subDistrictRow, id int64) subDistrictRow {
	t.Helper()
	for _, r := range rows {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("sub-district %d is not listed", id)
	return subDistrictRow{}
}

// -----------------------------------------------------------------------------
// Listing sub-districts
// -----------------------------------------------------------------------------

func TestIntegration_ListSubDistrictsWhenEmpty(t *testing.T) {
	resetDB(t)

	rec := call(t, integrationRouter(testDB), http.MethodGet, "/subdistricts", "")
	assertData(t, rec, http.StatusOK, `[]`)
}

func TestIntegration_ListSubDistrictsInIDOrder(t *testing.T) {
	resetDB(t)
	seedSubDistricts(t)

	rows := listSubDistricts(t, integrationRouter(testDB))
	if got := listedIDs(rows); !slices.Equal(got, []int64{1, 2, 3}) {
		t.Fatalf("sub-district IDs = %v, want [1 2 3]", got)
	}

	want := subDistrictRow{
		ID:                     1,
		Name:                   "Menteng",
		Slug:                   "menteng",
		FloodHistory:           ptr("Banjir 2020"),
		SafetyCrimeInformation: ptr("Aman di siang hari"),
		EstimatedDailySpending: ptr(1500000.5),
		StaticMapImageURL:      ptr("https://example.com/menteng.png"),
	}
	if !reflect.DeepEqual(rows[0], want) {
		t.Errorf("first sub-district = %+v, want %+v", rows[0], want)
	}
}

// -----------------------------------------------------------------------------
// Reading one sub-district
// -----------------------------------------------------------------------------

func TestIntegration_GetSubDistrict(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantCode   string
		want       string // data on success
	}{
		{name: "returns every field", target: "/subdistricts/1", wantStatus: http.StatusOK,
			want: `{"id":1,"name":"Menteng","slug":"menteng","flood_history":"Banjir 2020",` +
				`"safety_crime_information":"Aman di siang hari","estimated_daily_spending":1500000.5,` +
				`"static_map_image_url":"https://example.com/menteng.png"}`},
		{name: "unset fields come back as null", target: "/subdistricts/2", wantStatus: http.StatusOK,
			want: `{"id":2,"name":"Kemang","slug":"kemang","flood_history":null,"safety_crime_information":null,` +
				`"estimated_daily_spending":null,"static_map_image_url":null}`},
		{name: "unknown sub-district", target: "/subdistricts/99",
			wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "invalid id", target: "/subdistricts/abc",
			wantStatus: http.StatusBadRequest, wantCode: "INVALID_SUB_DISTRICT_ID"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedSubDistricts(t)

			rec := call(t, integrationRouter(testDB), http.MethodGet, tc.target, "")
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
				return
			}
			assertData(t, rec, tc.wantStatus, tc.want)
		})
	}
}

// -----------------------------------------------------------------------------
// Creating sub-districts
// -----------------------------------------------------------------------------

func TestIntegration_CreateSubDistrictRules(t *testing.T) {
	const tebet = `"name":"Tebet","slug":"tebet"`
	urlOfLength := func(n int) string {
		return "https://example.com/" + strings.Repeat("a", n-len("https://example.com/"))
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string // empty on success
		wantStored int    // sub-districts stored afterwards; the seed holds 3
	}{
		{name: "name and slug only", body: `{` + tebet + `}`, wantStatus: http.StatusCreated, wantStored: 4},
		{name: "every optional field at its limit", wantStatus: http.StatusCreated, wantStored: 4,
			body: `{` + tebet + `,"flood_history":"Banjir","safety_crime_information":"Aman",` +
				`"estimated_daily_spending":99999999.99,"static_map_image_url":"` + urlOfLength(255) + `"}`},
		{name: "name of 100 characters", body: `{"name":"` + strings.Repeat("a", 100) + `","slug":"tebet"}`,
			wantStatus: http.StatusCreated, wantStored: 4},
		{name: "name of 101 characters", body: `{"name":"` + strings.Repeat("a", 101) + `","slug":"tebet"}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 3},
		{name: "blank name", body: `{"name":"   ","slug":"tebet"}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 3},
		{name: "slug of 101 characters", body: `{"name":"Tebet","slug":"` + strings.Repeat("a", 101) + `"}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 3},
		{name: "blank slug", body: `{"name":"Tebet","slug":""}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 3},
		{name: "name already used", body: `{"name":"Menteng","slug":"tebet"}`,
			wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS", wantStored: 3},
		{name: "name already used in another case", body: `{"name":"MENTENG","slug":"tebet"}`,
			wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS", wantStored: 3},
		{name: "name already used in another case, with spaces around it", body: `{"name":"  kemang  ","slug":"tebet"}`,
			wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS", wantStored: 3},
		{name: "slug already used", body: `{"name":"Tebet","slug":"menteng"}`,
			wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS", wantStored: 3},
		{name: "slug differing only in case is allowed", body: `{"name":"Tebet","slug":"MENTENG"}`,
			wantStatus: http.StatusCreated, wantStored: 4},
		{name: "negative estimated daily spending", body: `{` + tebet + `,"estimated_daily_spending":-1}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 3},
		{name: "estimated daily spending too large for DECIMAL(10,2)", body: `{` + tebet + `,"estimated_daily_spending":100000000}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 3},
		{name: "static map image URL of 256 characters", body: `{` + tebet + `,"static_map_image_url":"` + urlOfLength(256) + `"}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedSubDistricts(t)

			rec := call(t, integrationRouter(testDB), http.MethodPost, "/subdistricts", tc.body)
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			} else if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}

			if got := countSubDistricts(t); got != tc.wantStored {
				t.Errorf("stored %d sub-districts, want %d", got, tc.wantStored)
			}
		})
	}
}

func TestIntegration_CreatedSubDistrictIsListedAsSent(t *testing.T) {
	resetDB(t)
	seedSubDistricts(t)
	router := integrationRouter(testDB)

	body := `{"id":999,"name":"Tebet","slug":"tebet",` +
		`"flood_history":"🌊 \"Banjir\" '; DROP TABLE sub_districts; --","safety_crime_information":null,` +
		`"estimated_daily_spending":1500000.50,"static_map_image_url":"https://example.com/tebet.png"}`
	rec := call(t, router, http.MethodPost, "/subdistricts", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d; body: %s", rec.Code, rec.Body)
	}

	var created subDistrictRow
	if err := json.Unmarshal(responseData(t, rec), &created); err != nil {
		t.Fatalf("created resource is not a sub-district: %v", err)
	}
	if created.ID != 4 {
		t.Fatalf("created ID = %d, want 4; the client-sent id must be ignored", created.ID)
	}

	listed := findSubDistrict(t, listSubDistricts(t, router), 4)
	want := subDistrictRow{
		ID:                     4,
		Name:                   "Tebet",
		Slug:                   "tebet",
		FloodHistory:           ptr("🌊 \"Banjir\" '; DROP TABLE sub_districts; --"),
		EstimatedDailySpending: ptr(1500000.5),
		StaticMapImageURL:      ptr("https://example.com/tebet.png"),
	}
	if !reflect.DeepEqual(listed, want) {
		t.Errorf("listed = %+v, want %+v", listed, want)
	}
	if !reflect.DeepEqual(listed, created) {
		t.Errorf("listed = %+v, but create responded with %+v", listed, created)
	}
}

func TestIntegration_ConcurrentCreatesWithSameNameStoreOne(t *testing.T) {
	resetDB(t)
	router := integrationRouter(testDB)

	const body = `{"name":"Tebet","slug":"tebet"}`
	const n = 5
	recs := make([]*httptest.ResponseRecorder, n)

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/subdistricts", body))
			recs[i] = rec
		}()
	}
	wg.Wait()

	if got := countSubDistricts(t); got != 1 {
		t.Errorf("stored %d sub-districts, want 1", got)
	}

	created := 0
	for _, rec := range recs {
		switch rec.Code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			assertError(t, rec, http.StatusConflict, "SUB_DISTRICT_EXISTS")
		default:
			t.Errorf("status %d, want 201 or 409; body: %s", rec.Code, rec.Body)
		}
	}
	if created != 1 {
		t.Errorf("%d requests created a sub-district, want exactly 1", created)
	}
}

// -----------------------------------------------------------------------------
// Updating sub-districts
// -----------------------------------------------------------------------------

func TestIntegration_PatchSubDistrict(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		body       string
		wantStatus int
		wantCode   string
		check      func(t *testing.T, rows []subDistrictRow) // nil means the sub-districts must not change
	}{
		{name: "renames a sub-district and keeps its other fields", target: "/subdistricts/1",
			body: `{"name":"Menteng Atas"}`, wantStatus: http.StatusOK,
			check: func(t *testing.T, rows []subDistrictRow) {
				sd := findSubDistrict(t, rows, 1)
				if sd.Name != "Menteng Atas" || sd.FloodHistory == nil || *sd.FloodHistory != "Banjir 2020" {
					t.Errorf("sub-district 1 = %+v, want name Menteng Atas and flood history kept", sd)
				}
			}},
		{name: "renaming to its own current name is not a duplicate", target: "/subdistricts/1",
			body: `{"name":"Menteng"}`, wantStatus: http.StatusOK, check: nil},
		{name: "slug is replaced", target: "/subdistricts/2", body: `{"slug":"kemang-raya"}`, wantStatus: http.StatusOK,
			check: func(t *testing.T, rows []subDistrictRow) {
				if sd := findSubDistrict(t, rows, 2); sd.Slug != "kemang-raya" {
					t.Errorf("slug = %q, want kemang-raya", sd.Slug)
				}
			}},
		{name: "null clears optional fields", target: "/subdistricts/1",
			body: `{"flood_history":null,"static_map_image_url":null}`, wantStatus: http.StatusOK,
			check: func(t *testing.T, rows []subDistrictRow) {
				sd := findSubDistrict(t, rows, 1)
				if sd.FloodHistory != nil || sd.StaticMapImageURL != nil {
					t.Errorf("sub-district 1 = %+v, want flood history and map URL cleared", sd)
				}
				if sd.SafetyCrimeInformation == nil {
					t.Errorf("safety information was cleared, want it kept")
				}
			}},
		{name: "sets estimated daily spending", target: "/subdistricts/2", body: `{"estimated_daily_spending":750000.25}`,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, rows []subDistrictRow) {
				sd := findSubDistrict(t, rows, 2)
				if sd.EstimatedDailySpending == nil || *sd.EstimatedDailySpending != 750000.25 {
					t.Errorf("estimated daily spending = %v, want 750000.25", sd.EstimatedDailySpending)
				}
			}},
		{name: "null clears estimated daily spending", target: "/subdistricts/1", body: `{"estimated_daily_spending":null}`,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, rows []subDistrictRow) {
				if sd := findSubDistrict(t, rows, 1); sd.EstimatedDailySpending != nil {
					t.Errorf("estimated daily spending = %v, want null", *sd.EstimatedDailySpending)
				}
			}},
		{name: "an empty body changes nothing", target: "/subdistricts/1", body: `{}`, wantStatus: http.StatusOK, check: nil},
		{name: "unknown sub-district", target: "/subdistricts/99", body: `{"name":"Kemang Timur"}`,
			wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND"},
		{name: "invalid body", target: "/subdistricts/1", body: `{"name":5}`,
			wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY"},
		{name: "null name", target: "/subdistricts/1", body: `{"name":null}`,
			wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY"},
		{name: "blank name", target: "/subdistricts/1", body: `{"name":"   "}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "name of 101 characters", target: "/subdistricts/1", body: `{"name":"` + strings.Repeat("a", 101) + `"}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "name used by another sub-district", target: "/subdistricts/2", body: `{"name":"Menteng"}`,
			wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS"},
		{name: "name used by another sub-district in another case", target: "/subdistricts/2", body: `{"name":"MENTENG"}`,
			wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS"},
		{name: "changes the case of its own name", target: "/subdistricts/1", body: `{"name":"menteng"}`,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, rows []subDistrictRow) {
				if sd := findSubDistrict(t, rows, 1); sd.Name != "menteng" {
					t.Errorf("name = %q, want menteng", sd.Name)
				}
			}},
		{name: "slug used by another sub-district", target: "/subdistricts/2", body: `{"slug":"menteng"}`,
			wantStatus: http.StatusConflict, wantCode: "SUB_DISTRICT_EXISTS"},
		{name: "negative estimated daily spending", target: "/subdistricts/1", body: `{"estimated_daily_spending":-1}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedSubDistricts(t)
			router := integrationRouter(testDB)
			before := listSubDistricts(t, router)

			rec := call(t, router, http.MethodPatch, tc.target, tc.body)
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			} else if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}

			after := listSubDistricts(t, router)
			if tc.check == nil {
				if !reflect.DeepEqual(after, before) {
					t.Errorf("sub-districts changed: got %+v, want %+v", after, before)
				}
				return
			}
			tc.check(t, after)
		})
	}
}

// -----------------------------------------------------------------------------
// Deleting sub-districts
// -----------------------------------------------------------------------------

func TestIntegration_DeleteSubDistrict(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantCode   string
		wantIDs    []int64
	}{
		{name: "deletes an unreferenced sub-district", target: "/subdistricts/2",
			wantStatus: http.StatusNoContent, wantIDs: []int64{1, 3}},
		{name: "unknown sub-district leaves the sub-districts", target: "/subdistricts/99",
			wantStatus: http.StatusNotFound, wantCode: "SUB_DISTRICT_NOT_FOUND", wantIDs: []int64{1, 2, 3}},
		{name: "invalid id leaves the sub-districts", target: "/subdistricts/abc",
			wantStatus: http.StatusBadRequest, wantCode: "INVALID_SUB_DISTRICT_ID", wantIDs: []int64{1, 2, 3}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedSubDistricts(t)
			router := integrationRouter(testDB)

			rec := call(t, router, http.MethodDelete, tc.target, "")
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			} else if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}

			if got := listedIDs(listSubDistricts(t, router)); !slices.Equal(got, tc.wantIDs) {
				t.Errorf("sub-district IDs = %v, want %v", got, tc.wantIDs)
			}
		})
	}
}

func TestIntegration_DeleteReferencedSubDistrict(t *testing.T) {
	resetDB(t)
	seedSubDistricts(t)
	seedPlaceIn(t, 1)
	router := integrationRouter(testDB)

	assertError(t, call(t, router, http.MethodDelete, "/subdistricts/1", ""),
		http.StatusConflict, "SUB_DISTRICT_IN_USE")

	if got := listedIDs(listSubDistricts(t, router)); !slices.Equal(got, []int64{1, 2, 3}) {
		t.Errorf("sub-district IDs = %v, want [1 2 3]; a referenced sub-district must not be deleted", got)
	}
}

func TestIntegration_DeleteSubDistrictTwice(t *testing.T) {
	resetDB(t)
	seedSubDistricts(t)
	router := integrationRouter(testDB)

	if rec := call(t, router, http.MethodDelete, "/subdistricts/2", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("first delete: status %d; body: %s", rec.Code, rec.Body)
	}

	assertError(t, call(t, router, http.MethodDelete, "/subdistricts/2", ""),
		http.StatusNotFound, "SUB_DISTRICT_NOT_FOUND")
}

func TestIntegration_NameAndSlugAreFreeAfterDelete(t *testing.T) {
	resetDB(t)
	seedSubDistricts(t)
	router := integrationRouter(testDB)

	if rec := call(t, router, http.MethodDelete, "/subdistricts/2", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: status %d; body: %s", rec.Code, rec.Body)
	}

	rec := call(t, router, http.MethodPost, "/subdistricts", `{"name":"Kemang","slug":"kemang"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create with the deleted name and slug: status %d; body: %s", rec.Code, rec.Body)
	}
}

// -----------------------------------------------------------------------------
// Failures
// -----------------------------------------------------------------------------

func TestIntegration_StorageFailureHidesDetails(t *testing.T) {
	resetDB(t)
	seedSubDistricts(t)

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
		{name: "list sub-districts", method: http.MethodGet, target: "/subdistricts", body: ""},
		{name: "get sub-district", method: http.MethodGet, target: "/subdistricts/1", body: ""},
		{name: "create sub-district", method: http.MethodPost, target: "/subdistricts", body: `{"name":"Tebet","slug":"tebet"}`},
		{name: "update sub-district", method: http.MethodPatch, target: "/subdistricts/1", body: `{"name":"Menteng Atas"}`},
		{name: "delete sub-district", method: http.MethodDelete, target: "/subdistricts/1", body: ""},
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
