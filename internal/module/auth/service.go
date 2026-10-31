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
	"rantau-go/internal/jwt"
	"rantau-go/internal/module/users"
)

// Service defines the operations required by the handler.
//
//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock
type Service interface {
	Register(ctx context.Context, u *users.User, password string) (*users.User, error)
	Login(ctx context.Context, email, password string) (*users.User, *Tokens, error)
	Refresh(ctx context.Context, refreshToken string) (*users.User, *Tokens, error)
	Logout(ctx context.Context, refreshToken string) error
}

type service struct {
	cfg   config.Config
	users users.Service
	store RefreshStore
}

func NewService(cfg config.Config, userService users.Service, store RefreshStore) Service {
	return &service{
		cfg:   cfg,
		users: userService,
		store: store,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

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
	// The address must be the bare address: a form such as "Ana <ana@x.com>"
	// parses too, but is not an email address.
	if addr, err := mail.ParseAddress(u.Email); err != nil || addr.Address != u.Email {
		return nil, ErrInvalidEmail
	}

	if len(password) < minPasswordLength {
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

// Refresh exchanges a valid refresh token for the user and a new token pair.
// The old refresh token is consumed, so it cannot be reused.
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

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
