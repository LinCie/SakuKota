package reports

import (
	"time"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/storage"
)

// MaxPendingPerUser is the number of reports a user may have waiting for an
// admin at once.
const MaxPendingPerUser = 5

// Filter narrows the reports returned by List. The zero value applies no filter.
type Filter struct {
	// Status keeps only the reports in this status. Empty keeps all of them.
	Status domain.ReportStatus
}

// CreateRequest is the API payload for reporting a place or a review. The
// author comes from the access token and the target from the URL.
type CreateRequest struct {
	Reason domain.ReportReason `json:"reason"`
	Note   *string             `json:"note"`
}

// Response is the API representation of a domain.Report. Who resolved a
// report is not exposed.
type Response struct {
	ID         int64               `json:"id"`
	PlaceID    int64               `json:"place_id"`
	ReviewID   *int64              `json:"review_id"`
	Reason     domain.ReportReason `json:"reason"`
	Note       *string             `json:"note"`
	Status     domain.ReportStatus `json:"status"`
	ResolvedAt *time.Time          `json:"resolved_at"`
	CreatedAt  time.Time           `json:"created_at"`
	User       AuthorResponse      `json:"user"`
}

// DetailResponse is a report with its content as it is now. Place is null
// once the place is deleted; Review is null for a place report or once the
// review is deleted.
type DetailResponse struct {
	Response
	Place  *PlaceResponse  `json:"place"`
	Review *ReviewResponse `json:"review"`
}

// PlaceResponse is the part of a reported domain.Place an admin judges it by.
type PlaceResponse struct {
	ID               int64                   `json:"id"`
	Name             string                  `json:"name"`
	Address          *string                 `json:"address"`
	ShortDescription *string                 `json:"short_description"`
	Category         CategoryResponse        `json:"category"`
	Medias           []storage.MediaResponse `json:"medias"`
}

// CategoryResponse is the API representation of a reported place's domain.Category.
type CategoryResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Code  string `json:"code"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// ReviewResponse is the API representation of a reported domain.Review.
type ReviewResponse struct {
	ID        int64                   `json:"id"`
	Rating    int                     `json:"rating"`
	Comment   *string                 `json:"comment"`
	CreatedAt time.Time               `json:"created_at"`
	User      AuthorResponse          `json:"user"`
	Medias    []storage.MediaResponse `json:"medias"`
}

// AuthorResponse is the API representation of a report's domain.Author.
type AuthorResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// toReport returns the report the user files against the place, or against
// its review when reviewID is not nil.
func toReport(userID, placeID int64, reviewID *int64, req CreateRequest) *domain.Report {
	return &domain.Report{
		UserID:   userID,
		PlaceID:  placeID,
		ReviewID: reviewID,
		Reason:   req.Reason,
		Note:     req.Note,
	}
}

// toResponse turns r into the JSON shape sent to clients.
func toResponse(r domain.Report) Response {
	return Response{
		ID:         r.ID,
		PlaceID:    r.PlaceID,
		ReviewID:   r.ReviewID,
		Reason:     r.Reason,
		Note:       r.Note,
		Status:     r.Status,
		ResolvedAt: r.ResolvedAt,
		CreatedAt:  r.CreatedAt,
		User: AuthorResponse{
			ID:   r.User.ID,
			Name: r.User.Name,
		},
	}
}

// toDetailResponse turns r and its content into the JSON shape sent to
// clients, with media keys turned into URLs by mediaURL.
func toDetailResponse(r domain.Report, mediaURL func(key string) string) DetailResponse {
	detail := DetailResponse{Response: toResponse(r)}

	if p := r.Place; p != nil {
		detail.Place = &PlaceResponse{
			ID:               p.ID,
			Name:             p.Name,
			Address:          p.Address,
			ShortDescription: p.ShortDescription,
			Category: CategoryResponse{
				ID:    p.Category.ID,
				Name:  p.Category.Name,
				Slug:  p.Category.Slug,
				Code:  p.Category.Code,
				Label: p.Category.Label,
				Color: p.Category.Color,
			},
			Medias: storage.ToMediaResponses(p.Medias, mediaURL),
		}
	}

	if rv := r.Review; rv != nil {
		detail.Review = &ReviewResponse{
			ID:        rv.ID,
			Rating:    rv.Rating,
			Comment:   rv.Comment,
			CreatedAt: rv.CreatedAt,
			User:      AuthorResponse{ID: rv.User.ID, Name: rv.User.Name},
			Medias:    storage.ToMediaResponses(rv.Medias, mediaURL),
		}
	}

	return detail
}
