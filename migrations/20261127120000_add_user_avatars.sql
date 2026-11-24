-- +goose Up
-- A user's profile photo. Replacing it deletes the old MEDIAS row and queues
-- its key in STORAGE_DELETIONS.
ALTER TABLE USERS ADD COLUMN avatar_media_id BIGINT UNIQUE REFERENCES MEDIAS(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE USERS DROP COLUMN avatar_media_id;
