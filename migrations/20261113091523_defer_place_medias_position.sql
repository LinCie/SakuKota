-- +goose Up
-- Deleting a media closes the gap by shifting later medias down one position.
-- A deferrable key is checked at the end of that statement, so the shift does
-- not collide with a position that has not moved yet.
ALTER TABLE PLACE_MEDIAS
    DROP CONSTRAINT place_medias_pkey,
    ADD CONSTRAINT place_medias_pkey PRIMARY KEY (place_id, position) DEFERRABLE INITIALLY IMMEDIATE;

-- +goose Down
ALTER TABLE PLACE_MEDIAS
    DROP CONSTRAINT place_medias_pkey,
    ADD CONSTRAINT place_medias_pkey PRIMARY KEY (place_id, position);
