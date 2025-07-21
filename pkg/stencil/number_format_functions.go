package stencil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// formatPattern represents a parsed format pattern
type formatPattern struct {
	raw       string
	segments  []formatSegment
}

type formatSegment struct {
	literal bool
	text    string
	spec    formatSpec
}

type formatSpec struct {
	index     int    // argument index (1-based, 0 means use next)
	flags     string // format flags
	width     string // field width
	precision string // precision
	verb      rune   // format verb
}

var formatSpecRegex = regexp.MustCompile(`%(?:(\d+)\$)?([-#+ 0,(\<]*)?(\d+)?(\.\d+)?([a-zA-Z%])`)

// parseFormatPattern parses a format pattern into segments
func parseFormatPattern(pattern string) (*formatPattern, error) {
	if pattern == "" {
		return nil, fmt.Errorf("empty format pattern")
	}

	result := &formatPattern{raw: pattern}
	lastEnd := 0
	
	matches := formatSpecRegex.FindAllStringSubmatchIndex(pattern, -1)
	for _, match := range matches {
		// Add literal text before this match
		if match[0] > lastEnd {
			result.segments = append(result.segments, formatSegment{
				literal: true,
				text:    pattern[lastEnd:match[0]],
			})
		}
		
		// Parse the format specifier
		spec := formatSpec{}
		
		// Index (match[2] and match[3])
		if match[2] >= 0 {
			spec.index, _ = strconv.Atoi(pattern[match[2]:match[3]])
		}
		
		// Flags (match[4] and match[5])
		if match[4] >= 0 {
			spec.flags = pattern[match[4]:match[5]]
		}
		
		// Width (match[6] and match[7])
		if match[6] >= 0 {
			spec.width = pattern[match[6]:match[7]]
		}
		
		// Precision (match[8] and match[9])
		if match[8] >= 0 {
			spec.precision = pattern[match[8]:match[9]]
		}
		
		// Verb (match[10] and match[11])
		if match[10] >= 0 {
			spec.verb = rune(pattern[match[10]])
		}
		
		result.segments = append(result.segments, formatSegment{
			literal: false,
			spec:    spec,
		})
		
		lastEnd = match[1]
	}
	
	// Add any remaining literal text
	if lastEnd < len(pattern) {
		result.segments = append(result.segments, formatSegment{
			literal: true,
			text:    pattern[lastEnd:],
		})
	}
	
	return result, nil
}

// convertFormatValue converts a value to the appropriate type for the format verb
func convertFormatValue(value interface{}, verb rune) (interface{}, error) {
	if value == nil {
		switch verb {
		case 'd', 'b', 'o', 'x', 'X':
			return 0, nil
		case 'e', 'E', 'f', 'F', 'g', 'G':
			return 0.0, nil
		case 's', 'q', 'v':
			return "null", nil
		case 'c':
			return rune(0), nil
		case 'p':
			return uintptr(0), nil
		case 'U':
			return 0, nil
		default:
			return nil, nil
		}
	}
	
	switch verb {
	case 'd', 'b', 'o', 'x', 'X':
		// Integer formats
		switch v := value.(type) {
		case int:
			return v, nil
		case int8:
			return int(v), nil
		case int16:
			return int(v), nil
		case int32:
			return int(v), nil
		case int64:
			return v, nil
		case uint:
			return int(v), nil
		case uint8:
			return int(v), nil
		case uint16:
			return int(v), nil
		case uint32:
			return int(v), nil
		case uint64:
			return int64(v), nil
		case float32:
			return int(v), nil
		case float64:
			return int(v), nil
		case string:
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				// Try parsing as float and convert
				f, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, fmt.Errorf("cannot convert %q to integer", v)
				}
				return int(f), nil
			}
			return i, nil
		default:
			// Try converting via string
			str := fmt.Sprintf("%v", v)
			return convertFormatValue(str, verb)
		}
		
	case 'e', 'E', 'f', 'F', 'g', 'G':
		// Float formats
		switch v := value.(type) {
		case float32:
			return float64(v), nil
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case int8:
			return float64(v), nil
		case int16:
			return float64(v), nil
		case int32:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case uint:
			return float64(v), nil
		case uint8:
			return float64(v), nil
		case uint16:
			return float64(v), nil
		case uint32:
			return float64(v), nil
		case uint64:
			return float64(v), nil
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert %q to float", v)
			}
			return f, nil
		default:
			// Try converting via string
			str := fmt.Sprintf("%v", v)
			return convertFormatValue(str, verb)
		}
		
	case 's', 'v':
		// String formats
		return FormatValue(value), nil
		
	case 'q':
		// Quoted string format
		return FormatValue(value), nil
		
	case 'c':
		// Character format
		switch v := value.(type) {
		case rune:
			return v, nil
		case int:
			return rune(v), nil
		case string:
			if len(v) > 0 {
				return []rune(v)[0], nil
			}
			return rune(0), nil
		default:
			return rune(0), nil
		}
		
	case 'p':
		// Pointer format
		return fmt.Sprintf("%p", value), nil
		
	case 'U':
		// Unicode format
		switch v := value.(type) {
		case rune:
			return v, nil
		case int:
			return rune(v), nil
		default:
			return rune(0), nil
		}
		
	case '%':
		// Literal percent
		return "%", nil
		
	default:
		return value, nil
	}
}

