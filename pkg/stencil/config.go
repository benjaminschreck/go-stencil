package stencil

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config contains all configuration options for the Stencil engine
type Config struct {
	// CacheMaxSize is the maximum number of templates to cache. 0 disables caching.
	CacheMaxSize int
	// CacheTTL is the time-to-live for cached templates. 0 means no expiration.
	CacheTTL time.Duration
	// LogLevel controls the verbosity of logging (debug, info, warn, error)
	LogLevel string
	// MaxRenderDepth controls the maximum depth of nested template includes/fragments
	MaxRenderDepth int
	// StrictMode enables strict template validation and error handling
	StrictMode bool
}

var (
	globalConfig      *Config
	globalConfigMutex sync.RWMutex
	configOnce        sync.Once
)

func init() {
	// Initialize global config from environment on first use
	configOnce.Do(func() {
		globalConfig = ConfigFromEnvironment()
	})
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		CacheMaxSize:   100,
		CacheTTL:       0,
		LogLevel:       "info",
		MaxRenderDepth: 100,
		StrictMode:     false,
	}
}

// ConfigFromEnvironment creates a configuration from environment variables
func ConfigFromEnvironment() *Config {
	config := DefaultConfig()

	// STENCIL_CACHE_MAX_SIZE
	if val := os.Getenv("STENCIL_CACHE_MAX_SIZE"); val != "" {
		if size, err := strconv.Atoi(val); err == nil {
			config.CacheMaxSize = size
		}
	}

	// STENCIL_CACHE_TTL
	if val := os.Getenv("STENCIL_CACHE_TTL"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.CacheTTL = duration
		}
	}

	// STENCIL_LOG_LEVEL
	if val := os.Getenv("STENCIL_LOG_LEVEL"); val != "" {
		config.LogLevel = val
	}

	// STENCIL_MAX_RENDER_DEPTH
	if val := os.Getenv("STENCIL_MAX_RENDER_DEPTH"); val != "" {
		if depth, err := strconv.Atoi(val); err == nil {
			config.MaxRenderDepth = depth
		}
	}

	// STENCIL_STRICT_MODE
	if val := os.Getenv("STENCIL_STRICT_MODE"); val != "" {
		config.StrictMode = parseBool(val)
	}

	return config
}

// NewConfigWithDefaults creates a new configuration with defaults applied to unset fields
func NewConfigWithDefaults(overrides *Config) *Config {
	defaults := DefaultConfig()
	
	if overrides == nil {
		return defaults
	}

	// Create a copy of the overrides
	config := *overrides

	// Apply defaults for zero values
	if config.CacheMaxSize == 0 && overrides.CacheMaxSize == 0 {
		config.CacheMaxSize = defaults.CacheMaxSize
	}
	
	if config.LogLevel == "" {
		config.LogLevel = defaults.LogLevel
	}
	
	if config.MaxRenderDepth == 0 {
		config.MaxRenderDepth = defaults.MaxRenderDepth
	}

	return &config
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.CacheMaxSize < 0 {
		return errors.New("cache max size cannot be negative")
	}

	if c.CacheTTL < 0 {
		return errors.New("cache TTL cannot be negative")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	
	if !validLogLevels[c.LogLevel] {
		return errors.New("invalid log level: " + c.LogLevel)
	}

	if c.MaxRenderDepth <= 0 {
		return errors.New("max render depth must be positive")
	}

	return nil
}

// GetGlobalConfig returns the global configuration
func GetGlobalConfig() *Config {
	globalConfigMutex.RLock()
	defer globalConfigMutex.RUnlock()

	if globalConfig == nil {
		return DefaultConfig()
	}
	
	// Return a copy to prevent modification
	configCopy := *globalConfig
	return &configCopy
}

// SetGlobalConfig sets the global configuration
func SetGlobalConfig(config *Config) {
	globalConfigMutex.Lock()
	globalConfig = config
	globalConfigMutex.Unlock()
	
	// Update logger based on new config (outside the lock to avoid deadlock)
	UpdateLoggerFromConfig()
}

// parseBool parses a boolean value from a string
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}