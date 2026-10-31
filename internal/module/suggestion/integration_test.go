//go:build integration

package suggestion_test

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
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"rantau-go/internal/database"
	"rantau-go/internal/module/place"
	"rantau-go/internal/module/suggestion"
	"rantau-go/internal/storage"
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

// The seeded users.
const (
	lin   int64 = 1
	sari  int64 = 2
	admin int64 = 3
)

// resetDB empties the tables and restarts their identity sequences, so each
// test starts with IDs 1, 2, 3, ...
func resetDB(t *testing.T) {
	t.Helper()
	mustExec(t, `TRUNCATE PLACE_SUGGESTIONS, PLACE_REVIEWS, PLACES, CATEGORY_FIELDS, CATEGORIES, SUB_DISTRICTS, USERS
		RESTART IDENTITY CASCADE`)
}

func mustExec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := testDB.Exec(query, args...); err != nil {
		t.Fatalf("exec failed: %v\nquery: %s", err, query)
	}
}

// seedCatalog creates sub-districts 1 (Menteng) and 2 (Kemang), categories 1
// (Warung) and 2 (Kopi), and users 1 (Lin), 2 (Sari), and 3 (Admin, an
// admin). Warung has a required "contact" text field and an optional "seats"
// number field. Kopi has no fields.
func seedCatalog(t *testing.T) {
	t.Helper()
	mustExec(t, `INSERT INTO SUB_DISTRICTS (name, slug) VALUES ('Menteng', 'menteng'), ('Kemang', 'kemang')`)
	mustExec(t, `INSERT INTO CATEGORIES (name, slug, code, label, color)
		VALUES ('Warung', 'warung', 'W', 'Warung', 'chili'), ('Kopi', 'kopi', 'K', 'Kopi', 'plum')`)
	mustExec(t, `INSERT INTO CATEGORY_FIELDS (category_id, key, label, data_type, required, sort_order)
		VALUES (1, 'contact', 'Contact', 'text', TRUE, 1), (1, 'seats', 'Seats', 'number', FALSE, 2)`)
	mustExec(t, `INSERT INTO USERS (name, email, password_hash, role) VALUES
		('Lin', 'lin@example.com', 'x', 'user'),
		('Sari', 'sari@example.com', 'x', 'user'),
		('Admin', 'admin@example.com', 'x', 'admin')`)
}

// integrationRouter wires the real services and repositories behind the HTTP
// handlers. Approving a suggestion goes through the real place service.
func integrationRouter(db *sqlx.DB) http.Handler {
	places := place.NewService(place.NewPostgresRepository(db), fakeImages{})
	return newRouter(suggestion.NewService(suggestion.NewPostgresRepository(db), places))
}

// fakeImages stands in for object storage. Suggestions never touch images.
type fakeImages struct{}

func (fakeImages) PresignUpload(_ context.Context, prefix, _ string) (storage.Upload, error) {
	return storage.Upload{Key: "uploads/" + prefix + "/new.webp"}, nil
}

func (fakeImages) Promote(_ context.Context, prefix, _ string) (string, error) {
	return prefix + "/new.webp", nil
}

func (fakeImages) Delete(context.Context, string) error { return nil }

func (fakeImages) PublicURL(key string) string { return "https://img.test/" + key }

// kopiBody is a minimal suggestion of a Kopi named name in Menteng.
func kopiBody(name string) string {
	return fmt.Sprintf(`{"name":%q,"sub_district_id":1,"category_id":2,`+
		`"location":{"type":"Point","coordinates":[106.8,-6.2]}}`, name)
}

// warungPlace is a final place for approving a suggestion: a Warung in
// Menteng with its required contact.
const warungPlace = `{"name":"Warung Bu Ani","sub_district_id":1,"category_id":1,` +
	`"location":{"type":"Point","coordinates":[106.83,-6.19]},` +
	`"address":"Jl. Sabang 12","attributes":{"contact":"0812","seats":12}}`

// submit sends the user's suggestion.
func submit(t *testing.T, router http.Handler, userID int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serve(t, router, authRequest(t, http.MethodPost, "/suggestions", body, userID))
}

