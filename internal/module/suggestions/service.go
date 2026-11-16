package suggestions

import (
	"context"
	"encoding/json"

	"rantau-go/internal/module/places"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/pkg/text"
)

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	// Create checks the name, location, and attributes, then saves the suggestion as pending.
	// Blank optional text is saved as nothing.
	Create(ctx context.Context, s *Suggestion) (*Suggestion, error)
	// GetSuggestion returns the suggestion with the given ID.
	GetSuggestion(ctx context.Context, id int64) (*Suggestion, error)
	// GetByUser returns the user's suggestions, newest first.
	GetByUser(ctx context.Context, userID int64) ([]Suggestion, error)
	// GetMany returns the suggestions matching the filter, oldest first.
	GetMany(ctx context.Context, filter SuggestionFilter) ([]Suggestion, error)
	// Approve creates the place p from a pending suggestion and marks the
	// suggestion approved as that place. p must have a location.
	Approve(ctx context.Context, id, adminID int64, p *places.Place) (*Suggestion, error)
	// Reject marks a pending suggestion rejected. The reason is trimmed and
	// stored as nil when blank, and must be at most 1000 characters.
	Reject(ctx context.Context, id, adminID int64, reason *string) (*Suggestion, error)
}

// PlaceService defines the place operations that suggestions require: checking the
// place text when a suggestion is submitted, and creating and deleting places on approval.
type PlaceService interface {
	// Create saves a new place, which approval uses to make a place from a suggestion.
	Create(ctx context.Context, p *places.Place) (*places.Place, error)
	// Delete removes the place again, which is used to undo a failed approval.
	Delete(ctx context.Context, id int64) error
	// NormalizeName trims the spaces around a place name, and rejects it if it is empty or too long.
	NormalizeName(ctx context.Context, name string) (string, error)
	// NormalizeOptionalFields trims the optional text of p and turns blank values into nothing.
	// It rejects a value longer than its column allows.
	NormalizeOptionalFields(ctx context.Context, p *places.Place) error
}

type service struct {
	repo   Repository
	places PlaceService
}

// NewService builds the suggestion service from the repository and the place service.
func NewService(repo Repository, places PlaceService) Service {
	return &service{
		repo:   repo,
		places: places,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

// Create checks the suggestion's fields, then saves it as pending.
func (s *service) Create(ctx context.Context, sg *Suggestion) (*Suggestion, error) {
	name, err := s.places.NormalizeName(ctx, sg.Name)
	if err != nil {
		return nil, err
	}

	if !postgis.InIndonesia(sg.Location) {
		return nil, ErrInvalidLocation
	}

	attributes, err := normalizeAttributes(sg.Attributes)
	if err != nil {
		return nil, err
	}

	// The optional place text follows the place rules, so the place service checks and trims it.
	place := &places.Place{
		Address:             sg.Address,
		OperatingHours:      sg.OperatingHours,
		EstimatedPriceRange: sg.EstimatedPriceRange,
		ShortDescription:    sg.ShortDescription,
	}
	if err := s.places.NormalizeOptionalFields(ctx, place); err != nil {
		return nil, err
	}

	note, ok := text.TrimOptionalMax(sg.Note, maxNoteLength)
	if !ok {
		return nil, ErrNoteTooLong
	}

	// Only the submitted fields are stored, so the suggestion starts pending
	// whatever the client sent for its review.
	return s.repo.Create(ctx, &Suggestion{
		UserID:              sg.UserID,
		SubDistrictID:       sg.SubDistrictID,
		CategoryID:          sg.CategoryID,
		Name:                name,
		Address:             place.Address,
		Location:            sg.Location,
		OperatingHours:      place.OperatingHours,
		EstimatedPriceRange: place.EstimatedPriceRange,
		ShortDescription:    place.ShortDescription,
		Attributes:          attributes,
		Note:                note,
		Status:              StatusPending,
	})
}

// GetSuggestion returns the suggestion with the given id.
func (s *service) GetSuggestion(ctx context.Context, id int64) (*Suggestion, error) {
	return s.repo.GetSuggestion(ctx, id)
}

// GetByUser returns the user's suggestions, newest first.
func (s *service) GetByUser(ctx context.Context, userID int64) ([]Suggestion, error) {
	return s.repo.GetByUser(ctx, userID)
}

// GetMany checks the status filter, then returns the matching suggestions.
func (s *service) GetMany(ctx context.Context, filter SuggestionFilter) ([]Suggestion, error) {
	switch filter.Status {
	case "", StatusPending, StatusApproved, StatusRejected:
	default:
		return nil, ErrInvalidStatus
	}

	return s.repo.GetMany(ctx, filter)
}

// Approve creates the place, then marks the suggestion approved as that place.
// If marking fails, the new place is deleted again, so it is not left unlinked.
func (s *service) Approve(ctx context.Context, id, adminID int64, p *places.Place) (*Suggestion, error) {
	// Checked before the suggestion is read: a place without a location is
	// refused without touching storage.
	if p.Location == nil {
		return nil, ErrInvalidLocation
	}

	current, err := s.repo.GetSuggestion(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status != StatusPending {
		return nil, ErrAlreadyReviewed
	}

	created, err := s.places.Create(ctx, p)
	if err != nil {
		return nil, err
	}

	approved, err := s.repo.MarkApproved(ctx, id, created.ID, adminID)
	if err != nil {
		// A failure to delete the place is ignored: the place is left without
		// a suggestion, but the approval itself has failed either way. The
		// delete ignores cancellation so a dropped request still cleans up.
		_ = s.places.Delete(context.WithoutCancel(ctx), created.ID)
		return nil, err
	}

	return approved, nil
}

// Reject marks a pending suggestion as rejected, after checking the reason's length.
func (s *service) Reject(ctx context.Context, id, adminID int64, reason *string) (*Suggestion, error) {
	trimmed, ok := text.TrimOptionalMax(reason, maxReasonLength)
	if !ok {
		return nil, ErrReasonTooLong
	}

	return s.repo.Reject(ctx, id, adminID, trimmed)
}

// normalizeAttributes checks that raw is a JSON object and returns it as sent.
// A missing or null value becomes an empty object.
func normalizeAttributes(raw json.RawMessage) (json.RawMessage, error) {
	var object map[string]json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &object); err != nil {
			return nil, ErrInvalidAttributes
		}
	}

	if object == nil {
		return json.RawMessage(`{}`), nil
	}

	return raw, nil
}
