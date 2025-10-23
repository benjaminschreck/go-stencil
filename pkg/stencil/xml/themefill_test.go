package xml

import (
	"encoding/xml"
	"strings"
	"testing"
)

// TestShadingThemeFillMarshaling tests that the ThemeFill attribute is preserved during XML marshaling
func TestShadingThemeFillMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		shading  Shading
		expected string
	}{
		{
			name: "shading with themeFill",
			shading: Shading{
				Val:       "clear",
				Color:     "auto",
				Fill:      "E8E8E8",
				ThemeFill: "background2",
			},
			expected: `w:themeFill="background2"`,
		},
		{
			name: "shading without themeFill",
			shading: Shading{
				Val:   "clear",
				Color: "auto",
				Fill:  "FFFFFF",
			},
			expected: `w:fill="FFFFFF"`,
		},
		{
			name: "shading with all attributes",
			shading: Shading{
				Val:       "solid",
				Color:     "000000",
				Fill:      "FFFF00",
				ThemeFill: "accent1",
			},
			expected: `w:themeFill="accent1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal the shading
			data, err := xml.Marshal(tt.shading)
			if err != nil {
				t.Fatalf("Failed to marshal shading: %v", err)
			}

			result := string(data)

			// Check if expected string is in the result
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected marshaled XML to contain %q, got: %s", tt.expected, result)
			}

			// If themeFill is set, verify it's in the output
			if tt.shading.ThemeFill != "" {
				expectedAttr := `w:themeFill="` + tt.shading.ThemeFill + `"`
				if !strings.Contains(result, expectedAttr) {
					t.Errorf("ThemeFill attribute not preserved. Expected %q in output: %s", expectedAttr, result)
				}
			}
		})
	}
}

// TestShadingThemeFillUnmarshaling tests that the ThemeFill attribute is correctly parsed from XML
func TestShadingThemeFillUnmarshaling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Shading
	}{
		{
			name:  "parse themeFill",
			input: `<w:shd w:val="clear" w:color="auto" w:fill="E8E8E8" w:themeFill="background2"/>`,
			expected: Shading{
				Val:       "clear",
				Color:     "auto",
				Fill:      "E8E8E8",
				ThemeFill: "background2",
			},
		},
		{
			name:  "parse without themeFill",
			input: `<w:shd w:val="clear" w:color="auto" w:fill="FFFFFF"/>`,
			expected: Shading{
				Val:   "clear",
				Color: "auto",
				Fill:  "FFFFFF",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var shading Shading
			err := xml.Unmarshal([]byte(tt.input), &shading)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if shading.Val != tt.expected.Val {
				t.Errorf("Val: got %q, want %q", shading.Val, tt.expected.Val)
			}
			if shading.Color != tt.expected.Color {
				t.Errorf("Color: got %q, want %q", shading.Color, tt.expected.Color)
			}
			if shading.Fill != tt.expected.Fill {
				t.Errorf("Fill: got %q, want %q", shading.Fill, tt.expected.Fill)
			}
			if shading.ThemeFill != tt.expected.ThemeFill {
				t.Errorf("ThemeFill: got %q, want %q", shading.ThemeFill, tt.expected.ThemeFill)
			}
		})
	}
}

// TestTableCellPropertiesWithThemeFill tests complete cell properties marshaling/unmarshaling
func TestTableCellPropertiesWithThemeFill(t *testing.T) {
	input := `<w:tcPr>
		<w:tcW w:type="dxa" w:w="2974"/>
		<w:shd w:val="clear" w:color="auto" w:fill="E8E8E8" w:themeFill="background2"/>
	</w:tcPr>`

	var props TableCellProperties
	err := xml.Unmarshal([]byte(input), &props)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify shading was parsed
	if props.Shading == nil {
		t.Fatal("Shading is nil")
	}

	if props.Shading.ThemeFill != "background2" {
		t.Errorf("ThemeFill: got %q, want %q", props.Shading.ThemeFill, "background2")
	}

	// Marshal it back
	data, err := xml.Marshal(props)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	result := string(data)

	// Verify themeFill is in the marshaled output
	if !strings.Contains(result, `w:themeFill="background2"`) {
		t.Errorf("ThemeFill not preserved in marshaled output: %s", result)
	}
}
