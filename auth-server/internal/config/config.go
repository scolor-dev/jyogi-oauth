package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        string
	Host        string
	DatabaseURL string
	RedisURL    string
	LogLevel    string

	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
	JWTIssuer         string
	JWTKID            string
	AccessTokenTTL    time.Duration

	RefreshTokenTTL    time.Duration
	RefreshTokenLength int

	SessionTTL          time.Duration
	SessionCookieName   string
	SessionCookieSecure bool
	SessionCookieDomain string

	CodeTTL    time.Duration
	CodeLength int

	Argon2Memory      uint32
	Argon2Iterations  uint32
	Argon2Parallelism uint8

	RateLimitLogin int
	RateLimitToken int
	RateLimitIP    int
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:        envOrDefault("AUTH_SERVER_PORT", "8080"),
		Host:        envOrDefault("AUTH_SERVER_HOST", "0.0.0.0"),
		DatabaseURL: os.Getenv("AUTH_DATABASE_URL"),
		RedisURL:    os.Getenv("AUTH_REDIS_URL"),
		LogLevel:    envOrDefault("AUTH_LOG_LEVEL", "info"),

		JWTPrivateKeyPath: os.Getenv("AUTH_JWT_PRIVATE_KEY_PATH"),
		JWTPublicKeyPath:  os.Getenv("AUTH_JWT_PUBLIC_KEY_PATH"),
		JWTIssuer:         envOrDefault("AUTH_JWT_ISSUER", "https://oauth.example.internal"),
		JWTKID:            envOrDefault("AUTH_JWT_KID", "key-1"),
		AccessTokenTTL:    time.Duration(envOrDefaultInt("AUTH_JWT_ACCESS_TOKEN_TTL", 900)) * time.Second,

		RefreshTokenTTL:    time.Duration(envOrDefaultInt("AUTH_REFRESH_TOKEN_TTL", 604800)) * time.Second,
		RefreshTokenLength: envOrDefaultInt("AUTH_REFRESH_TOKEN_LENGTH", 64),

		SessionTTL:          time.Duration(envOrDefaultInt("AUTH_SESSION_TTL", 86400)) * time.Second,
		SessionCookieName:   envOrDefault("AUTH_SESSION_COOKIE_NAME", "jyogi_sid"),
		SessionCookieSecure: envOrDefault("AUTH_SESSION_COOKIE_SECURE", "true") == "true",
		SessionCookieDomain: os.Getenv("AUTH_SESSION_COOKIE_DOMAIN"),

		CodeTTL:    time.Duration(envOrDefaultInt("AUTH_CODE_TTL", 600)) * time.Second,
		CodeLength: envOrDefaultInt("AUTH_CODE_LENGTH", 32),

		Argon2Memory:      uint32(envOrDefaultInt("AUTH_ARGON2_MEMORY", 65536)),
		Argon2Iterations:  uint32(envOrDefaultInt("AUTH_ARGON2_ITERATIONS", 3)),
		Argon2Parallelism: uint8(envOrDefaultInt("AUTH_ARGON2_PARALLELISM", 2)),

		RateLimitLogin: envOrDefaultInt("AUTH_RATE_LIMIT_LOGIN", 5),
		RateLimitToken: envOrDefaultInt("AUTH_RATE_LIMIT_TOKEN", 20),
		RateLimitIP:    envOrDefaultInt("AUTH_RATE_LIMIT_IP", 30),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("AUTH_DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("AUTH_REDIS_URL is required")
	}
	if cfg.JWTPrivateKeyPath == "" {
		return nil, fmt.Errorf("AUTH_JWT_PRIVATE_KEY_PATH is required")
	}
	if cfg.JWTPublicKeyPath == "" {
		return nil, fmt.Errorf("AUTH_JWT_PUBLIC_KEY_PATH is required")
	}

	return cfg, nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envOrDefaultInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
