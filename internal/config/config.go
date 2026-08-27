package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port            string
	Environment     string
	ShutdownTimeout time.Duration
	// TrustedProxies is the allowlist of proxy IPs/CIDRs Gin will honour
	// X-Forwarded-For from. Empty means "trust none" (the safe default behind
	// a single edge proxy like Railway's) - c.ClientIP() then returns the
	// immediate peer. Set it only if you need the real client IP in logs.
	TrustedProxies []string

	// Database
	DBHost string
	DBPort string
	DBName string
	DBUser string
	DBPass string
	// DBTLS maps to the go-sql-driver `tls` DSN param: "" / "false" (no TLS,
	// fine over Railway private networking), "true", "skip-verify", or
	// "preferred". Set "true" when connecting over a public MySQL proxy.
	DBTLS string
	// Connection pool sizing. Defaults suit a dedicated DB; lower
	// DBMaxOpenConns on managed plans with tight connection limits.
	DBMaxOpenConns int
	DBMaxIdleConns int

	// JWT
	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration

	// OAuth
	GoogleOAuthID     string
	GoogleOAuthSecret string
	OAuthCallbackURL  string

	// Midtrans
	MidtransServerKey string
	MidtransClientKey string
	MidtransIsSandbox bool

	// R2 (Cloudflare)
	R2Endpoint      string
	R2Bucket        string
	R2AccessKey     string
	R2SecretKey     string
	R2Region        string
	R2PublicBaseURL string

	// CORS - allowed origins for cross-site cookie-based auth (frontend + admin)
	CORSOrigins []string

	// Cookies - auth tokens are cross-site (Vercel <-> Railway), so cookies
	// need SameSite=None, which in turn mandates Secure. CookieSecure exists
	// mainly so local dev over plain http can opt out if needed.
	CookieSecure bool

	// Admin bootstrap - creates the very first admin on a fresh deploy, since
	// every admin-management route needs an existing admin session. Opt-in:
	// leave email+password empty to disable. No-op once that admin exists.
	AdminBootstrapEmail    string
	AdminBootstrapName     string
	AdminBootstrapPassword string
}

// LoadConfig loads configuration from .env (if present) and the environment,
// falling back to development defaults for anything unset.
func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	jwtAccessExpiry, err := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_EXPIRY: %w", err)
	}

	jwtRefreshExpiry, err := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRY: %w", err)
	}

	shutdownTimeout, err := time.ParseDuration(getEnv("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		return nil, fmt.Errorf("invalid SHUTDOWN_TIMEOUT: %w", err)
	}

	cfg := &Config{
		Port:            getEnv("PORT", "8000"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		ShutdownTimeout: shutdownTimeout,
		TrustedProxies:  splitAndTrim(getEnv("TRUSTED_PROXIES", "")),

		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "3306"),
		DBName:         getEnv("DB_NAME", "tixora"),
		DBUser:         getEnv("DB_USER", "root"),
		DBPass:         getEnv("DB_PASS", ""),
		DBTLS:          getEnv("DB_TLS", ""),
		DBMaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 100),
		DBMaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 10),

		JWTSecret:        getEnv("JWT_SECRET", "dev_secret_min_32_chars_long_change_me"),
		JWTAccessExpiry:  jwtAccessExpiry,
		JWTRefreshExpiry: jwtRefreshExpiry,

		GoogleOAuthID:     getEnv("GOOGLE_OAUTH_ID", ""),
		GoogleOAuthSecret: getEnv("GOOGLE_OAUTH_SECRET", ""),
		OAuthCallbackURL:  getEnv("OAUTH_CALLBACK_URL", "http://localhost:3000/api/auth/callback"),

		MidtransServerKey: getEnv("MIDTRANS_SERVER_KEY", ""),
		MidtransClientKey: getEnv("MIDTRANS_CLIENT_KEY", ""),
		MidtransIsSandbox: getEnvBool("MIDTRANS_IS_SANDBOX", true),

		R2Endpoint:      getEnv("R2_ENDPOINT", ""),
		R2Bucket:        getEnv("R2_BUCKET", "tixora"),
		R2AccessKey:     getEnv("R2_ACCESS_KEY", ""),
		R2SecretKey:     getEnv("R2_SECRET_KEY", ""),
		R2Region:        getEnv("R2_REGION", "auto"),
		R2PublicBaseURL: getEnv("R2_PUBLIC_BASE_URL", ""),

		CORSOrigins: splitAndTrim(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:3001")),

		CookieSecure: getEnvBool("COOKIE_SECURE", true),

		AdminBootstrapEmail:    getEnv("ADMIN_BOOTSTRAP_EMAIL", ""),
		AdminBootstrapName:     getEnv("ADMIN_BOOTSTRAP_NAME", "Administrator"),
		AdminBootstrapPassword: getEnv("ADMIN_BOOTSTRAP_PASSWORD", ""),
	}

	if cfg.Environment == "production" && cfg.JWTSecret == "dev_secret_min_32_chars_long_change_me" {
		return nil, fmt.Errorf("JWT_SECRET must be set in production")
	}

	return cfg, nil
}

// DatabaseDSN builds the MySQL connection string (DSN) for GORM.
func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local%s",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.DBName, c.tlsDSNParam(),
	)
}

// ServerDSN builds a MySQL connection string without a database name,
// used to connect to the server to create the database if it doesn't exist yet.
func (c *Config) ServerDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local%s",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.tlsDSNParam(),
	)
}

// tlsDSNParam returns the "&tls=..." fragment for the DSN, or "" when TLS is
// not configured. "false" is treated as unset since that's the driver default.
func (c *Config) tlsDSNParam() string {
	if c.DBTLS == "" || c.DBTLS == "false" {
		return ""
	}
	return "&tls=" + c.DBTLS
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	return value == "true" || value == "1"
}

// splitAndTrim splits a comma-separated list, trimming whitespace and
// dropping empty entries.
func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
