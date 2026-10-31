//go:build integration

package review_test

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
	"rantau-go/internal/module/review"
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
	mustExec(t, `TRUNCATE PLACE_REVIEWS, PLACES, CATEGORIES, SUB_DISTRICTS, USERS RESTART IDENTITY CASCADE`)
}

func mustExec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := testDB.Exec(query, args...); err != nil {
		t.Fatalf("exec failed: %v\nquery: %s", err, query)
	}
}

// seedCatalog creates places 1 (Warung Bu Ani) and 2 (Kopi Senja), and users
// 1 (Lin), 2 (Sari), and 3 (Budi).
func seedCatalog(t *testing.T) {
	t.Helper()
	mustExec(t, `INSERT INTO SUB_DISTRICTS (name, slug) VALUES ('Menteng', 'menteng')`)
	mustExec(t, `INSERT INTO CATEGORIES (name, slug, code, label, color) VALUES ('Warung', 'warung', 'W', 'Warung', 'chili')`)
	mustExec(t, `INSERT INTO PLACES (sub_district_id, category_id, name) VALUES (1, 1, 'Warung Bu Ani'), (1, 1, 'Kopi Senja')`)
	mustExec(t, `INSERT INTO USERS (name, email, password_hash) VALUES
		('Lin', 'lin@example.com', 'x'), ('Sari', 'sari@example.com', 'x'), ('Budi', 'budi@example.com', 'x')`)
}

// integrationRouter wires the real service and repository behind the HTTP handlers.
func integrationRouter(db *sqlx.DB) http.Handler {
	return newRouter(review.NewService(review.NewPostgresRepository(db)))
}

// postReview sends the user's review to target.
func postReview(t *testing.T, router http.Handler, target string, userID int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serve(t, router, authRequest(t, http.MethodPost, target, body, userID))
}

// mustPostReview sends the user's review to target and fails unless it is created.
func mustPostReview(t *testing.T, router http.Handler, target string, userID int64, body string) {
	t.Helper()
	if rec := postReview(t, router, target, userID, body); rec.Code != http.StatusCreated {
		t.Fatalf("POST %s as user %d: status %d; body: %s", target, userID, rec.Code, rec.Body)
	}
}

type authorRow struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type reviewRow struct {
	ID        int64     `json:"id"`
	PlaceID   int64     `json:"place_id"`
	Rating    int       `json:"rating"`
	Comment   *string   `json:"comment"`
	CreatedAt string    `json:"created_at"`
	User      authorRow `json:"user"`
}

