-- +goose Up
-- Postgres does not index the referencing side of a foreign key. These serve
-- the GET /places filters, the pending-suggestion count made on every submit,
-- and the foreign key checks and cascades that scan these tables when a
-- sub-district, user or place is deleted.
CREATE INDEX places_sub_district_id_idx ON PLACES (sub_district_id);
CREATE INDEX places_category_id_idx ON PLACES (category_id);
CREATE INDEX place_suggestions_user_id_status_idx ON PLACE_SUGGESTIONS (user_id, status);
CREATE INDEX place_suggestions_place_id_idx ON PLACE_SUGGESTIONS (place_id);

-- +goose Down
DROP INDEX place_suggestions_place_id_idx;
DROP INDEX place_suggestions_user_id_status_idx;
DROP INDEX places_category_id_idx;
DROP INDEX places_sub_district_id_idx;