// formatValue formats a single value according to a format spec
func formatValue(spec formatSpec, value interface{}, locale string) (string, error) {
	// Convert value to appropriate type
	converted, err := convertFormatValue(value, spec.verb)
	if err != nil {
		return "", err
	}
	
	// Build the format string
	var formatStr strings.Builder
	formatStr.WriteRune('%')
	formatStr.WriteString(spec.flags)
	formatStr.WriteString(spec.width)
	formatStr.WriteString(spec.precision)
	formatStr.WriteRune(spec.verb)
	
	// Apply formatting
	result := fmt.Sprintf(formatStr.String(), converted)
	
	// Apply locale-specific formatting for numeric values with comma flag
	if strings.Contains(spec.flags, ",") && locale != "" {
		result = applyLocaleNumberFormatting(result, locale)
	}
	
	return result, nil
}

// applyLocaleNumberFormatting applies locale-specific number formatting
func applyLocaleNumberFormatting(formatted string, locale string) string {
	// Extract language from locale
	lang := strings.Split(locale, "-")[0]
	
	switch lang {
	case "de": // German - period for thousands, comma for decimal
		// Replace commas with a temporary marker
		result := strings.ReplaceAll(formatted, ",", "§")
		// Replace periods with commas (decimal separator)
		result = strings.ReplaceAll(result, ".", ",")
		// Replace temporary markers with periods (thousands separator)
		result = strings.ReplaceAll(result, "§", ".")
		return result
		
	case "fr": // French - space for thousands, comma for decimal
		// Replace commas with spaces (thousands separator)
		result := strings.ReplaceAll(formatted, ",", " ")
		// Replace periods with commas (decimal separator)
		result = strings.ReplaceAll(result, ".", ",")
		return result
		
	case "hu": // Hungarian - space for thousands, comma for decimal
		// Same as French
		result := strings.ReplaceAll(formatted, ",", " ")
		result = strings.ReplaceAll(result, ".", ",")
		return result
		
	default:
		// Keep default formatting
		return formatted
	}
}

// getNumberFormatter returns a function to format numbers for a specific locale
func getNumberFormatter(locale string) func(float64, int) string {
	// Extract language from locale
	lang := strings.Split(strings.ToLower(locale), "-")[0]
	
	return func(value float64, decimals int) string {
		// Format with the specified number of decimals
		format := fmt.Sprintf("%%.%df", decimals)
		result := fmt.Sprintf(format, value)
		
		// Add thousands separators
		parts := strings.Split(result, ".")
		intPart := parts[0]
		decPart := ""
		if len(parts) > 1 {
			decPart = parts[1]
		}
		
		// Handle negative numbers
		negative := false
		if strings.HasPrefix(intPart, "-") {
			negative = true
			intPart = intPart[1:]
		}
		
		// Add thousands separators
		var formatted strings.Builder
		for i, digit := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				switch lang {
				case "de":
					formatted.WriteRune('.')
				case "fr", "hu":
					formatted.WriteRune(' ')
				default:
					formatted.WriteRune(',')
				}
			}
			formatted.WriteRune(digit)
		}
		
		// Rebuild the number
		result = formatted.String()
		if decPart != "" {
			switch lang {
			case "de", "fr", "hu":
				result += "," + decPart
			default:
				result += "." + decPart
			}
		}
		
		if negative {
			result = "-" + result
		}
		
		return result
	}
}