// listReviews calls GET target and returns the reviews in the order the API lists them.
func listReviews(t *testing.T, router http.Handler, target string) []reviewRow {
	t.Helper()
	rec := serve(t, router, jsonRequest(http.MethodGet, target, ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d; body: %s", target, rec.Code, rec.Body)
	}

	var rows []reviewRow
	if err := json.Unmarshal(responseData(t, rec), &rows); err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	return rows
}

// reviewIDs returns the ids of rows in order.
func reviewIDs(rows []reviewRow) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

// ratingTotals are the rating totals stored on a place.
type ratingTotals struct {
	Count int `db:"rating_count"`
	Sum   int `db:"rating_sum"`
}

func placeTotals(t *testing.T, placeID int64) ratingTotals {
	t.Helper()
	var totals ratingTotals
	err := testDB.Get(&totals, `SELECT rating_count, rating_sum FROM PLACES WHERE id = $1`, placeID)
	if err != nil {
		t.Fatalf("read totals of place %d: %v", placeID, err)
	}
	return totals
}

// reviewedTotals are the totals counted from the place's stored reviews.
func reviewedTotals(t *testing.T, placeID int64) ratingTotals {
	t.Helper()
	var totals ratingTotals
	err := testDB.Get(&totals, `SELECT COUNT(*) AS rating_count, COALESCE(SUM(rating), 0) AS rating_sum
		FROM PLACE_REVIEWS WHERE place_id = $1`, placeID)
	if err != nil {
		t.Fatalf("count reviews of place %d: %v", placeID, err)
	}
	return totals
}

// -----------------------------------------------------------------------------
// Creating reviews
// -----------------------------------------------------------------------------

func TestIntegration_CreatedReviewIsListedAsSent(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	rec := postReview(t, router, "/places/1/reviews", 1, `{"rating":4,"comment":"  Murah, porsi besar  "}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body)
	}

	var created reviewRow
	if err := json.Unmarshal(responseData(t, rec), &created); err != nil {
		t.Fatalf("created review: %v", err)
	}

	want := reviewRow{
		ID:        1,
		PlaceID:   1,
		Rating:    4,
		Comment:   ptr("Murah, porsi besar"),
		CreatedAt: created.CreatedAt,
		User:      authorRow{ID: 1, Name: "Lin"},
	}
	if !reflect.DeepEqual(created, want) {
		t.Errorf("created = %+v, want %+v", created, want)
	}
	if at, err := time.Parse(time.RFC3339Nano, created.CreatedAt); err != nil || at.IsZero() {
		t.Errorf("created_at = %q, want a timestamp", created.CreatedAt)
	}

	listed := listReviews(t, router, "/places/1/reviews")
	if len(listed) != 1 || !reflect.DeepEqual(listed[0], created) {
		t.Errorf("listed = %+v, but create responded with %+v", listed, created)
	}
}

func TestIntegration_ReviewsAddUpOnThePlace(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	mustPostReview(t, router, "/places/1/reviews", 1, `{"rating":5}`)
	mustPostReview(t, router, "/places/1/reviews", 2, `{"rating":4,"comment":"Enak"}`)
	mustPostReview(t, router, "/places/1/reviews", 3, `{"rating":2}`)

	if got, want := placeTotals(t, 1), (ratingTotals{Count: 3, Sum: 11}); got != want {
		t.Errorf("place 1 totals = %+v, want %+v", got, want)
	}
	if got, want := placeTotals(t, 2), (ratingTotals{Count: 0, Sum: 0}); got != want {
		t.Errorf("unreviewed place 2 totals = %+v, want %+v", got, want)
	}
}

func TestIntegration_CreateReviewRules(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		body       string
		wantStatus int
		wantCode   string       // empty when the review is created
		wantTotals ratingTotals // of place 1 afterwards
	}{
		{name: "rating only is stored", target: "/places/1/reviews", body: `{"rating":5}`,
			wantStatus: http.StatusCreated, wantCode: "", wantTotals: ratingTotals{Count: 1, Sum: 5}},
		{name: "rating and comment are stored", target: "/places/1/reviews", body: `{"rating":3,"comment":"Antre lama"}`,
			wantStatus: http.StatusCreated, wantCode: "", wantTotals: ratingTotals{Count: 1, Sum: 3}},
		{name: "blank comment is stored as a rating only", target: "/places/1/reviews", body: `{"rating":2,"comment":"   "}`,
			wantStatus: http.StatusCreated, wantCode: "", wantTotals: ratingTotals{Count: 1, Sum: 2}},
		{name: "rating below one", target: "/places/1/reviews", body: `{"rating":0}`,
			wantStatus: http.StatusBadRequest, wantCode: "validation_error", wantTotals: ratingTotals{}},
		{name: "rating above five", target: "/places/1/reviews", body: `{"rating":6}`,
			wantStatus: http.StatusBadRequest, wantCode: "validation_error", wantTotals: ratingTotals{}},
		{name: "missing rating", target: "/places/1/reviews", body: `{"comment":"Enak"}`,
			wantStatus: http.StatusBadRequest, wantCode: "validation_error", wantTotals: ratingTotals{}},
		{name: "comment too long", target: "/places/1/reviews", body: `{"rating":4,"comment":"` + strings.Repeat("a", 1001) + `"}`,
			wantStatus: http.StatusBadRequest, wantCode: "validation_error", wantTotals: ratingTotals{}},
		{name: "malformed body", target: "/places/1/reviews", body: `{"rating":`,
			wantStatus: http.StatusBadRequest, wantCode: "invalid_body", wantTotals: ratingTotals{}},
		{name: "unknown place", target: "/places/99/reviews", body: `{"rating":4}`,
			wantStatus: http.StatusNotFound, wantCode: "place_not_found", wantTotals: ratingTotals{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetDB(t)
			seedCatalog(t)

			rec := postReview(t, integrationRouter(testDB), tc.target, 1, tc.body)
			if tc.wantCode != "" {
				assertError(t, rec, tc.wantStatus, tc.wantCode)
			} else if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}

			if got := placeTotals(t, 1); got != tc.wantTotals {
				t.Errorf("place 1 totals = %+v, want %+v", got, tc.wantTotals)
			}
			if got := reviewedTotals(t, 1); got != tc.wantTotals {
				t.Errorf("stored reviews of place 1 add up to %+v, want %+v", got, tc.wantTotals)
			}
		})
	}
}

func TestIntegration_UserReviewsAPlaceOnce(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	mustPostReview(t, router, "/places/1/reviews", 1, `{"rating":5,"comment":"Murah"}`)

	rec := postReview(t, router, "/places/1/reviews", 1, `{"rating":1,"comment":"Berubah pikiran"}`)
	assertError(t, rec, http.StatusConflict, "already_reviewed")

	if got, want := placeTotals(t, 1), (ratingTotals{Count: 1, Sum: 5}); got != want {
		t.Errorf("place 1 totals = %+v, want %+v", got, want)
	}
	listed := listReviews(t, router, "/places/1/reviews")
	if len(listed) != 1 || listed[0].Rating != 5 {
		t.Errorf("listed = %+v, want only the first review", listed)
	}

	// The same user may still review another place.
	mustPostReview(t, router, "/places/2/reviews", 1, `{"rating":3}`)
}

func TestIntegration_ConcurrentReviewsAreAllCounted(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	mustExec(t, `INSERT INTO USERS (name, email, password_hash)
		SELECT 'User ' || n, 'user' || n || '@example.com', 'x' FROM generate_series(4, 20) AS n`)
	router := integrationRouter(testDB)

	// Users 1 to 20 each rate place 1 at once, with ratings 1 to 5 four times over.
	const n = 20
	reqs := make([]*http.Request, n)
	want := ratingTotals{Count: n}
	for i := range n {
		rating := i%5 + 1
		want.Sum += rating
		reqs[i] = authRequest(t, http.MethodPost, "/places/1/reviews", fmt.Sprintf(`{"rating":%d}`, rating), int64(i+1))
	}

	recs := make([]*httptest.ResponseRecorder, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, reqs[i])
			recs[i] = rec
		}()
	}
	wg.Wait()

	for i, rec := range recs {
		if rec.Code != http.StatusCreated {
			t.Errorf("user %d: status %d, want 201; body: %s", i+1, rec.Code, rec.Body)
		}
	}
	if got := placeTotals(t, 1); got != want {
		t.Errorf("place 1 totals = %+v, want %+v", got, want)
	}
	if got := reviewedTotals(t, 1); got != want {
		t.Errorf("stored reviews of place 1 add up to %+v, want %+v", got, want)
	}
}

func TestIntegration_ConcurrentDuplicateReviewsStoreOne(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	// User 1 sends the same review several times at once, like a double submit.
	const n = 5
	reqs := make([]*http.Request, n)
	for i := range n {
		reqs[i] = authRequest(t, http.MethodPost, "/places/1/reviews", `{"rating":4}`, 1)
	}

	recs := make([]*httptest.ResponseRecorder, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, reqs[i])
			recs[i] = rec
		}()
	}
	wg.Wait()

	created := 0
	for _, rec := range recs {
		switch rec.Code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			assertError(t, rec, http.StatusConflict, "already_reviewed")
		default:
			t.Errorf("status %d, want 201 or 409; body: %s", rec.Code, rec.Body)
		}
	}
	if created != 1 {
		t.Errorf("%d requests created a review, want exactly 1", created)
	}

	want := ratingTotals{Count: 1, Sum: 4}
	if got := placeTotals(t, 1); got != want {
		t.Errorf("place 1 totals = %+v, want %+v", got, want)
	}
	if got := reviewedTotals(t, 1); got != want {
		t.Errorf("stored reviews of place 1 add up to %+v, want %+v", got, want)
	}
}

// -----------------------------------------------------------------------------
// Listing reviews
// -----------------------------------------------------------------------------

func TestIntegration_ListReviews(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	mustExec(t, `INSERT INTO PLACES (sub_district_id, category_id, name) VALUES (1, 1, 'Toko Buku')`)
	// Reviews 2 and 3 of place 1 were created at the same moment.
	mustExec(t, `INSERT INTO PLACE_REVIEWS (place_id, user_id, rating, comment, created_at) VALUES
		(1, 1, 4, 'Murah', '2026-10-01T08:00:00Z'),
		(1, 2, 5, NULL, '2026-10-03T08:00:00Z'),
		(1, 3, 2, 'Antre lama', '2026-10-03T08:00:00Z'),
		(2, 1, 3, NULL, '2026-10-02T08:00:00Z')`)
	router := integrationRouter(testDB)

	tests := []struct {
		name   string
		target string
		want   []int64
	}{
		{name: "newest first, later id first on a tie", target: "/places/1/reviews", want: []int64{3, 2, 1}},
		{name: "only the place's own reviews", target: "/places/2/reviews", want: []int64{4}},
		{name: "place without reviews is an empty list", target: "/places/3/reviews", want: []int64{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := reviewIDs(listReviews(t, router, tc.target)); !slices.Equal(got, tc.want) {
				t.Errorf("review IDs = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIntegration_ListedReviewsCarryTheirAuthors(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	mustExec(t, `INSERT INTO PLACE_REVIEWS (place_id, user_id, rating, comment, created_at) VALUES
		(1, 1, 4, 'Murah', '2026-10-01T08:00:00Z'),
		(1, 2, 5, NULL, '2026-10-02T08:00:00Z')`)
	router := integrationRouter(testDB)

	rec := serve(t, router, jsonRequest(http.MethodGet, "/places/1/reviews", ""))
	if strings.Contains(rec.Body.String(), "@example.com") {
		t.Errorf("listing exposes author emails: %s", rec.Body)
	}

	listed := listReviews(t, router, "/places/1/reviews")
	want := []reviewRow{
		{ID: 2, PlaceID: 1, Rating: 5, Comment: nil, User: authorRow{ID: 2, Name: "Sari"}},
		{ID: 1, PlaceID: 1, Rating: 4, Comment: ptr("Murah"), User: authorRow{ID: 1, Name: "Lin"}},
	}
	wantCreatedAt := []time.Time{
		time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC),
	}
	if len(listed) != len(want) {
		t.Fatalf("listed %d reviews, want %d: %+v", len(listed), len(want), listed)
	}
	for i := range listed {
		at, err := time.Parse(time.RFC3339Nano, listed[i].CreatedAt)
		if err != nil || !at.Equal(wantCreatedAt[i]) {
			t.Errorf("review %d created_at = %q, want %s", listed[i].ID, listed[i].CreatedAt, wantCreatedAt[i])
		}
		listed[i].CreatedAt = ""
		if !reflect.DeepEqual(listed[i], want[i]) {
			t.Errorf("listed[%d] = %+v, want %+v", i, listed[i], want[i])
		}
	}
}

func TestIntegration_ListReviewsOfUnknownPlace(t *testing.T) {
	resetDB(t)
	seedCatalog(t)

	rec := serve(t, integrationRouter(testDB), jsonRequest(http.MethodGet, "/places/99/reviews", ""))
	assertError(t, rec, http.StatusNotFound, "place_not_found")
}

// -----------------------------------------------------------------------------
// Deleting places and failures
// -----------------------------------------------------------------------------

func TestIntegration_DeletingPlaceRemovesItsReviews(t *testing.T) {
	resetDB(t)
	seedCatalog(t)
	router := integrationRouter(testDB)

	mustPostReview(t, router, "/places/1/reviews", 1, `{"rating":5}`)
	mustPostReview(t, router, "/places/2/reviews", 2, `{"rating":3}`)

	mustExec(t, `DELETE FROM PLACES WHERE id = 1`)

	rec := serve(t, router, jsonRequest(http.MethodGet, "/places/1/reviews", ""))
	assertError(t, rec, http.StatusNotFound, "place_not_found")
	if got := reviewedTotals(t, 1); got != (ratingTotals{}) {
		t.Errorf("deleted place still has reviews adding up to %+v", got)
	}
	if got := reviewIDs(listReviews(t, router, "/places/2/reviews")); !slices.Equal(got, []int64{2}) {
		t.Errorf("place 2 review IDs = %v, want [2]", got)
	}
}

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
		body   string
	}{
		{name: "list reviews", method: http.MethodGet, body: ""},
		{name: "create review", method: http.MethodPost, body: `{"rating":4}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := serve(t, router, authRequest(t, tc.method, "/places/1/reviews", tc.body, 1))
			assertError(t, rec, http.StatusInternalServerError, "internal_error")
			if strings.Contains(rec.Body.String(), "database is closed") {
				t.Errorf("500 body exposes driver detail: %s", rec.Body)
			}
		})
	}
}
