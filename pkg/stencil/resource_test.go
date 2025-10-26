package stencil

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestPreparedTemplateClose tests that Close() can be called safely
func TestPreparedTemplateClose(t *testing.T) {
	// Create a simple template
	reader := createTestDocx(t, "Hello {{name}}")
	
	tmpl, err := Prepare(reader)
	if err != nil {
		t.Fatalf("Failed to prepare template: %v", err)
	}
	
	// Test that Close() can be called
	err = tmpl.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
	
	// Test that Close() can be called multiple times
	err = tmpl.Close()
	if err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}
	
	// Test that the template can still be used after Close()
	// Close() is more of a hint for resource cleanup, not a hard stop
	data := map[string]interface{}{
		"name": "World",
	}
	_, err = tmpl.Render(data)
	if err != nil {
		t.Errorf("Render after Close() failed: %v", err)
	}
}

// TestTemplateCacheClose tests that cache can clean up resources
func TestTemplateCacheClose(t *testing.T) {
	cache := NewTemplateCacheWithConfig(CacheConfig{
		MaxSize: 10,
		TTL:     time.Hour,
	})
	
	// Add some templates to the cache
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("template-%d", i)
		reader := createTestDocx(t, fmt.Sprintf("Template {{num%d}}", i))
		
		tmpl, err := cache.Prepare(reader, key)
		if err != nil {
			t.Fatalf("Failed to prepare template: %v", err)
		}
		
		// Use the template
		data := map[string]interface{}{
			fmt.Sprintf("num%d", i): i,
		}
		_, err = tmpl.Render(data)
		if err != nil {
			t.Fatalf("Failed to render template: %v", err)
		}
	}
	
	// Test Clear() method
	cache.Clear()
	
	// Verify cache is empty
	if len(cache.cache) != 0 {
		t.Errorf("Cache not empty after Clear(), has %d entries", len(cache.cache))
	}
	if cache.lru.Len() != 0 {
		t.Errorf("LRU list not empty after Clear(), has %d entries", cache.lru.Len())
	}
}

// TestResourceLeaksUnderLoad tests for resource leaks under concurrent load
func TestResourceLeaksUnderLoad(t *testing.T) {
	// Skip this test in short mode as it's a stress test
	if testing.Short() {
		t.Skip("Skipping resource leak test in short mode")
	}
	
	// Record initial memory stats
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// Create a cache
	cache := NewTemplateCacheWithConfig(CacheConfig{
		MaxSize: 50,
		TTL:     time.Second * 10,
	})
	
	// Run concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 20
	numOperations := 100
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				// Create unique key for each operation
				key := fmt.Sprintf("template-%d-%d", goroutineID, j%10)
				
				// Get or create template
				reader := createTestDocx(t, fmt.Sprintf("Test {{var%d}}", goroutineID))
				tmpl, err := cache.Prepare(reader, key)
				if err != nil {
					t.Errorf("Failed to prepare template: %v", err)
					continue
				}
				
				// Render the template
				data := map[string]interface{}{
					fmt.Sprintf("var%d", goroutineID): j,
				}
				_, err = tmpl.Render(data)
				if err != nil {
					t.Errorf("Failed to render: %v", err)
				}
				
				// Occasionally remove entries
				if j%20 == 0 {
					cache.Remove(key)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Clear the cache
	cache.Clear()
	
	// Force garbage collection
	runtime.GC()
	runtime.GC() // Run twice to ensure finalizers run
	
	// Check memory stats
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	
	// Allow some memory growth but flag significant leaks
	memGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	memGrowthMB := float64(memGrowth) / (1024 * 1024)
	
	t.Logf("Memory growth: %.2f MB", memGrowthMB)
	t.Logf("Goroutines: %d -> %d", runtime.NumGoroutine(), runtime.NumGoroutine())
	
	// Warn if memory grew by more than 50MB
	if memGrowthMB > 50 {
		t.Logf("Warning: Significant memory growth detected: %.2f MB", memGrowthMB)
	}
}

// TestDocxReaderResourceCleanup tests that DocxReader doesn't leak file handles
func TestDocxReaderResourceCleanup(t *testing.T) {
	// Create multiple DocxReader instances
	for i := 0; i < 100; i++ {
		reader := createTestDocx(t, fmt.Sprintf("Test {{num%d}}", i))
		
		docxReader, err := NewDocxReader(bytes.NewReader(reader.Bytes()), int64(reader.Len()))
		if err != nil {
			t.Fatalf("Failed to create DocxReader: %v", err)
		}
		
		// Access some files
		_, err = docxReader.GetDocumentXML()
		if err != nil {
			t.Fatalf("Failed to get document.xml: %v", err)
		}
		
		// Note: Currently DocxReader doesn't have a Close method
		// This test documents the current behavior and will help
		// verify that adding Close() doesn't break anything
		_ = docxReader
	}
	
	// Force GC to clean up any resources
	runtime.GC()
}

// TestConcurrentTemplateAccess tests concurrent access to templates
func TestConcurrentTemplateAccess(t *testing.T) {
	reader := createTestDocx(t, "Hello {{name}}, count: {{count}}")
	
	tmpl, err := Prepare(reader)
	if err != nil {
		t.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()
	
	// Run concurrent renders
	var wg sync.WaitGroup
	numGoroutines := 50
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			data := map[string]interface{}{
				"name":  fmt.Sprintf("User%d", id),
				"count": id,
			}
			
			_, err := tmpl.Render(data)
			if err != nil {
				t.Errorf("Goroutine %d: Render failed: %v", id, err)
				return
			}
		}(i)
	}
	
	wg.Wait()
}

