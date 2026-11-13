package suggestions

import (
	"encoding/json"
	"time"

	"rantau-go/internal/pkg/postgis"
)

// CreateSuggestionRequest is the API payload for suggesting a place. The
// author comes from the access token.
type CreateSuggestionRequest struct {
	SubDistrictID       int64          `json:"sub_district_id"`
	CategoryID          int64          `json:"category_id"`
	Name                string         `json:"name"`
	Address             *string        `json:"address"`
	Location            *postgis.Point `json:"location"`
	OperatingHours      *string        `json:"operating_hours"`
	EstimatedPriceRange *string        `json:"estimated_price_range"`
	ShortDescription    *string        `json:"short_description"`
	// Attributes is a JSON object keyed by the category's field keys.
	Attributes json.RawMessage `json:"attributes"`
	Note       *string         `json:"note"`
}

// RejectSuggestionRequest is the API payload for rejecting a suggestion.
type RejectSuggestionRequest struct {
	Reason *string `json:"reason"`
}

// SuggestionResponse is the API representation of a Suggestion.
type SuggestionResponse struct {
	ID                  int64           `json:"id"`
	SubDistrictID       int64           `json:"sub_district_id"`
	CategoryID          int64           `json:"category_id"`
	Name                string          `json:"name"`
	Address             *string         `json:"address"`
	Location            *postgis.Point  `json:"location"`
	OperatingHours      *string         `json:"operating_hours"`
	EstimatedPriceRange *string         `json:"estimated_price_range"`
	ShortDescription    *string         `json:"short_description"`
	Attributes          json.RawMessage `json:"attributes"`
	Note                *string         `json:"note"`
	Status              Status          `json:"status"`
	RejectionReason     *string         `json:"rejection_reason"`
	PlaceID             *int64          `json:"place_id"`
	ReviewedAt          *time.Time      `json:"reviewed_at"`
	CreatedAt           time.Time       `json:"created_at"`
	User                AuthorResponse  `json:"user"`
}

// AuthorResponse is the API representation of a Suggestion's Author.
type AuthorResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// -----------------------------------------------------------------------------
// Conversions
// -----------------------------------------------------------------------------

// toSuggestion returns the suggestion the user submits. The author comes from
// the token, not the request.
func toSuggestion(userID int64, req CreateSuggestionRequest) *Suggestion {
	return &Suggestion{
		UserID:              userID,
		SubDistrictID:       req.SubDistrictID,
		CategoryID:          req.CategoryID,
		Name:                req.Name,
		Address:             req.Address,
		Location:            req.Location,
		OperatingHours:      req.OperatingHours,
		EstimatedPriceRange: req.EstimatedPriceRange,
		ShortDescription:    req.ShortDescription,
		Attributes:          req.Attributes,
		Note:                req.Note,
	}
}

// toSuggestionResponse turns s into the JSON shape sent to clients.
func toSuggestionResponse(s Suggestion) SuggestionResponse {
	return SuggestionResponse{
		ID:                  s.ID,
		SubDistrictID:       s.SubDistrictID,
		CategoryID:          s.CategoryID,
		Name:                s.Name,
		Address:             s.Address,
		Location:            s.Location,
		OperatingHours:      s.OperatingHours,
		EstimatedPriceRange: s.EstimatedPriceRange,
		ShortDescription:    s.ShortDescription,
		Attributes:          s.Attributes,
		Note:                s.Note,
		Status:              s.Status,
		RejectionReason:     s.RejectionReason,
		PlaceID:             s.PlaceID,
		ReviewedAt:          s.ReviewedAt,
		CreatedAt:           s.CreatedAt,
		User: AuthorResponse{
			ID:   s.User.ID,
			Name: s.User.Name,
		},
	}
}

// toSuggestionResponses turns each suggestion into its response shape.
func toSuggestionResponses(suggestions []Suggestion) []SuggestionResponse {
	responses := make([]SuggestionResponse, 0, len(suggestions))

	for _, s := range suggestions {
		responses = append(responses, toSuggestionResponse(s))
	}

	return responses
}
