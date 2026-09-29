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
	"time"

	"rantau-go/internal/config"
	"rantau-go/internal/jwt"
)

var (
	ErrNotFound            = errors.New("user not found")
	ErrEmailTaken          = errors.New("email already registered")
	ErrInvalidName         = errors.New("name is required")
	ErrInvalidEmail        = errors.New("invalid email")
	ErrWeakPassword        = errors.New("password must be at least 8 characters")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

const refreshTokenBytes = 32

// Repository defines the persistence operations required by the service.
type Repository interface {
	Create(ctx context.Context, u *User) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
}

// PasswordHasher defines the password hashing operations required by the service.
type PasswordHasher interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password, encodedHash string) (bool, error)
}

// RefreshStore defines the refresh token storage operations required by the
// service. Tokens are stored by their hash, never in raw form.
type RefreshStore interface {
	Save(ctx context.Context, tokenHash string, userID int64, ttl time.Duration) error
	// Consume returns the token's user ID and removes the token. It returns
	// ErrInvalidRefreshToken if the token is unknown or expired.
	Consume(ctx context.Context, tokenHash string) (int64, error)
	Delete(ctx context.Context, tokenHash string) error
}

type service struct {
	cfg    config.Config
	repo   Repository
	hasher PasswordHasher
	store  RefreshStore
}

func NewService(cfg config.Config, repo Repository, hasher PasswordHasher, store RefreshStore) Service {
	return &service{
		cfg:    cfg,
		repo:   repo,
		hasher: hasher,
		store:  store,
	}
}

// -----------------------------------------------------------------------------
// service
// -----------------------------------------------------------------------------

func (s *service) Register(ctx context.Context, u *User, password string) (*User, error) {
	u.Name = strings.TrimSpace(u.Name)
	if u.Name == "" {
		return nil, ErrInvalidName
	}

	u.Email = normalizeEmail(u.Email)
	if _, err := mail.ParseAddress(u.Email); err != nil {
		return nil, ErrInvalidEmail
	}

	if len(password) < minPasswordLength {
		return nil, ErrWeakPassword
	}

	hash, err := s.hasher.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u.PasswordHash = hash

	return s.repo.Create(ctx, u)
}

func (s *service) Login(ctx context.Context, email, password string) (*User, *Tokens, error) {
	u, err := s.repo.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	ok, err := s.hasher.VerifyPassword(password, u.PasswordHash)
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

// Refresh exchanges a valid refresh token for a new token pair. The old
// refresh token is consumed, so it cannot be reused.
func (s *service) Refresh(ctx context.Context, refreshToken string) (*Tokens, error) {
	userID, err := s.store.Consume(ctx, hashToken(refreshToken))
	if err != nil {
		return nil, err
	}

	// Reload the user so the new access token reflects their current role.
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	return s.issueTokens(ctx, u)
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	return s.store.Delete(ctx, hashToken(refreshToken))
}

func (s *service) issueTokens(ctx context.Context, u *User) (*Tokens, error) {
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
