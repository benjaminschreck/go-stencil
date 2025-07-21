package stencil

import (
	"fmt"
)

// replaceImageFunc implements the replaceImage function
// It accepts a data URI string and returns an image replacement marker
func replaceImageFunc(args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("replaceImage requires exactly 1 argument")
	}

	dataURI, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("replaceImage argument must be a string")
	}

	// Parse the data URI
	mimeType, data, err := parseDataURI(dataURI)
	if err != nil {
		return nil, err
	}

	// Create and return an image replacement marker
	// This will be processed later in the rendering pipeline
	return &imageReplacementMarker{
		mimeType: mimeType,
		data:     data,
	}, nil
}

// imageReplacementMarker is returned by replaceImage function
// It's processed during the post-processing phase
type imageReplacementMarker struct {
	mimeType string
	data     []byte
}

// String implements the Stringer interface for imageReplacementMarker
func (m *imageReplacementMarker) String() string {
	return fmt.Sprintf("[Image: %s, %d bytes]", m.mimeType, len(m.data))
}

// registerImageFunctions registers all image-related functions
func registerImageFunctions(registry FunctionRegistry) {
	// replaceImage function
	replaceImageFn := NewSimpleFunction("replaceImage", 1, 1, replaceImageFunc)
	registry.RegisterFunction(replaceImageFn)
}