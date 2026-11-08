package auth

const (
	refreshCookieName = "refresh_token"
	// refreshCookiePath scopes the cookie to the auth routes, the only ones
	// that read it. A proxy serving the API under a prefix must rewrite it
	// (see web/vite.config.ts).
	refreshCookiePath = "/v1/auth"
)

// Lengths in characters, counted after trimming spaces. They match the
// VARCHAR columns of USERS.
const (
	maxNameLength  = 100
	maxEmailLength = 255
)

// Password lengths in characters. The cap is a sanity limit: it keeps absurd
// input away from the hash, and a login with a longer password cannot be right.
const (
	minPasswordLength = 8
	maxPasswordLength = 128
)

// maxConcurrentHashes bounds how many argon2id hashes run at once. Each one
// allocates argonMemory, so unbounded logins could exhaust memory.
const maxConcurrentHashes = 4

const refreshTokenBytes = 32

// Argon2id parameters, following the OWASP recommendation.
const (
	argonTime    uint32 = 2
	argonMemory  uint32 = 19 * 1024
	argonThreads uint8  = 1
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)