// mustSubmit sends the user's suggestion, fails unless it is created, and
// returns its id.
func mustSubmit(t *testing.T, router http.Handler, userID int64, body string) int64 {
	t.Helper()
	rec := submit(t, router, userID, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("submit as user %d: status %d; body: %s", userID, rec.Code, rec.Body)
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(responseData(t, rec), &created); err != nil {
		t.Fatalf("created suggestion has no id: %v", err)
	}
	return created.ID
}

// approve sends the admin's approval of suggestion id as the place in body.
func approve(t *testing.T, router http.Handler, id int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serve(t, router, adminRequest(t, http.MethodPost, fmt.Sprintf("/suggestions/%d/approve", id), body, admin))
}

// reject sends the admin's rejection of suggestion id.
func reject(t *testing.T, router http.Handler, id int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serve(t, router, adminRequest(t, http.MethodPost, fmt.Sprintf("/suggestions/%d/reject", id), body, admin))
}

type authorRow struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type suggestionRow struct {
	ID                  int64           `json:"id"`
	SubDistrictID       int64           `json:"sub_district_id"`
	CategoryID          int64           `json:"category_id"`
	Name                string          `json:"name"`
	Address             *string         `json:"address"`
	Location            json.RawMessage `json:"location"`
	OperatingHours      *string         `json:"operating_hours"`
	EstimatedPriceRange *string         `json:"estimated_price_range"`
	ShortDescription    *string         `json:"short_description"`
	Attributes          json.RawMessage `json:"attributes"`
	Note                *string         `json:"note"`
	Status              string          `json:"status"`
	RejectionReason     *string         `json:"rejection_reason"`
	PlaceID             *int64          `json:"place_id"`
	ReviewedAt          *string         `json:"reviewed_at"`
	CreatedAt           string          `json:"created_at"`
	User                authorRow       `json:"user"`
}

// decodeSuggestion returns the suggestion in a successful response.
func decodeSuggestion(t *testing.T, rec *httptest.ResponseRecorder) suggestionRow {
	t.Helper()
	var row suggestionRow
	if err := json.Unmarshal(responseData(t, rec), &row); err != nil {
		t.Fatalf("response is not a suggestion: %v; body: %s", err, rec.Body)
	}
	return row
}

// list sends req and returns the suggestions in the order the API lists them.
func list(t *testing.T, router http.Handler, req *http.Request) []suggestionRow {
	t.Helper()
	rec := serve(t, router, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s %s: status %d; body: %s", req.Method, req.URL, rec.Code, rec.Body)
	}

	var rows []suggestionRow
	if err := json.Unmarshal(responseData(t, rec), &rows); err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL, err)
	}
	return rows
}

// mine lists the user's own suggestions.
func mine(t *testing.T, router http.Handler, userID int64) []suggestionRow {
	t.Helper()
	return list(t, router, authRequest(t, http.MethodGet, "/suggestions/mine", "", userID))
}

// suggestionIDs returns the ids of rows in order.
func suggestionIDs(rows []suggestionRow) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

// reviewState is the stored review state of a suggestion.
type reviewState struct {
	Status          string     `db:"status"`
	PlaceID         *int64     `db:"place_id"`
	ReviewedBy      *int64     `db:"reviewed_by"`
	ReviewedAt      *time.Time `db:"reviewed_at"`
	RejectionReason *string    `db:"rejection_reason"`
}

func storedState(t *testing.T, id int64) reviewState {
	t.Helper()
	var s reviewState
	err := testDB.Get(&s, `SELECT status, place_id, reviewed_by, reviewed_at, rejection_reason
		FROM PLACE_SUGGESTIONS WHERE id = $1`, id)
	if err != nil {
		t.Fatalf("read suggestion %d: %v", id, err)
	}
	return s
}

func countRows(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := testDB.Get(&n, query, args...); err != nil {
		t.Fatalf("count failed: %v\nquery: %s", err, query)
	}
	return n
}

func countPlaces(t *testing.T) int {
	t.Helper()
	return countRows(t, `SELECT COUNT(*) FROM PLACES`)
}

func countSuggestions(t *testing.T) int {
	t.Helper()
	return countRows(t, `SELECT COUNT(*) FROM PLACE_SUGGESTIONS`)
}

func countPending(t *testing.T, userID int64) int {
	t.Helper()
	return countRows(t, `SELECT COUNT(*) FROM PLACE_SUGGESTIONS WHERE user_id = $1 AND status = 'pending'`, userID)
}

// placeRecord is a stored place as approval created it.
type placeRecord struct {
	Name          string  `db:"name"`
	SubDistrictID int64   `db:"sub_district_id"`
	CategoryID    int64   `db:"category_id"`
	Address       *string `db:"address"`
	Lon           float64 `db:"lon"`
	Lat           float64 `db:"lat"`
	Attributes    string  `db:"attributes"`
}

func storedPlace(t *testing.T, id int64) placeRecord {
	t.Helper()
	var p placeRecord
	err := testDB.Get(&p, `SELECT name, sub_district_id, category_id, address,
			ST_X(location) AS lon, ST_Y(location) AS lat, attributes::text AS attributes
		FROM PLACES WHERE id = $1`, id)
	if err != nil {
		t.Fatalf("read place %d: %v", id, err)
	}
	return p
}

// sendAll sends reqs through router at once and returns their responses in
// the same order.
func sendAll(router http.Handler, reqs []*http.Request) []*httptest.ResponseRecorder {
	recs := make([]*httptest.ResponseRecorder, len(reqs))
	var wg sync.WaitGroup
	for i := range reqs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, reqs[i])
			recs[i] = rec
		}()
	}
	wg.Wait()
	return recs
}

// -----------------------------------------------------------------------------
// Submitting suggestions
// -----------------------------------------------------------------------------

