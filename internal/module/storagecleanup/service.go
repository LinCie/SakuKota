package storagecleanup

import (
	"context"
	"log/slog"
)

// Service defines the cleanup operations.
type Service interface {
	// Run deletes one batch of due objects from storage.
	Run(ctx context.Context) error
}

type service struct {
	repo  Repository
	media MediaStorage
}

// NewService builds the cleanup service from the queue and the media storage.
func NewService(repo Repository, media MediaStorage) Service {
	return &service{
		repo:  repo,
		media: media,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

// Run gets the due keys and deletes their objects. A failed delete is
// rescheduled and does not fail the run; a failure to update the queue does.
func (s *service) Run(ctx context.Context) error {
	keys, err := s.repo.GetDue(ctx, batchSize)
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err := s.delete(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

// delete removes the object for key, then takes key off the queue.
// If the object cannot be removed, key is rescheduled and only queue errors are returned.
func (s *service) delete(ctx context.Context, key string) error {
	if err := s.media.Delete(ctx, key); err != nil {
		slog.ErrorContext(ctx, "delete storage object", "key", key, "err", err)
		return s.repo.Reschedule(ctx, key, retryDelay, err.Error())
	}

	return s.repo.Remove(ctx, key)
}
