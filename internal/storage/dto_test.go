package storage_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/domain"
	"rantau-go/internal/storage"
)

func TestToMediaResponses(t *testing.T) {
	url := func(key string) string { return "https://cdn.test/" + key }

	t.Run("InOrder", func(t *testing.T) {
		medias := []domain.Media{
			{ID: 4, Key: "places/7/a.webp", ContentType: "image/webp", Size: 2048, Position: 1},
			{ID: 2, Key: "places/7/b.png", ContentType: "image/png", Size: 1024, Position: 2},
		}

		got := storage.ToMediaResponses(medias, url)

		want := []storage.MediaResponse{
			{ID: 4, URL: "https://cdn.test/places/7/a.webp", ContentType: "image/webp", Size: 2048, Position: 1},
			{ID: 2, URL: "https://cdn.test/places/7/b.png", ContentType: "image/png", Size: 1024, Position: 2},
		}
		assert.Equal(t, want, got)
	})

	t.Run("JSONFields", func(t *testing.T) {
		got := storage.ToMediaResponses([]domain.Media{
			{ID: 4, Key: "places/7/a.webp", ContentType: "image/webp", Size: 2048, Position: 1},
		}, url)

		body, err := json.Marshal(got)
		require.NoError(t, err)
		wantJSON := `[{"id":4,"url":"https://cdn.test/places/7/a.webp","content_type":"image/webp","size":2048,"position":1}]`
		assert.JSONEq(t, wantJSON, string(body))
	})

	t.Run("EmptyArray", func(t *testing.T) {
		for _, medias := range [][]domain.Media{nil, {}} {
			body, err := json.Marshal(storage.ToMediaResponses(medias, url))
			require.NoError(t, err)
			assert.JSONEq(t, `[]`, string(body), "medias = %#v", medias)
		}
	})
}
