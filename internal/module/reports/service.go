package reports

import (
	"context"
	"errors"

	"rantau-go/internal/domain"
	"rantau-go/internal/pkg/text"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// Repository is the persistence the service consumes.
type Repository interface {
	// Create saves the report as pending and returns it with its author. It
	// returns domain.ErrReportLimitReached at the limit,
	// domain.ErrAlreadyReported if the user's report of the same content is
	// pending, domain.ErrPlaceNotFound for a report of an unknown place, or
	// domain.ErrReviewNotFound for a report of a review the place lacks.
	Create(ctx context.Context, r *domain.Report) (*domain.Report, error)
	// Fetch returns the report with its author, or domain.ErrReportNotFound.
	Fetch(ctx context.Context, id int64) (*domain.Report, error)
	// List returns the reports matching the filter with their authors, oldest
	// first.
	List(ctx context.Context, filter Filter) ([]domain.Report, error)
	// Resolve marks the pending report id with status and returns it with its
	// author. It returns domain.ErrReportNotFound, or
	// domain.ErrReportAlreadyResolved if the report is not pending.
	Resolve(ctx context.Context, id, adminID int64, status domain.ReportStatus) (*domain.Report, error)
	// ResolveContent marks removed every pending report of the place, and of
	// its reviews, when reviewID is nil; otherwise of the review only.
	ResolveContent(ctx context.Context, placeID int64, reviewID *int64, adminID int64) error
}

// Transactor runs functions in one database transaction.
type Transactor interface {
	// WithinTx runs fn in a transaction carried by the context passed to fn.
	// It commits if fn returns nil, and otherwise rolls back and returns fn's
	// error.
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// PlaceService is the place operations that showing and removing a reported
// place require.
type PlaceService interface {
	// Fetch returns the place with its category and medias, or
	// domain.ErrPlaceNotFound.
	Fetch(ctx context.Context, id int64) (*domain.Place, error)
	// Delete removes the place with its reviews and medias, or returns
	// domain.ErrPlaceNotFound.
	Delete(ctx context.Context, id int64) error
}

// ReviewService is the review operations that showing and removing a reported
// review require.
type ReviewService interface {
	// FetchByID returns the place's review with its author and medias, or
	// domain.ErrReviewNotFound.
	FetchByID(ctx context.Context, placeID, reviewID int64) (*domain.Review, error)
	// Delete removes the place's review, its medias, and its rating, or
	// returns domain.ErrReviewNotFound.
	Delete(ctx context.Context, placeID, reviewID int64) error
}

// service implements the report use cases.
type service struct {
	repo    Repository
	tx      Transactor
	places  PlaceService
	reviews ReviewService
}

// NewService returns a service that stores reports in repo and removes
// reported content through places and reviews, in one transaction run by tx.
func NewService(repo Repository, tx Transactor, places PlaceService, reviews ReviewService) Service {
	return &service{
		repo:    repo,
		tx:      tx,
		places:  places,
		reviews: reviews,
	}
}

// Create checks the reason and the note (at most domain.MaxReportNoteLength
// characters), then saves the report as pending. A blank note is saved as
// nothing.
func (s *service) Create(ctx context.Context, r *domain.Report) (*domain.Report, error) {
	switch r.Reason {
	case domain.ReportReasonSpam, domain.ReportReasonOffensive, domain.ReportReasonFake, domain.ReportReasonOther:
	default:
		return nil, domain.ErrReportReasonInvalid
	}

	note, ok := text.TrimOptionalMax(r.Note, domain.MaxReportNoteLength)
	if !ok {
		return nil, domain.ErrReportNoteTooLong
	}
	r.Note = note

	return s.repo.Create(ctx, r)
}

// Fetch returns the report with its content as it is now: the place, and the
// review for a review report. Content that is gone is left nil. It returns
// domain.ErrReportNotFound if there is no such report.
func (s *service) Fetch(ctx context.Context, id int64) (*domain.Report, error) {
	report, err := s.repo.Fetch(ctx, id)
	if err != nil {
		return nil, err
	}

	report.Place, err = s.places.Fetch(ctx, report.PlaceID)
	if err != nil && !errors.Is(err, domain.ErrPlaceNotFound) {
		return nil, err
	}

	if report.ReviewID != nil {
		report.Review, err = s.reviews.FetchByID(ctx, report.PlaceID, *report.ReviewID)
		if err != nil && !errors.Is(err, domain.ErrReviewNotFound) {
			return nil, err
		}
	}

	return report, nil
}

// List returns the reports matching the filter, oldest first.
// It returns domain.ErrReportStatusInvalid for an unknown status.
func (s *service) List(ctx context.Context, filter Filter) ([]domain.Report, error) {
	switch filter.Status {
	case "", domain.ReportStatusPending, domain.ReportStatusRemoved, domain.ReportStatusDismissed:
	default:
		return nil, domain.ErrReportStatusInvalid
	}

	return s.repo.List(ctx, filter)
}

// Remove deletes the content of a pending report and marks removed the report
// and every other pending report of that content, in one transaction. Content
// already deleted another way still resolves the report. If the delete fails,
// the report stays pending.
func (s *service) Remove(ctx context.Context, id, adminID int64) (*domain.Report, error) {
	var removed *domain.Report
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// Resolving first locks the report, so a concurrent resolve waits and
		// then fails without touching the content.
		var err error
		removed, err = s.repo.Resolve(ctx, id, adminID, domain.ReportStatusRemoved)
		if err != nil {
			return err
		}

		if err := s.deleteContent(ctx, removed); err != nil {
			return err
		}

		return s.repo.ResolveContent(ctx, removed.PlaceID, removed.ReviewID, adminID)
	})
	if err != nil {
		return nil, err
	}

	return removed, nil
}

// Dismiss marks a pending report dismissed and keeps its content.
func (s *service) Dismiss(ctx context.Context, id, adminID int64) (*domain.Report, error) {
	return s.repo.Resolve(ctx, id, adminID, domain.ReportStatusDismissed)
}

// deleteContent deletes the review or place r reports. Content that is
// already gone is not an error.
func (s *service) deleteContent(ctx context.Context, r *domain.Report) error {
	var err error
	if r.ReviewID == nil {
		err = s.places.Delete(ctx, r.PlaceID)
	} else {
		err = s.reviews.Delete(ctx, r.PlaceID, *r.ReviewID)
	}

	if errors.Is(err, domain.ErrPlaceNotFound) || errors.Is(err, domain.ErrReviewNotFound) {
		return nil
	}

	return err
}
