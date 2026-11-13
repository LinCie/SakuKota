-- +goose Up
-- Deleting a media closes the gap by shifting later medias down one position.
-- A deferrable key is checked at the end of that statement, so the shift does
-- not collide with a position that has not moved yet.
ALTER TABLE SUB_DISTRICT_MEDIAS
    DROP CONSTRAINT sub_district_medias_pkey,
    ADD CONSTRAINT sub_district_medias_pkey PRIMARY KEY (sub_district_id, position) DEFERRABLE INITIALLY IMMEDIATE;

-- +goose Down
ALTER TABLE SUB_DISTRICT_MEDIAS
    DROP CONSTRAINT sub_district_medias_pkey,
    ADD CONSTRAINT sub_district_medias_pkey PRIMARY KEY (sub_district_id, position);
