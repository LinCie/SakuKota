-- +goose Up
ALTER TABLE PLACES ADD COLUMN category_id BIGINT REFERENCES CATEGORIES(id);

UPDATE PLACES p
SET category_id = (
    SELECT MIN(pc.category_id)
    FROM PLACE_CATEGORIES pc
    WHERE pc.place_id = p.id
);

ALTER TABLE PLACES ALTER COLUMN category_id SET NOT NULL;

DROP TABLE PLACE_CATEGORIES;

-- +goose Down
CREATE TABLE PLACE_CATEGORIES (
    place_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    PRIMARY KEY (place_id, category_id),
    FOREIGN KEY (place_id) REFERENCES PLACES(id),
    FOREIGN KEY (category_id) REFERENCES CATEGORIES(id)
);

INSERT INTO PLACE_CATEGORIES (place_id, category_id)
SELECT id, category_id FROM PLACES;

ALTER TABLE PLACES DROP COLUMN category_id;