func TestIntegration_SubmittedSuggestionIsReadBackAsSent(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	rec := submit(t, router, lin, `{"name":"  Warung Bu Ani  ","sub_district_id":1,"category_id":1,`+
		`"location":{"type":"Point","coordinates":[106.83,-6.19]},`+
		`"address":"  Jl. Sabang 12  ","operating_hours":"07:00-21:00","estimated_price_range":"   ",`+
		`"attributes":{"seats":12},"note":"  Murah, porsi besar  "}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body)
	}
	created := decodeSuggestion(t, rec)

	// Attributes missing Warung's required contact are accepted: they are
	// only checked against the category when an admin approves.
	assertJSON(t, created.Location, `{"type":"Point","coordinates":[106.83,-6.19]}`)
	assertJSON(t, created.Attributes, `{"seats":12}`)
	if at, err := time.Parse(time.RFC3339Nano, created.CreatedAt); err != nil || at.IsZero() {
		t.Errorf("created_at = %q, want a timestamp", created.CreatedAt)
	}

	got := created
	got.Location, got.Attributes, got.CreatedAt = nil, nil, ""
	want := suggestionRow{
		ID:             1,
		SubDistrictID:  1,
		CategoryID:     1,
		Name:           "Warung Bu Ani",
		Address:        ptr("Jl. Sabang 12"),
		OperatingHours: ptr("07:00-21:00"),
		Note:           ptr("Murah, porsi besar"),
		Status:         "pending",
		User:           authorRow{ID: lin, Name: "Lin"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("created = %+v, want %+v", got, want)
	}

	read := serve(t, router, adminRequest(t, http.MethodGet, "/suggestions/1", "", admin))
	if read.Code != http.StatusOK {
		t.Fatalf("GET /suggestions/1: status %d; body: %s", read.Code, read.Body)
	}
	if got := decodeSuggestion(t, read); !reflect.DeepEqual(got, created) {
		t.Errorf("read back %+v, but create responded with %+v", got, created)
	}

	if listed := mine(t, router, lin); len(listed) != 1 || !reflect.DeepEqual(listed[0], created) {
		t.Errorf("Lin's suggestions = %+v, want only %+v", listed, created)
	}
	if countPlaces(t) != 0 {
		t.Error("submitting a suggestion created a place")
	}
}

func TestIntegration_CreateSuggestionRules(t *testing.T) {
	const kopi = `"name":"Kopi Senja","sub_district_id":1,"category_id":2`
	const jakarta = `"location":{"type":"Point","coordinates":[106.8,-6.2]}`

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string // empty on success
		wantStored int
	}{
		{name: "minimal suggestion", body: `{` + kopi + `,` + jakarta + `}`, wantStatus: http.StatusCreated, wantCode: "", wantStored: 1},
		{name: "blank name", body: `{"name":"  ","sub_district_id":1,"category_id":2,` + jakarta + `}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "name too long", body: `{"name":"` + strings.Repeat("a", 151) + `","sub_district_id":1,"category_id":2,` + jakarta + `}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "missing location", body: `{` + kopi + `}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "null location", body: `{` + kopi + `,"location":null}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "location in Tokyo", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[139.7,35.7]}}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "swapped Jakarta coordinates", body: `{` + kopi + `,"location":{"type":"Point","coordinates":[-6.2,106.8]}}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY", wantStored: 0},
		{name: "unknown sub-district", body: `{"name":"Kopi Senja","sub_district_id":99,"category_id":2,` + jakarta + `}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "unknown category", body: `{"name":"Kopi Senja","sub_district_id":1,"category_id":99,` + jakarta + `}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "attributes not an object", body: `{` + kopi + `,` + jakarta + `,"attributes":[1,2]}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "attributes unknown to the category are accepted", body: `{` + kopi + `,` + jakarta + `,"attributes":{"parking":true}}`, wantStatus: http.StatusCreated, wantCode: "", wantStored: 1},
		{name: "note too long", body: `{` + kopi + `,` + jakarta + `,"note":"` + strings.Repeat("a", 1001) + `"}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantStored: 0},
		{name: "malformed body", body: `{"name":`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY", wantStored: 0},
		{name: "unknown top-level keys are ignored", body: `{` + kopi + `,` + jakarta + `,"id":999,"created_at":"yesterday"}`, wantStatus: http.StatusCreated, wantCode: "", wantStored: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)

			rec := submit(t, integrationRouter(testDB), lin, tc.body)
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			} else if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}

			if got := countSuggestions(t); got != tc.wantStored {
				t.Errorf("stored %d suggestions, want %d", got, tc.wantStored)
			}
		})
	}
}

