package stencil

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

// Function represents a callable function in templates
type Function interface {
	// Call executes the function with the given arguments
	Call(args ...interface{}) (interface{}, error)
	
	// Name returns the function name
	Name() string
	
	// MinArgs returns the minimum number of arguments required
	MinArgs() int
	
	// MaxArgs returns the maximum number of arguments allowed (-1 for unlimited)
	MaxArgs() int
}

// FunctionRegistry manages available functions
type FunctionRegistry interface {
	// RegisterFunction adds a function to the registry
	RegisterFunction(fn Function) error
	
	// GetFunction retrieves a function by name
	GetFunction(name string) (Function, bool)
	
	// ListFunctions returns all registered function names
	ListFunctions() []string
}

// DefaultFunctionRegistry is the default implementation of FunctionRegistry
type DefaultFunctionRegistry struct {
	functions map[string]Function
	mutex     sync.RWMutex
}

// NewFunctionRegistry creates a new function registry
func NewFunctionRegistry() *DefaultFunctionRegistry {
	return &DefaultFunctionRegistry{
		functions: make(map[string]Function),
	}
}

func (r *DefaultFunctionRegistry) RegisterFunction(fn Function) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	name := fn.Name()
	if name == "" {
		return fmt.Errorf("function name cannot be empty")
	}
	
	r.functions[name] = fn
	return nil
}

func (r *DefaultFunctionRegistry) GetFunction(name string) (Function, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	fn, exists := r.functions[name]
	return fn, exists
}

func (r *DefaultFunctionRegistry) ListFunctions() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	names := make([]string, 0, len(r.functions))
	for name := range r.functions {
		names = append(names, name)
	}
	return names
}

// GlobalFunctionRegistry is the default global registry
var globalRegistry *DefaultFunctionRegistry
var registryOnce sync.Once

// GetDefaultFunctionRegistry returns the default global function registry
func GetDefaultFunctionRegistry() FunctionRegistry {
	registryOnce.Do(func() {
		globalRegistry = NewFunctionRegistry()
		// Register basic functions
		registerBasicFunctions(globalRegistry)
	})
	return globalRegistry
}

// SimpleFunctionImpl provides a basic implementation of Function
type SimpleFunctionImpl struct {
	name    string
	minArgs int
	maxArgs int
	handler func(args ...interface{}) (interface{}, error)
}

func NewSimpleFunction(name string, minArgs, maxArgs int, handler func(args ...interface{}) (interface{}, error)) Function {
	return &SimpleFunctionImpl{
		name:    name,
		minArgs: minArgs,
		maxArgs: maxArgs,
		handler: handler,
	}
}

func (f *SimpleFunctionImpl) Call(args ...interface{}) (interface{}, error) {
	// Validate argument count
	argCount := len(args)
	if argCount < f.minArgs {
		return nil, fmt.Errorf("function %s requires at least %d arguments, got %d", f.name, f.minArgs, argCount)
	}
	if f.maxArgs >= 0 && argCount > f.maxArgs {
		return nil, fmt.Errorf("function %s accepts at most %d arguments, got %d", f.name, f.maxArgs, argCount)
	}
	
	return f.handler(args...)
}

func (f *SimpleFunctionImpl) Name() string {
	return f.name
}

func (f *SimpleFunctionImpl) MinArgs() int {
	return f.minArgs
}

func (f *SimpleFunctionImpl) MaxArgs() int {
	return f.maxArgs
}

