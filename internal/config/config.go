package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL        string
	Port               int
	JWTSecret          string
	JWTExpiry          time.Duration
	RefreshTokenExpiry time.Duration
	CookieSecure       bool
	CORSAllowedOrigin  string
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

func NewConfig() (Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	port := 8080
	if value := os.Getenv("PORT"); value != "" {
		parsedPort, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, err
		}
		port = parsedPort
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	minutes := 5
	if v := os.Getenv("JWT_EXPIRES_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			minutes = n
		}
	}
	jwtExpiry := time.Duration(minutes) * time.Minute

	days := 30
	if v := os.Getenv("REFRESH_TOKEN_EXPIRES_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			days = n
		}
	}
	refreshTokenExpiry := time.Duration(days) * 24 * time.Hour

	// Secure by default; set COOKIE_SECURE=false only for local development
	// over plain HTTP.
	cookieSecure := true
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cookieSecure = b
		}
	}

	corsAllowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if corsAllowedOrigin == "" {
		corsAllowedOrigin = "http://localhost:5173"
	}

	storage, err := newStorageConfig()
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabaseURL:        databaseURL,
		Port:               port,
		JWTSecret:          jwtSecret,
		JWTExpiry:          jwtExpiry,
		RefreshTokenExpiry: refreshTokenExpiry,
		CookieSecure:       cookieSecure,
		CORSAllowedOrigin:  corsAllowedOrigin,
		Storage:            storage,
	}, nil
}

func newStorageConfig() (StorageConfig, error) {
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
