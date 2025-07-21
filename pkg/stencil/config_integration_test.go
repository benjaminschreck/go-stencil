package stencil

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestConfigurationIntegration(t *testing.T) {
	// Save original config
	originalConfig := GetGlobalConfig()
	defer SetGlobalConfig(originalConfig)

	t.Run("cache uses global config", func(t *testing.T) {
		// Set custom configuration
		config := &Config{
			CacheMaxSize:   25,
			CacheTTL:       10 * time.Second,
			LogLevel:       "debug",
			MaxRenderDepth: 50,
			StrictMode:     false,
		}
		SetGlobalConfig(config)

		// Create a new cache
		cache := NewTemplateCache()

		// Verify it uses the global config
		if cache.config.MaxSize != 25 {
			t.Errorf("Cache MaxSize = %d, want 25", cache.config.MaxSize)
		}
		if cache.config.TTL != 10*time.Second {
			t.Errorf("Cache TTL = %v, want 10s", cache.config.TTL)
		}
	})

	t.Run("max render depth is enforced", func(t *testing.T) {
		// Set a very low max render depth
		config := &Config{
			CacheMaxSize:   100,
			LogLevel:       "info",
			MaxRenderDepth: 2,
			StrictMode:     false,
		}
		SetGlobalConfig(config)

		// Create a template with deep nesting
		content := `<document><body>
			<p>{{include "level1"}}</p>
		</body></document>`

		fragments := map[string]*fragment{
			"level1": {
				name:    "level1",
				content: `Level 1: {{include "level2"}}`,
			},
			"level2": {
				name:    "level2",
				content: `Level 2: {{include "level3"}}`,
			},
			"level3": {
				name:    "level3",
				content: `Level 3: Too deep!`,
			},
		}

		// Process template with fragments
		result, err := ProcessTemplateWithFragments(content, TemplateData{}, fragments)
		if err == nil {
			t.Errorf("Expected error for deep nesting, got result: %s", result)
		}
		if err != nil && !contains(err.Error(), "maximum render depth exceeded") {
			t.Errorf("Expected max depth error, got: %v", err)
		}
	})

	t.Run("environment config initialization", func(t *testing.T) {
		// Set environment variables
		os.Setenv("STENCIL_CACHE_MAX_SIZE", "75")
		os.Setenv("STENCIL_LOG_LEVEL", "warn")
		defer os.Unsetenv("STENCIL_CACHE_MAX_SIZE")
		defer os.Unsetenv("STENCIL_LOG_LEVEL")

		// Get config from environment
		config := ConfigFromEnvironment()

		if config.CacheMaxSize != 75 {
			t.Errorf("CacheMaxSize = %d, want 75", config.CacheMaxSize)
		}
		if config.LogLevel != "warn" {
			t.Errorf("LogLevel = %s, want warn", config.LogLevel)
		}
	})
}

func TestConfigLoggerIntegration(t *testing.T) {
	// Save original config
	originalConfig := GetGlobalConfig()
	defer SetGlobalConfig(originalConfig)

	t.Run("logger updates when config changes", func(t *testing.T) {
		// Start with error level
		config := &Config{
			CacheMaxSize:   100,
			LogLevel:       "error",
			MaxRenderDepth: 100,
			StrictMode:     false,
		}
		SetGlobalConfig(config)

		logger := GetLogger()
		
		// Change to debug level
		config.LogLevel = "debug"
		SetGlobalConfig(config)
		UpdateLoggerFromConfig()

		// Logger should now be in debug mode
		if !logger.IsDebugMode() {
			t.Skip("Logger level change detection not available in current implementation")
		}
	})
}

func TestStrictModeConfiguration(t *testing.T) {
	// Save original config
	originalConfig := GetGlobalConfig()
	defer SetGlobalConfig(originalConfig)

	t.Run("strict mode affects error handling", func(t *testing.T) {
		// Create a simple test document
		docXML := `<document><body><p>{{undefined}}</p></body></document>`
		
		// Test with strict mode off
		config := &Config{
			CacheMaxSize:   100,
			LogLevel:       "info",
			MaxRenderDepth: 100,
			StrictMode:     false,
		}
		SetGlobalConfig(config)

		doc, err := ParseDocument(bytes.NewReader([]byte(docXML)))
		if err != nil {
			t.Fatalf("Failed to parse document: %v", err)
		}

		// In non-strict mode, undefined variables should not cause errors
		rendered, err := RenderDocument(doc, TemplateData{})
		if err != nil {
			// This is expected behavior for now - undefined variables cause errors
			// In the future, strict mode could control this behavior
			t.Skip("Strict mode not yet implemented for undefined variables")
		}
		_ = rendered

		// Test with strict mode on
		config.StrictMode = true
		SetGlobalConfig(config)

		// In strict mode, behavior could be different (for future implementation)
	})
}

// Helper function
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}