// registerBasicFunctions registers the basic built-in functions
func registerBasicFunctions(registry *DefaultFunctionRegistry) {
	// Register date functions
	registerDateFunctions(registry)
	
	// Register number format functions
	registerNumberFormatFunctions(registry)
	
	// Register HTML functions
	registerHTMLFunction(registry)
	
	// Register XML functions
	registerXMLFunction(registry)
	
	// Register table row functions
	registerTableRowFunctions(registry)
	
	// Register table column functions
	registerTableColumnFunctions(registry)
	
	// Register image functions
	registerImageFunctions(registry)
	
	// Register link functions
	registerLinkFunctions(registry)
	
	// empty() function - checks if a value is empty
	emptyFn := NewSimpleFunction("empty", 1, 1, func(args ...interface{}) (interface{}, error) {
		return isEmpty(args[0]), nil
	})
	registry.RegisterFunction(emptyFn)
	
	// coalesce() function - returns first non-empty value
	coalesceFn := NewSimpleFunction("coalesce", 1, -1, func(args ...interface{}) (interface{}, error) {
		for _, arg := range args {
			if !isEmpty(arg) {
				return arg, nil
			}
		}
		return nil, nil
	})
	registry.RegisterFunction(coalesceFn)
	
	// list() function - creates a list from arguments
	listFn := NewSimpleFunction("list", 0, -1, func(args ...interface{}) (interface{}, error) {
		return args, nil
	})
	registry.RegisterFunction(listFn)
	
	// data() function - returns the entire data context
	dataFn := NewSimpleFunction("data", 0, 0, func(args ...interface{}) (interface{}, error) {
		// Note: The actual data will be injected during evaluation
		// This is a placeholder that will be handled specially
		return nil, fmt.Errorf("data() function requires special handling")
	})
	registry.RegisterFunction(dataFn)
	
	// map() function - extracts values from a collection by path
	mapFn := NewSimpleFunction("map", 2, 2, func(args ...interface{}) (interface{}, error) {
		path, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("first parameter of map() must be a string")
		}
		
		collection := args[1]
		return mapExtract(path, collection)
	})
	registry.RegisterFunction(mapFn)
	
	// str() function - converts value to string
	strFn := NewSimpleFunction("str", 1, 1, func(args ...interface{}) (interface{}, error) {
		if args[0] == nil {
			return "", nil
		}
		return FormatValue(args[0]), nil
	})
	registry.RegisterFunction(strFn)
	
	// integer() function - converts value to integer
	integerFn := NewSimpleFunction("integer", 1, 1, func(args ...interface{}) (interface{}, error) {
		return toInteger(args[0])
	})
	registry.RegisterFunction(integerFn)
	
	// decimal() function - converts value to decimal (float64)
	decimalFn := NewSimpleFunction("decimal", 1, 1, func(args ...interface{}) (interface{}, error) {
		return toDecimal(args[0])
	})
	registry.RegisterFunction(decimalFn)
	
	// lowercase() function - converts string to lowercase
	lowercaseFn := NewSimpleFunction("lowercase", 1, 1, func(args ...interface{}) (interface{}, error) {
		if args[0] == nil {
			return nil, nil
		}
		return strings.ToLower(FormatValue(args[0])), nil
	})
	registry.RegisterFunction(lowercaseFn)
	
	// uppercase() function - converts string to uppercase
	uppercaseFn := NewSimpleFunction("uppercase", 1, 1, func(args ...interface{}) (interface{}, error) {
		if args[0] == nil {
			return nil, nil
		}
		return strings.ToUpper(FormatValue(args[0])), nil
	})
	registry.RegisterFunction(uppercaseFn)
	
	// titlecase() function - converts string to title case
	titlecaseFn := NewSimpleFunction("titlecase", 1, 1, func(args ...interface{}) (interface{}, error) {
		if args[0] == nil {
			return nil, nil
		}
		return toTitleCase(FormatValue(args[0])), nil
	})
	registry.RegisterFunction(titlecaseFn)
	
	// join() function - joins collection elements with optional separator
	joinFn := NewSimpleFunction("join", 1, 2, func(args ...interface{}) (interface{}, error) {
		// Get the collection
		collection := args[0]
		if collection == nil {
			return "", nil
		}
		
		// Convert to slice
		items, err := toSlice(collection)
		if err != nil {
			// If it's not a collection, check if it's a single value we should convert
			// The original Stencil requires a collection, so we return error
			return nil, fmt.Errorf("first parameter must be a collection")
		}
		
		// Get separator if provided
		separator := ""
		if len(args) > 1 {
			if args[1] == nil {
				separator = ""
			} else if sep, ok := args[1].(string); ok {
				separator = sep
			} else {
				return nil, fmt.Errorf("second parameter must be a string")
			}
		}
		
		// Join the items
		var result []string
		for _, item := range items {
			if item != nil {
				result = append(result, FormatValue(item))
			}
		}
		
		return strings.Join(result, separator), nil
	})
	registry.RegisterFunction(joinFn)
	
	// joinAnd() function - joins with two separators
	joinAndFn := NewSimpleFunction("joinAnd", 3, 3, func(args ...interface{}) (interface{}, error) {
		// Get the collection
		collection := args[0]
		if collection == nil {
			return "", nil
		}
		
		// Convert to slice
		items, err := toSlice(collection)
		if err != nil {
			return nil, fmt.Errorf("first parameter must be a collection")
		}
		
		// Get separators
		sep1, ok1 := args[1].(string)
		if !ok1 {
			return nil, fmt.Errorf("second parameter must be a string")
		}
		
		sep2, ok2 := args[2].(string)
		if !ok2 {
			return nil, fmt.Errorf("third parameter must be a string")
		}
		
		// Filter nil values and convert to strings
		var strItems []string
		for _, item := range items {
			if item != nil {
				strItems = append(strItems, FormatValue(item))
			}
		}
		
		// Join based on count
		switch len(strItems) {
		case 0:
			return "", nil
		case 1:
			return strItems[0], nil
		case 2:
			return strItems[0] + sep2 + strItems[1], nil
		default:
			// Join all but last with sep1, then add last with sep2
			return strings.Join(strItems[:len(strItems)-1], sep1) + sep2 + strItems[len(strItems)-1], nil
		}
	})
	registry.RegisterFunction(joinAndFn)
	
	// replace() function - replaces all occurrences of pattern with replacement
	replaceFn := NewSimpleFunction("replace", 3, 3, func(args ...interface{}) (interface{}, error) {
		// Get text
		text := ""
		if args[0] != nil {
			text = FormatValue(args[0])
		}
		
		// Get pattern - if nil, don't do any replacement
		if args[1] == nil {
			return text, nil
		}
		pattern := FormatValue(args[1])
		
		// Get replacement
		replacement := ""
		if args[2] != nil {
			replacement = FormatValue(args[2])
		}
		
		// Perform replacement
		return strings.ReplaceAll(text, pattern, replacement), nil
	})
	registry.RegisterFunction(replaceFn)
	
	// length() function - returns the length of a value
	lengthFn := NewSimpleFunction("length", 1, 1, func(args ...interface{}) (interface{}, error) {
		val := args[0]
		if val == nil {
			return 0, nil
		}
		
		switch v := val.(type) {
		case string:
			// For strings, return the number of runes (Unicode code points)
			return len([]rune(v)), nil
		case []interface{}:
			return len(v), nil
		case []string:
			return len(v), nil
		case []int:
			return len(v), nil
		case []float64:
			return len(v), nil
		case []bool:
			return len(v), nil
		case []map[string]interface{}:
			return len(v), nil
		case map[string]interface{}:
			return len(v), nil
		default:
			// For other types, convert to string and return its length
			str := FormatValue(v)
			return len([]rune(str)), nil
		}
	})
	registry.RegisterFunction(lengthFn)
	
	// round() function - rounds a number to the nearest integer
	roundFn := NewSimpleFunction("round", 1, 1, func(args ...interface{}) (interface{}, error) {
		return mathRound(args[0])
	})
	registry.RegisterFunction(roundFn)
	
	// floor() function - rounds down to closest smaller integer
	floorFn := NewSimpleFunction("floor", 1, 1, func(args ...interface{}) (interface{}, error) {
		return mathFloor(args[0])
	})
	registry.RegisterFunction(floorFn)
	
	// ceil() function - rounds up to closest bigger integer
	ceilFn := NewSimpleFunction("ceil", 1, 1, func(args ...interface{}) (interface{}, error) {
		return mathCeil(args[0])
	})
	registry.RegisterFunction(ceilFn)
	
	// sum() function - sums numbers in a list
	sumFn := NewSimpleFunction("sum", 1, 1, func(args ...interface{}) (interface{}, error) {
		return sumList(args[0])
	})
	registry.RegisterFunction(sumFn)
	
	// contains() function - checks if a list contains a value
	containsFn := NewSimpleFunction("contains", 2, 2, func(args ...interface{}) (interface{}, error) {
		return containsValue(args[0], args[1])
	})
	registry.RegisterFunction(containsFn)
	
	// pageBreak() function - inserts a page break
	pageBreakFn := NewSimpleFunction("pageBreak", 0, 0, func(args ...interface{}) (interface{}, error) {
		// Create a page break using the Break struct with type="page"
		pageBreak := &Break{Type: "page"}
		return &OOXMLFragment{Content: pageBreak}, nil
	})
	registry.RegisterFunction(pageBreakFn)
	
	// range() function - creates a range of numbers
	rangeFn := NewSimpleFunction("range", 1, 3, func(args ...interface{}) (interface{}, error) {
		return createRange(args...)
	})
	registry.RegisterFunction(rangeFn)
	
	// switch() function - pattern matching with case values
	switchFn := NewSimpleFunction("switch", 3, -1, func(args ...interface{}) (interface{}, error) {
		return switchFunction(args...)
	})
	registry.RegisterFunction(switchFn)
}