func TestIntegration_SubmittedSuggestionIsPending(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	mustExec(t, `INSERT INTO PLACES (sub_district_id, category_id, name) VALUES (1, 2, 'Kopi Senja')`)
	router := integrationRouter(testDB)

	// The client claims the suggestion is already approved as place 1.
	rec := submit(t, router, lin, `{"name":"Kopi Pagi","sub_district_id":1,"category_id":2,`+
		`"location":{"type":"Point","coordinates":[106.8,-6.2]},`+
		`"status":"approved","place_id":1,"reviewed_by":3,"rejection_reason":"x","user_id":2}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body)
	}

	created := decodeSuggestion(t, rec)
	if created.Status != "pending" || created.PlaceID != nil || created.ReviewedAt != nil || created.RejectionReason != nil {
		t.Errorf("created = %+v, want a pending suggestion with no review", created)
	}
	if created.User.ID != lin {
		t.Errorf("author = %d, want Lin from the token", created.User.ID)
	}
	if got, want := storedState(t, created.ID), (reviewState{Status: "pending"}); !reflect.DeepEqual(got, want) {
		t.Errorf("stored state = %+v, want %+v", got, want)
	}
}

// -----------------------------------------------------------------------------
// Pending limit
// -----------------------------------------------------------------------------

func TestIntegration_PendingLimit(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	for i := range suggestion.MaxPendingPerUser {
		mustSubmit(t, router, lin, kopiBody(fmt.Sprintf("Kopi %d", i+1)))
	}

	// A sixth pending suggestion is refused.
	rec := submit(t, router, lin, kopiBody("Kopi 6"))
	assertError(t, rec, http.StatusTooManyRequests, "TOO_MANY_PENDING_SUGGESTIONS")
	if got := countPending(t, lin); got != suggestion.MaxPendingPerUser {
		t.Fatalf("Lin has %d pending suggestions, want %d", got, suggestion.MaxPendingPerUser)
	}

	// The limit is per user.
	mustSubmit(t, router, sari, kopiBody("Kopi Sari"))

	// A rejected suggestion no longer counts.
	if rec := reject(t, router, 1, `{}`); rec.Code != http.StatusOK {
		t.Fatalf("reject: status %d; body: %s", rec.Code, rec.Body)
	}
	mustSubmit(t, router, lin, kopiBody("Kopi 6"))
	assertError(t, submit(t, router, lin, kopiBody("Kopi 7")), http.StatusTooManyRequests, "TOO_MANY_PENDING_SUGGESTIONS")

	// Neither does an approved one.
	if rec := approve(t, router, 2, kopiBody("Kopi 2")); rec.Code != http.StatusOK {
		t.Fatalf("approve: status %d; body: %s", rec.Code, rec.Body)
	}
	mustSubmit(t, router, lin, kopiBody("Kopi 7"))

	if got := countPending(t, lin); got != suggestion.MaxPendingPerUser {
		t.Errorf("Lin has %d pending suggestions, want %d", got, suggestion.MaxPendingPerUser)
	}
}

func TestIntegration_ConcurrentSubmitsRespectPendingLimit(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	// Lin sends twice the limit at once.
	n := 2 * suggestion.MaxPendingPerUser
	reqs := make([]*http.Request, n)
	for i := range n {
		reqs[i] = authRequest(t, http.MethodPost, "/suggestions", kopiBody(fmt.Sprintf("Kopi %d", i+1)), lin)
	}

	created := 0
	for _, rec := range sendAll(router, reqs) {
		switch rec.Code {
		case http.StatusCreated:
			created++
		case http.StatusTooManyRequests:
			assertError(t, rec, http.StatusTooManyRequests, "TOO_MANY_PENDING_SUGGESTIONS")
		default:
			t.Errorf("status %d, want 201 or 429; body: %s", rec.Code, rec.Body)
		}
	}
	if created != suggestion.MaxPendingPerUser {
		t.Errorf("%d requests created a suggestion, want exactly %d", created, suggestion.MaxPendingPerUser)
	}
	if got := countPending(t, lin); got != suggestion.MaxPendingPerUser {
		t.Errorf("Lin has %d pending suggestions, want %d", got, suggestion.MaxPendingPerUser)
	}
}

// -----------------------------------------------------------------------------
// Approving suggestions
// -----------------------------------------------------------------------------

func TestIntegration_ApproveCreatesThePlaceTheAdminSent(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	// Lin's suggestion has a typo and misses Warung's required contact. The
	// admin corrects both when approving.
	id := mustSubmit(t, router, lin, `{"name":"Warung Bu Anni","sub_district_id":2,"category_id":1,`+
		`"location":{"type":"Point","coordinates":[106.8,-6.2]},"note":"Murah"}`)

	rec := approve(t, router, id, warungPlace)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
	}

	approved := decodeSuggestion(t, rec)
	if approved.Status != "approved" || approved.PlaceID == nil || *approved.PlaceID != 1 {
		t.Fatalf("approved = %+v, want status approved with place 1", approved)
	}
	if approved.ReviewedAt == nil {
		t.Error("reviewed_at is null, want the approval time")
	}
	// The suggestion keeps what Lin sent; the place has what the admin sent.
	if approved.Name != "Warung Bu Anni" || approved.SubDistrictID != 2 {
		t.Errorf("suggestion changed to %q in sub-district %d, want Lin's original", approved.Name, approved.SubDistrictID)
	}

	if got := countPlaces(t); got != 1 {
		t.Fatalf("stored %d places, want 1", got)
	}
	p := storedPlace(t, 1)
	wantPlace := placeRecord{
		Name: "Warung Bu Ani", SubDistrictID: 1, CategoryID: 1, Address: ptr("Jl. Sabang 12"),
		Lon: 106.83, Lat: -6.19, Attributes: p.Attributes,
	}
	if !reflect.DeepEqual(p, wantPlace) {
		t.Errorf("stored place = %+v, want %+v", p, wantPlace)
	}
	assertJSON(t, json.RawMessage(p.Attributes), `{"contact":"0812","seats":12}`)

	state := storedState(t, id)
	if state.Status != "approved" || state.PlaceID == nil || *state.PlaceID != 1 ||
		state.ReviewedBy == nil || *state.ReviewedBy != admin || state.ReviewedAt == nil || state.RejectionReason != nil {
		t.Errorf("stored state = %+v, want approved as place 1 by admin %d", state, admin)
	}

	// Lin sees the outcome.
	if listed := mine(t, router, lin); len(listed) != 1 || listed[0].Status != "approved" {
		t.Errorf("Lin's suggestions = %+v, want the approved one", listed)
	}
}

func TestIntegration_ApproveRules(t *testing.T) {
	const menteng = `"sub_district_id":1`
	const jakarta = `"location":{"type":"Point","coordinates":[106.8,-6.2]}`

	tests := []struct {
		name       string
		target     int64 // suggestion to approve
		body       string
		wantStatus int
		wantCode   string // empty on success
		wantPlaces int
		wantState  string // stored status of suggestion 1 afterwards
	}{
		{name: "valid place", target: 1, body: warungPlace,
			wantStatus: http.StatusOK, wantCode: "", wantPlaces: 1, wantState: "approved"},
		{name: "missing required attribute", target: 1,
			body:       `{"name":"Warung Bu Ani",` + menteng + `,"category_id":1,` + jakarta + `,"attributes":{"seats":2}}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantPlaces: 0, wantState: "pending"},
		{name: "unknown attribute key", target: 1,
			body:       `{"name":"Kopi Senja",` + menteng + `,"category_id":2,` + jakarta + `,"attributes":{"parking":true}}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantPlaces: 0, wantState: "pending"},
		{name: "missing location", target: 1,
			body:       `{"name":"Kopi Senja",` + menteng + `,"category_id":2}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantPlaces: 0, wantState: "pending"},
		{name: "location in Tokyo", target: 1,
			body:       `{"name":"Kopi Senja",` + menteng + `,"category_id":2,"location":{"type":"Point","coordinates":[139.7,35.7]}}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantPlaces: 0, wantState: "pending"},
		{name: "blank name", target: 1,
			body:       `{"name":"  ",` + menteng + `,"category_id":2,` + jakarta + `}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantPlaces: 0, wantState: "pending"},
		{name: "unknown sub-district", target: 1,
			body:       `{"name":"Kopi Senja","sub_district_id":99,"category_id":2,` + jakarta + `}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantPlaces: 0, wantState: "pending"},
		{name: "unknown category", target: 1,
			body:       `{"name":"Kopi Senja",` + menteng + `,"category_id":99,` + jakarta + `}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantPlaces: 0, wantState: "pending"},
		{name: "malformed body", target: 1, body: `{"name":`,
			wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY", wantPlaces: 0, wantState: "pending"},
		{name: "unknown suggestion", target: 99, body: warungPlace,
			wantStatus: http.StatusNotFound, wantCode: "SUGGESTION_NOT_FOUND", wantPlaces: 0, wantState: "pending"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			router := integrationRouter(testDB)
			mustSubmit(t, router, lin, kopiBody("Kopi Senja"))

			rec := approve(t, router, tc.target, tc.body)
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			} else if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}

			if got := countPlaces(t); got != tc.wantPlaces {
				t.Errorf("stored %d places, want %d", got, tc.wantPlaces)
			}
			if got := storedState(t, 1).Status; got != tc.wantState {
				t.Errorf("suggestion 1 status = %q, want %q", got, tc.wantState)
			}
		})
	}
}

