package stencil

import (
	"strings"
	"testing"
)

func TestParseControlStructures(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantAST  string
		wantErr  bool
	}{
		{
			name:    "simple text",
			content: "Hello World",
			wantAST: `[Text("Hello World")]`,
		},
		{
			name:    "simple variable",
			content: "Hello {{name}}",
			wantAST: `[Text("Hello ") Expression(Variable(name))]`,
		},
		{
			name:    "simple if statement",
			content: "{{if condition}}Yes{{end}}",
			wantAST: `[If(Variable(condition))]`,
		},
		{
			name:    "if-else statement",
			content: "{{if condition}}Yes{{else}}No{{end}}",
			wantAST: `[If(Variable(condition)) Else]`,
		},
		{
			name:    "if-elsif-else statement",
			content: "{{if x > 5}}Big{{elsif x > 0}}Small{{else}}Zero{{end}}",
			wantAST: `[If(BinaryOp(Variable(x) > Literal(5))) ElsIf(BinaryOp(Variable(x) > Literal(0))) Else]`,
		},
		{
			name:    "unless statement",
			content: "{{unless condition}}No{{end}}",
			wantAST: `[Unless(Variable(condition))]`,
		},
		{
			name:    "unless-else statement",
			content: "{{unless condition}}No{{else}}Yes{{end}}",
			wantAST: `[Unless(Variable(condition))]`,
		},
		{
			name:    "simple for loop",
			content: "{{for item in items}}{{item}}{{end}}",
			wantAST: `[For(item in Variable(items))]`,
		},
		{
			name:    "indexed for loop",
			content: "{{for i, item in items}}{{i}}: {{item}}{{end}}",
			wantAST: `[For(i, item in Variable(items))]`,
		},
		{
			name:    "nested if statements",
			content: "{{if outer}}{{if inner}}Both{{end}}{{end}}",
			wantAST: `[If(Variable(outer))]`,
		},
		{
			name:    "mixed content",
			content: "Start {{if show}}{{name}}{{end}} End",
			wantAST: `[Text("Start ") If(Variable(show)) Text(" End")]`,
		},
		{
			name:    "complex expression in if",
			content: "{{if age >= 18 & hasLicense}}Can drive{{end}}",
			wantAST: `[If(BinaryOp(BinaryOp(Variable(age) >= Literal(18)) & Variable(hasLicense)))]`,
		},
		{
			name:    "arithmetic in variable",
			content: "Total: {{price * quantity + tax}}",
			wantAST: `[Text("Total: ") Expression(BinaryOp(BinaryOp(Variable(price) * Variable(quantity)) + Variable(tax)))]`,
		},
		{
			name:    "string comparison with German quotes",
			content: "{{if haftung == \u201EHaftung klar individuelle Quote\u201C}}Match{{end}}",
			wantAST: `[If(BinaryOp(Variable(haftung) == Literal("Haftung klar individuelle Quote")))]`,
		},
		{
			name:    "string comparison with French quotes",
			content: `{{if status == »active«}}Active{{end}}`,
			wantAST: `[If(BinaryOp(Variable(status) == Literal("active")))]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structures, err := ParseControlStructures(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseControlStructures() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				got := formatStructures(structures)
				if got != tt.wantAST {
					t.Errorf("ParseControlStructures() = %v, want %v", got, tt.wantAST)
				}
			}
		})
	}
}

func TestIfNodeRender(t *testing.T) {
	tests := []struct {
		name    string
		node    *IfNode
		data    TemplateData
		want    string
		wantErr bool
	}{
		{
			name: "simple if true",
			node: &IfNode{
				Condition: &LiteralNode{Value: true},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Yes"},
				},
			},
			data: TemplateData{},
			want: "Yes",
		},
		{
			name: "simple if false",
			node: &IfNode{
				Condition: &LiteralNode{Value: false},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Yes"},
				},
			},
			data: TemplateData{},
			want: "",
		},
		{
			name: "if-else true",
			node: &IfNode{
				Condition: &LiteralNode{Value: true},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Yes"},
				},
				ElseBody: []ControlStructure{
					&TextNode{Content: "No"},
				},
			},
			data: TemplateData{},
			want: "Yes",
		},
		{
			name: "if-else false",
			node: &IfNode{
				Condition: &LiteralNode{Value: false},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Yes"},
				},
				ElseBody: []ControlStructure{
					&TextNode{Content: "No"},
				},
			},
			data: TemplateData{},
			want: "No",
		},
		{
			name: "if with variable condition",
			node: &IfNode{
				Condition: &VariableNode{Name: "show"},
				ThenBody: []ControlStructure{
					&ExpressionContentNode{
						Expression: &VariableNode{Name: "message"},
					},
				},
			},
			data: TemplateData{
				"show":    true,
				"message": "Hello World",
			},
			want: "Hello World",
		},
		{
			name: "if with arithmetic condition",
			node: &IfNode{
				Condition: &BinaryOpNode{
					Left:     &VariableNode{Name: "age"},
					Operator: ">=",
					Right:    &LiteralNode{Value: 18},
				},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Adult"},
				},
				ElseBody: []ControlStructure{
					&TextNode{Content: "Minor"},
				},
			},
			data: TemplateData{
				"age": 25,
			},
			want: "Adult",
		},
		{
			name: "if-elsif-else first condition true",
			node: &IfNode{
				Condition: &BinaryOpNode{
					Left:     &VariableNode{Name: "score"},
					Operator: ">=",
					Right:    &LiteralNode{Value: 90},
				},
				ThenBody: []ControlStructure{
					&TextNode{Content: "A"},
				},
				ElsIfs: []*ElsIfNode{
					{
						Condition: &BinaryOpNode{
							Left:     &VariableNode{Name: "score"},
							Operator: ">=",
							Right:    &LiteralNode{Value: 80},
						},
						Body: []ControlStructure{
							&TextNode{Content: "B"},
						},
					},
				},
				ElseBody: []ControlStructure{
					&TextNode{Content: "F"},
				},
			},
			data: TemplateData{
				"score": 95,
			},
			want: "A",
		},
		{
			name: "if-elsif-else second condition true",
			node: &IfNode{
				Condition: &BinaryOpNode{
					Left:     &VariableNode{Name: "score"},
					Operator: ">=",
					Right:    &LiteralNode{Value: 90},
				},
				ThenBody: []ControlStructure{
					&TextNode{Content: "A"},
				},
				ElsIfs: []*ElsIfNode{
					{
						Condition: &BinaryOpNode{
							Left:     &VariableNode{Name: "score"},
							Operator: ">=",
							Right:    &LiteralNode{Value: 80},
						},
						Body: []ControlStructure{
							&TextNode{Content: "B"},
						},
					},
				},
				ElseBody: []ControlStructure{
					&TextNode{Content: "F"},
				},
			},
			data: TemplateData{
				"score": 85,
			},
			want: "B",
		},
		{
			name: "if-elsif-else else condition",
			node: &IfNode{
				Condition: &BinaryOpNode{
					Left:     &VariableNode{Name: "score"},
					Operator: ">=",
					Right:    &LiteralNode{Value: 90},
				},
				ThenBody: []ControlStructure{
					&TextNode{Content: "A"},
				},
				ElsIfs: []*ElsIfNode{
					{
						Condition: &BinaryOpNode{
							Left:     &VariableNode{Name: "score"},
							Operator: ">=",
							Right:    &LiteralNode{Value: 80},
						},
						Body: []ControlStructure{
							&TextNode{Content: "B"},
						},
					},
				},
				ElseBody: []ControlStructure{
					&TextNode{Content: "F"},
				},
			},
			data: TemplateData{
				"score": 65,
			},
			want: "F",
		},
		{
			name: "string equality with German quotes",
			node: &IfNode{
				Condition: &BinaryOpNode{
					Left:     &VariableNode{Name: "haftung"},
					Operator: "==",
					Right:    &LiteralNode{Value: "Haftung klar individuelle Quote"},
				},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Match"},
				},
				ElseBody: []ControlStructure{
					&TextNode{Content: "No match"},
				},
			},
			data: TemplateData{
				"haftung": "Haftung klar individuelle Quote",
			},
			want: "Match",
		},
		{
			name: "string equality with German quotes - no match",
			node: &IfNode{
				Condition: &BinaryOpNode{
					Left:     &VariableNode{Name: "haftung"},
					Operator: "==",
					Right:    &LiteralNode{Value: "Haftung klar individuelle Quote"},
				},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Match"},
				},
				ElseBody: []ControlStructure{
					&TextNode{Content: "No match"},
				},
			},
			data: TemplateData{
				"haftung": "Different value",
			},
			want: "No match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.node.Render(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("IfNode.Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IfNode.Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnlessNodeRender(t *testing.T) {
	tests := []struct {
		name    string
		node    *UnlessNode
		data    TemplateData
		want    string
		wantErr bool
	}{
		{
			name: "unless true (should not render)",
			node: &UnlessNode{
				Condition: &LiteralNode{Value: true},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Hidden"},
				},
			},
			data: TemplateData{},
			want: "",
		},
		{
			name: "unless false (should render)",
			node: &UnlessNode{
				Condition: &LiteralNode{Value: false},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Visible"},
				},
			},
			data: TemplateData{},
			want: "Visible",
		},
		{
			name: "unless-else with true condition",
			node: &UnlessNode{
				Condition: &LiteralNode{Value: true},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Hidden"},
				},
				ElseBody: []ControlStructure{
					&TextNode{Content: "Else"},
				},
			},
			data: TemplateData{},
			want: "Else",
		},
		{
			name: "unless-else with false condition",
			node: &UnlessNode{
				Condition: &LiteralNode{Value: false},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Visible"},
				},
				ElseBody: []ControlStructure{
					&TextNode{Content: "Else"},
				},
			},
			data: TemplateData{},
			want: "Visible",
		},
		{
			name: "unless with variable condition",
			node: &UnlessNode{
				Condition: &VariableNode{Name: "hide"},
				ThenBody: []ControlStructure{
					&TextNode{Content: "Shown when hide is false"},
				},
			},
			data: TemplateData{
				"hide": false,
			},
			want: "Shown when hide is false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.node.Render(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnlessNode.Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UnlessNode.Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseForSyntax(t *testing.T) {
	tests := []struct {
		name     string
		forStr   string
		wantVar  string
		wantIdx  string
		wantColl string
		wantErr  bool
	}{
		{
			name:     "simple for loop",
			forStr:   "item in items",
			wantVar:  "item",
			wantIdx:  "",
			wantColl: "Variable(items)",
		},
		{
			name:     "indexed for loop",
			forStr:   "i, item in items",
			wantVar:  "item",
			wantIdx:  "i",
			wantColl: "Variable(items)",
		},
		{
			name:     "for with spaces",
			forStr:   " item  in  items ",
			wantVar:  "item",
			wantIdx:  "",
			wantColl: "Variable(items)",
		},
		{
			name:     "indexed for with spaces",
			forStr:   " i , item  in  items ",
			wantVar:  "item",
			wantIdx:  "i",
			wantColl: "Variable(items)",
		},
		{
			name:     "for with expression collection",
			forStr:   "item in getData().items",
			wantVar:  "item",
			wantIdx:  "",
			wantColl: "FieldAccess(FunctionCall(getData, []).items)",
		},
		{
			name:    "invalid syntax - no in",
			forStr:  "item items",
			wantErr: true,
		},
		{
			name:    "invalid syntax - too many variables",
			forStr:  "a, b, c in items",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseForSyntax(tt.forStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseForSyntax() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if got.Variable != tt.wantVar {
					t.Errorf("parseForSyntax() variable = %v, want %v", got.Variable, tt.wantVar)
				}
				if got.IndexVar != tt.wantIdx {
					t.Errorf("parseForSyntax() index = %v, want %v", got.IndexVar, tt.wantIdx)
				}
				if got.Collection.String() != tt.wantColl {
					t.Errorf("parseForSyntax() collection = %v, want %v", got.Collection.String(), tt.wantColl)
				}
			}
		})
	}
}

func TestControlStructureIntegration(t *testing.T) {
	tests := []struct {
		name    string
		content string
		data    TemplateData
		want    string
		wantErr bool
	}{
		{
			name:    "simple if true",
			content: "{{if show}}Hello{{end}}",
			data: TemplateData{
				"show": true,
			},
			want: "Hello",
		},
		{
			name:    "simple if false",
			content: "{{if show}}Hello{{end}}",
			data: TemplateData{
				"show": false,
			},
			want: "",
		},
		{
			name:    "if-else true",
			content: "{{if show}}Yes{{else}}No{{end}}",
			data: TemplateData{
				"show": true,
			},
			want: "Yes",
		},
		{
			name:    "if-else false",
			content: "{{if show}}Yes{{else}}No{{end}}",
			data: TemplateData{
				"show": false,
			},
			want: "No",
		},
		{
			name:    "unless true",
			content: "{{unless hide}}Visible{{end}}",
			data: TemplateData{
				"hide": false,
			},
			want: "Visible",
		},
		{
			name:    "unless false",
			content: "{{unless hide}}Visible{{end}}",
			data: TemplateData{
				"hide": true,
			},
			want: "",
		},
		{
			name:    "mixed content with if",
			content: "Start {{if show}}{{name}}{{end}} End",
			data: TemplateData{
				"show": true,
				"name": "World",
			},
			want: "Start World End",
		},
		{
			name:    "arithmetic condition",
			content: "{{if age >= 18}}Adult{{else}}Minor{{end}}",
			data: TemplateData{
				"age": 25,
			},
			want: "Adult",
		},
		{
			name:    "nested if statements",
			content: "{{if outer}}{{if inner}}Both{{else}}Outer only{{end}}{{end}}",
			data: TemplateData{
				"outer": true,
				"inner": true,
			},
			want: "Both",
		},
		{
			name:    "elsif chain",
			content: "{{if score >= 90}}A{{elsif score >= 80}}B{{elsif score >= 70}}C{{else}}F{{end}}",
			data: TemplateData{
				"score": 85,
			},
			want: "B",
		},
		{
			name:    "simple for loop",
			content: "{{for item in items}}{{item}} {{end}}",
			data: TemplateData{
				"items": []interface{}{"a", "b", "c"},
			},
			want: "a b c ",
		},
		{
			name:    "indexed for loop",
			content: "{{for i, item in items}}{{i}}:{{item}} {{end}}",
			data: TemplateData{
				"items": []interface{}{"x", "y"},
			},
			want: "0:x 1:y ",
		},
		{
			name:    "for loop with nested expressions",
			content: "{{for item in items}}[{{item.name}}] {{end}}",
			data: TemplateData{
				"items": []interface{}{
					map[string]interface{}{"name": "Item1"},
					map[string]interface{}{"name": "Item2"},
				},
			},
			want: "[Item1] [Item2] ",
		},
		{
			name:    "nested for loops",
			content: "{{for row in matrix}}{{for col in row}}{{col}} {{end}}| {{end}}",
			data: TemplateData{
				"matrix": []interface{}{
					[]interface{}{1, 2},
					[]interface{}{3, 4},
				},
			},
			want: "1 2 | 3 4 | ",
		},
		{
			name:    "for loop with conditionals",
			content: "{{for item in items}}{{if item > 2}}{{item}} {{end}}{{end}}",
			data: TemplateData{
				"items": []interface{}{1, 2, 3, 4},
			},
			want: "3 4 ",
		},
		{
			name:    "empty collection",
			content: "{{for item in items}}{{item}}{{end}}",
			data: TemplateData{
				"items": []interface{}{},
			},
			want: "",
		},
		{
			name:    "for loop over string",
			content: "{{for char in text}}[{{char}}]{{end}}",
			data: TemplateData{
				"text": "Hi",
			},
			want: "[H][i]",
		},
		{
			name:    "for loop over map",
			content: "{{for pair in data}}{{pair.key}}={{pair.value}} {{end}}",
			data: TemplateData{
				"data": map[string]interface{}{
					"x": 42,
				},
			},
			want: "x=42 ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structures, err := ParseControlStructures(tt.content)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ParseControlStructures() error = %v", err)
				}
				return
			}

			got, err := renderControlBody(structures, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderControlBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("renderControlBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to format structures for testing
func formatStructures(structures []ControlStructure) string {
	var parts []string
	for _, s := range structures {
		parts = append(parts, s.String())
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func TestTruthiness(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  bool
	}{
		{"nil", nil, false},
		{"true", true, true},
		{"false", false, false},
		{"zero int", 0, false},
		{"positive int", 1, true},
		{"negative int", -1, true},
		{"zero float", 0.0, false},
		{"positive float", 1.5, true},
		{"empty string", "", false},
		{"non-empty string", "hello", true},
		{"empty slice", []interface{}{}, false},
		{"non-empty slice", []interface{}{1}, true},
		{"empty map", map[string]interface{}{}, false},
		{"non-empty map", map[string]interface{}{"key": "value"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTruthy(tt.value)
			if got != tt.want {
				t.Errorf("isTruthy(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestRangeInLoops(t *testing.T) {
	tests := []struct {
		name    string
		content string
		data    TemplateData
		want    string
	}{
		{
			name:    "for loop with range(5)",
			content: "{{for i in range(5)}}{{i}} {{end}}",
			data:    TemplateData{},
			want:    "0 1 2 3 4 ",
		},
		{
			name:    "for loop with range(start, end)",
			content: "{{for i in range(2, 6)}}{{i}} {{end}}",
			data:    TemplateData{},
			want:    "2 3 4 5 ",
		},
		{
			name:    "for loop with range and step",
			content: "{{for i in range(0, 10, 3)}}{{i}} {{end}}",
			data:    TemplateData{},
			want:    "0 3 6 9 ",
		},
		{
			name:    "indexed for loop with range",
			content: "{{for idx, val in range(3, 6)}}{{idx}}:{{val}} {{end}}",
			data:    TemplateData{},
			want:    "0:3 1:4 2:5 ",
		},
		{
			name:    "range with variables",
			content: "{{for i in range(start, end)}}{{i}} {{end}}",
			data:    TemplateData{"start": 1, "end": 4},
			want:    "1 2 3 ",
		},
		{
			name:    "range with calculation",
			content: "{{for i in range(count * 2)}}{{i}} {{end}}",
			data:    TemplateData{"count": 3},
			want:    "0 1 2 3 4 5 ",
		},
		{
			name:    "nested loops with range",
			content: "{{for i in range(2)}}[{{for j in range(3)}}{{j}}{{end}}] {{end}}",
			data:    TemplateData{},
			want:    "[012] [012] ",
		},
		{
			name:    "empty range in loop",
			content: "{{for i in range(0)}}{{i}} {{end}}empty",
			data:    TemplateData{},
			want:    "empty",
		},
		{
			name:    "descending range in loop",
			content: "{{for i in range(5, 0, -1)}}{{i}} {{end}}",
			data:    TemplateData{},
			want:    "5 4 3 2 1 ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structures, err := ParseControlStructures(tt.content)
			if err != nil {
				t.Fatalf("ParseControlStructures() error = %v", err)
			}

			result, err := renderControlBody(structures, tt.data)
			if err != nil {
				t.Fatalf("renderControlBody() error = %v", err)
			}

			if result != tt.want {
				t.Errorf("renderControlBody() = %q, want %q", result, tt.want)
			}
		})
	}
}