// getCurrencyFormatter returns a function to format currency for a specific locale
func getCurrencyFormatter(locale string) func(float64) string {
	// Parse locale
	parts := strings.Split(strings.ToUpper(locale), "-")
	lang := strings.ToLower(parts[0])
	country := ""
	if len(parts) > 1 {
		country = parts[1]
	}
	
	// Determine currency symbol and placement
	var symbol string
	var before bool
	var space bool
	
	switch {
	case lang == "en" && country == "US":
		symbol = "$"
		before = true
		space = false
	case lang == "en" && country == "GB":
		symbol = "£"
		before = true
		space = false
	case lang == "de" || (lang == "fr" && country == "FR"):
		symbol = "€"
		before = false
		space = true
	case lang == "ja":
		symbol = "¥"
		before = true
		space = false
	case lang == "hu":
		symbol = "Ft"
		before = false
		space = true
	default:
		// Default to dollar
		symbol = "$"
		before = true
		space = false
	}
	
	// Get number formatter for locale
	formatter := getNumberFormatter(locale)
	
	return func(value float64) string {
		// Determine decimals (Japanese Yen has no decimals)
		decimals := 2
		if lang == "ja" {
			decimals = 0
		}
		
		// Format the number
		negative := value < 0
		if negative {
			value = -value
		}
		
		formatted := formatter(value, decimals)
		
		// Add currency symbol
		var result string
		if before {
			if space {
				result = symbol + " " + formatted
			} else {
				result = symbol + formatted
			}
		} else {
			if space {
				result = formatted + " " + symbol
			} else {
				result = formatted + symbol
			}
		}
		
		// Handle negative values
		if negative {
			if before {
				result = "-" + result
			} else {
				result = "-" + formatted
				if space {
					result += " " + symbol
				} else {
					result += symbol
				}
			}
		}
		
		return result
	}
}

// getPercentFormatter returns a function to format percentages for a specific locale
func getPercentFormatter(locale string) func(float64) string {
	// Parse locale
	lang := strings.ToLower(strings.Split(locale, "-")[0])
	
	// Get number formatter for locale
	formatter := getNumberFormatter(locale)
	
	return func(value float64) string {
		// Convert to percentage (multiply by 100)
		percentage := value * 100
		
		// Format based on locale
		var decimals int
		if percentage == float64(int(percentage)) {
			decimals = 0
		} else {
			decimals = 2
		}
		
		formatted := formatter(percentage, decimals)
		
		// Add percent symbol
		switch lang {
		case "de", "fr":
			return formatted + " %"
		default:
			return formatted + "%"
		}
	}
}