func TestIntegration_ReviewedSuggestionCannotBeReviewedAgain(t *testing.T) {
	tests := []struct {
		name       string
		first      func(t *testing.T, router http.Handler) *httptest.ResponseRecorder
		second     func(t *testing.T, router http.Handler) *httptest.ResponseRecorder
		wantState  string
		wantPlaces int
	}{
		{
			name:       "approve an approved suggestion",
			first:      func(t *testing.T, r http.Handler) *httptest.ResponseRecorder { return approve(t, r, 1, warungPlace) },
			second:     func(t *testing.T, r http.Handler) *httptest.ResponseRecorder { return approve(t, r, 1, warungPlace) },
			wantState:  "approved",
			wantPlaces: 1,
		},
		{
			name:       "reject an approved suggestion",
			first:      func(t *testing.T, r http.Handler) *httptest.ResponseRecorder { return approve(t, r, 1, warungPlace) },
			second:     func(t *testing.T, r http.Handler) *httptest.ResponseRecorder { return reject(t, r, 1, `{}`) },
			wantState:  "approved",
			wantPlaces: 1,
		},
		{
			name:       "approve a rejected suggestion",
			first:      func(t *testing.T, r http.Handler) *httptest.ResponseRecorder { return reject(t, r, 1, `{}`) },
			second:     func(t *testing.T, r http.Handler) *httptest.ResponseRecorder { return approve(t, r, 1, warungPlace) },
			wantState:  "rejected",
			wantPlaces: 0,
		},
		{
			name: "reject a rejected suggestion",
			first: func(t *testing.T, r http.Handler) *httptest.ResponseRecorder {
				return reject(t, r, 1, `{"reason":"Tutup"}`)
			},
			second: func(t *testing.T, r http.Handler) *httptest.ResponseRecorder {
				return reject(t, r, 1, `{"reason":"Lain"}`)
			},
			wantState:  "rejected",
			wantPlaces: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			router := integrationRouter(testDB)
			mustSubmit(t, router, lin, kopiBody("Kopi Senja"))

			if rec := tc.first(t, router); rec.Code != http.StatusOK {
				t.Fatalf("first review: status %d; body: %s", rec.Code, rec.Body)
			}
			before := storedState(t, 1)

			assertError(t, tc.second(t, router), http.StatusConflict, "ALREADY_REVIEWED")

			if got := storedState(t, 1); !reflect.DeepEqual(got, before) {
				t.Errorf("stored state changed from %+v to %+v", before, got)
			}
			if got := storedState(t, 1).Status; got != tc.wantState {
				t.Errorf("status = %q, want %q", got, tc.wantState)
			}
			if got := countPlaces(t); got != tc.wantPlaces {
				t.Errorf("stored %d places, want %d", got, tc.wantPlaces)
			}
		})
	}
}

