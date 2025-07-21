package stencil

import (
	"container/list"
	"errors"
	"io"
	"sync"
	"time"
)

// CacheConfig contains configuration options for the template cache
type CacheConfig struct {
	// MaxSize is the maximum number of templates to cache. 0 disables caching.
	MaxSize int
	// TTL is the time-to-live for cached templates. 0 means no expiration.
	TTL time.Duration
}

// TemplateCache provides caching for prepared templates
type TemplateCache struct {
	mu       sync.RWMutex
	cache    map[string]*cacheEntry
	lru      *list.List
	config   CacheConfig
}

type cacheEntry struct {
	key      string
	template *PreparedTemplate
	expiry   time.Time
	element  *list.Element
}

// NewTemplateCache creates a new template cache with default configuration
func NewTemplateCache() *TemplateCache {
	config := GetGlobalConfig()
	return NewTemplateCacheWithConfig(CacheConfig{
		MaxSize: config.CacheMaxSize,
		TTL:     config.CacheTTL,
	})
}

// NewTemplateCacheWithConfig creates a new template cache with the given configuration
func NewTemplateCacheWithConfig(config CacheConfig) *TemplateCache {
	return &TemplateCache{
		cache:  make(map[string]*cacheEntry),
		lru:    list.New(),
		config: config,
	}
}

// Prepare retrieves a template from cache or prepares a new one
func (tc *TemplateCache) Prepare(reader io.Reader, key string) (*PreparedTemplate, error) {
	// Check if caching is disabled
	if tc.config.MaxSize == 0 {
		if reader == nil {
			return nil, errors.New("cache is disabled and no reader provided")
		}
		return Prepare(reader)
	}
	
	// Try to get from cache first
	tc.mu.RLock()
	entry, exists := tc.cache[key]
	tc.mu.RUnlock()
	
	if exists {
		// Check if entry has expired
		if tc.config.TTL > 0 && time.Now().After(entry.expiry) {
			// Entry has expired, remove it
			tc.Remove(key)
		} else {
			// Move to front of LRU list
			tc.mu.Lock()
			tc.lru.MoveToFront(entry.element)
			tc.mu.Unlock()
			return entry.template, nil
		}
	}
	
	// Not in cache or expired, need to prepare
	if reader == nil {
		return nil, errors.New("template not in cache and no reader provided")
	}
	
	// Prepare the template
	prepared, err := Prepare(reader)
	if err != nil {
		return nil, err
	}
	
	// Add to cache
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	// Check if we need to evict
	if tc.lru.Len() >= tc.config.MaxSize {
		// Evict least recently used
		oldest := tc.lru.Back()
		if oldest != nil {
			oldEntry := oldest.Value.(*cacheEntry)
			// Close the evicted template
			if oldEntry.template != nil {
				oldEntry.template.Close()
			}
			delete(tc.cache, oldEntry.key)
			tc.lru.Remove(oldest)
		}
	}
	
	// Create new entry
	entry = &cacheEntry{
		key:      key,
		template: prepared,
	}
	
	if tc.config.TTL > 0 {
		entry.expiry = time.Now().Add(tc.config.TTL)
	}
	
	// Add to LRU list
	element := tc.lru.PushFront(entry)
	entry.element = element
	
	// Add to cache map
	tc.cache[key] = entry
	
	return prepared, nil
}

// Get retrieves a template from cache without preparing a new one
func (tc *TemplateCache) Get(key string) (*PreparedTemplate, bool) {
	tc.mu.RLock()
	entry, exists := tc.cache[key]
	tc.mu.RUnlock()
	
	if !exists {
		return nil, false
	}
	
	// Check expiry
	if tc.config.TTL > 0 && time.Now().After(entry.expiry) {
		tc.Remove(key)
		return nil, false
	}
	
	// Move to front of LRU
	tc.mu.Lock()
	tc.lru.MoveToFront(entry.element)
	tc.mu.Unlock()
	
	return entry.template, true
}

// Set adds a template to the cache
func (tc *TemplateCache) Set(key string, template *PreparedTemplate) {
	// Check if caching is disabled
	if tc.config.MaxSize == 0 {
		return
	}
	
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	// Check if key already exists
	if existing, exists := tc.cache[key]; exists {
		// Update existing entry
		existing.template = template
		existing.expiry = time.Now().Add(tc.config.TTL)
		tc.lru.MoveToFront(existing.element)
		return
	}
	
	// Check if we need to evict
	if tc.lru.Len() >= tc.config.MaxSize {
		// Evict least recently used
		oldest := tc.lru.Back()
		if oldest != nil {
			oldEntry := oldest.Value.(*cacheEntry)
			delete(tc.cache, oldEntry.key)
			tc.lru.Remove(oldest)
			if oldEntry.template != nil {
				oldEntry.template.Close()
			}
		}
	}
	
	// Create new entry
	expiry := time.Time{}
	if tc.config.TTL > 0 {
		expiry = time.Now().Add(tc.config.TTL)
	}
	
	entry := &cacheEntry{
		key:      key,
		template: template,
		expiry:   expiry,
	}
	
	// Add to LRU list
	element := tc.lru.PushFront(entry)
	entry.element = element
	
	// Add to cache map
	tc.cache[key] = entry
}

// Remove removes a template from the cache and closes it
func (tc *TemplateCache) Remove(key string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	entry, exists := tc.cache[key]
	if !exists {
		return
	}
	
	// Close the template
	if entry.template != nil {
		entry.template.Close()
	}
	
	delete(tc.cache, key)
	tc.lru.Remove(entry.element)
}

// Clear removes all templates from the cache and closes them
func (tc *TemplateCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	// Close all cached templates
	for _, entry := range tc.cache {
		if entry.template != nil {
			entry.template.Close()
		}
	}
	
	tc.cache = make(map[string]*cacheEntry)
	tc.lru = list.New()
}

// Size returns the current number of cached templates
func (tc *TemplateCache) Size() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return len(tc.cache)
}

// Close closes all templates in the cache and clears it
func (tc *TemplateCache) Close() error {
	tc.Clear()
	return nil
}

// defaultCache is a global cache instance for convenience
var defaultCache = NewTemplateCache()

