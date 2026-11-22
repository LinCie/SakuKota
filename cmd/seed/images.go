package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand/v2"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"rantau-go/internal/storage"
)

// imageSeeder uploads the seed images and records them in the seed transaction.
type imageSeeder struct {
	tx    *sqlx.Tx
	store *storage.S3
}

// seedImages gives every place of the seeded sub-districts, and each of those
// sub-districts, 2-3 images of one random color. Reruns skip whatever already
// has its images. An object uploaded before a failed run stays in the bucket
// without a row.
func (s imageSeeder) seedImages(ctx context.Context) error {
	var placeIDs []int64
	err := s.tx.SelectContext(ctx, &placeIDs, `
		SELECT p.id
		FROM PLACES p
		JOIN SUB_DISTRICTS sd ON sd.id = p.sub_district_id
		WHERE sd.slug IN ('setiabudi', 'mampang-prapatan', 'tebet')
		  AND NOT EXISTS (SELECT 1 FROM PLACE_MEDIAS pm WHERE pm.place_id = p.id)
		ORDER BY p.id`)
	if err != nil {
		return fmt.Errorf("list places without medias: %w", err)
	}
	for _, id := range placeIDs {
		err := s.addColorMedias(ctx, fmt.Sprintf("places/%d", id),
			`INSERT INTO PLACE_MEDIAS (place_id, media_id, position) VALUES ($1, $2, $3)`, id)
		if err != nil {
			return fmt.Errorf("seed medias of place %d: %w", id, err)
		}
	}

	var subDistricts []struct {
		ID   int64  `db:"id"`
		Slug string `db:"slug"`
	}
	err = s.tx.SelectContext(ctx, &subDistricts, `
		SELECT sd.id, sd.slug
		FROM SUB_DISTRICTS sd
		WHERE sd.slug IN ('setiabudi', 'mampang-prapatan', 'tebet')
		  AND NOT EXISTS (SELECT 1 FROM SUB_DISTRICT_MEDIAS sm WHERE sm.sub_district_id = sd.id)
		ORDER BY sd.id`)
	if err != nil {
		return fmt.Errorf("list sub-districts without medias: %w", err)
	}
	for _, sd := range subDistricts {
		err := s.addColorMedias(ctx, fmt.Sprintf("subdistricts/%d/medias", sd.ID),
			`INSERT INTO SUB_DISTRICT_MEDIAS (sub_district_id, media_id, position) VALUES ($1, $2, $3)`, sd.ID)
		if err != nil {
			return fmt.Errorf("seed medias of sub-district %s: %w", sd.Slug, err)
		}
	}

	return nil
}

// addColorMedias stores 2 or 3 single-color images under prefix and links
// them to the owner with linkSQL, which takes the owner id, media id, and position.
func (s imageSeeder) addColorMedias(ctx context.Context, prefix, linkSQL string, ownerID int64) error {
	count := 2 + rand.IntN(2)
	for position := 1; position <= count; position++ {
		body, err := colorPNG()
		if err != nil {
			return err
		}
		mediaID, err := s.addMedia(ctx, prefix, body)
		if err != nil {
			return err
		}
		if _, err := s.tx.ExecContext(ctx, linkSQL, ownerID, mediaID, position); err != nil {
			return err
		}
	}

	return nil
}

// addMedia uploads the PNG body to {prefix}/{uuid}.png, the key the API gives
// an upload, and returns the id of its new MEDIAS row.
func (s imageSeeder) addMedia(ctx context.Context, prefix string, body []byte) (int64, error) {
	key := prefix + "/" + uuid.NewString() + ".png"
	if err := s.store.Put(ctx, key, "image/png", body); err != nil {
		return 0, err
	}

	var id int64
	err := s.tx.GetContext(ctx, &id,
		`INSERT INTO MEDIAS (key, content_type, size_bytes) VALUES ($1, 'image/png', $2) RETURNING id`,
		key, len(body))
	return id, err
}

// colorPNG returns a 640x480 PNG filled with one random color. With a single
// palette entry it is under 200 bytes.
func colorPNG() ([]byte, error) {
	fill := color.RGBA{R: uint8(rand.IntN(256)), G: uint8(rand.IntN(256)), B: uint8(rand.IntN(256)), A: 255}
	img := image.NewPaletted(image.Rect(0, 0, 640, 480), color.Palette{fill})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
