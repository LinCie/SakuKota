package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/config"
	"rantau-go/internal/domain"
	"rantau-go/internal/module/auth"
	"rantau-go/internal/module/auth/mock"
)

const (
	testEmail    = "ana@example.com"
	testPassword = "correct-horse"
	// testPasswordHash is an argon2id hash of testPassword in the format hashPassword produces.
	testPasswordHash = "$argon2id$v=19$m=19456,t=2,p=1$MDEyMzQ1Njc4OWFiY2RlZg$Y7M1bhWqqijLLRFsFRE4eQcEvQVz3Bj9bWBHHIpCniI"
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

// hashToken is the SHA-256 hex digest the service keys refresh tokens by.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// newService uses the real password hashing; the user service and the refresh
// store are mocks that fail the test on any call without an expectation.
func newService(t *testing.T) (*auth.Service, *mock.MockUserService, *mock.MockRefreshStore) {
	t.Helper()
	ctrl := gomock.NewController(t)
	users := mock.NewMockUserService(ctrl)
	store := mock.NewMockRefreshStore(ctrl)
	return auth.NewService(testConfig(), users, store), users, store
}

// expectSave expects one Save of a refresh token for userID and returns a
// pointer to the token hash it was given, which is set once Save is called.
func expectSave(store *mock.MockRefreshStore, userID int64) *string {
	saved := new(string)
	store.EXPECT().Save(gomock.Any(), gomock.Any(), userID, testConfig().RefreshTokenExpiry).DoAndReturn(
		func(_ context.Context, tokenHash string, _ int64, _ time.Duration) error {
			*saved = tokenHash
			return nil
		})
	return saved
}

// issueSession logs in testEmail and returns the issued tokens.
func issueSession(t *testing.T, svc *auth.Service, users *mock.MockUserService, store *mock.MockRefreshStore) *auth.Tokens {
	t.Helper()

	users.EXPECT().FetchByEmail(gomock.Any(), testEmail).Return(&domain.User{ID: 7, Email: testEmail, PasswordHash: testPasswordHash}, nil)
	expectSave(store, 7)

	_, tokens, err := svc.Login(t.Context(), testEmail, testPassword)
	require.NoError(t, err, "Login(%q)", testEmail)
	return tokens
}

func TestService_Register_NormalizesAndStoresHashedUser(t *testing.T) {
	svc, users, _ := newService(t)

	var stored *domain.User
	users.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) (*domain.User, error) {
		stored = u
		return &domain.User{ID: 1, Name: u.Name, Email: u.Email, PasswordHash: u.PasswordHash}, nil
	})

	u, err := svc.Register(t.Context(), &domain.User{Name: "  Ana  ", Email: "  ANA@Example.COM "}, testPassword)
	require.NoError(t, err)
	require.NotNil(t, stored, "Register() did not store the user")
	assert.EqualValues(t, 1, u.ID)
	assert.Equal(t, "Ana", stored.Name)
	assert.Equal(t, testEmail, stored.Email)
	assert.True(t, strings.HasPrefix(stored.PasswordHash, "$argon2id$"), "stored PasswordHash = %q, want an argon2id hash", stored.PasswordHash)
}

func TestService_Register_RejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		userName string
		email    string
		password string
		want     error
	}{
		{name: "blank_name", userName: "   ", email: testEmail, password: "password1", want: domain.ErrUserNameRequired},
		{name: "malformed_email", userName: "Ana", email: "not-an-email", password: "password1", want: domain.ErrEmailInvalid},
		{name: "empty_email", userName: "Ana", email: "", password: "password1", want: domain.ErrEmailInvalid},
		{name: "7_character_password", userName: "Ana", email: testEmail, password: "1234567", want: domain.ErrPasswordTooShort},
		{name: "4_multibyte_characters_are_fewer_than_8", userName: "Ana", email: testEmail, password: strings.Repeat("é", 4), want: domain.ErrPasswordTooShort},
		{name: "101_character_name", userName: strings.Repeat("a", 101), email: testEmail, password: "password1", want: domain.ErrUserNameTooLong},
		{name: "101_multibyte_character_name", userName: strings.Repeat("é", 101), email: testEmail, password: "password1", want: domain.ErrUserNameTooLong},
		{name: "email_over_255_characters", userName: "Ana", email: strings.Repeat("a", 245) + "@example.com", password: "password1", want: domain.ErrEmailTooLong},
		{name: "display_name_is_not_a_bare_address", userName: "Ana", email: "Ana <ana@example.com>", password: "password1", want: domain.ErrEmailInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Invalid input must never reach storage: the mock has no expectations.
			svc, _, _ := newService(t)

			_, err := svc.Register(t.Context(), &domain.User{Name: tc.userName, Email: tc.email}, tc.password)
			assert.ErrorIs(t, err, tc.want, "Register(%q, %q, %q)", tc.userName, tc.email, tc.password)
		})
	}
}

