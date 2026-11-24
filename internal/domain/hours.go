package domain

import (
	"cmp"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"slices"
)

// Limits of a place's operating hours.
const (
	// MaxPeriodsPerDay is the most opening periods one day may have, such as a morning and an evening shift.
	MaxPeriodsPerDay = 3
	minutesPerDay    = 24 * 60
)

// OperatingPeriod is one stretch of a weekday during which a place is open.
type OperatingPeriod struct {
	// Day is the weekday, 1 for Monday through 7 for Sunday.
	Day int `json:"day"`
	// Open and Close are HH:MM times of the place's local day. Close may be
	// 24:00; a place open past midnight has a period on each day.
	Open  string `json:"open"`
	Close string `json:"close"`
}

// OperatingHours are the periods a place is open, stored as a JSONB array.
// A place that is closed all week or whose hours are unknown has none (nil).
type OperatingHours []OperatingPeriod

// Scan reads a JSON array from the database (database/sql's Scanner).
func (h *OperatingHours) Scan(src any) error {
	switch value := src.(type) {
	case nil:
		*h = nil
		return nil
	case []byte:
		return json.Unmarshal(value, h)
	case string:
		return json.Unmarshal([]byte(value), h)
	default:
		return fmt.Errorf("place: cannot scan %T into OperatingHours", src)
	}
}

// Value writes the hours as a JSON array, or NULL if there are none (database/sql's Valuer).
func (h OperatingHours) Value() (driver.Value, error) {
	if len(h) == 0 {
		return nil, nil
	}

	return json.Marshal(h)
}

// NormalizeOperatingHours checks the periods and returns them sorted by day and
// opening time, in HH:MM form. No periods give nil. It returns an error
// wrapping ErrPlaceOperatingHoursInvalid if a day or time is out of range, a
// period does not open before it closes, periods of one day overlap, or a day
// has more than MaxPeriodsPerDay periods. The input is not changed.
func NormalizeOperatingHours(hours OperatingHours) (OperatingHours, error) {
	if len(hours) == 0 {
		return nil, nil
	}

	type span struct {
		day, open, close int
	}
	spans := make([]span, len(hours))
	for i, p := range hours {
		if p.Day < 1 || p.Day > 7 {
			return nil, fmt.Errorf("%w: day must be 1 (Monday) to 7 (Sunday)", ErrPlaceOperatingHoursInvalid)
		}
		open, ok := parseClock(p.Open, false)
		if !ok {
			return nil, fmt.Errorf("%w: open %q must be a time between 00:00 and 23:59 as HH:MM", ErrPlaceOperatingHoursInvalid, p.Open)
		}
		closing, ok := parseClock(p.Close, true)
		if !ok {
			return nil, fmt.Errorf("%w: close %q must be a time between 00:00 and 24:00 as HH:MM", ErrPlaceOperatingHoursInvalid, p.Close)
		}
		if open >= closing {
			return nil, fmt.Errorf("%w: open %s must be before close %s; split a period past midnight over two days", ErrPlaceOperatingHoursInvalid, p.Open, p.Close)
		}
		spans[i] = span{day: p.Day, open: open, close: closing}
	}

	slices.SortFunc(spans, func(a, b span) int {
		return cmp.Or(cmp.Compare(a.day, b.day), cmp.Compare(a.open, b.open))
	})

	normalized := make(OperatingHours, len(spans))
	perDay := 0
	for i, s := range spans {
		if i > 0 && spans[i-1].day == s.day {
			perDay++
			if s.open < spans[i-1].close {
				return nil, fmt.Errorf("%w: periods on day %d overlap", ErrPlaceOperatingHoursInvalid, s.day)
			}
		} else {
			perDay = 1
		}
		if perDay > MaxPeriodsPerDay {
			return nil, fmt.Errorf("%w: day %d has more than %d periods", ErrPlaceOperatingHoursInvalid, s.day, MaxPeriodsPerDay)
		}
		normalized[i] = OperatingPeriod{Day: s.day, Open: formatClock(s.open), Close: formatClock(s.close)}
	}

	return normalized, nil
}

// parseClock returns the minutes since midnight of an HH:MM time. It accepts
// 24:00 only if allowEnd is set.
func parseClock(s string, allowEnd bool) (int, bool) {
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	for _, i := range []int{0, 1, 3, 4} {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}

	hour := int(s[0]-'0')*10 + int(s[1]-'0')
	minute := int(s[3]-'0')*10 + int(s[4]-'0')
	total := hour*60 + minute
	if minute > 59 || total > minutesPerDay || (total == minutesPerDay && !allowEnd) {
		return 0, false
	}

	return total, true
}

// formatClock is the HH:MM form of minutes since midnight.
func formatClock(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}