// isEmpty checks if a value is considered empty
func isEmpty(val interface{}) bool {
	if val == nil {
		return true
	}
	
	switch v := val.(type) {
	case bool:
		return !v
	case int, int8, int16, int32, int64:
		return v == 0
	case uint, uint8, uint16, uint32, uint64:
		return v == 0
	case float32, float64:
		return v == 0.0
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case []string:
		return len(v) == 0
	case []int:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false // Non-nil objects are not empty
	}
}

// OOXMLFragment represents a fragment of OOXML content to be inserted
type OOXMLFragment struct {
	Content interface{} // The OOXML content (e.g., Break, etc.)
}

// FunctionProvider interface allows for providing custom functions during template rendering
type FunctionProvider interface {
	// ProvideFunctions returns a map of function name to Function implementation
	ProvideFunctions() map[string]Function
}


// CreateRegistryWithProvider creates a new registry and registers functions from a provider
func CreateRegistryWithProvider(provider FunctionProvider) (FunctionRegistry, error) {
	registry := NewFunctionRegistry()
	
	// Register basic functions first
	registerBasicFunctions(registry)
	
	// Register functions from provider
	functions := provider.ProvideFunctions()
	for _, fn := range functions {
		if err := registry.RegisterFunction(fn); err != nil {
			return nil, err
		}
	}
	
	return registry, nil
}

