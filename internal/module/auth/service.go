package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"

	"rantau-go/internal/config"
	"rantau-go/internal/module/users"
	"rantau-go/internal/pkg/jwt"
)

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	// Register checks the sign-up details, then saves the user with a hashed password.
	Register(ctx context.Context, u *users.User, password string) (*users.User, error)
	// Login checks the email and password, then issues a new pair of tokens.
	Login(ctx context.Context, email, password string) (*users.User, *Tokens, error)
	// Refresh trades a refresh token for the user and a new token pair.
	// The old refresh token stops working.
	Refresh(ctx context.Context, refreshToken string) (*users.User, *Tokens, error)
	// Logout revokes a refresh token so it cannot be used again.
	Logout(ctx context.Context, refreshToken string) error
}

type service struct {
	cfg   config.Config
	users users.Service
	store RefreshStore
}

// -----------------------------------------------------------------------------
// Exported
// -----------------------------------------------------------------------------

func NewService(cfg config.Config, userService users.Service, store RefreshStore) Service {
	return &service{
		cfg:   cfg,
		users: userService,
		store: store,
	}
}

func (s *service) Register(ctx context.Context, u *users.User, password string) (*users.User, error) {
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
	if !emailPattern.MatchString(u.Email) {
		return nil, ErrInvalidEmail
	}

	if utf8.RuneCountInString(password) < minPasswordLength {
		return nil, ErrWeakPassword
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = hash

	return s.users.Create(ctx, u)
}

func (s *service) Login(ctx context.Context, email, password string) (*users.User, *Tokens, error) {
	u, err := s.users.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	ok, err := verifyPassword(password, u.PasswordHash)
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

func (s *service) Refresh(ctx context.Context, refreshToken string) (*users.User, *Tokens, error) {
	userID, err := s.store.Consume(ctx, hashToken(refreshToken))
	if err != nil {
		return nil, nil, err
	}

	// Reload the user so the new access token reflects their current role.
	u, err := s.users.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
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

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	return s.store.Delete(ctx, hashToken(refreshToken))
}

// -----------------------------------------------------------------------------
// Unexported
// -----------------------------------------------------------------------------

// issueTokens makes a new access token and refresh token for u.
// The refresh token is saved by its hash, never as it is.
func (s *service) issueTokens(ctx context.Context, u *users.User) (*Tokens, error) {
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

// hashToken returns the SHA-256 of token in hex, so raw tokens are never stored.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// normalizeEmail trims spaces and lowercases email, so case does not matter when logging in.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
