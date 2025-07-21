package stencil

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.CacheMaxSize != 100 {
		t.Errorf("DefaultConfig CacheMaxSize = %d, want 100", config.CacheMaxSize)
	}

	if config.CacheTTL != 0 {
		t.Errorf("DefaultConfig CacheTTL = %v, want 0", config.CacheTTL)
	}

	if config.LogLevel != "info" {
		t.Errorf("DefaultConfig LogLevel = %s, want info", config.LogLevel)
	}

	if config.MaxRenderDepth != 100 {
		t.Errorf("DefaultConfig MaxRenderDepth = %d, want 100", config.MaxRenderDepth)
	}

	if config.StrictMode {
		t.Errorf("DefaultConfig StrictMode = true, want false")
	}
}

func TestConfigFromEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		check   func(t *testing.T, config *Config)
	}{
		{
			name: "cache max size",
			envVars: map[string]string{
				"STENCIL_CACHE_MAX_SIZE": "50",
			},
			check: func(t *testing.T, config *Config) {
				if config.CacheMaxSize != 50 {
					t.Errorf("CacheMaxSize = %d, want 50", config.CacheMaxSize)
				}
			},
		},
		{
			name: "cache TTL",
			envVars: map[string]string{
				"STENCIL_CACHE_TTL": "5m",
			},
			check: func(t *testing.T, config *Config) {
				if config.CacheTTL != 5*time.Minute {
					t.Errorf("CacheTTL = %v, want 5m", config.CacheTTL)
				}
			},
		},
		{
			name: "log level",
			envVars: map[string]string{
				"STENCIL_LOG_LEVEL": "debug",
			},
			check: func(t *testing.T, config *Config) {
				if config.LogLevel != "debug" {
					t.Errorf("LogLevel = %s, want debug", config.LogLevel)
				}
			},
		},
		{
			name: "max render depth",
			envVars: map[string]string{
				"STENCIL_MAX_RENDER_DEPTH": "200",
			},
			check: func(t *testing.T, config *Config) {
				if config.MaxRenderDepth != 200 {
					t.Errorf("MaxRenderDepth = %d, want 200", config.MaxRenderDepth)
				}
			},
		},
		{
			name: "strict mode",
			envVars: map[string]string{
				"STENCIL_STRICT_MODE": "true",
			},
			check: func(t *testing.T, config *Config) {
				if !config.StrictMode {
					t.Errorf("StrictMode = false, want true")
				}
			},
		},
		{
			name: "multiple environment variables",
			envVars: map[string]string{
				"STENCIL_CACHE_MAX_SIZE": "25",
				"STENCIL_LOG_LEVEL":      "error",
				"STENCIL_STRICT_MODE":    "true",
			},
			check: func(t *testing.T, config *Config) {
				if config.CacheMaxSize != 25 {
					t.Errorf("CacheMaxSize = %d, want 25", config.CacheMaxSize)
				}
				if config.LogLevel != "error" {
					t.Errorf("LogLevel = %s, want error", config.LogLevel)
				}
				if !config.StrictMode {
					t.Errorf("StrictMode = false, want true")
				}
			},
		},
		{
			name: "invalid cache max size",
			envVars: map[string]string{
				"STENCIL_CACHE_MAX_SIZE": "invalid",
			},
			check: func(t *testing.T, config *Config) {
				if config.CacheMaxSize != 100 {
					t.Errorf("CacheMaxSize = %d, want 100 (default)", config.CacheMaxSize)
				}
			},
		},
		{
			name: "invalid cache TTL",
			envVars: map[string]string{
				"STENCIL_CACHE_TTL": "invalid",
			},
			check: func(t *testing.T, config *Config) {
				if config.CacheTTL != 0 {
					t.Errorf("CacheTTL = %v, want 0 (default)", config.CacheTTL)
				}
			},
		},
		{
			name: "invalid max render depth",
			envVars: map[string]string{
				"STENCIL_MAX_RENDER_DEPTH": "not-a-number",
			},
			check: func(t *testing.T, config *Config) {
				if config.MaxRenderDepth != 100 {
					t.Errorf("MaxRenderDepth = %d, want 100 (default)", config.MaxRenderDepth)
				}
			},
		},
		{
			name: "empty strict mode",
			envVars: map[string]string{
				"STENCIL_STRICT_MODE": "",
			},
			check: func(t *testing.T, config *Config) {
				if config.StrictMode {
					t.Errorf("StrictMode = true, want false (default)")
				}
			},
		},
		{
			name: "case insensitive boolean",
			envVars: map[string]string{
				"STENCIL_STRICT_MODE": "TRUE",
			},
			check: func(t *testing.T, config *Config) {
				if !config.StrictMode {
					t.Errorf("StrictMode = false, want true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for key := range tt.envVars {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Get config
			config := ConfigFromEnvironment()

			// Run test-specific checks
			tt.check(t, config)

			// Clean up
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestNewConfigWithDefaults(t *testing.T) {
	overrides := &Config{
		CacheMaxSize: 200,
		LogLevel:     "debug",
	}

	config := NewConfigWithDefaults(overrides)

	if config.CacheMaxSize != 200 {
		t.Errorf("CacheMaxSize = %d, want 200", config.CacheMaxSize)
	}

	if config.LogLevel != "debug" {
		t.Errorf("LogLevel = %s, want debug", config.LogLevel)
	}

	// Check that defaults are applied for unset fields
	if config.MaxRenderDepth != 100 {
		t.Errorf("MaxRenderDepth = %d, want 100 (default)", config.MaxRenderDepth)
	}

	if config.StrictMode {
		t.Errorf("StrictMode = true, want false (default)")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		valid  bool
	}{
		{
			name:   "valid config",
			config: DefaultConfig(),
			valid:  true,
		},
		{
			name: "negative cache size",
			config: &Config{
				CacheMaxSize:   -1,
				LogLevel:       "info",
				MaxRenderDepth: 100,
			},
			valid: false,
		},
		{
			name: "negative cache TTL",
			config: &Config{
				CacheMaxSize:   100,
				CacheTTL:       -1 * time.Second,
				LogLevel:       "info",
				MaxRenderDepth: 100,
			},
			valid: false,
		},
		{
			name: "invalid log level",
			config: &Config{
				CacheMaxSize:   100,
				LogLevel:       "invalid",
				MaxRenderDepth: 100,
			},
			valid: false,
		},
		{
			name: "zero max render depth",
			config: &Config{
				CacheMaxSize:   100,
				LogLevel:       "info",
				MaxRenderDepth: 0,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.valid && err != nil {
				t.Errorf("Validate() returned error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Validate() returned nil, want error")
			}
		})
	}
}

func TestGlobalConfig(t *testing.T) {
	// Save original config
	originalConfig := GetGlobalConfig()

	// Test setting global config
	newConfig := &Config{
		CacheMaxSize:   50,
		LogLevel:       "debug",
		MaxRenderDepth: 200,
		StrictMode:     true,
	}

	SetGlobalConfig(newConfig)

	retrievedConfig := GetGlobalConfig()
	if retrievedConfig.CacheMaxSize != 50 {
		t.Errorf("Global CacheMaxSize = %d, want 50", retrievedConfig.CacheMaxSize)
	}
	if retrievedConfig.LogLevel != "debug" {
		t.Errorf("Global LogLevel = %s, want debug", retrievedConfig.LogLevel)
	}

	// Restore original config
	SetGlobalConfig(originalConfig)
}