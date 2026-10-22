package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"rantau-go/internal/config"
	"rantau-go/internal/module/auth"
	"rantau-go/internal/module/auth/mock"
)

const (
	testEmail    = "ana@example.com"
	testPassword = "correct-horse"
	testHash     = "hashed"
)

var errBoom = errors.New("boom")

func testConfig() config.Config {
	return config.Config{
		JWTSecret:          "test-secret",
		JWTExpiry:          15 * time.Minute,
		RefreshTokenExpiry: 24 * time.Hour,
		CookieSecure:       true,
	}
}

func newService(t *testing.T) (auth.Service, *mock.MockRepository, *mock.MockPasswordHasher, *mock.MockRefreshStore) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mock.NewMockRepository(ctrl)
	hasher := mock.NewMockPasswordHasher(ctrl)
	store := mock.NewMockRefreshStore(ctrl)
	return auth.NewService(testConfig(), repo, hasher, store), repo, hasher, store
}

// issueSession logs in testEmail and returns the issued tokens along with the
// key the refresh store was asked to save them under.
func issueSession(t *testing.T, svc auth.Service, repo *mock.MockRepository, hasher *mock.MockPasswordHasher, store *mock.MockRefreshStore) (*auth.Tokens, string) {
	t.Helper()

	repo.EXPECT().GetByEmail(gomock.Any(), testEmail).
		Return(&auth.User{ID: 7, Email: testEmail, PasswordHash: testHash}, nil)
	hasher.EXPECT().VerifyPassword(testPassword, testHash).Return(true, nil)

	var key string
	store.EXPECT().Save(gomock.Any(), gomock.Any(), int64(7), gomock.Any()).DoAndReturn(
		func(_ context.Context, k string, _ int64, _ time.Duration) error {
			key = k
			return nil
		})

	_, tokens, err := svc.Login(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	return tokens, key
}

// -----------------------------------------------------------------------------
// Register
// -----------------------------------------------------------------------------

func TestService_Register_NormalizesAndStoresHashedUser(t *testing.T) {
	svc, repo, hasher, _ := newService(t)

	hasher.EXPECT().HashPassword(testPassword).Return(testHash, nil)
	repo.EXPECT().Create(gomock.Any(), &auth.User{
		Name:         "Ana",
		Email:        testEmail,
		PasswordHash: testHash,
	}).Return(&auth.User{ID: 1, Name: "Ana", Email: testEmail, PasswordHash: testHash}, nil)

	u, err := svc.Register(context.Background(), &auth.User{Name: "  Ana  ", Email: "  ANA@Example.COM "}, testPassword)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if u.ID != 1 {
		t.Errorf("ID = %d, want 1", u.ID)
	}
}

func TestService_Register_RejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		userName string
		email    string
		password string
		want     error
	}{
		{name: "blank name", userName: "   ", email: testEmail, password: "password1", want: auth.ErrInvalidName},
		{name: "malformed email", userName: "Ana", email: "not-an-email", password: "password1", want: auth.ErrInvalidEmail},
		{name: "empty email", userName: "Ana", email: "", password: "password1", want: auth.ErrInvalidEmail},
		{name: "7-character password", userName: "Ana", email: testEmail, password: "1234567", want: auth.ErrWeakPassword},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: invalid input must never reach hashing or storage.
			svc, _, _, _ := newService(t)

			_, err := svc.Register(context.Background(), &auth.User{Name: tc.userName, Email: tc.email}, tc.password)
			if !errors.Is(err, tc.want) {
				t.Errorf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestService_Register_AcceptsEightCharacterPassword(t *testing.T) {
	svc, repo, hasher, _ := newService(t)

	hasher.EXPECT().HashPassword("12345678").Return(testHash, nil)
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&auth.User{ID: 1}, nil)

	if _, err := svc.Register(context.Background(), &auth.User{Name: "Ana", Email: testEmail}, "12345678"); err != nil {
		t.Errorf("register: %v", err)
	}
}

func TestService_Register_DuplicateEmailIsReturned(t *testing.T) {
	svc, repo, hasher, _ := newService(t)

	hasher.EXPECT().HashPassword(testPassword).Return(testHash, nil)
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, auth.ErrEmailTaken)

	_, err := svc.Register(context.Background(), &auth.User{Name: "Ana", Email: testEmail}, testPassword)
	if !errors.Is(err, auth.ErrEmailTaken) {
		t.Errorf("error = %v, want %v", err, auth.ErrEmailTaken)
	}
}

