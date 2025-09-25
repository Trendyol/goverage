//go:build goverage

package goverage

import (
	"os"
	"time"
)

const (
	defaultPort      = "7777"
	coverageEndpoint = "/v1/cover/profile"

	portEnvVar     = "COVERAGE_HTTP_PORT"
	coverDirEnvVar = "GOCOVERDIR"

	temporaryCoverageFile = "/tmp/coverage.out"
	coverageFileMode      = 0o755

	requestTimeout    = 60 * time.Second
	readTimeout       = 15 * time.Second
	readHeaderTimeout = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second

	contentTypeTextPlain = "text/plain; charset=utf-8"
	modePrefix           = "mode:"
)

type Config struct {
	Port     string
	CoverDir string
}

func NewConfig() *Config {
	return &Config{
		Port:     getEnvOrDefault(portEnvVar, defaultPort),
		CoverDir: os.Getenv(coverDirEnvVar),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
