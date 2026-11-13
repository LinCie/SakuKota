package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"

	"rantau-go/internal/config"
	"rantau-go/internal/module/user"
	"rantau-go/internal/pkg/jwt"
)

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	// Register checks the sign-up details, then saves the user with a hashed password.
	Register(ctx context.Context, u *user.User, password string) (*user.User, error)
	// Login checks the email and password, then issues a new pair of tokens.
	Login(ctx context.Context, email, password string) (*user.User, *Tokens, error)
	// Refresh trades a refresh token for the user and a new token pair.
	// The old refresh token stops working.
	Refresh(ctx context.Context, refreshToken string) (*user.User, *Tokens, error)
	// Logout revokes a refresh token so it cannot be used again.
	Logout(ctx context.Context, refreshToken string) error
}

type service struct {
	cfg   config.Config
	users user.Service
	store RefreshStore
	// hashSlots holds one token per running password hash.
	hashSlots chan struct{}
}

// NewService builds the auth service from the config, the user service, and the refresh store.
func NewService(cfg config.Config, userService user.Service, store RefreshStore) Service {
	return &service{
		cfg:       cfg,
		users:     userService,
		store:     store,
		hashSlots: make(chan struct{}, maxConcurrentHashes),
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

// Register checks the name, email, and password, then saves the user with a hashed password.
func (s *service) Register(ctx context.Context, u *user.User, password string) (*user.User, error) {
	u.Name = strings.TrimSpace(u.Name)
	switch {
	case u.Name == "":
		return nil, ErrInvalidName
	case utf8.RuneCountInString(u.Name) > maxNameLength:
		return nil, ErrNameTooLong
	}

	u.Email = normalizeEmail(u.Email)
	if utf8.RuneCountInString(u.Email) > maxEmailLength {
		return nil, ErrEmailTooLong
	}
	// The address must be the bare address: a form such as "Ana <ana@x.com>"
	// parses too, but is not an email address.
	if addr, err := mail.ParseAddress(u.Email); err != nil || addr.Address != u.Email {
		return nil, ErrInvalidEmail
	}

	switch n := utf8.RuneCountInString(password); {
	case n < minPasswordLength:
		return nil, ErrWeakPassword
	case n > maxPasswordLength:
		return nil, ErrPasswordTooLong
	}

	hash, err := s.hash(ctx, password)
	if err != nil {
		return nil, err
	}

	u.PasswordHash = hash

	return s.users.Create(ctx, u)
}

// Login checks the email and password, then issues a new pair of tokens for the user.
func (s *service) Login(ctx context.Context, email, password string) (*user.User, *Tokens, error) {
	// No stored password is this long, so there is nothing to verify.
	if utf8.RuneCountInString(password) > maxPasswordLength {
		return nil, nil, ErrInvalidCredentials
	}

	u, err := s.users.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	ok, err := s.verify(ctx, password, u.PasswordHash)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, nil, err
	}

	return u, tokens, nil
}

// Refresh exchanges a valid refresh token for the user and a new token pair.
// The old refresh token is consumed, so it cannot be reused.
func (s *service) Refresh(ctx context.Context, refreshToken string) (*user.User, *Tokens, error) {
	userID, err := s.store.Consume(ctx, hashToken(refreshToken))
	if err != nil {
		return nil, nil, err
	}

	// Reload the user so the new access token reflects their current role.
	u, err := s.users.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, nil, ErrInvalidRefreshToken
		}
		return nil, nil, err
	}

	tokens, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, nil, err
	}

	return u, tokens, nil
}

// Logout deletes the refresh token, so it cannot be used again.
func (s *service) Logout(ctx context.Context, refreshToken string) error {
	return s.store.Delete(ctx, hashToken(refreshToken))
}

// issueTokens makes a new access token and refresh token for u.
// The refresh token is saved by its hash, never as it is.
func (s *service) issueTokens(ctx context.Context, u *user.User) (*Tokens, error) {
	accessToken, err := jwt.GenerateToken(s.cfg, u.ID, u.Role)
	if err != nil {
		return nil, err
	}

	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	refreshToken := base64.RawURLEncoding.EncodeToString(b)

	if err := s.store.Save(ctx, hashToken(refreshToken), u.ID, s.cfg.RefreshTokenExpiry); err != nil {
		return nil, err
	}

	return &Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// hash turns password into an argon2id hash, waiting for a free hash slot first.
func (s *service) hash(ctx context.Context, password string) (string, error) {
	release, err := s.acquireHashSlot(ctx)
	if err != nil {
		return "", err
	}
	defer release()

	return hashPassword(password)
}

// verify checks password against encodedHash, waiting for a free hash slot first.
func (s *service) verify(ctx context.Context, password, encodedHash string) (bool, error) {
	release, err := s.acquireHashSlot(ctx)
	if err != nil {
		return false, err
	}
	defer release()

	return verifyPassword(password, encodedHash)
}

// acquireHashSlot waits for a free slot, or for ctx to end.
// It returns the function that gives the slot back.
func (s *service) acquireHashSlot(ctx context.Context) (release func(), err error) {
	select {
	case s.hashSlots <- struct{}{}:
		return func() { <-s.hashSlots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// hashToken returns the SHA-256 of token in hex, so raw tokens are never stored.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// normalizeEmail trims spaces and lowercases email, so case does not matter when logging in.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