// CallFunction is a helper to call a function by name with arguments
func CallFunction(name string, data TemplateData, args ...interface{}) (interface{}, error) {
	// Special handling for data() function
	if name == "data" && len(args) == 0 {
		return data, nil
	}
	
	var registry FunctionRegistry
	if reg, ok := data["__functions__"]; ok {
		if funcReg, ok := reg.(FunctionRegistry); ok {
			registry = funcReg
		}
	}
	
	if registry == nil {
		registry = GetDefaultFunctionRegistry()
	}
	
	fn, exists := registry.GetFunction(name)
	if !exists {
		return nil, fmt.Errorf("unknown function: %s", name)
	}
	
	return fn.Call(args...)
}

// mapExtract extracts values from a collection following a path
func mapExtract(path string, data interface{}) (interface{}, error) {
	if data == nil {
		return []interface{}{}, nil
	}
	
	// Split path by dots
	parts := strings.Split(path, ".")
	
	// Start with the initial data wrapped in a slice
	var current []interface{}
	
	// Convert initial data to slice
	switch v := data.(type) {
	case []interface{}:
		current = v
	case []string:
		current = make([]interface{}, len(v))
		for i, item := range v {
			current[i] = item
		}
	case []int:
		current = make([]interface{}, len(v))
		for i, item := range v {
			current[i] = item
		}
	case []map[string]interface{}:
		current = make([]interface{}, len(v))
		for i, item := range v {
			current[i] = item
		}
	default:
		// If not a slice, wrap in a slice
		current = []interface{}{data}
	}
	
	// Process each part of the path
	for _, part := range parts {
		if part == "" {
			continue
		}
		
		var next []interface{}
		
		for _, item := range current {
			if item == nil {
				continue
			}
			
			// For maps, extract the field value
			switch v := item.(type) {
			case map[string]interface{}:
				if val, exists := v[part]; exists {
					// If the value is a slice, flatten it
					switch vv := val.(type) {
					case []interface{}:
						next = append(next, vv...)
					case []string:
						for _, s := range vv {
							next = append(next, s)
						}
					case []int:
						for _, i := range vv {
							next = append(next, i)
						}
					case []map[string]interface{}:
						for _, m := range vv {
							next = append(next, m)
						}
					default:
						next = append(next, val)
					}
				}
			case []interface{}:
				// If we have a nested array, we need to flatten
				for _, nested := range v {
					result, _ := mapExtract(part, nested)
					if resultSlice, ok := result.([]interface{}); ok {
						next = append(next, resultSlice...)
					}
				}
			}
		}
		
		current = next
	}
	
	return current, nil
}

// toInteger converts various types to integer
func toInteger(val interface{}) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	
	switch v := val.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		// Try to parse as integer first
		if i, err := strconv.Atoi(v); err == nil {
			return i, nil
		}
		// Try to parse as float and convert
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return int(f), nil
		}
		return nil, fmt.Errorf("cannot convert string %q to integer", v)
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to integer", val)
	}
}

// toDecimal converts various types to decimal (float64)
func toDecimal(val interface{}) (interface{}, error) {
	if val == nil {
		return nil, nil
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
		// Try to parse as float
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, nil
		}
		return nil, fmt.Errorf("cannot convert string %q to decimal", v)
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to decimal", val)
	}
}

