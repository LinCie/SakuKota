package storagecleanup_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/module/storagecleanup"
	"rantau-go/internal/module/storagecleanup/mock"
)

const (
	keyA = "places/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
	keyB = "places/8/9f8e7d6c-5b4a-4321-8fed-cba987654321.png"
)

// newMocks returns a repository and object storage that fail the test on any
// call they do not expect.
func newMocks(t *testing.T) (*mock.MockRepository, *mock.MockImageStorage) {
	t.Helper()
	ctrl := gomock.NewController(t)
	return mock.NewMockRepository(ctrl), mock.NewMockImageStorage(ctrl)
}

func TestService_Run_DeletesDueObjects(t *testing.T) {
	repo, objects := newMocks(t)
	gomock.InOrder(
		repo.EXPECT().GetDue(gomock.Any(), 100).Return([]string{keyA, keyB}, nil),
		objects.EXPECT().Delete(gomock.Any(), keyA).Return(nil),
		repo.EXPECT().Remove(gomock.Any(), keyA).Return(nil),
		objects.EXPECT().Delete(gomock.Any(), keyB).Return(nil),
		repo.EXPECT().Remove(gomock.Any(), keyB).Return(nil),
	)

	require.NoError(t, storagecleanup.NewService(repo, objects).Run(context.Background()))
}

func TestService_Run_EmptyQueue(t *testing.T) {
	// Nothing is due, so nothing may be deleted.
	repo, objects := newMocks(t)
	repo.EXPECT().GetDue(gomock.Any(), 100).Return(nil, nil)

	require.NoError(t, storagecleanup.NewService(repo, objects).Run(context.Background()))
}

func TestService_Run_GetDueFailure(t *testing.T) {
	// Nothing may be deleted when the queue cannot be read.
	boom := errors.New("connection reset")
	repo, objects := newMocks(t)
	repo.EXPECT().GetDue(gomock.Any(), 100).Return(nil, boom)

	err := storagecleanup.NewService(repo, objects).Run(context.Background())
	require.ErrorIs(t, err, boom)
}

func TestService_Run_DeleteFailureIsRescheduled(t *testing.T) {
	// A failed delete is retried an hour later. It does not fail the run, and
	// the run carries on with the next key.
	repo, objects := newMocks(t)
	gomock.InOrder(
		repo.EXPECT().GetDue(gomock.Any(), 100).Return([]string{keyA, keyB}, nil),
		objects.EXPECT().Delete(gomock.Any(), keyA).Return(errors.New("connection reset")),
		repo.EXPECT().Reschedule(gomock.Any(), keyA, time.Hour, "connection reset").Return(nil),
		objects.EXPECT().Delete(gomock.Any(), keyB).Return(nil),
		repo.EXPECT().Remove(gomock.Any(), keyB).Return(nil),
	)

	require.NoError(t, storagecleanup.NewService(repo, objects).Run(context.Background()))
}

func TestService_Run_QueueUpdateFailure(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name   string
		expect func(repo *mock.MockRepository, objects *mock.MockImageStorage)
	}{
		{
			name: "remove failure is returned",
			expect: func(repo *mock.MockRepository, objects *mock.MockImageStorage) {
				objects.EXPECT().Delete(gomock.Any(), keyA).Return(nil)
				repo.EXPECT().Remove(gomock.Any(), keyA).Return(boom)
			},
		},
		{
			name: "reschedule failure is returned",
			expect: func(repo *mock.MockRepository, objects *mock.MockImageStorage) {
				objects.EXPECT().Delete(gomock.Any(), keyA).Return(errors.New("timeout"))
				repo.EXPECT().Reschedule(gomock.Any(), keyA, time.Hour, "timeout").Return(boom)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, objects := newMocks(t)
			repo.EXPECT().GetDue(gomock.Any(), 100).Return([]string{keyA}, nil)
			tc.expect(repo, objects)

			err := storagecleanup.NewService(repo, objects).Run(context.Background())
			require.ErrorIs(t, err, boom)
		})
	}
}
