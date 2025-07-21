package stencil

import (
	"fmt"
	"strings"
)

// LinkReplacementMarker marks a location where a link should be replaced
type LinkReplacementMarker struct {
	URL string
}

func (l LinkReplacementMarker) isMarker() bool {
	return true
}

func (l LinkReplacementMarker) String() string {
	return fmt.Sprintf("[LINK:%s]", l.URL)
}

func replaceLinkFunc(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("replaceLink expects exactly 1 argument, got %d", len(args))
	}
	
	url, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("replaceLink expects a string argument")
	}
	
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("replaceLink: URL cannot be empty")
	}
	
	return LinkReplacementMarker{URL: url}, nil
}

func registerLinkFunctions(registry *DefaultFunctionRegistry) {
	// replaceLink function
	replaceLinkFn := NewSimpleFunction("replaceLink", 1, 1, replaceLinkFunc)
	registry.RegisterFunction(replaceLinkFn)
}