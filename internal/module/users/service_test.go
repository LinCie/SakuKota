package users_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/domain"
	"rantau-go/internal/module/users"
	"rantau-go/internal/module/users/mock"
)

const testEmail = "ana@example.com"

var errBoom = errors.New("boom")

func newService(t *testing.T) (*users.Service, *mock.MockRepository) {
	t.Helper()
	repo := mock.NewMockRepository(gomock.NewController(t))
	return users.NewService(repo), repo
}

func TestService_Create_ReturnsCreatedUser(t *testing.T) {
	svc, repo := newService(t)
	in := &domain.User{Name: "Ana", Email: testEmail, PasswordHash: "hash"}
	want := &domain.User{ID: 1, Name: "Ana", Email: testEmail, PasswordHash: "hash", Role: "user"}
	repo.EXPECT().Create(gomock.Any(), in).Return(want, nil)

	got, err := svc.Create(t.Context(), in)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_Create_PropagatesErrors(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{name: "EmailTaken", want: domain.ErrEmailTaken},
		{name: "Boom", want: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newService(t)
			in := &domain.User{Name: "Ana", Email: testEmail}
			repo.EXPECT().Create(gomock.Any(), in).Return(nil, tc.want)

			_, err := svc.Create(t.Context(), in)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestService_FetchByEmail_ReturnsUser(t *testing.T) {
	svc, repo := newService(t)
	want := &domain.User{ID: 7, Email: testEmail}
	repo.EXPECT().FetchByEmail(gomock.Any(), testEmail).Return(want, nil)

	got, err := svc.FetchByEmail(t.Context(), testEmail)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_FetchByEmail_PropagatesErrors(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{name: "NotFound", want: domain.ErrUserNotFound},
		{name: "Boom", want: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newService(t)
			repo.EXPECT().FetchByEmail(gomock.Any(), testEmail).Return(nil, tc.want)

			_, err := svc.FetchByEmail(t.Context(), testEmail)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestService_Fetch_ReturnsUser(t *testing.T) {
	svc, repo := newService(t)
	want := &domain.User{ID: 7, Email: testEmail}
	repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(want, nil)

	got, err := svc.Fetch(t.Context(), 7)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_Fetch_PropagatesErrors(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{name: "NotFound", want: domain.ErrUserNotFound},
		{name: "Boom", want: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newService(t)
			repo.EXPECT().Fetch(gomock.Any(), int64(7)).Return(nil, tc.want)

			_, err := svc.Fetch(t.Context(), 7)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}
