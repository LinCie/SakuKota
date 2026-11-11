-- +goose Up
-- The place service validates attributes by data type, so a type it does not
-- know must not be storable.
ALTER TABLE CATEGORY_FIELDS ADD CONSTRAINT category_fields_data_type_check
    CHECK (data_type IN ('text', 'number', 'boolean', 'select', 'range', 'date'));

-- +goose Down
ALTER TABLE CATEGORY_FIELDS DROP CONSTRAINT category_fields_data_type_check;
