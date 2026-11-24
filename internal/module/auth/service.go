package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/sync/semaphore"

	"sakukota-go/internal/config"
	"sakukota-go/internal/domain"
	"sakukota-go/internal/pkg/jwt"
)

//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock

// UserService is the user operations that authentication needs.
type UserService interface {
	// Create saves a new user. It returns domain.ErrEmailTaken if the email is registered.
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	// FetchByEmail returns the user with the email, or domain.ErrUserNotFound.
	FetchByEmail(ctx context.Context, email string) (*domain.User, error)
	// Fetch returns the user with the id, or domain.ErrUserNotFound.
	Fetch(ctx context.Context, id int64) (*domain.User, error)
}

// RefreshStore keeps refresh tokens by their hash.
type RefreshStore interface {
	// Save stores the token for the user until ttl has passed.
	Save(ctx context.Context, tokenHash string, userID int64, ttl time.Duration) error
	// Consume removes the token and returns its user. It returns
	// domain.ErrInvalidRefreshToken if the token is unknown or expired.
	Consume(ctx context.Context, tokenHash string) (int64, error)
	// Delete removes the token if it is stored.
	Delete(ctx context.Context, tokenHash string) error
}

// Tokens is an access token and refresh token pair issued to a domain.User.
type Tokens struct {
	AccessToken  string
	RefreshToken string
}

// service holds the business rules of registration and sessions.
type service struct {
	cfg   config.Config
	users UserService
	store RefreshStore
	// hashes bounds concurrent argon2id hashes, each holding argonMemory and a core.
	hashes *semaphore.Weighted
}

// NewService returns a service that issues tokens signed with cfg, loads users
// from userService, and keeps refresh tokens in store.
func NewService(cfg config.Config, userService UserService, store RefreshStore) Service {
	return &service{
		cfg:   cfg,
		users: userService,
		store: store,
		// Half the cores, leaving the rest for other requests.
		hashes: semaphore.NewWeighted(int64(max(1, runtime.GOMAXPROCS(0)/2))),
	}
}

// Register checks the sign-up details, then saves the user with a hashed password.
// Hashing waits its turn among concurrent hashes, and returns ctx's error if ctx ends first.
func (s *service) Register(ctx context.Context, u *domain.User, password string) (*domain.User, error) {
	name, err := domain.NormalizeUserName(u.Name)
	if err != nil {
		return nil, err
	}
	u.Name = name

	u.Email = normalizeEmail(u.Email)
	if utf8.RuneCountInString(u.Email) > domain.MaxEmailLength {
		return nil, domain.ErrEmailTooLong
	}
	if !emailPattern.MatchString(u.Email) {
		return nil, domain.ErrEmailInvalid
	}

	if utf8.RuneCountInString(password) < domain.MinPasswordLength {
		return nil, domain.ErrPasswordTooShort
	}

	if err := s.hashes.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	hash, err := hashPassword(password)
	s.hashes.Release(1)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = hash

	return s.users.Create(ctx, u)
}

// Login checks the email and password, then issues a new pair of tokens.
// Checking the password waits its turn to hash, like Register.
func (s *service) Login(ctx context.Context, email, password string) (*domain.User, *Tokens, error) {
	u, err := s.users.FetchByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, nil, domain.ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if err := s.hashes.Acquire(ctx, 1); err != nil {
		return nil, nil, err
	}
	ok, err := verifyPassword(password, u.PasswordHash)
	s.hashes.Release(1)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, domain.ErrInvalidCredentials
	}

	tokens, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, nil, err
	}

	return u, tokens, nil
}

// Refresh trades a refresh token for the user and a new token pair.
// The old refresh token stops working.
func (s *service) Refresh(ctx context.Context, refreshToken string) (*domain.User, *Tokens, error) {
	userID, err := s.store.Consume(ctx, hashToken(refreshToken))
	if err != nil {
		return nil, nil, err
	}

	// Reload the user so the new access token reflects their current role.
	u, err := s.users.Fetch(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, nil, domain.ErrInvalidRefreshToken
		}
		return nil, nil, err
	}

	tokens, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, nil, err
	}

	return u, tokens, nil
}

// Logout revokes a refresh token so it cannot be used again.
func (s *service) Logout(ctx context.Context, refreshToken string) error {
	return s.store.Delete(ctx, hashToken(refreshToken))
}

// issueTokens makes a new access token and refresh token for u.
// The refresh token is saved by its hash, never as it is.
func (s *service) issueTokens(ctx context.Context, u *domain.User) (*Tokens, error) {
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
