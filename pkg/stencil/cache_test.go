package stencil

import (
	"archive/zip"
	"bytes"
	"io"
	"sync"
	"testing"
	"time"
)

func TestTemplateCache_Basic(t *testing.T) {
	cache := NewTemplateCache()
	
	// Create a test template
	templateData := createTestDocx(t, "{{name}}")
	reader := bytes.NewReader(templateData.Bytes())
	
	// First preparation should create new template
	prepared1, err := cache.Prepare(reader, "test-key")
	if err != nil {
		t.Fatalf("Failed to prepare template: %v", err)
	}
	
	// Second preparation with same key should return cached template
	prepared2, err := cache.Prepare(nil, "test-key") // nil reader since it should use cache
	if err != nil {
		t.Fatalf("Failed to get cached template: %v", err)
	}
	
	// Should be the same object
	if prepared1 != prepared2 {
		t.Error("Expected cached template to be the same object")
	}
}

func TestTemplateCache_DifferentKeys(t *testing.T) {
	cache := NewTemplateCache()
	
	// Create two different templates
	template1 := createTestDocx(t, "{{name1}}")
	template2 := createTestDocx(t, "{{name2}}")
	
	prepared1, err := cache.Prepare(bytes.NewReader(template1.Bytes()), "key1")
	if err != nil {
		t.Fatalf("Failed to prepare template1: %v", err)
	}
	
	prepared2, err := cache.Prepare(bytes.NewReader(template2.Bytes()), "key2")
	if err != nil {
		t.Fatalf("Failed to prepare template2: %v", err)
	}
	
	// Should be different objects
	if prepared1 == prepared2 {
		t.Error("Expected different templates for different keys")
	}
	
	// Verify each can be rendered independently
	data1 := TemplateData{"name1": "Alice"}
	data2 := TemplateData{"name2": "Bob"}
	
	output1, err := prepared1.Render(data1)
	if err != nil {
		t.Fatalf("Failed to render template1: %v", err)
	}
	
	output2, err := prepared2.Render(data2)
	if err != nil {
		t.Fatalf("Failed to render template2: %v", err)
	}
	
	// Read and parse outputs to verify
	output1Bytes, err := io.ReadAll(output1)
	if err != nil {
		t.Fatalf("Failed to read output1: %v", err)
	}
	
	output2Bytes, err := io.ReadAll(output2)
	if err != nil {
		t.Fatalf("Failed to read output2: %v", err)
	}
	
	// Parse the DOCX outputs to check content
	reader1, err := zip.NewReader(bytes.NewReader(output1Bytes), int64(len(output1Bytes)))
	if err != nil {
		t.Fatalf("Failed to parse output1 as zip: %v", err)
	}
	
	reader2, err := zip.NewReader(bytes.NewReader(output2Bytes), int64(len(output2Bytes)))
	if err != nil {
		t.Fatalf("Failed to parse output2 as zip: %v", err)
	}
	
	// Find and read document.xml in both outputs
	var doc1Content, doc2Content []byte
	for _, f := range reader1.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open document.xml in output1: %v", err)
			}
			doc1Content, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read document.xml in output1: %v", err)
			}
			break
		}
	}
	
	for _, f := range reader2.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open document.xml in output2: %v", err)
			}
			doc2Content, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read document.xml in output2: %v", err)
			}
			break
		}
	}
	
	// Verify outputs contain expected values
	if !bytes.Contains(doc1Content, []byte("Alice")) {
		t.Errorf("Template1 output should contain 'Alice', got: %s", doc1Content)
	}
	if !bytes.Contains(doc2Content, []byte("Bob")) {
		t.Errorf("Template2 output should contain 'Bob', got: %s", doc2Content)
	}
}

func TestTemplateCache_Clear(t *testing.T) {
	cache := NewTemplateCache()
	
	templateData := createTestDocx(t, "{{name}}")
	
	// Add template to cache
	prepared1, err := cache.Prepare(bytes.NewReader(templateData.Bytes()), "test-key")
	if err != nil {
		t.Fatalf("Failed to prepare template: %v", err)
	}
	
	// Clear cache
	cache.Clear()
	
	// Try to get from cache - should need to prepare again
	prepared2, err := cache.Prepare(bytes.NewReader(templateData.Bytes()), "test-key")
	if err != nil {
		t.Fatalf("Failed to prepare template after clear: %v", err)
	}
	
	// Should be different objects since cache was cleared
	if prepared1 == prepared2 {
		t.Error("Expected new template after cache clear")
	}
}

