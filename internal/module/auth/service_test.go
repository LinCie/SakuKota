package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/config"
	"rantau-go/internal/module/auth"
	"rantau-go/internal/module/users"
	usermock "rantau-go/internal/module/users/mock"
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

// newService uses the real password hashing and refresh store; only the
// user service is mocked.
func newService(t *testing.T) (auth.Service, *usermock.MockService) {
	t.Helper()
	userSvc := usermock.NewMockService(gomock.NewController(t))
	return auth.NewService(testConfig(), userSvc, auth.NewMemoryRefreshStore()), userSvc
}

// issueSession logs in testEmail and returns the issued tokens.
func issueSession(t *testing.T, svc auth.Service, userSvc *usermock.MockService) *auth.Tokens {
	t.Helper()

	userSvc.EXPECT().GetByEmail(gomock.Any(), testEmail).
		Return(&users.User{ID: 7, Email: testEmail, PasswordHash: testPasswordHash}, nil)

	_, tokens, err := svc.Login(context.Background(), testEmail, testPassword)
	require.NoError(t, err, "login")
	return tokens
}

// -----------------------------------------------------------------------------
// Register
// -----------------------------------------------------------------------------

func TestService_Register_NormalizesAndStoresHashedUser(t *testing.T) {
	svc, userSvc := newService(t)

	var stored *users.User
	userSvc.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, u *users.User) (*users.User, error) {
			stored = u
			return &users.User{ID: 1, Name: u.Name, Email: u.Email, PasswordHash: u.PasswordHash}, nil
		})

	u, err := svc.Register(context.Background(), &users.User{Name: "  Ana  ", Email: "  ANA@Example.COM "}, testPassword)
	require.NoError(t, err, "register")
	assert.Equal(t, int64(1), u.ID)
	assert.Equal(t, "Ana", stored.Name)
	assert.Equal(t, testEmail, stored.Email)
	assert.True(t, strings.HasPrefix(stored.PasswordHash, "$argon2id$"),
		"stored password is not an argon2id hash: %q", stored.PasswordHash)
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
		{name: "4 multibyte characters are fewer than 8 characters", userName: "Ana", email: testEmail, password: strings.Repeat("é", 4), want: auth.ErrWeakPassword},
		{name: "129-character password", userName: "Ana", email: testEmail, password: strings.Repeat("a", 129), want: auth.ErrPasswordTooLong},
		{name: "101-character name", userName: strings.Repeat("a", 101), email: testEmail, password: "password1", want: auth.ErrNameTooLong},
		{name: "101 multibyte-character name", userName: strings.Repeat("é", 101), email: testEmail, password: "password1", want: auth.ErrNameTooLong},
		{name: "email over 255 characters", userName: "Ana", email: strings.Repeat("a", 245) + "@example.com", password: "password1", want: auth.ErrEmailTooLong},
		{name: "display name is not a bare address", userName: "Ana", email: "Ana <ana@example.com>", password: "password1", want: auth.ErrInvalidEmail},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: invalid input must never reach storage.
			svc, _ := newService(t)

			_, err := svc.Register(context.Background(), &users.User{Name: tc.userName, Email: tc.email}, tc.password)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestService_Register_AcceptsEightCharacterPassword(t *testing.T) {
	svc, userSvc := newService(t)

	userSvc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&users.User{ID: 1}, nil)

	_, err := svc.Register(context.Background(), &users.User{Name: "Ana", Email: testEmail}, "12345678")
	assert.NoError(t, err, "register")
}

func TestService_Register_PasswordLengthLimitsCountCharacters(t *testing.T) {
	for _, password := range []string{strings.Repeat("a", 128), strings.Repeat("é", 8), strings.Repeat("é", 128)} {
		svc, userSvc := newService(t)

		userSvc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&users.User{ID: 1}, nil)

		_, err := svc.Register(context.Background(), &users.User{Name: "Ana", Email: testEmail}, password)
		assert.NoError(t, err, "register with %d bytes", len(password))
	}
}

func TestService_Login_PasswordOverMaximumIsInvalidCredentials(t *testing.T) {
	// No expectations: the password is refused before storage is read.
	svc, _ := newService(t)

	_, _, err := svc.Login(context.Background(), testEmail, strings.Repeat("a", 129))
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestService_Register_AcceptsHundredCharacterName(t *testing.T) {
	svc, userSvc := newService(t)

	userSvc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&users.User{ID: 1}, nil)

	_, err := svc.Register(context.Background(), &users.User{Name: strings.Repeat("a", 100), Email: testEmail}, "password1")
	assert.NoError(t, err, "register")
}

func TestService_Register_DuplicateEmailIsReturned(t *testing.T) {
	svc, userSvc := newService(t)

	userSvc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, users.ErrEmailTaken)

	_, err := svc.Register(context.Background(), &users.User{Name: "Ana", Email: testEmail}, testPassword)
	assert.ErrorIs(t, err, users.ErrEmailTaken)
}

// -----------------------------------------------------------------------------
// Login
// -----------------------------------------------------------------------------