// registerNumberFormatFunctions registers all number formatting functions
func registerNumberFormatFunctions(registry *DefaultFunctionRegistry) {
	// format() function - formats values using printf-style patterns
	formatFn := NewSimpleFunction("format", 1, -1, func(args ...interface{}) (interface{}, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("format() expects at least 1 argument")
		}
		
		// Handle nil pattern
		if args[0] == nil {
			return nil, nil
		}
		
		pattern := FormatValue(args[0])
		
		// Parse the pattern
		parsed, err := parseFormatPattern(pattern)
		if err != nil {
			return nil, err
		}
		
		// Count format specifiers
		specCount := 0
		for _, seg := range parsed.segments {
			if !seg.literal && seg.spec.verb != '%' {
				specCount++
			}
		}
		
		// Check argument count
		if len(args)-1 < specCount {
			return nil, fmt.Errorf("format() expects %d values for pattern but got %d", specCount, len(args)-1)
		}
		
		// Build result
		var result strings.Builder
		argIndex := 1
		
		for _, seg := range parsed.segments {
			if seg.literal {
				result.WriteString(seg.text)
			} else if seg.spec.verb == '%' {
				result.WriteRune('%')
			} else {
				// Get the value for this specifier
				var value interface{}
				if seg.spec.index > 0 {
					// Explicit index
					if seg.spec.index > len(args)-1 {
						return nil, fmt.Errorf("format() index %d out of range", seg.spec.index)
					}
					value = args[seg.spec.index]
				} else {
					// Use next argument
					if argIndex >= len(args) {
						return nil, fmt.Errorf("format() not enough arguments")
					}
					value = args[argIndex]
					argIndex++
				}
				
				// Format the value
				formatted, err := formatValue(seg.spec, value, "")
				if err != nil {
					return nil, err
				}
				result.WriteString(formatted)
			}
		}
		
		return result.String(), nil
	})
	registry.RegisterFunction(formatFn)
	
	// formatWithLocale() function - formats with locale support
	formatWithLocaleFn := NewSimpleFunction("formatWithLocale", 2, -1, func(args ...interface{}) (interface{}, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("formatWithLocale() expects at least 2 arguments")
		}
		
		// Handle nil arguments
		if args[0] == nil || args[1] == nil {
			return nil, nil
		}
		
		locale := FormatValue(args[0])
		pattern := FormatValue(args[1])
		
		// Parse the pattern
		parsed, err := parseFormatPattern(pattern)
		if err != nil {
			return nil, err
		}
		
		// Count format specifiers
		specCount := 0
		for _, seg := range parsed.segments {
			if !seg.literal && seg.spec.verb != '%' {
				specCount++
			}
		}
		
		// Check argument count
		if len(args)-2 < specCount {
			return nil, fmt.Errorf("formatWithLocale() expects %d values for pattern but got %d", specCount, len(args)-2)
		}
		
		// Build result
		var result strings.Builder
		argIndex := 2
		
		for _, seg := range parsed.segments {
			if seg.literal {
				result.WriteString(seg.text)
			} else if seg.spec.verb == '%' {
				result.WriteRune('%')
			} else {
				// Get the value for this specifier
				var value interface{}
				if seg.spec.index > 0 {
					// Explicit index (adjust for locale parameter)
					realIndex := seg.spec.index + 1
					if realIndex >= len(args) {
						return nil, fmt.Errorf("formatWithLocale() index %d out of range", seg.spec.index)
					}
					value = args[realIndex]
				} else {
					// Use next argument
					if argIndex >= len(args) {
						return nil, fmt.Errorf("formatWithLocale() not enough arguments")
					}
					value = args[argIndex]
					argIndex++
				}
				
				// Format the value with locale
				formatted, err := formatValue(seg.spec, value, locale)
				if err != nil {
					return nil, err
				}
				result.WriteString(formatted)
			}
		}
		
		return result.String(), nil
	})
	registry.RegisterFunction(formatWithLocaleFn)
	
	// currency() function - formats number as currency
	currencyFn := NewSimpleFunction("currency", 1, 2, func(args ...interface{}) (interface{}, error) {
		if args[0] == nil {
			return nil, nil
		}
		
		// Convert to float
		var value float64
		switch v := args[0].(type) {
		case float64:
			value = v
		case float32:
			value = float64(v)
		case int:
			value = float64(v)
		case int64:
			value = float64(v)
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert %q to number", v)
			}
			value = f
		default:
			// Try converting via string
			str := fmt.Sprintf("%v", v)
			f, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert value to number")
			}
			value = f
		}
		
		// Get locale
		locale := "en-US" // Default
		if len(args) > 1 && args[1] != nil {
			locale = FormatValue(args[1])
		}
		
		// Format as currency
		formatter := getCurrencyFormatter(locale)
		return formatter(value), nil
	})
	registry.RegisterFunction(currencyFn)
	
	// percent() function - formats number as percentage
	percentFn := NewSimpleFunction("percent", 1, 2, func(args ...interface{}) (interface{}, error) {
		if args[0] == nil {
			return nil, nil
		}
		
		// Convert to float
		var value float64
		switch v := args[0].(type) {
		case float64:
			value = v
		case float32:
			value = float64(v)
		case int:
			value = float64(v)
		case int64:
			value = float64(v)
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert %q to number", v)
			}
			value = f
		default:
			// Try converting via string
			str := fmt.Sprintf("%v", v)
			f, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert value to number")
			}
			value = f
		}
		
		// Get locale
		locale := "en-US" // Default
		if len(args) > 1 && args[1] != nil {
			locale = FormatValue(args[1])
		}
		
		// Format as percentage
		formatter := getPercentFormatter(locale)
		return formatter(value), nil
	})
	registry.RegisterFunction(percentFn)
}

// toNumber converts various types to float64
func toNumber(val interface{}) (float64, error) {
	if val == nil {
		return 0, nil
	}
	
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", val)
	}
}