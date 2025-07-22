package stencil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Regular expressions for parsing field access
var (
	// Matches array/map access like [0], ['key'], ["key"]
	bracketRegex = regexp.MustCompile(`^\[([^\]]+)\]`)
	// Matches dot notation like .field
	dotRegex = regexp.MustCompile(`^\.([^.\[]+)`)
)

// EvaluateVariable evaluates a variable expression with support for nested field access
func EvaluateVariable(expression string, data TemplateData) (interface{}, error) {
	// Trim whitespace
	expression = strings.TrimSpace(expression)
	
	logger := GetLogger()
	if logger.IsDebugMode() {
		logger.DebugExpression(expression, data)
	}
	
	// Handle nil or empty data
	if data == nil {
		return nil, nil
	}
	
	// Parse the expression into field access parts
	parts, err := parseFieldAccess(expression)
	if err != nil {
		return nil, err
	}
	
	// Start with the data and navigate through the parts
	current := interface{}(data)
	
	for _, part := range parts {
		switch part.Type {
		case fieldTypeIdentifier:
			// Handle map field access
			current = accessMapField(current, part.Value)
		case fieldTypeBracket:
			// Handle bracket notation (could be array index or map key)
			current = accessBracketField(current, part.Value)
		}
		
		// If we hit nil at any point, return nil
		if current == nil {
			return nil, nil
		}
	}
	
	if logger.IsDebugMode() {
		logger.WithField("result", current).Debug("Variable evaluation complete")
	}
	
	return current, nil
}

// fieldAccessPart represents a part of a field access expression
type fieldAccessPart struct {
	Type  fieldAccessType
	Value string
}

type fieldAccessType int

const (
	fieldTypeIdentifier fieldAccessType = iota
	fieldTypeBracket
)

// parseFieldAccess parses an expression into field access parts
func parseFieldAccess(expression string) ([]fieldAccessPart, error) {
	var parts []fieldAccessPart
	remaining := expression
	
	// Parse the initial identifier
	if remaining == "" {
		return nil, nil
	}
	
	// Find the first part (before any . or [)
	idx := strings.IndexAny(remaining, ".[")
	if idx == -1 {
		// Simple identifier
		parts = append(parts, fieldAccessPart{
			Type:  fieldTypeIdentifier,
			Value: remaining,
		})
		return parts, nil
	}
	
	// Add the initial identifier
	if idx > 0 {
		parts = append(parts, fieldAccessPart{
			Type:  fieldTypeIdentifier,
			Value: remaining[:idx],
		})
		remaining = remaining[idx:]
	}
	
	// Parse remaining parts
	for remaining != "" {
		if strings.HasPrefix(remaining, ".") {
			// Dot notation
			matches := dotRegex.FindStringSubmatch(remaining)
			if len(matches) < 2 {
				return nil, NewEvaluationError(expression, fmt.Errorf("invalid dot notation"))
			}
			parts = append(parts, fieldAccessPart{
				Type:  fieldTypeIdentifier,
				Value: matches[1],
			})
			remaining = remaining[len(matches[0]):]
		} else if strings.HasPrefix(remaining, "[") {
			// Bracket notation
			matches := bracketRegex.FindStringSubmatch(remaining)
			if len(matches) < 2 {
				return nil, NewEvaluationError(expression, fmt.Errorf("invalid bracket notation"))
			}
			parts = append(parts, fieldAccessPart{
				Type:  fieldTypeBracket,
				Value: matches[1],
			})
			remaining = remaining[len(matches[0]):]
		} else {
			return nil, NewEvaluationError(expression, fmt.Errorf("unexpected character"))
		}
	}
	
	return parts, nil
}

// accessMapField accesses a field in a map-like structure
func accessMapField(current interface{}, field string) interface{} {
	if current == nil {
		return nil
	}
	
	switch v := current.(type) {
	case TemplateData:
		return v[field]
	case map[string]interface{}:
		return v[field]
	case map[string]string:
		return v[field]
	case map[string]int:
		return v[field]
	case map[string]float64:
		return v[field]
	case map[string]bool:
		return v[field]
	default:
		// Not a map-like structure
		return nil
	}
}

// accessBracketField accesses a field using bracket notation
func accessBracketField(current interface{}, key string) interface{} {
	if current == nil {
		return nil
	}
	
	// Try to parse as integer for array access
	if idx, err := strconv.Atoi(key); err == nil {
		return accessArrayIndex(current, idx)
	}
	
	// Otherwise treat as string key (remove quotes if present)
	key = strings.Trim(key, `'"`)
	return accessMapField(current, key)
}

// accessArrayIndex accesses an array element by index
func accessArrayIndex(current interface{}, index int) interface{} {
	if current == nil {
		return nil
	}
	
	switch v := current.(type) {
	case []interface{}:
		// Handle negative indices (Python-style)
		if index < 0 {
			index = len(v) + index
		}
		if index >= 0 && index < len(v) {
			return v[index]
		}
	case []string:
		if index < 0 {
			index = len(v) + index
		}
		if index >= 0 && index < len(v) {
			return v[index]
		}
	case []int:
		if index < 0 {
			index = len(v) + index
		}
		if index >= 0 && index < len(v) {
			return v[index]
		}
	case []float64:
		if index < 0 {
			index = len(v) + index
		}
		if index >= 0 && index < len(v) {
			return v[index]
		}
	case []bool:
		if index < 0 {
			index = len(v) + index
		}
		if index >= 0 && index < len(v) {
			return v[index]
		}
	case []map[string]interface{}:
		if index < 0 {
			index = len(v) + index
		}
		if index >= 0 && index < len(v) {
			return v[index]
		}
	}
	
	// Not an array or out of bounds
	return nil
}

// FormatValue converts a value to its string representation
func FormatValue(value interface{}) string {
	if value == nil {
		return ""
	}
	
	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32:
		// Use strconv.FormatFloat with 'g' format and precision 10 for cleaner representation
		return strconv.FormatFloat(float64(v), 'g', 10, 32)
	case float64:
		// Use strconv.FormatFloat with 'g' format and precision 15 for cleaner representation
		// This removes unnecessary trailing zeros and handles precision issues
		return strconv.FormatFloat(v, 'g', 15, 64)
	case bool:
		return fmt.Sprintf("%v", v)
	default:
		// For complex types, use fmt.Sprintf
		return fmt.Sprintf("%v", v)
	}
}