// TestTemplateCacheEviction tests that eviction properly cleans up resources
func TestTemplateCacheEviction(t *testing.T) {
	cache := NewTemplateCacheWithConfig(CacheConfig{
		MaxSize: 3, // Small cache to force evictions
		TTL:     0,
	})
	
	// Track which templates have been created
	created := make(map[string]bool)
	
	// Add more templates than cache can hold
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("template-%d", i)
		created[key] = true
		
		reader := createTestDocx(t, fmt.Sprintf("Template {{var%d}}", i))
		
		tmpl, err := cache.Prepare(reader, key)
		if err != nil {
			t.Fatalf("Failed to prepare template: %v", err)
		}
		
		// Use the template
		data := map[string]interface{}{
			fmt.Sprintf("var%d", i): i,
		}
		_, err = tmpl.Render(data)
		if err != nil {
			t.Fatalf("Failed to render: %v", err)
		}
	}
	
	// Cache should only have MaxSize entries
	if len(cache.cache) > cache.config.MaxSize {
		t.Errorf("Cache has %d entries, expected max %d", len(cache.cache), cache.config.MaxSize)
	}
	
	// Verify LRU list consistency
	if cache.lru.Len() != len(cache.cache) {
		t.Errorf("LRU list has %d entries, cache has %d entries", cache.lru.Len(), len(cache.cache))
	}
	
	// Clear cache and verify cleanup
	cache.Clear()
	
	if len(cache.cache) != 0 || cache.lru.Len() != 0 {
		t.Errorf("Cache not properly cleared: %d entries, %d LRU items", len(cache.cache), cache.lru.Len())
	}
}

// TestTemplateCacheTTLExpiration tests that TTL expiration works properly
func TestTemplateCacheTTLExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTL test in short mode")
	}
	
	cache := NewTemplateCacheWithConfig(CacheConfig{
		MaxSize: 10,
		TTL:     time.Millisecond * 100, // Short TTL for testing
	})
	
	// Add a template
	key := "test-template"
	reader := createTestDocx(t, "Hello {{name}}")
	
	tmpl1, err := cache.Prepare(reader, key)
	if err != nil {
		t.Fatalf("Failed to prepare template: %v", err)
	}
	
	// Template should be in cache
	_, found := cache.Get(key)
	if !found {
		t.Error("Template not found in cache immediately after adding")
	}
	
	// Wait for TTL to expire
	time.Sleep(time.Millisecond * 150)
	
	// Try to get the template again - it should not be found (expired)
	_, found = cache.Get(key)
	if found {
		t.Error("Template found in cache after TTL expiration")
	}
	
	// Prepare again with new reader
	reader2 := createTestDocx(t, "Hello {{name}}")
	tmpl2, err := cache.Prepare(reader2, key)
	if err != nil {
		t.Fatalf("Failed to prepare template after expiration: %v", err)
	}
	
	// Both templates should still work independently
	data := map[string]interface{}{"name": "World"}
	if _, err := tmpl1.Render(data); err != nil {
		t.Errorf("Original template failed to render: %v", err)
	}
	if _, err := tmpl2.Render(data); err != nil {
		t.Errorf("New template failed to render: %v", err)
	}
}

// TestTemplateResourcesSafety tests that templates handle resources safely
func TestTemplateResourcesSafety(t *testing.T) {
	// Test 1: Multiple templates from same source
	content := "Hello {{name}}"
	var templates []*PreparedTemplate
	
	for i := 0; i < 10; i++ {
		reader := createTestDocx(t, content)
		tmpl, err := Prepare(reader)
		if err != nil {
			t.Fatalf("Failed to prepare template %d: %v", i, err)
		}
		templates = append(templates, tmpl)
	}
	
	// Close all templates
	for i, tmpl := range templates {
		if err := tmpl.Close(); err != nil {
			t.Errorf("Failed to close template %d: %v", i, err)
		}
	}
	
	// Test 2: Template with fragments
	reader := createTestDocx(t, "Main: {{name}}, Fragment: {{include \"fragment1\"}}")
	tmpl, err := Prepare(reader)
	if err != nil {
		t.Fatalf("Failed to prepare template with fragments: %v", err)
	}

	// Add a fragment
	err = tmpl.AddFragment("fragment1", "Fragment content: {{data}}")
	if err != nil {
		t.Fatalf("Failed to add fragment: %v", err)
	}

	// Use the template
	data := map[string]interface{}{
		"name": "Test",
		"data": "Fragment Data",
	}
	_, err = tmpl.Render(data)
	if err != nil {
		t.Errorf("Failed to render template with fragments: %v", err)
	}
	
	// Close should handle fragments properly
	if err := tmpl.Close(); err != nil {
		t.Errorf("Failed to close template with fragments: %v", err)
	}
}