func TestTemplateCache_Remove(t *testing.T) {
	cache := NewTemplateCache()
	
	templateData := createTestDocx(t, "{{name}}")
	
	// Add multiple templates
	_, err := cache.Prepare(bytes.NewReader(templateData.Bytes()), "key1")
	if err != nil {
		t.Fatalf("Failed to prepare template: %v", err)
	}
	
	prepared2, err := cache.Prepare(bytes.NewReader(templateData.Bytes()), "key2")
	if err != nil {
		t.Fatalf("Failed to prepare template: %v", err)
	}
	
	// Remove key1
	cache.Remove("key1")
	
	// key2 should still be cached
	cachedPrepared2, err := cache.Prepare(nil, "key2")
	if err != nil {
		t.Fatalf("Failed to get cached template: %v", err)
	}
	
	if prepared2 != cachedPrepared2 {
		t.Error("key2 should still be cached after removing key1")
	}
}

func TestTemplateCache_ConcurrentAccess(t *testing.T) {
	cache := NewTemplateCache()
	templateData := createTestDocx(t, "{{name}}")
	
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	
	// Simulate concurrent access
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			key := "shared-key"
			if id%2 == 0 {
				key = "key-" + string(rune('0'+id))
			}
			
			// Prepare template
			_, err := cache.Prepare(bytes.NewReader(templateData.Bytes()), key)
			if err != nil {
				errors <- err
				return
			}
			
			// Render template
			data := TemplateData{"name": "User" + string(rune('0'+id))}
			prepared, err := cache.Prepare(nil, key)
			if err != nil {
				// May need to prepare if not in cache
				prepared, err = cache.Prepare(bytes.NewReader(templateData.Bytes()), key)
				if err != nil {
					errors <- err
					return
				}
			}
			
			_, err = prepared.Render(data)
			if err != nil {
				errors <- err
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestTemplateCache_Configuration(t *testing.T) {
	// Test with size limit
	config := CacheConfig{
		MaxSize: 2,
		TTL:     0, // No TTL for this test
	}
	cache := NewTemplateCacheWithConfig(config)
	
	template1 := createTestDocx(t, "{{name1}}")
	template2 := createTestDocx(t, "{{name2}}")
	template3 := createTestDocx(t, "{{name3}}")
	
	// Add templates up to limit
	_, err := cache.Prepare(bytes.NewReader(template1.Bytes()), "key1")
	if err != nil {
		t.Fatalf("Failed to prepare template1: %v", err)
	}
	
	_, err = cache.Prepare(bytes.NewReader(template2.Bytes()), "key2")
	if err != nil {
		t.Fatalf("Failed to prepare template2: %v", err)
	}
	
	// Add third template (should evict oldest)
	_, err = cache.Prepare(bytes.NewReader(template3.Bytes()), "key3")
	if err != nil {
		t.Fatalf("Failed to prepare template3: %v", err)
	}
	
	// key1 should be evicted
	_, err = cache.Prepare(nil, "key1")
	if err == nil {
		t.Error("Expected key1 to be evicted from cache")
	}
	
	// key2 and key3 should still be in cache
	_, err = cache.Prepare(nil, "key2")
	if err != nil {
		t.Error("Expected key2 to still be in cache")
	}
	
	_, err = cache.Prepare(nil, "key3")
	if err != nil {
		t.Error("Expected key3 to still be in cache")
	}
}

func TestTemplateCache_TTL(t *testing.T) {
	config := CacheConfig{
		MaxSize: 10,
		TTL:     100 * time.Millisecond,
	}
	cache := NewTemplateCacheWithConfig(config)
	
	templateData := createTestDocx(t, "{{name}}")
	
	// Add template
	_, err := cache.Prepare(bytes.NewReader(templateData.Bytes()), "ttl-key")
	if err != nil {
		t.Fatalf("Failed to prepare template: %v", err)
	}
	
	// Should be in cache immediately
	_, err = cache.Prepare(nil, "ttl-key")
	if err != nil {
		t.Error("Expected template to be in cache immediately after adding")
	}
	
	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)
	
	// Should no longer be in cache
	_, err = cache.Prepare(nil, "ttl-key")
	if err == nil {
		t.Error("Expected template to be evicted after TTL")
	}
}

func TestTemplateCache_Disabled(t *testing.T) {
	// Create cache with size 0 (disabled)
	config := CacheConfig{
		MaxSize: 0,
	}
	cache := NewTemplateCacheWithConfig(config)
	
	templateData := createTestDocx(t, "{{name}}")
	
	// Prepare template
	prepared1, err := cache.Prepare(bytes.NewReader(templateData.Bytes()), "key1")
	if err != nil {
		t.Fatalf("Failed to prepare template: %v", err)
	}
	
	// Try to get from cache - should fail since cache is disabled
	_, err = cache.Prepare(nil, "key1")
	if err == nil {
		t.Error("Expected error when cache is disabled")
	}
	
	// Should still be able to prepare new templates
	prepared2, err := cache.Prepare(bytes.NewReader(templateData.Bytes()), "key1")
	if err != nil {
		t.Fatalf("Failed to prepare template with disabled cache: %v", err)
	}
	
	// Should be different objects
	if prepared1 == prepared2 {
		t.Error("Expected different objects when cache is disabled")
	}
}