func TestService_Login_NormalizesEmailAndIssuesTokens(t *testing.T) {
	svc, userSvc := newService(t)

	userSvc.EXPECT().GetByEmail(gomock.Any(), testEmail).
		Return(&users.User{ID: 7, Email: testEmail, PasswordHash: testPasswordHash}, nil)

	u, tokens, err := svc.Login(context.Background(), "  ANA@Example.com ", testPassword)
	require.NoError(t, err, "login")
	assert.Equal(t, int64(7), u.ID)
	assert.NotEmpty(t, tokens.AccessToken, "login must return an access token")
	assert.NotEmpty(t, tokens.RefreshToken, "login must return a refresh token")
}

func TestService_Login_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*usermock.MockService)
		password string
	}{
		{
			name: "unknown email",
			setup: func(userSvc *usermock.MockService) {
				userSvc.EXPECT().GetByEmail(gomock.Any(), testEmail).Return(nil, users.ErrNotFound)
			},
			password: testPassword,
		},
		{
			name: "wrong password",
			setup: func(userSvc *usermock.MockService) {
				userSvc.EXPECT().GetByEmail(gomock.Any(), testEmail).
					Return(&users.User{ID: 7, Email: testEmail, PasswordHash: testPasswordHash}, nil)
			},
			password: "wrong-horse",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, userSvc := newService(t)
			tc.setup(userSvc)

			u, tokens, err := svc.Login(context.Background(), testEmail, tc.password)
			assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
			assert.Nil(t, u, "failed login returned a user")
			assert.Nil(t, tokens, "failed login returned tokens")
		})
	}
}

func TestService_Login_RepositoryFailureIsNotBadCredentials(t *testing.T) {
	svc, userSvc := newService(t)

	userSvc.EXPECT().GetByEmail(gomock.Any(), testEmail).Return(nil, errBoom)

	_, _, err := svc.Login(context.Background(), testEmail, testPassword)
	assert.ErrorIs(t, err, errBoom)
	assert.NotErrorIs(t, err, auth.ErrInvalidCredentials, "storage failure was reported as invalid credentials")
}

func TestService_Login_CorruptHashIsNotBadCredentials(t *testing.T) {
	svc, userSvc := newService(t)

	userSvc.EXPECT().GetByEmail(gomock.Any(), testEmail).
		Return(&users.User{ID: 7, Email: testEmail, PasswordHash: "corrupt"}, nil)

	_, _, err := svc.Login(context.Background(), testEmail, testPassword)
	require.Error(t, err, "login with a corrupt stored hash succeeded")
	assert.NotErrorIs(t, err, auth.ErrInvalidCredentials, "corrupt stored hash was reported as invalid credentials")
}

// -----------------------------------------------------------------------------
// Refresh
// -----------------------------------------------------------------------------

func TestService_Refresh_ConsumesOldSessionAndIssuesNewTokens(t *testing.T) {
	svc, userSvc := newService(t)
	first := issueSession(t, svc, userSvc)

	userSvc.EXPECT().GetUser(gomock.Any(), int64(7)).
		Return(&users.User{ID: 7, Email: testEmail, PasswordHash: testPasswordHash}, nil)

	u, second, err := svc.Refresh(context.Background(), first.RefreshToken)
	require.NoError(t, err, "refresh")
	assert.Equal(t, int64(7), u.ID)
	assert.NotEmpty(t, second.AccessToken, "refresh did not return an access token")
	assert.NotEqual(t, first.RefreshToken, second.RefreshToken, "refresh token was not rotated")

	// The old refresh token was consumed, so reusing it must fail.
	_, _, err = svc.Refresh(context.Background(), first.RefreshToken)
	assert.ErrorIs(t, err, auth.ErrInvalidRefreshToken, "reused refresh token")
}

func TestService_Refresh_UnknownTokenIsRejected(t *testing.T) {
	svc, _ := newService(t)

	_, _, err := svc.Refresh(context.Background(), "never-issued")
	assert.ErrorIs(t, err, auth.ErrInvalidRefreshToken)
}

func TestService_Refresh_DeletedUserIsRejected(t *testing.T) {
	svc, userSvc := newService(t)
	first := issueSession(t, svc, userSvc)

	userSvc.EXPECT().GetUser(gomock.Any(), int64(7)).Return(nil, users.ErrNotFound)

	// No session may be issued for a missing user.
	_, _, err := svc.Refresh(context.Background(), first.RefreshToken)
	assert.ErrorIs(t, err, auth.ErrInvalidRefreshToken)
}

func TestService_Refresh_RepositoryFailureIsReturned(t *testing.T) {
	svc, userSvc := newService(t)
	first := issueSession(t, svc, userSvc)

	userSvc.EXPECT().GetUser(gomock.Any(), int64(7)).Return(nil, errBoom)

	_, _, err := svc.Refresh(context.Background(), first.RefreshToken)
	assert.ErrorIs(t, err, errBoom)
	assert.NotErrorIs(t, err, auth.ErrInvalidRefreshToken, "storage failure was reported as an invalid refresh token")
}

// -----------------------------------------------------------------------------
// Logout
// -----------------------------------------------------------------------------

func TestService_Logout_RevokesSession(t *testing.T) {
	svc, userSvc := newService(t)
	first := issueSession(t, svc, userSvc)

	require.NoError(t, svc.Logout(context.Background(), first.RefreshToken), "logout")

	_, _, err := svc.Refresh(context.Background(), first.RefreshToken)
	assert.ErrorIs(t, err, auth.ErrInvalidRefreshToken, "refresh after logout")
}