// toTitleCase converts a string to title case (capitalize first letter of each word)
func toTitleCase(s string) string {
	// Handle empty string
	if s == "" {
		return ""
	}
	
	// Split the string by spaces while preserving the spaces
	var result strings.Builder
	words := strings.Fields(s)
	
	// If no words found (e.g., only spaces), return original
	if len(words) == 0 {
		return s
	}
	
	// Find the position of each word and capitalize it
	lastEnd := 0
	for _, word := range words {
		// Find where this word starts in the original string
		start := strings.Index(s[lastEnd:], word) + lastEnd
		
		// Add any spaces/characters between words
		if start > lastEnd {
			result.WriteString(s[lastEnd:start])
		}
		
		// Capitalize the first letter of the word
		if len(word) > 0 {
			// Handle Unicode properly
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			// Make the rest lowercase
			for i := 1; i < len(runes); i++ {
				runes[i] = unicode.ToLower(runes[i])
			}
			result.WriteString(string(runes))
		}
		
		lastEnd = start + len(word)
	}
	
	// Add any trailing spaces/characters
	if lastEnd < len(s) {
		result.WriteString(s[lastEnd:])
	}
	
	return result.String()
}

// mathRound rounds a number to the nearest integer
func mathRound(val interface{}) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	
	// Convert to float64
	num, err := toNumber(val)
	if err != nil {
		return nil, err
	}
	
	// Round and return as int to match Go expression evaluation
	return int(math.Round(num)), nil
}

// mathFloor rounds down to the closest smaller integer
func mathFloor(val interface{}) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	
	// Convert to float64
	num, err := toNumber(val)
	if err != nil {
		return nil, err
	}
	
	// Floor and return as int to match Go expression evaluation
	return int(math.Floor(num)), nil
}

// mathCeil rounds up to the closest bigger integer
func mathCeil(val interface{}) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	
	// Convert to float64
	num, err := toNumber(val)
	if err != nil {
		return nil, err
	}
	
	// Ceil and return as int to match Go expression evaluation
	return int(math.Ceil(num)), nil
}

// sumList sums all numbers in a list
func sumList(val interface{}) (interface{}, error) {
	if val == nil {
		return 0, nil
	}
	
	// Check if it's a valid list type (not string, which gets converted to char array)
	switch val.(type) {
	case string:
		return nil, fmt.Errorf("sum() requires a list, got %T", val)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return nil, fmt.Errorf("sum() requires a list, got %T", val)
	case float32, float64:
		return nil, fmt.Errorf("sum() requires a list, got %T", val)
	case bool:
		return nil, fmt.Errorf("sum() requires a list, got %T", val)
	}
	
	// Convert to slice
	items, err := toSlice(val)
	if err != nil {
		return nil, fmt.Errorf("sum() requires a list, got %T", val)
	}
	
	// Handle empty slice
	if len(items) == 0 {
		return 0, nil
	}
	
	// Sum the items
	var sum float64 = 0
	hasFloat := false
	
	for _, item := range items {
		if item == nil {
			continue // Skip nil values
		}
		
		// Convert item to number
		num, err := toNumber(item)
		if err != nil {
			return nil, fmt.Errorf("sum() cannot convert item %v to number: %w", item, err)
		}
		
		sum += num
		
		// Check if we have any float values
		if _, isFloat := item.(float64); isFloat {
			hasFloat = true
		}
		if _, isFloat := item.(float32); isFloat {
			hasFloat = true
		}
		if str, isStr := item.(string); isStr {
			if _, err := strconv.ParseFloat(str, 64); err == nil && strings.Contains(str, ".") {
				hasFloat = true
			}
		}
	}
	
	// Return int if all inputs were integers, float otherwise
	if !hasFloat && sum == float64(int(sum)) {
		return int(sum), nil
	}
	return sum, nil
}

