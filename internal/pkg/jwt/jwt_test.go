package jwt_test

import (
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/config"
	"rantau-go/internal/pkg/jwt"
)

const secret = "0123456789abcdef0123456789abcdef"

func testConfig() config.Config {
	return config.Config{JWTSecret: secret, JWTExpiry: 5 * time.Minute}
}

func TestValidateToken_AcceptsIssuedToken(t *testing.T) {
	token, err := jwt.GenerateToken(testConfig(), 7, "admin")
	require.NoError(t, err)

	claims, err := jwt.ValidateToken(testConfig(), token)
	require.NoError(t, err)
	assert.Equal(t, "7", claims.Subject)
	assert.Equal(t, "admin", claims.Role)
}

func TestValidateToken_RejectsTokensNotIssuedByThisService(t *testing.T) {
	valid := func() gojwt.RegisteredClaims {
		return gojwt.RegisteredClaims{
			Issuer:    "rantau-go",
			Subject:   "7",
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Minute)),
		}
	}
	sign := func(method gojwt.SigningMethod, key any, c gojwt.RegisteredClaims) string {
		s, err := gojwt.NewWithClaims(method, jwt.Claims{Role: "admin", RegisteredClaims: c}).SignedString(key)
		require.NoError(t, err, "sign token")
		return s
	}

	noIssuer := valid()
	noIssuer.Issuer = ""
	otherIssuer := valid()
	otherIssuer.Issuer = "someone-else"
	noExpiry := valid()
	noExpiry.ExpiresAt = nil
	expired := valid()
	expired.ExpiresAt = gojwt.NewNumericDate(time.Now().Add(-time.Minute))

	tests := []struct {
		name  string
		token string
	}{
		{name: "HS512", token: sign(gojwt.SigningMethodHS512, []byte(secret), valid())},
		{name: "AlgNone", token: sign(gojwt.SigningMethodNone, gojwt.UnsafeAllowNoneSignatureType, valid())},
		{name: "OtherSecret", token: sign(gojwt.SigningMethodHS256, []byte("another-secret-another-secret-00"), valid())},
		{name: "NoIssuer", token: sign(gojwt.SigningMethodHS256, []byte(secret), noIssuer)},
		{name: "OtherIssuer", token: sign(gojwt.SigningMethodHS256, []byte(secret), otherIssuer)},
		{name: "NoExpiry", token: sign(gojwt.SigningMethodHS256, []byte(secret), noExpiry)},
		{name: "Expired", token: sign(gojwt.SigningMethodHS256, []byte(secret), expired)},
		{name: "Garbage", token: "garbage"},
	}

	// The control: the same helper with valid claims is accepted.
	_, err := jwt.ValidateToken(testConfig(), sign(gojwt.SigningMethodHS256, []byte(secret), valid()))
	require.NoError(t, err, "control token")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := jwt.ValidateToken(testConfig(), tc.token)
			assert.Error(t, err)
		})
	}
}
