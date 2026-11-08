package jwt

import (
	"errors"
	"fmt"
	"time"

	"rantau-go/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

// issuer names this service in the tokens it signs, so a token signed with the
// same secret by something else is not accepted.
const issuer = "rantau-go"

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(cfg config.Config, userID int64, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   fmt.Sprint(userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.JWTExpiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func ValidateToken(cfg config.Config, t string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(t, claims, func(*jwt.Token) (any, error) {
		return []byte(cfg.JWTSecret), nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("token invalid")
	}

	return claims, nil
}
