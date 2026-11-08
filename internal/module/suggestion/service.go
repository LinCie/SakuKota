package suggestion

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"rantau-go/internal/module/place"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/pkg/text"
)

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	// Create stores the suggestion as pending. The name is trimmed and must be
	// 1 to 150 characters, and the location must lie within Indonesia. Blank
	// optional text is stored as nil. Attributes must be a JSON object.
	Create(ctx context.Context, s *Suggestion) (*Suggestion, error)
	// GetSuggestion returns the suggestion with the given ID.
	GetSuggestion(ctx context.Context, id int64) (*Suggestion, error)
	// GetByUser returns the user's suggestions, newest first.
	GetByUser(ctx context.Context, userID int64) ([]Suggestion, error)
	// GetMany returns the suggestions matching the filter, oldest first.
	GetMany(ctx context.Context, filter SuggestionFilter) ([]Suggestion, error)
	// Approve creates the place p from a pending suggestion and marks the
	// suggestion approved as that place. p must have a location.
	Approve(ctx context.Context, id, adminID int64, p *place.Place) (*Suggestion, error)
	// Reject marks a pending suggestion rejected. The reason is trimmed and
	// stored as nil when blank, and must be at most 1000 characters.
	Reject(ctx context.Context, id, adminID int64, reason *string) (*Suggestion, error)
}

// PlaceService defines the place operations that approving a suggestion requires.
type PlaceService interface {
	Create(ctx context.Context, p *place.Place) (*place.Place, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo   Repository
	places PlaceService
}

func NewService(repo Repository, places PlaceService) Service {
	return &service{
		repo:   repo,
		places: places,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

func (s *service) Create(ctx context.Context, sg *Suggestion) (*Suggestion, error) {
	name, err := normalizeName(sg.Name)
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

	address, err := trimOptionalWithLimit(sg.Address, place.MaxAddressLength, ErrAddressTooLong)
	if err != nil {
		return nil, err
	}

	operatingHours, err := trimOptionalWithLimit(sg.OperatingHours, place.MaxOperatingHoursLength, ErrOperatingHoursTooLong)
	if err != nil {
		return nil, err
	}

	estimatedPriceRange, err := trimOptionalWithLimit(sg.EstimatedPriceRange, place.MaxEstimatedPriceRangeLength, ErrEstimatedPriceRangeTooLong)
	if err != nil {
		return nil, err
	}

	note, err := trimOptionalWithLimit(sg.Note, maxNoteLength, ErrNoteTooLong)
	if err != nil {
		return nil, err
	}

	// Only the submitted fields are stored, so the suggestion starts pending
	// whatever the client sent for its review.
	return s.repo.Create(ctx, &Suggestion{
		UserID:              sg.UserID,
		SubDistrictID:       sg.SubDistrictID,
		CategoryID:          sg.CategoryID,
		Name:                name,
		Address:             address,
		Location:            sg.Location,
		OperatingHours:      operatingHours,
		EstimatedPriceRange: estimatedPriceRange,
		ShortDescription:    text.TrimOptional(sg.ShortDescription),
		Attributes:          attributes,
		Note:                note,
		Status:              StatusPending,
	})
}

func (s *service) GetSuggestion(ctx context.Context, id int64) (*Suggestion, error) {
	return s.repo.GetSuggestion(ctx, id)
}

func (s *service) GetByUser(ctx context.Context, userID int64) ([]Suggestion, error) {
	return s.repo.GetByUser(ctx, userID)
}

func (s *service) GetMany(ctx context.Context, filter SuggestionFilter) ([]Suggestion, error) {
	switch filter.Status {
	case "", StatusPending, StatusApproved, StatusRejected:
	default:
		return nil, ErrInvalidStatus
	}

	return s.repo.GetMany(ctx, filter)
}

// Approve creates the place first and then marks the suggestion approved as
// it. If the suggestion cannot be marked, the place is deleted again so that no
// place is left without an approved suggestion.
func (s *service) Approve(ctx context.Context, id, adminID int64, p *place.Place) (*Suggestion, error) {
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
		// a suggestion, but the approval itself has failed either way.
		_ = s.places.Delete(ctx, created.ID)
		return nil, err
	}

	return approved, nil
}

func (s *service) Reject(ctx context.Context, id, adminID int64, reason *string) (*Suggestion, error) {
	trimmed, err := trimOptionalWithLimit(reason, maxReasonLength, ErrReasonTooLong)
	if err != nil {
		return nil, err
	}

	return s.repo.Reject(ctx, id, adminID, trimmed)
}

// normalizeName trims name and checks it against the length of PLACE_SUGGESTIONS.name.
func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", ErrInvalidName
	case utf8.RuneCountInString(name) > place.MaxNameLength:
		return "", ErrNameTooLong
	}

	return name, nil
}

// trimOptionalWithLimit trims value and stores blank text as nil. It reports
// tooLong if what is left is longer than limit characters.
func trimOptionalWithLimit(value *string, limit int, tooLong error) (*string, error) {
	trimmed, ok := text.TrimOptionalMax(value, limit)
	if !ok {
		return nil, tooLong
	}

	return trimmed, nil
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
