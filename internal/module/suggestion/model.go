package suggestion

import (
	"encoding/json"
	"time"

	"rantau-go/internal/pkg/postgis"
)

// Status is where a suggestion stands in its review.
type Status string

// Suggestion represents a row in the PLACE_SUGGESTIONS table, joined with its author.
type Suggestion struct {
	ID            int64          `db:"id"`
	UserID        int64          `db:"user_id"`
	SubDistrictID int64          `db:"sub_district_id"`
	CategoryID    int64          `db:"category_id"`
	Name          string         `db:"name"`
	Address       *string        `db:"address"`
	Location      *postgis.Point `db:"location"`
	// OperatingHours, EstimatedPriceRange and ShortDescription describe the place as the user knows it.
	OperatingHours      *string `db:"operating_hours"`
	EstimatedPriceRange *string `db:"estimated_price_range"`
	ShortDescription    *string `db:"short_description"`
	// Attributes is a JSON object of the category's custom field values. It
	// is checked against the category when the suggestion is approved.
	Attributes json.RawMessage `db:"attributes"`
	// Note is the user's message to the reviewer.
	Note   *string `db:"note"`
	Status Status  `db:"status"`
	// RejectionReason is nil unless the suggestion was rejected with a reason.
	RejectionReason *string `db:"rejection_reason"`
	// PlaceID is the place the suggestion was approved as, or nil if it has none.
	PlaceID    *int64     `db:"place_id"`
	ReviewedBy *int64     `db:"reviewed_by"`
	ReviewedAt *time.Time `db:"reviewed_at"`
	CreatedAt  time.Time  `db:"created_at"`
	User       Author     `db:"user"`
}

// Author is the user who submitted a Suggestion, as shown next to it.
type Author struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

// SuggestionFilter narrows the suggestions returned by GetMany. The zero value applies no filter.
type SuggestionFilter struct {
	// Status keeps only the suggestions in this status. Empty keeps all of them.
	Status Status
}