// containsValue checks if a list contains a specific value using string comparison
func containsValue(searchVal, listVal interface{}) (interface{}, error) {
	// Handle nil list
	if listVal == nil {
		return false, nil
	}
	
	// Check if it's a valid list type (not string, which gets converted to char array)
	switch listVal.(type) {
	case string:
		return nil, fmt.Errorf("contains() second parameter must be a list, got %T", listVal)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return nil, fmt.Errorf("contains() second parameter must be a list, got %T", listVal)
	case float32, float64:
		return nil, fmt.Errorf("contains() second parameter must be a list, got %T", listVal)
	case bool:
		return nil, fmt.Errorf("contains() second parameter must be a list, got %T", listVal)
	}
	
	// Convert to slice
	items, err := toSlice(listVal)
	if err != nil {
		return nil, fmt.Errorf("contains() second parameter must be a list, got %T", listVal)
	}
	
	// Convert search value to string for comparison
	searchStr := FormatValue(searchVal)
	
	// Check each item in the list
	for _, item := range items {
		itemStr := FormatValue(item)
		if searchStr == itemStr {
			return true, nil
		}
	}
	
	return false, nil
}

// createRange creates a range of numbers based on the arguments provided
func createRange(args ...interface{}) (interface{}, error) {
	switch len(args) {
	case 1:
		// range(n) - from 0 to n-1
		end, err := toNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("range() argument must be a number")
		}
		return rangeNumbers(0, int(end), 1)
		
	case 2:
		// range(start, end) - from start to end-1
		start, err := toNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("range() first argument must be a number")
		}
		end, err := toNumber(args[1])
		if err != nil {
			return nil, fmt.Errorf("range() second argument must be a number")
		}
		return rangeNumbers(int(start), int(end), 1)
		
	case 3:
		// range(start, end, step) - from start to end-1 with step
		start, err := toNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("range() first argument must be a number")
		}
		end, err := toNumber(args[1])
		if err != nil {
			return nil, fmt.Errorf("range() second argument must be a number")
		}
		step, err := toNumber(args[2])
		if err != nil {
			return nil, fmt.Errorf("range() third argument must be a number")
		}
		stepInt := int(step)
		if stepInt == 0 {
			return nil, fmt.Errorf("range() step cannot be zero")
		}
		return rangeNumbers(int(start), int(end), stepInt)
		
	default:
		return nil, fmt.Errorf("range() requires 1-3 arguments, got %d", len(args))
	}
}

// rangeNumbers generates a slice of integers from start to end (exclusive) with given step
func rangeNumbers(start, end, step int) ([]interface{}, error) {
	if step == 0 {
		return nil, fmt.Errorf("step cannot be zero")
	}
	
	var result []interface{}
	
	if step > 0 {
		// Positive step: start < end
		for i := start; i < end; i += step {
			result = append(result, i)
		}
	} else {
		// Negative step: start > end
		for i := start; i > end; i += step {
			result = append(result, i)
		}
	}
	
	return result, nil
}

// switchFunction implements switch-case functionality matching the original Stencil behavior
func switchFunction(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("switch() requires at least 3 arguments (expression, case, value), got %d", len(args))
	}
	
	// Get the expression to match against
	expr := args[0]
	
	// Iterate through case-value pairs (starting from index 1, step by 2)
	for i := 1; i < len(args); i += 2 {
		// Make sure we have both case and value
		if i+1 >= len(args) {
			// This means we have an odd number of arguments after the expression
			// The last argument should be the default value
			if len(args)%2 == 0 {
				// Even total arguments means the last one is a default value
				return args[len(args)-1], nil
			} else {
				// Odd total arguments means no default value
				return nil, nil
			}
		}
		
		caseValue := args[i]
		returnValue := args[i+1]
		
		// Check for match using the same logic as the original Stencil
		if matchesCase(expr, caseValue) {
			return returnValue, nil
		}
	}
	
	// No match found - check if there's a default value
	if len(args)%2 == 0 {
		// Even number of total arguments means the last argument is a default value
		return args[len(args)-1], nil
	} else {
		// Odd number of total arguments means no default value
		return nil, nil
	}
}

// matchesCase checks if the expression matches the case value using the same logic as original Stencil
func matchesCase(expr, caseValue interface{}) (result bool) {
	// Handle nil cases: both nil should match
	if expr == nil && caseValue == nil {
		return true
	}
	
	// If only one is nil, they don't match
	if expr == nil || caseValue == nil {
		return false
	}
	
	// For non-nil values, use Go's equality comparison
	// But we need to handle uncomparable types (slices, maps, functions)
	// which will panic if compared with ==
	defer func() {
		// If we panic due to uncomparable types, recover and return false
		if r := recover(); r != nil {
			// This happens when comparing uncomparable types like slices, maps, functions
			// In this case, they don't match (same behavior as original Stencil)
			result = false
		}
	}()
	
	// This handles all comparable primitive types (string, int, float, bool, etc.)
	// For uncomparable types (slices, maps), it will panic and be caught by recover
	return expr == caseValue
}