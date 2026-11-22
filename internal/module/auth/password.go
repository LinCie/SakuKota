package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// errInvalidHash means a stored password hash is not a valid argon2id hash.
var errInvalidHash = errors.New("invalid argon2id hash")

// hashPassword turns password into an argon2id hash.
// The result is in PHC format: $argon2id$v=19$m=...,t=...,p=...$salt$hash
func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// verifyPassword reports whether password matches the encoded argon2id hash.
// shortcut: the m, t, p in the hash are not read back, the argon constants are
// used. Changing them means rehashing every stored password.
func verifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, errInvalidHash
	}

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, errInvalidHash
	}

	otherKey := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return subtle.ConstantTimeCompare(key, otherKey) == 1, nil
}
