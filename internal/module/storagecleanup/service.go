package storagecleanup

import (
	"context"
	"log/slog"
	"time"
)

// Service defines the cleanup operations.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	// Run deletes one batch of due objects from storage.
	Run(ctx context.Context) error
}

type service struct {
	repo    Repository
	objects ObjectStorage
}

func NewService(repo Repository, objects ObjectStorage) *service {
	return &service{
		repo:    repo,
		objects: objects,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

// Run claims due deletions and deletes their objects. A failed delete is
// rescheduled and does not fail the run; a failure to update the queue does.
func (s *service) Run(ctx context.Context) error {
	deletions, err := s.repo.ClaimDue(ctx, batchSize)
	if err != nil {
		return err
	}

	for _, d := range deletions {
		if err := s.delete(ctx, d); err != nil {
			return err
		}
	}

	return nil
}

// delete deletes the object of d and takes it off the queue, or reschedules it
// if the object cannot be deleted. Only a failure to update the queue is
// returned.
func (s *service) delete(ctx context.Context, d Deletion) error {
	if err := s.objects.Delete(ctx, d.Key); err != nil {
		slog.ErrorContext(ctx, "delete storage object", "key", d.Key, "attempt", d.Attempts+1, "err", err)
		return s.repo.Reschedule(ctx, d.Key, retryDelay(d.Attempts), err.Error())
	}

	return s.repo.Remove(ctx, d.Key)
}

// retryDelay returns how long to wait before retrying a delete that has now
// failed attempts+1 times.
func retryDelay(attempts int) time.Duration {
	delay := baseRetryDelay
	for range attempts {
		delay *= 2
		if delay >= maxRetryDelay {
			return maxRetryDelay
		}
	}

	return delay
}
