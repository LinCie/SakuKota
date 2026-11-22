package storagecleanup

import (
	"context"
	"log/slog"
	"time"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// Repository is the deletion queue the Service needs.
type Repository interface {
	// Remove takes key off the queue.
	Remove(ctx context.Context, key string) error
	// ListDue returns up to limit keys that are due, longest due first.
	ListDue(ctx context.Context, limit int) ([]string, error)
	// Reschedule records a failed attempt to delete key and makes it due again
	// after retryIn.
	Reschedule(ctx context.Context, key string, retryIn time.Duration, lastErr string) error
}

// MediaStorage is the object storage the Service deletes from.
type MediaStorage interface {
	// Delete removes the object. Deleting a missing object is not an error.
	Delete(ctx context.Context, key string) error
}

// Service deletes queued storage objects.
type Service struct {
	repo  Repository
	media MediaStorage
}

// NewService builds the cleanup service from the queue and the media storage.
func NewService(repo Repository, media MediaStorage) *Service {
	return &Service{repo: repo, media: media}
}

// Run deletes one batch of due objects from storage. A failed delete is
// rescheduled and does not fail the run; a failure to update the queue does.
func (s *Service) Run(ctx context.Context) error {
	keys, err := s.repo.ListDue(ctx, batchSize)
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
func (s *Service) delete(ctx context.Context, key string) error {
	if err := s.media.Delete(ctx, key); err != nil {
		slog.ErrorContext(ctx, "delete storage object", "key", key, "err", err)
		return s.repo.Reschedule(ctx, key, retryDelay, err.Error())
	}

	return s.repo.Remove(ctx, key)
}