func TestService_Register_AcceptsEightCharacterPassword(t *testing.T) {
	svc, users, _ := newService(t)
	users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&domain.User{ID: 1}, nil)

	_, err := svc.Register(t.Context(), &domain.User{Name: "Ana", Email: testEmail}, "12345678")
	assert.NoError(t, err, "Register() with an 8-character password")
}

func TestService_Register_AcceptsHundredCharacterName(t *testing.T) {
	svc, users, _ := newService(t)
	users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&domain.User{ID: 1}, nil)

	_, err := svc.Register(t.Context(), &domain.User{Name: strings.Repeat("a", 100), Email: testEmail}, "password1")
	assert.NoError(t, err, "Register() with a 100-character name")
}

func TestService_Register_DuplicateEmailIsReturned(t *testing.T) {
	svc, users, _ := newService(t)
	users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, domain.ErrEmailTaken)

	_, err := svc.Register(t.Context(), &domain.User{Name: "Ana", Email: testEmail}, testPassword)
	assert.ErrorIs(t, err, domain.ErrEmailTaken)
}

func TestService_Login_NormalizesEmailAndIssuesTokens(t *testing.T) {
	svc, users, store := newService(t)
	users.EXPECT().FetchByEmail(gomock.Any(), testEmail).Return(&domain.User{ID: 7, Email: testEmail, PasswordHash: testPasswordHash}, nil)
	savedHash := expectSave(store, 7)

	u, tokens, err := svc.Login(t.Context(), "  ANA@Example.com ", testPassword)
	require.NoError(t, err)
	assert.EqualValues(t, 7, u.ID)
	assert.NotEmpty(t, tokens.AccessToken, "Login() returned no access token")
	require.NotEmpty(t, tokens.RefreshToken, "Login() returned no refresh token")
	assert.Equal(t, hashToken(tokens.RefreshToken), *savedHash, "the refresh token is saved by its hash")
}

func TestService_Login_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name     string
		user     *domain.User
		fetchErr error
		password string
	}{
		{name: "unknown_email", user: nil, fetchErr: domain.ErrUserNotFound, password: testPassword},
		{name: "wrong_password", user: &domain.User{ID: 7, Email: testEmail, PasswordHash: testPasswordHash}, fetchErr: nil, password: "wrong-horse"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, users, _ := newService(t)
			users.EXPECT().FetchByEmail(gomock.Any(), testEmail).Return(tc.user, tc.fetchErr)

			u, tokens, err := svc.Login(t.Context(), testEmail, tc.password)
			assert.ErrorIs(t, err, domain.ErrInvalidCredentials, "Login(%q, %q)", testEmail, tc.password)
			assert.Nil(t, u, "Login(%q, %q) user", testEmail, tc.password)
			assert.Nil(t, tokens, "Login(%q, %q) tokens", testEmail, tc.password)
		})
	}
}

func TestService_Login_RepositoryFailureIsNotBadCredentials(t *testing.T) {
	svc, users, _ := newService(t)
	users.EXPECT().FetchByEmail(gomock.Any(), testEmail).Return(nil, errBoom)

	_, _, err := svc.Login(t.Context(), testEmail, testPassword)
	assert.ErrorIs(t, err, errBoom)
	assert.NotErrorIs(t, err, domain.ErrInvalidCredentials, "Login() reported a storage failure as invalid credentials")
}

func TestService_Login_CorruptHashIsNotBadCredentials(t *testing.T) {
	svc, users, _ := newService(t)
	users.EXPECT().FetchByEmail(gomock.Any(), testEmail).Return(&domain.User{ID: 7, Email: testEmail, PasswordHash: "corrupt"}, nil)

	_, _, err := svc.Login(t.Context(), testEmail, testPassword)
	require.Error(t, err, "Login() with a corrupt stored hash succeeded")
	assert.NotErrorIs(t, err, domain.ErrInvalidCredentials, "Login() reported a corrupt stored hash as invalid credentials")
}