func TestService_Register_HashFailureStopsBeforeStorage(t *testing.T) {
	svc, _, hasher, _ := newService(t)

	hasher.EXPECT().HashPassword(testPassword).Return("", errBoom)

	_, err := svc.Register(context.Background(), &auth.User{Name: "Ana", Email: testEmail}, testPassword)
	if !errors.Is(err, errBoom) {
		t.Errorf("error = %v, want %v", err, errBoom)
	}
}

// -----------------------------------------------------------------------------
// Login
// -----------------------------------------------------------------------------

func TestService_Login_NormalizesEmailAndIssuesTokens(t *testing.T) {
	svc, repo, hasher, store := newService(t)

	repo.EXPECT().GetByEmail(gomock.Any(), testEmail).
		Return(&auth.User{ID: 7, Email: testEmail, PasswordHash: testHash}, nil)
	hasher.EXPECT().VerifyPassword(testPassword, testHash).Return(true, nil)

	var key string
	store.EXPECT().Save(gomock.Any(), gomock.Any(), int64(7), testConfig().RefreshTokenExpiry).DoAndReturn(
		func(_ context.Context, k string, _ int64, _ time.Duration) error {
			key = k
			return nil
		})

	u, tokens, err := svc.Login(context.Background(), "  ANA@Example.com ", testPassword)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if u.ID != 7 {
		t.Errorf("user ID = %d, want 7", u.ID)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Error("login must return both an access token and a refresh token")
	}
	if key == tokens.RefreshToken {
		t.Error("raw refresh token was passed to the store")
	}
}

func TestService_Login_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*mock.MockRepository, *mock.MockPasswordHasher)
	}{
		{
			name: "unknown email",
			setup: func(repo *mock.MockRepository, _ *mock.MockPasswordHasher) {
				repo.EXPECT().GetByEmail(gomock.Any(), testEmail).Return(nil, auth.ErrNotFound)
			},
		},
		{
			name: "wrong password",
			setup: func(repo *mock.MockRepository, hasher *mock.MockPasswordHasher) {
				repo.EXPECT().GetByEmail(gomock.Any(), testEmail).
					Return(&auth.User{ID: 7, Email: testEmail, PasswordHash: testHash}, nil)
				hasher.EXPECT().VerifyPassword(gomock.Any(), testHash).Return(false, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No store expectations: a failed login must not create a session.
			svc, repo, hasher, _ := newService(t)
			tc.setup(repo, hasher)

			u, tokens, err := svc.Login(context.Background(), testEmail, testPassword)
			if !errors.Is(err, auth.ErrInvalidCredentials) {
				t.Errorf("error = %v, want %v", err, auth.ErrInvalidCredentials)
			}
			if u != nil || tokens != nil {
				t.Error("failed login returned a user or tokens")
			}
		})
	}
}

func TestService_Login_RepositoryFailureIsNotBadCredentials(t *testing.T) {
	svc, repo, _, _ := newService(t)

	repo.EXPECT().GetByEmail(gomock.Any(), testEmail).Return(nil, errBoom)

	_, _, err := svc.Login(context.Background(), testEmail, testPassword)
	if !errors.Is(err, errBoom) {
		t.Errorf("error = %v, want %v", err, errBoom)
	}
	if errors.Is(err, auth.ErrInvalidCredentials) {
		t.Error("storage failure was reported as invalid credentials")
	}
}

func TestService_Login_HashVerificationFailureIsReturned(t *testing.T) {
	svc, repo, hasher, _ := newService(t)

	repo.EXPECT().GetByEmail(gomock.Any(), testEmail).
		Return(&auth.User{ID: 7, Email: testEmail, PasswordHash: "corrupt"}, nil)
	hasher.EXPECT().VerifyPassword(testPassword, "corrupt").Return(false, errBoom)

	_, _, err := svc.Login(context.Background(), testEmail, testPassword)
	if !errors.Is(err, errBoom) {
		t.Errorf("error = %v, want %v", err, errBoom)
	}
	if errors.Is(err, auth.ErrInvalidCredentials) {
		t.Error("hash verification failure was reported as invalid credentials")
	}
}

