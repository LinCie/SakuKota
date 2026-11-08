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
	"rantau-go/internal/module/users"
	"rantau-go/internal/pkg/jwt"
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
	// hashSlots holds one token per running password hash.
	hashSlots chan struct{}
}

func NewService(cfg config.Config, userService users.Service, store RefreshStore) Service {
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

func (s *service) Login(ctx context.Context, email, password string) (*users.User, *Tokens, error) {
	// No stored password is this long, so there is nothing to verify.
	if utf8.RuneCountInString(password) > maxPasswordLength {
		return nil, nil, ErrInvalidCredentials
	}

	u, err := s.users.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
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

// hash hashes password once a hash slot is free.
func (s *service) hash(ctx context.Context, password string) (string, error) {
	release, err := s.acquireHashSlot(ctx)
	if err != nil {
		return "", err
	}
	defer release()

	return hashPassword(password)
}

// verify checks password against encodedHash once a hash slot is free.
func (s *service) verify(ctx context.Context, password, encodedHash string) (bool, error) {
	release, err := s.acquireHashSlot(ctx)
	if err != nil {
		return false, err
	}
	defer release()

	return verifyPassword(password, encodedHash)
}

// acquireHashSlot waits for a free slot, or for ctx to end, and returns the
// function that frees the slot.
func (s *service) acquireHashSlot(ctx context.Context) (release func(), err error) {
	select {
	case s.hashSlots <- struct{}{}:
		return func() { <-s.hashSlots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
