package pgtest

import (
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// InsertUser inserts a user named name, with the email name@example.com, and
// returns its id.
func InsertUser(t testing.TB, db *sqlx.DB, name string) int64 {
	t.Helper()

	var id int64
	err := db.Get(&id,
		`INSERT INTO USERS (name, email, password_hash) VALUES ($1, $2, 'hash') RETURNING id`,
		name, strings.ToLower(name)+"@example.com")
	require.NoError(t, err, "insert user %q", name)

	return id
}

// InsertSubDistrict inserts a sub-district with the name, also used as its
// slug, and returns its id.
func InsertSubDistrict(t testing.TB, db *sqlx.DB, name string) int64 {
	t.Helper()

	var id int64
	err := db.Get(&id, `INSERT INTO SUB_DISTRICTS (name, slug) VALUES ($1, $1) RETURNING id`, name)
	require.NoError(t, err, "insert sub-district %q", name)

	return id
}

// InsertPlace inserts a place with only its required columns and returns its id.
func InsertPlace(t testing.TB, db *sqlx.DB, subDistrictID, categoryID int64, name string) int64 {
	t.Helper()

	var id int64
	err := db.Get(&id,
		`INSERT INTO PLACES (sub_district_id, category_id, name) VALUES ($1, $2, $3) RETURNING id`,
		subDistrictID, categoryID, name)
	require.NoError(t, err, "insert place %q", name)

	return id
}

// CategoryID returns the id of a category the migrations define.
func CategoryID(t testing.TB, db *sqlx.DB, slug string) int64 {
	t.Helper()

	var id int64
	err := db.Get(&id, `SELECT id FROM CATEGORIES WHERE slug = $1`, slug)
	require.NoError(t, err, "select category %q", slug)

	return id
}

// QueuedKeys returns the storage keys waiting for deletion, in key order.
func QueuedKeys(t testing.TB, db *sqlx.DB) []string {
	t.Helper()

	keys := []string{}
	err := db.Select(&keys, `SELECT key FROM STORAGE_DELETIONS ORDER BY key`)
	require.NoError(t, err, "select storage deletions")

	return keys
}
