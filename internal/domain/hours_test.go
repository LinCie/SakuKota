package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sakukota-go/internal/domain"
)

func TestNormalizeOperatingHours(t *testing.T) {
	period := func(day int, open, closing string) domain.OperatingPeriod {
		return domain.OperatingPeriod{Day: day, Open: open, Close: closing}
	}

	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name string
			in   domain.OperatingHours
			want domain.OperatingHours
		}{
			{name: "none", in: nil, want: nil},
			{name: "empty_is_nil", in: domain.OperatingHours{}, want: nil},
			{name: "single", in: domain.OperatingHours{period(1, "08:00", "20:00")}, want: domain.OperatingHours{period(1, "08:00", "20:00")}},
			{name: "whole_day", in: domain.OperatingHours{period(7, "00:00", "24:00")}, want: domain.OperatingHours{period(7, "00:00", "24:00")}},
			{
				name: "sorted_by_day_then_open",
				in:   domain.OperatingHours{period(2, "17:00", "21:00"), period(1, "08:00", "12:00"), period(2, "08:00", "12:00")},
				want: domain.OperatingHours{period(1, "08:00", "12:00"), period(2, "08:00", "12:00"), period(2, "17:00", "21:00")},
			},
			{
				name: "touching_periods",
				in:   domain.OperatingHours{period(3, "08:00", "12:00"), period(3, "12:00", "16:00")},
				want: domain.OperatingHours{period(3, "08:00", "12:00"), period(3, "12:00", "16:00")},
			},
			{
				name: "three_periods_a_day",
				in:   domain.OperatingHours{period(4, "06:00", "08:00"), period(4, "10:00", "12:00"), period(4, "14:00", "16:00")},
				want: domain.OperatingHours{period(4, "06:00", "08:00"), period(4, "10:00", "12:00"), period(4, "14:00", "16:00")},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := domain.NormalizeOperatingHours(tt.in)
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name string
			in   domain.OperatingHours
		}{
			{name: "day_zero", in: domain.OperatingHours{period(0, "08:00", "20:00")}},
			{name: "day_eight", in: domain.OperatingHours{period(8, "08:00", "20:00")}},
			{name: "open_not_a_time", in: domain.OperatingHours{period(1, "8:00", "20:00")}},
			{name: "open_letters", in: domain.OperatingHours{period(1, "ab:cd", "20:00")}},
			{name: "open_24", in: domain.OperatingHours{period(1, "24:00", "24:00")}},
			{name: "open_minute_60", in: domain.OperatingHours{period(1, "08:60", "20:00")}},
			{name: "close_after_24", in: domain.OperatingHours{period(1, "08:00", "24:01")}},
			{name: "close_empty", in: domain.OperatingHours{period(1, "08:00", "")}},
			{name: "open_equals_close", in: domain.OperatingHours{period(1, "08:00", "08:00")}},
			{name: "open_after_close", in: domain.OperatingHours{period(1, "22:00", "02:00")}},
			{name: "overlap", in: domain.OperatingHours{period(1, "08:00", "12:00"), period(1, "11:59", "16:00")}},
			{name: "contained", in: domain.OperatingHours{period(1, "08:00", "20:00"), period(1, "10:00", "12:00")}},
			{
				name: "four_periods_a_day",
				in:   domain.OperatingHours{period(1, "01:00", "02:00"), period(1, "03:00", "04:00"), period(1, "05:00", "06:00"), period(1, "07:00", "08:00")},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := domain.NormalizeOperatingHours(tt.in)
				assert.ErrorIs(t, err, domain.ErrPlaceOperatingHoursInvalid)
				assert.Nil(t, got)
			})
		}
	})

	t.Run("same_time_on_different_days", func(t *testing.T) {
		in := domain.OperatingHours{period(1, "08:00", "20:00"), period(2, "08:00", "20:00")}
		got, err := domain.NormalizeOperatingHours(in)
		require.NoError(t, err)
		assert.Equal(t, in, got)
	})

	t.Run("input_unchanged", func(t *testing.T) {
		in := domain.OperatingHours{period(2, "08:00", "12:00"), period(1, "08:00", "12:00")}
		_, err := domain.NormalizeOperatingHours(in)
		require.NoError(t, err)
		assert.Equal(t, domain.OperatingHours{period(2, "08:00", "12:00"), period(1, "08:00", "12:00")}, in)
	})
}

func TestOperatingHours_Value(t *testing.T) {
	t.Run("none_is_null", func(t *testing.T) {
		got, err := domain.OperatingHours(nil).Value()
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("round_trip", func(t *testing.T) {
		hours := domain.OperatingHours{{Day: 1, Open: "08:00", Close: "20:00"}}
		value, err := hours.Value()
		require.NoError(t, err)
		assert.JSONEq(t, `[{"day":1,"open":"08:00","close":"20:00"}]`, string(value.([]byte)))

		var got domain.OperatingHours
		require.NoError(t, got.Scan(value))
		assert.Equal(t, hours, got)
	})

	t.Run("scan_null", func(t *testing.T) {
		got := domain.OperatingHours{{Day: 1, Open: "08:00", Close: "20:00"}}
		require.NoError(t, got.Scan(nil))
		assert.Nil(t, got)
	})

	t.Run("scan_unsupported", func(t *testing.T) {
		var got domain.OperatingHours
		assert.Error(t, got.Scan(42))
	})
}
