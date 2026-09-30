-- +goose Up
CREATE TABLE CATEGORY_FIELDS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category_id BIGINT NOT NULL,
    key VARCHAR(64) NOT NULL,
    label VARCHAR(100) NOT NULL,
    data_type VARCHAR(20) NOT NULL,
    required BOOLEAN NOT NULL DEFAULT FALSE,
    options JSONB,
    sort_order INTEGER NOT NULL,
    UNIQUE (category_id, key),
    FOREIGN KEY (category_id) REFERENCES CATEGORIES(id)
);

ALTER TABLE PLACES
    ADD COLUMN attributes JSONB NOT NULL DEFAULT '{}'
        CHECK (jsonb_typeof(attributes) = 'object');

-- +goose Down
ALTER TABLE PLACES DROP COLUMN attributes;
DROP TABLE CATEGORY_FIELDS;
