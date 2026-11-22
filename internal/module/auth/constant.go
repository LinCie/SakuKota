package auth

import "regexp"

const (
	refreshCookieName = "refresh_token"
	// refreshCookiePath scopes the cookie to the auth routes, the only ones
	// that read it. A proxy serving the API under a prefix must rewrite it
	// (see web/vite.config.ts).
	refreshCookiePath = "/v1/auth"
)

// emailPattern accepts one @ with text on both sides and no spaces.
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)

// refreshKeyPrefix namespaces refresh token keys in Redis.
const refreshKeyPrefix = "auth:refresh:"

// refreshTokenBytes is the number of random bytes in a refresh token.
const refreshTokenBytes = 32

// Argon2id parameters, following the OWASP recommendation.
const (
	argonTime    uint32 = 2
	argonMemory  uint32 = 19 * 1024
	argonThreads uint8  = 1
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)
