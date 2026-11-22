package suggestions

import (
	"context"
	"encoding/json"

	"rantau-go/internal/domain"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/pkg/text"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// Repository is the persistence the service consumes.
type Repository interface {
	// Create saves the suggestion as pending and returns it with its author.
	// It returns domain.ErrSuggestionLimitReached at the limit, or
	// domain.ErrSubDistrictNotFound or domain.ErrCategoryNotFound.
	Create(ctx context.Context, s *domain.Suggestion) (*domain.Suggestion, error)
	// Fetch returns the suggestion with its author, or domain.ErrSuggestionNotFound.
	Fetch(ctx context.Context, id int64) (*domain.Suggestion, error)
	// ListByUser returns the user's suggestions with their author, newest first.
	ListByUser(ctx context.Context, userID int64) ([]domain.Suggestion, error)
	// List returns the suggestions matching the filter with their authors,
	// oldest first.
	List(ctx context.Context, filter Filter) ([]domain.Suggestion, error)
	// MarkApproved marks the suggestion approved and links it to the place placeID.
	// It returns domain.ErrSuggestionNotFound, or domain.ErrSuggestionAlreadyReviewed
	// if it is not pending.
	MarkApproved(ctx context.Context, id, placeID, adminID int64) (*domain.Suggestion, error)
	// Reject marks the suggestion rejected with the reason. It returns
	// domain.ErrSuggestionNotFound, or domain.ErrSuggestionAlreadyReviewed if the
	// suggestion is not pending.
	Reject(ctx context.Context, id, adminID int64, reason *string) (*domain.Suggestion, error)
}

// PlaceService is the place operations that approving a suggestion requires.
type PlaceService interface {
	// Create saves a new place, which approval uses to make a place from a suggestion.
	Create(ctx context.Context, p *domain.Place) (*domain.Place, error)
}

// Transactor runs functions in one database transaction.
type Transactor interface {
	// WithinTx runs fn in a transaction carried by the context passed to fn.
	// It commits if fn returns nil, and otherwise rolls back and returns fn's
	// error.
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Service implements the suggestion use cases.
type Service struct {
	repo   Repository
	tx     Transactor
	places PlaceService
}

// NewService returns a Service that stores suggestions in repo and creates
// approved places through places, approving in one transaction run by tx.
func NewService(repo Repository, tx Transactor, places PlaceService) *Service {
	return &Service{
		repo:   repo,
		tx:     tx,
		places: places,
	}
}

// Create checks the name, location, and attributes, then saves the suggestion as pending.
// Blank optional text is saved as nothing.
func (s *Service) Create(ctx context.Context, sg *domain.Suggestion) (*domain.Suggestion, error) {
	name, err := domain.NormalizePlaceName(sg.Name)
	if err != nil {
		return nil, err
	}

	if !postgis.InIndonesia(sg.Location) {
		return nil, domain.ErrSuggestionLocationInvalid
	}

	attributes, err := normalizeAttributes(sg.Attributes)
	if err != nil {
		return nil, err
	}

	// The optional place text follows the place rules, so they check and trim it.
	place := &domain.Place{
		Address:             sg.Address,
		OperatingHours:      sg.OperatingHours,
		EstimatedPriceRange: sg.EstimatedPriceRange,
		ShortDescription:    sg.ShortDescription,
	}
	if err := domain.NormalizePlaceFields(place); err != nil {
		return nil, err
	}

	note, ok := text.TrimOptionalMax(sg.Note, domain.MaxSuggestionNoteLength)
	if !ok {
		return nil, domain.ErrSuggestionNoteTooLong
	}

	sg.Name = name
	sg.Address = place.Address
	sg.OperatingHours = place.OperatingHours
	sg.EstimatedPriceRange = place.EstimatedPriceRange
	sg.ShortDescription = place.ShortDescription
	sg.Attributes = attributes
	sg.Note = note

	return s.repo.Create(ctx, sg)
}

// Fetch returns the suggestion with the given ID.
func (s *Service) Fetch(ctx context.Context, id int64) (*domain.Suggestion, error) {
	return s.repo.Fetch(ctx, id)
}

// ListByUser returns the user's suggestions, newest first.
func (s *Service) ListByUser(ctx context.Context, userID int64) ([]domain.Suggestion, error) {
	return s.repo.ListByUser(ctx, userID)
}

// List returns the suggestions matching the filter, oldest first.
// It returns domain.ErrSuggestionStatusInvalid for an unknown status.
func (s *Service) List(ctx context.Context, filter Filter) ([]domain.Suggestion, error) {
	switch filter.Status {
	case "", domain.StatusPending, domain.StatusApproved, domain.StatusRejected:
	default:
		return nil, domain.ErrSuggestionStatusInvalid
	}

	return s.repo.List(ctx, filter)
}

// Approve creates the place p from a pending suggestion and marks the
// suggestion approved as that place, in one transaction. p must have a
// location. If either step fails, neither is kept.
func (s *Service) Approve(ctx context.Context, id, adminID int64, p *domain.Place) (*domain.Suggestion, error) {
	// Checked before the suggestion is read: a place without a location is
	// refused without touching storage.
	if p.Location == nil {
		return nil, domain.ErrSuggestionLocationInvalid
	}

	current, err := s.repo.Fetch(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status != domain.StatusPending {
		return nil, domain.ErrSuggestionAlreadyReviewed
	}

	var approved *domain.Suggestion
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		created, err := s.places.Create(ctx, p)
		if err != nil {
			return err
		}

		approved, err = s.repo.MarkApproved(ctx, id, created.ID, adminID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return approved, nil
}

// Reject marks a pending suggestion rejected. The reason is trimmed and
// stored as nil when blank, and must be at most 1000 characters.
func (s *Service) Reject(ctx context.Context, id, adminID int64, reason *string) (*domain.Suggestion, error) {
	trimmed, ok := text.TrimOptionalMax(reason, domain.MaxSuggestionReasonLength)
	if !ok {
		return nil, domain.ErrSuggestionReasonTooLong
	}

	return s.repo.Reject(ctx, id, adminID, trimmed)
}

// normalizeAttributes checks that raw is a JSON object and returns it as sent.
// A missing or null value becomes an empty object.
func normalizeAttributes(raw json.RawMessage) (json.RawMessage, error) {
	var object map[string]json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &object); err != nil {
			return nil, domain.ErrSuggestionAttributesInvalid
		}
	}

	if object == nil {
		return json.RawMessage(`{}`), nil
	}

	return raw, nil
}
