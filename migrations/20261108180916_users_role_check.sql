-- +goose Up
-- The role is read from this column into the access token and compared by the
-- admin guard, so a misspelled role must not be storable.
ALTER TABLE USERS ADD CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'));

-- +goose Down
ALTER TABLE USERS DROP CONSTRAINT users_role_check;
