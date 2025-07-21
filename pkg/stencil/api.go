// Package stencil provides a powerful template engine for Microsoft Word documents (DOCX).
// It enables dynamic document generation by processing templates with placeholders, control structures,
// and built-in functions.
//
// Basic Usage:
//
//	// Prepare a template from a file
//	tmpl, err := stencil.PrepareFile("template.docx")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer tmpl.Close()
//	
//	// Render with data
//	data := stencil.TemplateData{
//	    "name": "John Doe",
//	    "items": []map[string]interface{}{
//	        {"product": "Widget", "price": 19.99},
//	        {"product": "Gadget", "price": 29.99},
//	    },
//	}
//	
//	output, err := tmpl.Render(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	
//	// Save the result
//	result, err := os.Create("output.docx")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer result.Close()
//	
//	_, err = io.Copy(result, output)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Template Syntax:
//
// Variables: {{name}}, {{customer.address}}, {{price * 1.2}}
//
// Conditionals: {{if condition}}...{{else}}...{{end}}
//
// Loops: {{for item in items}}...{{end}}
//
// Functions: {{uppercase(name)}}, {{format("%.2f", price)}}
//
// For more information on template syntax and available functions, see the README.
package stencil

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Engine provides the main API for working with templates.
// Use New() to create a new engine instance.
type Engine struct {
	config   *Config
	cache    *TemplateCache
	registry FunctionRegistry
}

// New creates a new template engine with default configuration.
func New() *Engine {
	return &Engine{
		config:   GetGlobalConfig(),
		cache:    defaultCache,
		registry: GetDefaultFunctionRegistry(),
	}
}

// NewWithConfig creates a new template engine with custom configuration.
func NewWithConfig(config *Config) *Engine {
	return &Engine{
		config:   config,
		cache:    NewTemplateCache(),
		registry: NewFunctionRegistry(),
	}
}

// PrepareFile loads and compiles a template from a file path.
// The template is cached if caching is enabled in the configuration.
func (e *Engine) PrepareFile(path string) (*PreparedTemplate, error) {
	// Check cache first if enabled
	if e.config.CacheMaxSize > 0 && e.cache != nil {
		if tmpl, ok := e.cache.Get(path); ok {
			return tmpl, nil
		}
	}

	// Open and prepare the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open template file: %w", err)
	}
	defer file.Close()

	tmpl, err := e.Prepare(file)
	if err != nil {
		return nil, err
	}

	// Store in cache if enabled
	if e.config.CacheMaxSize > 0 && e.cache != nil {
		e.cache.Set(path, tmpl)
	}

	return tmpl, nil
}

// Prepare loads and compiles a template from an io.Reader.
func (e *Engine) Prepare(r io.Reader) (*PreparedTemplate, error) {
	// For now, use the global prepare function
	// In future, we might want to inject engine-specific settings
	return prepare(r)
}

// RegisterFunction adds a custom function that can be used in templates.
// The function name must be a valid identifier and not conflict with built-in functions.
func (e *Engine) RegisterFunction(name string, fn Function) error {
	return e.registry.RegisterFunction(fn)
}

// RegisterFunctionsFromProvider registers all functions from a provider.
// This is useful for adding a suite of related functions at once.
func (e *Engine) RegisterFunctionsFromProvider(provider FunctionProvider) error {
	functions := provider.ProvideFunctions()
	for name, fn := range functions {
		if err := e.registry.RegisterFunction(fn); err != nil {
			return fmt.Errorf("failed to register function %s: %w", name, err)
		}
	}
	return nil
}

// Config returns the engine's configuration.
func (e *Engine) Config() *Config {
	return e.config
}

// SetConfig updates the engine's configuration.
// Note that some settings (like cache size) may not take effect immediately.
func (e *Engine) SetConfig(config *Config) {
	e.config = config
}

// ClearCache removes all templates from the cache.
func (e *Engine) ClearCache() {
	if e.cache != nil {
		e.cache.Clear()
	}
}

// Close releases any resources held by the engine.
func (e *Engine) Close() error {
	// Currently no resources to release, but kept for future use
	return nil
}

// Option represents a configuration option for the engine.
type Option func(*Engine)

// WithConfig returns an option that sets the engine configuration.
func WithConfig(config *Config) Option {
	return func(e *Engine) {
		e.config = config
	}
}

// WithCache returns an option that sets the cache size (0 disables caching).
func WithCache(maxSize int) Option {
	return func(e *Engine) {
		e.config.CacheMaxSize = maxSize
	}
}

// WithFunction returns an option that registers a custom function.
func WithFunction(name string, fn Function) Option {
	return func(e *Engine) {
		e.registry.RegisterFunction(fn)
	}
}

// WithFunctionProvider returns an option that registers functions from a provider.
func WithFunctionProvider(provider FunctionProvider) Option {
	return func(e *Engine) {
		e.RegisterFunctionsFromProvider(provider)
	}
}

// NewWithOptions creates a new engine with the specified options.
func NewWithOptions(opts ...Option) *Engine {
	engine := New()
	for _, opt := range opts {
		opt(engine)
	}
	return engine
}

// DefaultEngine is the global default engine instance.
// It uses the global configuration and function registry.
var DefaultEngine = New()

// Module-level convenience functions that use the default engine.

// PrepareFile loads and compiles a template from a file path using the default engine.
func PrepareFile(path string) (*PreparedTemplate, error) {
	return DefaultEngine.PrepareFile(path)
}

// Prepare loads and compiles a template from an io.Reader using the default engine.
func Prepare(r io.Reader) (*PreparedTemplate, error) {
	return DefaultEngine.Prepare(r)
}


// RegisterGlobalFunction adds a custom function to the global function registry.
func RegisterGlobalFunction(name string, fn Function) error {
	return DefaultEngine.RegisterFunction(name, fn)
}

// RegisterFunctionsFromProvider registers functions from a provider in the global registry.
func RegisterFunctionsFromProvider(provider FunctionProvider) error {
	return DefaultEngine.RegisterFunctionsFromProvider(provider)
}

// PrepareWithCache loads and compiles a template with caching support.
// This is a convenience function that uses the default engine.
func PrepareWithCache(path string) (*PreparedTemplate, error) {
	return DefaultEngine.PrepareFile(path)
}

// ClearCache clears the global template cache.
func ClearCache() {
	DefaultEngine.ClearCache()
}

// SetCacheConfig updates the global cache configuration.
func SetCacheConfig(maxSize int, ttl time.Duration) {
	config := GetGlobalConfig()
	config.CacheMaxSize = maxSize
	config.CacheTTL = ttl
	SetGlobalConfig(config)
}