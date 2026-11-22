package storagecleanup_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/module/storagecleanup"
	"rantau-go/internal/module/storagecleanup/mock"
)

const (
	keyA = "places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
	keyB = "places/8/9f8e7d6c-5b4a-4321-8fed-cba987654321.png"
)

var errReset = errors.New("connection reset")

// newMocks returns the queue and object storage mocks.
func newMocks(t *testing.T) (*mock.MockRepository, *mock.MockMediaStorage) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mock.NewMockRepository(ctrl)
	objects := mock.NewMockMediaStorage(ctrl)
	return repo, objects
}

func TestService_Run_DeletesDueObjects(t *testing.T) {
	repo, objects := newMocks(t)
	svc := storagecleanup.NewService(repo, objects)
	gomock.InOrder(
		repo.EXPECT().ListDue(gomock.Any(), 100).Return([]string{keyA, keyB}, nil),
		objects.EXPECT().Delete(gomock.Any(), keyA).Return(nil),
		repo.EXPECT().Remove(gomock.Any(), keyA).Return(nil),
		objects.EXPECT().Delete(gomock.Any(), keyB).Return(nil),
		repo.EXPECT().Remove(gomock.Any(), keyB).Return(nil),
	)

	assert.NoError(t, svc.Run(t.Context()))
}

func TestService_Run_EmptyQueue(t *testing.T) {
	// Nothing is due, so nothing may be deleted.
	repo, objects := newMocks(t)
	svc := storagecleanup.NewService(repo, objects)
	repo.EXPECT().ListDue(gomock.Any(), 100).Return(nil, nil)

	assert.NoError(t, svc.Run(t.Context()))
}

func TestService_Run_ListDueFailure(t *testing.T) {
	// Nothing may be deleted when the queue cannot be read.
	repo, objects := newMocks(t)
	svc := storagecleanup.NewService(repo, objects)
	repo.EXPECT().ListDue(gomock.Any(), 100).Return(nil, errReset)

	err := svc.Run(t.Context())
	assert.ErrorIs(t, err, errReset)
}

func TestService_Run_DeleteFailureIsRescheduled(t *testing.T) {
	// A failed delete is retried an hour later. It does not fail the run, and
	// the run carries on with the next key.
	repo, objects := newMocks(t)
	svc := storagecleanup.NewService(repo, objects)
	gomock.InOrder(
		repo.EXPECT().ListDue(gomock.Any(), 100).Return([]string{keyA, keyB}, nil),
		objects.EXPECT().Delete(gomock.Any(), keyA).Return(errReset),
		repo.EXPECT().Reschedule(gomock.Any(), keyA, time.Hour, "connection reset").Return(nil),
		objects.EXPECT().Delete(gomock.Any(), keyB).Return(nil),
		repo.EXPECT().Remove(gomock.Any(), keyB).Return(nil),
	)

	assert.NoError(t, svc.Run(t.Context()))
}

func TestService_Run_QueueUpdateFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name  string
		setup func(repo *mock.MockRepository, objects *mock.MockMediaStorage)
	}{
		{
			name: "RemoveFailure",
			setup: func(repo *mock.MockRepository, objects *mock.MockMediaStorage) {
				gomock.InOrder(
					objects.EXPECT().Delete(gomock.Any(), keyA).Return(nil),
					repo.EXPECT().Remove(gomock.Any(), keyA).Return(boom),
				)
			},
		},
		{
			name: "RescheduleFailure",
			setup: func(repo *mock.MockRepository, objects *mock.MockMediaStorage) {
				gomock.InOrder(
					objects.EXPECT().Delete(gomock.Any(), keyA).Return(errors.New("timeout")),
					repo.EXPECT().Reschedule(gomock.Any(), keyA, time.Hour, "timeout").Return(boom),
				)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, objects := newMocks(t)
			svc := storagecleanup.NewService(repo, objects)
			repo.EXPECT().ListDue(gomock.Any(), 100).Return([]string{keyA}, nil)
			tc.setup(repo, objects)

			err := svc.Run(t.Context())
			assert.ErrorIs(t, err, boom)
		})
	}
}