func TestIntegration_ConcurrentApprovesCreateOnePlace(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)
	mustSubmit(t, router, lin, kopiBody("Kopi Senja"))

	// Several admins approve the same suggestion at once, like a double click.
	const n = 5
	reqs := make([]*http.Request, n)
	for i := range n {
		reqs[i] = adminRequest(t, http.MethodPost, "/suggestions/1/approve", warungPlace, admin)
	}

	approved := 0
	for _, rec := range sendAll(router, reqs) {
		switch rec.Code {
		case http.StatusOK:
			approved++
		case http.StatusConflict:
			assertError(t, rec, http.StatusConflict, "ALREADY_REVIEWED")
		default:
			t.Errorf("status %d, want 200 or 409; body: %s", rec.Code, rec.Body)
		}
	}
	if approved != 1 {
		t.Errorf("%d requests approved the suggestion, want exactly 1", approved)
	}

	// Places created by the losing requests are removed again.
	if got := countPlaces(t); got != 1 {
		t.Fatalf("stored %d places, want 1", got)
	}
	var placeID int64
	if err := testDB.Get(&placeID, `SELECT id FROM PLACES`); err != nil {
		t.Fatalf("read place: %v", err)
	}
	if state := storedState(t, 1); state.PlaceID == nil || *state.PlaceID != placeID {
		t.Errorf("suggestion points at place %v, want the stored place %d", state.PlaceID, placeID)
	}
}

// -----------------------------------------------------------------------------
// Rejecting suggestions
// -----------------------------------------------------------------------------

func TestIntegration_RejectRules(t *testing.T) {
	tests := []struct {
		name       string
		target     int64
		body       string
		wantStatus int
		wantCode   string  // empty on success
		wantState  string  // stored status of suggestion 1 afterwards
		wantReason *string // stored reason of suggestion 1 afterwards
	}{
		{name: "reason is trimmed", target: 1, body: `{"reason":"  Sudah tutup  "}`,
			wantStatus: http.StatusOK, wantCode: "", wantState: "rejected", wantReason: ptr("Sudah tutup")},
		{name: "no reason", target: 1, body: `{}`,
			wantStatus: http.StatusOK, wantCode: "", wantState: "rejected", wantReason: nil},
		{name: "blank reason is stored as none", target: 1, body: `{"reason":"   "}`,
			wantStatus: http.StatusOK, wantCode: "", wantState: "rejected", wantReason: nil},
		{name: "reason too long", target: 1, body: `{"reason":"` + strings.Repeat("a", 1001) + `"}`,
			wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantState: "pending", wantReason: nil},
		{name: "malformed body", target: 1, body: `{"reason":`,
			wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY", wantState: "pending", wantReason: nil},
		{name: "unknown suggestion", target: 99, body: `{}`,
			wantStatus: http.StatusNotFound, wantCode: "SUGGESTION_NOT_FOUND", wantState: "pending", wantReason: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)
			router := integrationRouter(testDB)
			mustSubmit(t, router, lin, kopiBody("Kopi Senja"))

			rec := reject(t, router, tc.target, tc.body)
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			} else if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}

			state := storedState(t, 1)
			if state.Status != tc.wantState {
				t.Errorf("status = %q, want %q", state.Status, tc.wantState)
			}
			if !reflect.DeepEqual(state.RejectionReason, tc.wantReason) {
				t.Errorf("reason = %v, want %v", state.RejectionReason, tc.wantReason)
			}
			if countPlaces(t) != 0 {
				t.Error("rejecting created a place")
			}
		})
	}
}

