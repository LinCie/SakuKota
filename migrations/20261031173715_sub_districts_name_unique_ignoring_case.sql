-- +goose Up
CREATE UNIQUE INDEX sub_districts_name_lower_key ON SUB_DISTRICTS (LOWER(name));

-- +goose Down
DROP INDEX sub_districts_name_lower_key;