func TestService_Refresh_ConsumesOldSessionAndIssuesNewTokens(t *testing.T) {
	svc, users, store := newService(t)
	first := issueSession(t, svc, users, store)

	// The old refresh token is consumed once, so reusing it must fail.
	oldHash := hashToken(first.RefreshToken)
	var newSaved *string
	gomock.InOrder(
		store.EXPECT().Consume(gomock.Any(), oldHash).Return(int64(7), nil),
		users.EXPECT().Fetch(gomock.Any(), int64(7)).Return(&domain.User{ID: 7, Email: testEmail, PasswordHash: testPasswordHash}, nil),
		store.EXPECT().Save(gomock.Any(), gomock.Any(), int64(7), testConfig().RefreshTokenExpiry).DoAndReturn(
			func(_ context.Context, tokenHash string, _ int64, _ time.Duration) error {
				newSaved = &tokenHash
				return nil
			}),
		store.EXPECT().Consume(gomock.Any(), oldHash).Return(int64(0), domain.ErrInvalidRefreshToken),
	)

	u, second, err := svc.Refresh(t.Context(), first.RefreshToken)
	require.NoError(t, err)
	assert.EqualValues(t, 7, u.ID)
	assert.NotEmpty(t, second.AccessToken, "Refresh() returned no access token")
	assert.NotEqual(t, first.RefreshToken, second.RefreshToken, "Refresh() did not rotate the refresh token")
	require.NotNil(t, newSaved, "Refresh() did not save the new refresh token")
	assert.Equal(t, hashToken(second.RefreshToken), *newSaved, "the new refresh token is saved by its hash")

	_, _, err = svc.Refresh(t.Context(), first.RefreshToken)
	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken, "Refresh() with a reused token")
}

func TestService_Refresh_UnknownTokenIsRejected(t *testing.T) {
	svc, _, store := newService(t)
	store.EXPECT().Consume(gomock.Any(), hashToken("never-issued")).Return(int64(0), domain.ErrInvalidRefreshToken)

	_, _, err := svc.Refresh(t.Context(), "never-issued")
	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken, "Refresh(%q)", "never-issued")
}

func TestService_Refresh_DeletedUserIsRejected(t *testing.T) {
	svc, users, store := newService(t)
	first := issueSession(t, svc, users, store)

	// No session may be issued for a missing user: there is no second Save.
	store.EXPECT().Consume(gomock.Any(), hashToken(first.RefreshToken)).Return(int64(7), nil)
	users.EXPECT().Fetch(gomock.Any(), int64(7)).Return(nil, domain.ErrUserNotFound)

	_, _, err := svc.Refresh(t.Context(), first.RefreshToken)
	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
}

func TestService_Refresh_RepositoryFailureIsReturned(t *testing.T) {
	svc, users, store := newService(t)
	first := issueSession(t, svc, users, store)

	store.EXPECT().Consume(gomock.Any(), hashToken(first.RefreshToken)).Return(int64(7), nil)
	users.EXPECT().Fetch(gomock.Any(), int64(7)).Return(nil, errBoom)

	_, _, err := svc.Refresh(t.Context(), first.RefreshToken)
	assert.ErrorIs(t, err, errBoom)
	assert.NotErrorIs(t, err, domain.ErrInvalidRefreshToken, "Refresh() reported a storage failure as an invalid refresh token")
}

func TestService_Logout_RevokesSession(t *testing.T) {
	svc, users, store := newService(t)
	first := issueSession(t, svc, users, store)

	hash := hashToken(first.RefreshToken)
	gomock.InOrder(
		store.EXPECT().Delete(gomock.Any(), hash).Return(nil),
		store.EXPECT().Consume(gomock.Any(), hash).Return(int64(0), domain.ErrInvalidRefreshToken),
	)

	require.NoError(t, svc.Logout(t.Context(), first.RefreshToken))

	_, _, err := svc.Refresh(t.Context(), first.RefreshToken)
	assert.ErrorIs(t, err, domain.ErrInvalidRefreshToken, "Refresh() after Logout")
}
