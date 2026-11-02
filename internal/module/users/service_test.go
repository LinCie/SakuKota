package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/module/users"
	"rantau-go/internal/module/users/mock"
)

const testEmail = "ana@example.com"

var errBoom = errors.New("boom")

func newService(t *testing.T) (users.Service, *mock.MockRepository) {
	t.Helper()
	repo := mock.NewMockRepository(gomock.NewController(t))
	return users.NewService(repo), repo
}

// -----------------------------------------------------------------------------
// Create
// -----------------------------------------------------------------------------

func TestService_Create_ReturnsCreatedUser(t *testing.T) {
	svc, repo := newService(t)
	in := &users.User{Name: "Ana", Email: testEmail, PasswordHash: "hash"}
	want := &users.User{ID: 1, Name: "Ana", Email: testEmail, PasswordHash: "hash", Role: "user"}
	repo.EXPECT().Create(gomock.Any(), in).Return(want, nil)

	got, err := svc.Create(context.Background(), in)
	require.NoError(t, err, "create")
	assert.Equal(t, want, got)
}

func TestService_Create_PropagatesErrors(t *testing.T) {
	for _, want := range []error{users.ErrEmailTaken, errBoom} {
		t.Run(want.Error(), func(t *testing.T) {
			svc, repo := newService(t)
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, want)

			_, err := svc.Create(context.Background(), &users.User{Name: "Ana", Email: testEmail})
			assert.ErrorIs(t, err, want)
		})
	}
}

// -----------------------------------------------------------------------------
// GetByEmail
// -----------------------------------------------------------------------------

func TestService_GetByEmail_ReturnsUser(t *testing.T) {
	svc, repo := newService(t)
	want := &users.User{ID: 7, Email: testEmail}
	repo.EXPECT().GetByEmail(gomock.Any(), testEmail).Return(want, nil)

	got, err := svc.GetByEmail(context.Background(), testEmail)
	require.NoError(t, err, "get by email")
	assert.Equal(t, want, got)
}

func TestService_GetByEmail_PropagatesErrors(t *testing.T) {
	for _, want := range []error{users.ErrNotFound, errBoom} {
		t.Run(want.Error(), func(t *testing.T) {
			svc, repo := newService(t)
			repo.EXPECT().GetByEmail(gomock.Any(), testEmail).Return(nil, want)

			_, err := svc.GetByEmail(context.Background(), testEmail)
			assert.ErrorIs(t, err, want)
		})
	}
}

// -----------------------------------------------------------------------------
// GetUser
// -----------------------------------------------------------------------------

func TestService_GetUser_ReturnsUser(t *testing.T) {
	svc, repo := newService(t)
	want := &users.User{ID: 7, Email: testEmail}
	repo.EXPECT().GetUser(gomock.Any(), int64(7)).Return(want, nil)

	got, err := svc.GetUser(context.Background(), 7)
	require.NoError(t, err, "get by id")
	assert.Equal(t, want, got)
}

func TestService_GetUser_PropagatesErrors(t *testing.T) {
	for _, want := range []error{users.ErrNotFound, errBoom} {
		t.Run(want.Error(), func(t *testing.T) {
			svc, repo := newService(t)
			repo.EXPECT().GetUser(gomock.Any(), int64(7)).Return(nil, want)

			_, err := svc.GetUser(context.Background(), 7)
			assert.ErrorIs(t, err, want)
		})
	}
}
