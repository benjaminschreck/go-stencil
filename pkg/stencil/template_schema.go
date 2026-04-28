package stencil

// Type describes a template value type used by TemplateSchema validation.
type Type struct {
	kind     string
	fields   TemplateSchema
	element  *Type
	nullable bool
}

// TemplateType is the public type accepted by TemplateSchema values.
type TemplateType = Type

// TemplateSchema describes the data available to a template using the same
// shape as TemplateData, but with types instead of render-time values.
type TemplateSchema map[string]TemplateType

var (
	String = Type{kind: semanticKindString}
	Number = Type{kind: semanticKindNumber}
	Bool   = Type{kind: semanticKindBool}
	Any    = Type{kind: semanticKindAny}
)

// Object declares a nested object with named fields.
func Object(fields TemplateSchema) TemplateType {
	return Type{
		kind:   semanticKindObject,
		fields: fields,
	}
}

// List declares an array/slice whose elements have the given type.
func List(element TemplateType) TemplateType {
	return Type{
		kind:    semanticKindArray,
		element: &element,
	}
}

// Nullable marks a type as accepting nil/null values.
func Nullable(t TemplateType) TemplateType {
	t.nullable = true
	return t
}

func validationSchemaFromTemplateSchema(schema TemplateSchema) ValidationSchema {
	fields := make([]FieldDefinition, 0)
	appendTemplateSchemaFields(&fields, "", schema, false)
	return ValidationSchema{Fields: fields}
}

func appendTemplateSchemaFields(fields *[]FieldDefinition, prefix string, schema TemplateSchema, nullable bool) {
	for name, typ := range schema {
		if name == "" {
			continue
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		appendTemplateTypeField(fields, path, typ, nullable)
	}
}

func appendTemplateTypeField(fields *[]FieldDefinition, path string, typ TemplateType, inheritedNullable bool) {
	nullable := inheritedNullable || typ.nullable
	switch typ.kind {
	case semanticKindArray:
		elementKind := semanticKindAny
		if typ.element != nil {
			elementKind = typ.element.kind
			if elementKind == "" || elementKind == semanticKindArray {
				elementKind = semanticKindAny
			}
		}
		*fields = append(*fields, FieldDefinition{
			Path:       path,
			Type:       elementKind,
			Nullable:   nullable,
			Collection: true,
		})
		if typ.element != nil && len(typ.element.fields) > 0 {
			appendTemplateSchemaFields(fields, path, typ.element.fields, typ.element.nullable)
		}
	case semanticKindObject:
		*fields = append(*fields, FieldDefinition{
			Path:     path,
			Type:     semanticKindObject,
			Nullable: nullable,
		})
		appendTemplateSchemaFields(fields, path, typ.fields, false)
	default:
		kind := typ.kind
		if kind == "" {
			kind = semanticKindAny
		}
		*fields = append(*fields, FieldDefinition{
			Path:     path,
			Type:     kind,
			Nullable: nullable,
		})
	}
}
