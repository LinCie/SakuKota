package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// minJWTSecretBytes is the least a JWT_SECRET may hold: HS256 wants a key as
// long as its 256-bit hash (RFC 7518, section 3.2).
const minJWTSecretBytes = 32

type Config struct {
	DatabaseURL        string
	RedisURL           string
	Port               int
	JWTSecret          string
	JWTExpiry          time.Duration
	RefreshTokenExpiry time.Duration
	CookieSecure       bool
	Storage            StorageConfig
}

// StorageConfig locates the S3-compatible bucket that holds uploaded images.
type StorageConfig struct {
	// Endpoint is the S3 API the server calls.
	Endpoint string
	// PublicEndpoint is the S3 API that clients upload to. Presigned URLs are
	// signed for this host.
	PublicEndpoint string
	// PublicURL is the base URL that objects are served from.
	PublicURL       string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

// DatabaseURL returns DATABASE_URL, which every command requires.
func DatabaseURL() (string, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return "", errors.New("DATABASE_URL is required")
	}

	return databaseURL, nil
}

// NewConfig reads the configuration of the API server.
func NewConfig() (Config, error) {
	databaseURL, err := DatabaseURL()
	if err != nil {
		return Config{}, err
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return Config{}, errors.New("REDIS_URL is required")
	}

	port := 8080
	if value := os.Getenv("PORT"); value != "" {
		parsedPort, err := strconv.Atoi(value)
		if err != nil || parsedPort <= 0 {
			return Config{}, errors.New("PORT must be a positive integer")
		}
		port = parsedPort
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	if len(jwtSecret) < minJWTSecretBytes {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least %d bytes (try: openssl rand -hex 32)", minJWTSecretBytes)
	}
	minutes := 5
	if v := os.Getenv("JWT_EXPIRES_MINUTES"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, errors.New("JWT_EXPIRES_MINUTES must be a positive integer")
		}
		minutes = n
	}
	jwtExpiry := time.Duration(minutes) * time.Minute

	days := 30
	if v := os.Getenv("REFRESH_TOKEN_EXPIRES_DAYS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, errors.New("REFRESH_TOKEN_EXPIRES_DAYS must be a positive integer")
		}
		days = n
	}
	refreshTokenExpiry := time.Duration(days) * 24 * time.Hour

	// Secure by default; set COOKIE_SECURE=false only for local development
	// over plain HTTP.
	cookieSecure := true
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, errors.New("COOKIE_SECURE must be a boolean")
		}
		cookieSecure = b
	}

	storage, err := NewStorageConfig()
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabaseURL:        databaseURL,
		RedisURL:           redisURL,
		Port:               port,
		JWTSecret:          jwtSecret,
		JWTExpiry:          jwtExpiry,
		RefreshTokenExpiry: refreshTokenExpiry,
		CookieSecure:       cookieSecure,
		Storage:            storage,
	}, nil
}

// NewStorageConfig reads the location of the image bucket.
func NewStorageConfig() (StorageConfig, error) {
	storage := StorageConfig{
		Endpoint:        os.Getenv("STORAGE_ENDPOINT"),
		PublicEndpoint:  os.Getenv("STORAGE_PUBLIC_ENDPOINT"),
		PublicURL:       os.Getenv("STORAGE_PUBLIC_URL"),
		Region:          os.Getenv("STORAGE_REGION"),
		Bucket:          os.Getenv("STORAGE_BUCKET"),
		AccessKeyID:     os.Getenv("STORAGE_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("STORAGE_SECRET_ACCESS_KEY"),
	}

	if storage.PublicEndpoint == "" {
		storage.PublicEndpoint = storage.Endpoint
	}
	if storage.Region == "" {
		storage.Region = "garage"
	}

	required := []struct{ name, value string }{
		{"STORAGE_ENDPOINT", storage.Endpoint},
		{"STORAGE_PUBLIC_URL", storage.PublicURL},
		{"STORAGE_BUCKET", storage.Bucket},
		{"STORAGE_ACCESS_KEY_ID", storage.AccessKeyID},
		{"STORAGE_SECRET_ACCESS_KEY", storage.SecretAccessKey},
	}
	for _, v := range required {
		if v.value == "" {
			return StorageConfig{}, errors.New(v.name + " is required")
		}
	}

	return storage, nil
}
