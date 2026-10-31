-- +goose Up
CREATE UNIQUE INDEX categories_name_lower_key ON CATEGORIES (LOWER(name));

-- +goose Down
DROP INDEX categories_name_lower_key;