func TestService_Login_StoreFailureReturnsNoTokens(t *testing.T) {
	svc, repo, hasher, store := newService(t)

	repo.EXPECT().GetByEmail(gomock.Any(), testEmail).
		Return(&auth.User{ID: 7, Email: testEmail, PasswordHash: testHash}, nil)
	hasher.EXPECT().VerifyPassword(testPassword, testHash).Return(true, nil)
	store.EXPECT().Save(gomock.Any(), gomock.Any(), int64(7), gomock.Any()).Return(errBoom)

	u, tokens, err := svc.Login(context.Background(), testEmail, testPassword)
	if !errors.Is(err, errBoom) {
		t.Errorf("error = %v, want %v", err, errBoom)
	}
	if u != nil || tokens != nil {
		t.Error("login returned a user or tokens after the session could not be saved")
	}
}

// -----------------------------------------------------------------------------
// Refresh
// -----------------------------------------------------------------------------

func TestService_Refresh_ConsumesOldSessionAndIssuesNewTokens(t *testing.T) {
	svc, repo, hasher, store := newService(t)
	first, key := issueSession(t, svc, repo, hasher, store)

	store.EXPECT().Consume(gomock.Any(), key).Return(int64(7), nil)
	repo.EXPECT().GetByID(gomock.Any(), int64(7)).
		Return(&auth.User{ID: 7, Email: testEmail, PasswordHash: testHash}, nil)
	store.EXPECT().Save(gomock.Any(), gomock.Any(), int64(7), gomock.Any()).Return(nil)

	u, second, err := svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if u.ID != 7 {
		t.Errorf("user ID = %d, want 7", u.ID)
	}
	if second.AccessToken == "" {
		t.Error("refresh did not return an access token")
	}
	if second.RefreshToken == first.RefreshToken {
		t.Error("refresh token was not rotated")
	}
}

func TestService_Refresh_UnknownTokenIsRejected(t *testing.T) {
	svc, _, _, store := newService(t)

	store.EXPECT().Consume(gomock.Any(), gomock.Any()).Return(int64(0), auth.ErrInvalidRefreshToken)

	_, _, err := svc.Refresh(context.Background(), "never-issued")
	if !errors.Is(err, auth.ErrInvalidRefreshToken) {
		t.Errorf("error = %v, want %v", err, auth.ErrInvalidRefreshToken)
	}
}

func TestService_Refresh_DeletedUserIsRejected(t *testing.T) {
	svc, repo, hasher, store := newService(t)
	first, key := issueSession(t, svc, repo, hasher, store)

	store.EXPECT().Consume(gomock.Any(), key).Return(int64(7), nil)
	repo.EXPECT().GetByID(gomock.Any(), int64(7)).Return(nil, auth.ErrNotFound)

	// No Save expectation: no session may be issued for a missing user.
	_, _, err := svc.Refresh(context.Background(), first.RefreshToken)
	if !errors.Is(err, auth.ErrInvalidRefreshToken) {
		t.Errorf("error = %v, want %v", err, auth.ErrInvalidRefreshToken)
	}
}

func TestService_Refresh_RepositoryFailureIsReturned(t *testing.T) {
	svc, repo, hasher, store := newService(t)
	first, key := issueSession(t, svc, repo, hasher, store)

	store.EXPECT().Consume(gomock.Any(), key).Return(int64(7), nil)
	repo.EXPECT().GetByID(gomock.Any(), int64(7)).Return(nil, errBoom)

	_, _, err := svc.Refresh(context.Background(), first.RefreshToken)
	if !errors.Is(err, errBoom) {
		t.Errorf("error = %v, want %v", err, errBoom)
	}
	if errors.Is(err, auth.ErrInvalidRefreshToken) {
		t.Error("storage failure was reported as an invalid refresh token")
	}
}

// -----------------------------------------------------------------------------
// Logout
// -----------------------------------------------------------------------------

func TestService_Logout_RevokesSessionKey(t *testing.T) {
	svc, repo, hasher, store := newService(t)
	first, key := issueSession(t, svc, repo, hasher, store)

	store.EXPECT().Delete(gomock.Any(), key).Return(nil)

	if err := svc.Logout(context.Background(), first.RefreshToken); err != nil {
		t.Errorf("logout: %v", err)
	}
}

func TestService_Logout_StoreFailureIsReturned(t *testing.T) {
	svc, repo, hasher, store := newService(t)
	first, _ := issueSession(t, svc, repo, hasher, store)

	store.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errBoom)

	if err := svc.Logout(context.Background(), first.RefreshToken); !errors.Is(err, errBoom) {
		t.Errorf("error = %v, want %v", err, errBoom)
	}
}
