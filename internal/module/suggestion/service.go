package suggestion

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"rantau-go/internal/module/place"
	"rantau-go/internal/postgis"
)

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	// Create stores the suggestion as pending. The name is trimmed and must be
	// 1 to 150 characters, and the location must lie within Indonesia. Blank
	// optional text is stored as nil. Attributes must be a JSON object.
	Create(ctx context.Context, s *Suggestion) (*Suggestion, error)
	// Get returns the suggestion with the given ID.
	Get(ctx context.Context, id int64) (*Suggestion, error)
	// GetMine returns the user's suggestions, newest first.
	GetMine(ctx context.Context, userID int64) ([]Suggestion, error)
	// GetMany returns the suggestions matching the filter, oldest first.
	GetMany(ctx context.Context, filter Filter) ([]Suggestion, error)
	// Approve creates the place p from a pending suggestion and marks the
	// suggestion approved as that place. p must have a location.
	Approve(ctx context.Context, id, reviewerID int64, p *place.Place) (*Suggestion, error)
	// Reject marks a pending suggestion rejected. The reason is trimmed and
	// stored as nil when blank, and must be at most 1000 characters.
	Reject(ctx context.Context, id, reviewerID int64, reason *string) (*Suggestion, error)
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

func NewService(repo Repository, places PlaceService) *service {
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

	if err := validateLocation(sg.Location); err != nil {
		return nil, err
	}

	attributes, err := normalizeAttributes(sg.Attributes)
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
		Address:             trimOptional(sg.Address),
		Location:            sg.Location,
		OperatingHours:      trimOptional(sg.OperatingHours),
		EstimatedPriceRange: trimOptional(sg.EstimatedPriceRange),
		ShortDescription:    trimOptional(sg.ShortDescription),
		Attributes:          attributes,
		Note:                note,
		Status:              StatusPending,
	})
}

func (s *service) Get(ctx context.Context, id int64) (*Suggestion, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetMine(ctx context.Context, userID int64) ([]Suggestion, error) {
	return s.repo.GetByUser(ctx, userID)
}

func (s *service) GetMany(ctx context.Context, filter Filter) ([]Suggestion, error) {
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
func (s *service) Approve(ctx context.Context, id, reviewerID int64, p *place.Place) (*Suggestion, error) {
	// Checked before the suggestion is read: a place without a location is
	// refused without touching storage.
	if p.Location == nil {
		return nil, ErrInvalidLocation
	}

	current, err := s.repo.GetByID(ctx, id)
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

	approved, err := s.repo.MarkApproved(ctx, id, created.ID, reviewerID)
	if err != nil {
		// A failure to delete the place is ignored: the place is left without
		// a suggestion, but the approval itself has failed either way.
		_ = s.places.Delete(ctx, created.ID)
		return nil, err
	}

	return approved, nil
}

func (s *service) Reject(ctx context.Context, id, reviewerID int64, reason *string) (*Suggestion, error) {
	reason, err := trimOptionalWithLimit(reason, maxReasonLength, ErrReasonTooLong)
	if err != nil {
		return nil, err
	}

	return s.repo.Reject(ctx, id, reviewerID, reason)
}

// normalizeName trims name and checks it against the length of PLACE_SUGGESTIONS.name.
func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", ErrInvalidName
	case utf8.RuneCountInString(name) > maxNameLength:
		return "", ErrNameTooLong
	}

	return name, nil
}

// trimOptional trims text and returns nil if nothing is left.
func trimOptional(text *string) *string {
	if text == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*text)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

// trimOptionalWithLimit trims text like trimOptional, and reports tooLong if
// what is left is longer than limit characters.
func trimOptionalWithLimit(text *string, limit int, tooLong error) (*string, error) {
	trimmed := trimOptional(text)
	if trimmed != nil && utf8.RuneCountInString(*trimmed) > limit {
		return nil, tooLong
	}

	return trimmed, nil
}

// validateLocation reports ErrInvalidLocation unless loc is set and lies within Indonesia.
func validateLocation(loc *postgis.Point) error {
	if loc == nil || loc.X() < indonesiaMinLongitude || loc.X() > indonesiaMaxLongitude ||
		loc.Y() < indonesiaMinLatitude || loc.Y() > indonesiaMaxLatitude {
		return ErrInvalidLocation
	}

	return nil
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