func TestIntegration_RejectedSuggestionShowsTheReasonToItsAuthor(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)
	mustSubmit(t, router, lin, kopiBody("Kopi Senja"))

	rec := reject(t, router, 1, `{"reason":"Sudah tutup"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	rejected := decodeSuggestion(t, rec)
	if rejected.Status != "rejected" || rejected.RejectionReason == nil || *rejected.RejectionReason != "Sudah tutup" ||
		rejected.ReviewedAt == nil || rejected.PlaceID != nil {
		t.Errorf("rejected = %+v, want status rejected with the reason", rejected)
	}

	state := storedState(t, 1)
	if state.ReviewedBy == nil || *state.ReviewedBy != admin || state.ReviewedAt == nil {
		t.Errorf("stored state = %+v, want reviewed by admin %d", state, admin)
	}

	listed := mine(t, router, lin)
	if len(listed) != 1 || !reflect.DeepEqual(listed[0], rejected) {
		t.Errorf("Lin's suggestions = %+v, want only %+v", listed, rejected)
	}
}

// -----------------------------------------------------------------------------
// Listing suggestions
// -----------------------------------------------------------------------------

// seedSuggestions inserts place 1 and five suggestions with IDs 1 to 5:
//
//	1 Lin    approved as place 1   created 2026-10-01
//	2 Lin    pending               created 2026-10-03
//	3 Sari   rejected              created 2026-10-03
//	4 Lin    pending               created 2026-10-03
//	5 Sari   pending               created 2026-10-05
func seedSuggestions(t *testing.T) {
	t.Helper()
	mustExec(t, `INSERT INTO PLACES (sub_district_id, category_id, name) VALUES (1, 2, 'Kopi Senja')`)
	mustExec(t, `INSERT INTO PLACE_SUGGESTIONS (user_id, sub_district_id, category_id, name, location,
			status, place_id, reviewed_by, reviewed_at, rejection_reason, created_at) VALUES
		(1, 1, 2, 'Kopi Senja', ST_SetSRID(ST_MakePoint(106.8, -6.2), 4326),
			'approved', 1, 3, '2026-10-02T09:00:00Z', NULL, '2026-10-01T08:00:00Z'),
		(1, 1, 2, 'Kopi Pagi', ST_SetSRID(ST_MakePoint(106.8, -6.2), 4326),
			'pending', NULL, NULL, NULL, NULL, '2026-10-03T08:00:00Z'),
		(2, 1, 2, 'Kopi Sore', ST_SetSRID(ST_MakePoint(106.8, -6.2), 4326),
			'rejected', NULL, 3, '2026-10-04T09:00:00Z', 'Duplikat', '2026-10-03T08:00:00Z'),
		(1, 2, 2, 'Kopi Malam', ST_SetSRID(ST_MakePoint(106.8, -6.2), 4326),
			'pending', NULL, NULL, NULL, NULL, '2026-10-03T08:00:00Z'),
		(2, 2, 1, 'Warung Sari', ST_SetSRID(ST_MakePoint(106.8, -6.2), 4326),
			'pending', NULL, NULL, NULL, NULL, '2026-10-05T08:00:00Z')`)
}

func TestIntegration_ListMySuggestions(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	seedSuggestions(t)
	router := integrationRouter(testDB)

	tests := []struct {
		name   string
		userID int64
		want   []int64
	}{
		{name: "newest first, later id first on a tie", userID: lin, want: []int64{4, 2, 1}},
		{name: "only the user's own suggestions", userID: sari, want: []int64{5, 3}},
		{name: "user without suggestions gets an empty list", userID: admin, want: []int64{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := suggestionIDs(mine(t, router, tc.userID)); !slices.Equal(got, tc.want) {
				t.Errorf("suggestion IDs = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIntegration_ListSuggestions(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	seedSuggestions(t)
	router := integrationRouter(testDB)

	tests := []struct {
		name   string
		target string
		want   []int64
	}{
		{name: "oldest first, earlier id first on a tie", target: "/suggestions", want: []int64{1, 2, 3, 4, 5}},
		{name: "pending queue", target: "/suggestions?status=pending", want: []int64{2, 4, 5}},
		{name: "approved", target: "/suggestions?status=approved", want: []int64{1}},
		{name: "rejected", target: "/suggestions?status=rejected", want: []int64{3}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := suggestionIDs(list(t, router, adminRequest(t, http.MethodGet, tc.target, "", admin)))
			if !slices.Equal(got, tc.want) {
				t.Errorf("suggestion IDs = %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("unknown status", func(t *testing.T) {
		rec := serve(t, router, adminRequest(t, http.MethodGet, "/suggestions?status=deleted", "", admin))
		assertError(t, rec, http.StatusBadRequest, "INVALID_STATUS")
	})
}

func TestIntegration_ListedSuggestionsCarryTheirAuthors(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	seedSuggestions(t)
	router := integrationRouter(testDB)

	for _, req := range []*http.Request{
		adminRequest(t, http.MethodGet, "/suggestions", "", admin),
		adminRequest(t, http.MethodGet, "/suggestions/3", "", admin),
		authRequest(t, http.MethodGet, "/suggestions/mine", "", lin),
	} {
		rec := serve(t, router, req)
		if strings.Contains(rec.Body.String(), "@example.com") {
			t.Errorf("%s exposes emails: %s", req.URL, rec.Body)
		}
	}

	listed := list(t, router, adminRequest(t, http.MethodGet, "/suggestions?status=rejected", "", admin))
	if len(listed) != 1 {
		t.Fatalf("listed %d rejected suggestions, want 1", len(listed))
	}
	got := listed[0]
	if got.User != (authorRow{ID: sari, Name: "Sari"}) {
		t.Errorf("author = %+v, want Sari", got.User)
	}
	if got.RejectionReason == nil || *got.RejectionReason != "Duplikat" {
		t.Errorf("rejection_reason = %v, want Duplikat", got.RejectionReason)
	}
	if got.ReviewedAt == nil {
		t.Fatal("reviewed_at is null")
	}
	if at, err := time.Parse(time.RFC3339Nano, *got.ReviewedAt); err != nil || !at.Equal(time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("reviewed_at = %q, want 2026-10-04T09:00:00Z", *got.ReviewedAt)
	}
	assertJSON(t, got.Location, `{"type":"Point","coordinates":[106.8,-6.2]}`)
	assertJSON(t, got.Attributes, `{}`)
}

func TestIntegration_GetUnknownSuggestion(t *testing.T) {
	resetDB(t)
	seedCatalog(t)

	rec := serve(t, integrationRouter(testDB), adminRequest(t, http.MethodGet, "/suggestions/99", "", admin))
	assertError(t, rec, http.StatusNotFound, "SUGGESTION_NOT_FOUND")
}

// -----------------------------------------------------------------------------
// Deleting what suggestions refer to
// -----------------------------------------------------------------------------

func TestIntegration_DeletingSubDistrictRemovesItsSuggestions(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	mustSubmit(t, router, lin, kopiBody("Kopi Menteng"))
	mustSubmit(t, router, lin, `{"name":"Kopi Kemang","sub_district_id":2,"category_id":2,`+
		`"location":{"type":"Point","coordinates":[106.8,-6.2]}}`)

	// Suggestions do not keep a sub-district from being deleted.
	mustExec(t, `DELETE FROM SUB_DISTRICTS WHERE id = 2`)

	rec := serve(t, router, adminRequest(t, http.MethodGet, "/suggestions/2", "", admin))
	assertError(t, rec, http.StatusNotFound, "SUGGESTION_NOT_FOUND")
	if got := suggestionIDs(mine(t, router, lin)); !slices.Equal(got, []int64{1}) {
		t.Errorf("Lin's suggestion IDs = %v, want [1]", got)
	}
}

func TestIntegration_DeletingUserRemovesTheirSuggestions(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	mustSubmit(t, router, lin, kopiBody("Kopi Lin"))
	mustSubmit(t, router, sari, kopiBody("Kopi Sari"))

	mustExec(t, `DELETE FROM USERS WHERE id = $1`, lin)

	got := suggestionIDs(list(t, router, adminRequest(t, http.MethodGet, "/suggestions", "", admin)))
	if !slices.Equal(got, []int64{2}) {
		t.Errorf("suggestion IDs = %v, want only Sari's [2]", got)
	}
}

func TestIntegration_DeletingApprovedPlaceKeepsTheSuggestion(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	id := mustSubmit(t, router, lin, kopiBody("Kopi Senja"))
	if rec := approve(t, router, id, warungPlace); rec.Code != http.StatusOK {
		t.Fatalf("approve: status %d; body: %s", rec.Code, rec.Body)
	}

	mustExec(t, `DELETE FROM PLACES WHERE id = 1`)

	state := storedState(t, id)
	if state.Status != "approved" || state.PlaceID != nil {
		t.Errorf("stored state = %+v, want still approved with no place", state)
	}
	if listed := mine(t, router, lin); len(listed) != 1 || listed[0].PlaceID != nil {
		t.Errorf("Lin's suggestions = %+v, want the approved one with place_id null", listed)
	}
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
		name string
		req  *http.Request
	}{
		{name: "create suggestion", req: authRequest(t, http.MethodPost, "/suggestions", kopiBody("Kopi Senja"), lin)},
		{name: "list my suggestions", req: authRequest(t, http.MethodGet, "/suggestions/mine", "", lin)},
		{name: "list suggestions", req: adminRequest(t, http.MethodGet, "/suggestions", "", admin)},
		{name: "get suggestion", req: adminRequest(t, http.MethodGet, "/suggestions/1", "", admin)},
		{name: "approve suggestion", req: adminRequest(t, http.MethodPost, "/suggestions/1/approve", warungPlace, admin)},
		{name: "reject suggestion", req: adminRequest(t, http.MethodPost, "/suggestions/1/reject", `{}`, admin)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := serve(t, router, tc.req)
			assertError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
			if strings.Contains(rec.Body.String(), "database is closed") {
				t.Errorf("500 body exposes driver detail: %s", rec.Body)
			}
		})
	}
}
