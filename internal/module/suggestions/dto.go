package suggestions

import (
	"encoding/json"
	"time"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/pkg/postgis"
)

// CreateRequest is the API payload for suggesting a place. The author comes
// from the access token.
type CreateRequest struct {
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

// ApproveRequest is the API payload for approving a suggestion: the place to
// create, in the same shape as the body of POST /places.
type ApproveRequest struct {
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
}

// RejectRequest is the API payload for rejecting a suggestion.
type RejectRequest struct {
	Reason *string `json:"reason"`
}

// Response is the API representation of a domain.Suggestion.
type Response struct {
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
	Status              domain.Status   `json:"status"`
	RejectionReason     *string         `json:"rejection_reason"`
	PlaceID             *int64          `json:"place_id"`
	ReviewedAt          *time.Time      `json:"reviewed_at"`
	CreatedAt           time.Time       `json:"created_at"`
	User                AuthorResponse  `json:"user"`
}

// AuthorResponse is the API representation of a suggestion's domain.Author.
type AuthorResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// toSuggestion returns the suggestion the user submits. The author comes from
// the token, not the request.
func toSuggestion(userID int64, req CreateRequest) *domain.Suggestion {
	return &domain.Suggestion{
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

// toPlace returns the place the admin approves the suggestion as.
func toPlace(req ApproveRequest) *domain.Place {
	return &domain.Place{
		SubDistrictID:       req.SubDistrictID,
		CategoryID:          req.CategoryID,
		Name:                req.Name,
		Address:             req.Address,
		Location:            req.Location,
		OperatingHours:      req.OperatingHours,
		EstimatedPriceRange: req.EstimatedPriceRange,
		ShortDescription:    req.ShortDescription,
		Attributes:          req.Attributes,
	}
}

// toResponse turns s into the JSON shape sent to clients.
func toResponse(s domain.Suggestion) Response {
	return Response{
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
