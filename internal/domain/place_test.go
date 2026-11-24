package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sakukota-go/internal/domain"
)

func TestNormalizePlaceFields_PriceRange(t *testing.T) {
	tests := []struct {
		name     string
		min, max *int
		wantErr  bool
	}{
		{name: "none"},
		{name: "range", min: new(9000), max: new(12000)},
		{name: "single_price", min: new(3500), max: new(3500)},
		{name: "free", min: new(0), max: new(0)},
		{name: "at_cap", min: new(0), max: new(domain.MaxPlacePrice)},
		{name: "only_min", min: new(1), wantErr: true},
		{name: "only_max", max: new(1), wantErr: true},
		{name: "negative", min: new(-1), max: new(5), wantErr: true},
		{name: "min_above_max", min: new(6), max: new(5), wantErr: true},
		{name: "above_cap", min: new(0), max: new(domain.MaxPlacePrice + 1), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &domain.Place{PriceMin: tt.min, PriceMax: tt.max}
			err := domain.NormalizePlaceFields(p)
			if tt.wantErr {
				assert.ErrorIs(t, err, domain.ErrPlacePriceRangeInvalid)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.min, p.PriceMin)
			assert.Equal(t, tt.max, p.PriceMax)
		})
	}
}

func TestNormalizePlaceFields_OperatingHours(t *testing.T) {
	t.Run("sorted_and_cleaned", func(t *testing.T) {
		p := &domain.Place{OperatingHours: domain.OperatingHours{{Day: 2, Open: "08:00", Close: "09:00"}, {Day: 1, Open: "08:00", Close: "09:00"}}}
		require.NoError(t, domain.NormalizePlaceFields(p))
		assert.Equal(t, 1, p.OperatingHours[0].Day)
	})

	t.Run("invalid", func(t *testing.T) {
		p := &domain.Place{OperatingHours: domain.OperatingHours{{Day: 9, Open: "08:00", Close: "09:00"}}}
		assert.True(t, domain.IsPlaceValidationError(domain.NormalizePlaceFields(p)))
	})
}
