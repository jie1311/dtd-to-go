package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// StructGenerator generates Go structs from DTD elements
type StructGenerator struct {
	packageName  string
	elements     map[string]*DTDElement
	elementOrder []string
}

// NewStructGenerator creates a new struct generator
func NewStructGenerator(packageName string, elements map[string]*DTDElement, elementOrder []string) *StructGenerator {
	return &StructGenerator{
		packageName:  packageName,
		elements:     elements,
		elementOrder: elementOrder,
	}
}

// GenerateStructs generates Go struct code for all elements
func (g *StructGenerator) GenerateStructs() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("package %s\n\n", g.packageName))
	builder.WriteString("import \"encoding/xml\"\n\n")

	// Generate structs for each element in declaration order
	for _, elementName := range g.elementOrder {
		if element, exists := g.elements[elementName]; exists {
			// Skip generating struct for simple elements (they'll be string fields)
			if !g.isSimpleElement(elementName) {
				structCode := g.generateStruct(element)
				builder.WriteString(structCode)
				builder.WriteString("\n")
			}
		}
	}

	return builder.String()
}

// generateStruct generates a Go struct for a single DTD element
func (g *StructGenerator) generateStruct(element *DTDElement) string {
	var builder strings.Builder

	structName := g.toGoStructName(element.Name)

	builder.WriteString(fmt.Sprintf("// %s represents the <%s> element\n", structName, element.Name))
	builder.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	// Add XML name annotation
	builder.WriteString(fmt.Sprintf("\tXMLName xml.Name `xml:\"%s\"`\n", element.Name))

	// Add attributes as struct fields
	for _, attr := range element.Attributes {
		fieldName := g.toGoFieldName(attr.Name)
		fieldType := g.getGoType(attr.Type)
		xmlTag := g.getXMLTag(attr.Name, attr.Required, true)

		builder.WriteString(fmt.Sprintf("\t%s %s `xml:\"%s\"`\n", fieldName, fieldType, xmlTag))
	}

	// Add content fields based on element content model
	contentFields := g.parseContentModel(element.Content)
	for _, field := range contentFields {
		builder.WriteString(fmt.Sprintf("\t%s\n", field))
	}

	// Add text content field if element can contain text
	if g.canContainText(element.Content) {
		builder.WriteString("\tText string `xml:\",chardata\"`\n")
	}

	builder.WriteString("}")

	return builder.String()
}

// parseContentModel parses the DTD content model and returns Go struct fields
func (g *StructGenerator) parseContentModel(content string) []string {
	var fields []string

	original := strings.TrimSpace(content)
	// Detect group-level repetition like (a | b | c)* or (a, b)+
	groupRepeating := false
	if strings.HasSuffix(original, ")*") || strings.HasSuffix(original, ")+") {
		groupRepeating = true
	}

	// Handle different content models
	if content == "EMPTY" {
		return fields
	}

	if content == "ANY" {
		fields = append(fields, "Content string `xml:\",innerxml\"`")
		return fields
	}

	if strings.Contains(content, "#PCDATA") {
		return fields // Text content handled separately
	}

	// Skip complex content models with entity references
	if strings.Contains(content, "%") {
		return fields
	}

	// Clean up the content model
	// If group-level repetition, strip trailing occurrence indicator for parsing child names
	if groupRepeating && (strings.HasSuffix(content, ")*") || strings.HasSuffix(content, ")+")) {
		// remove trailing )* or )+
		content = content[:len(content)-2]
	}
	content = strings.Trim(content, "()")
	content = strings.TrimSpace(content)

	// Handle choice (|) and sequence (,) operators
	var elementNames []string

	// Simplified parsing - extract element names
	// Remove occurrence indicators and extract basic element names
	parts := regexp.MustCompile(`[,|]`).Split(content, -1)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Remove occurrence indicators
		part = regexp.MustCompile(`[+*?]`).ReplaceAllString(part, "")
		// Remove parentheses
		part = strings.Trim(part, "()")
		part = strings.TrimSpace(part)

		if part != "" && !strings.Contains(part, "#PCDATA") && !strings.Contains(part, "%") {
			// Split further if there are nested structures
			subParts := strings.Fields(part)
			for _, subPart := range subParts {
				subPart = regexp.MustCompile(`[+*?(),]`).ReplaceAllString(subPart, "")
				subPart = strings.TrimSpace(subPart)
				if subPart != "" && !strings.Contains(subPart, "#PCDATA") {
					elementNames = append(elementNames, subPart)
				}
			}
		}
	}

	// Remove duplicates
	uniqueNames := make(map[string]bool)
	for _, name := range elementNames {
		if !uniqueNames[name] {
			uniqueNames[name] = true
			fieldName := g.toGoFieldName(name)
			structType := g.toGoStructName(name)

			// Determine if this should be a slice based on occurrence indicators or choice groups
			isSlice := groupRepeating || strings.Contains(original, name+"*") || strings.Contains(original, name+"+") || strings.Contains(original, "|")

			// Check if element is simple (just contains text)
			if g.isSimpleElement(name) {
				if isSlice {
					fields = append(fields, fmt.Sprintf("%s []string `xml:\"%s,omitempty\"`", fieldName, name))
				} else {
					fields = append(fields, fmt.Sprintf("%s *string `xml:\"%s,omitempty\"`", fieldName, name))
				}
			} else {
				if isSlice {
					fields = append(fields, fmt.Sprintf("%s []%s `xml:\"%s,omitempty\"`", fieldName, structType, name))
				} else {
					fields = append(fields, fmt.Sprintf("%s *%s `xml:\"%s,omitempty\"`", fieldName, structType, name))
				}
			}
		}
	}

	return fields
}

// isSimpleElement determines if an element should be treated as a simple string field
func (g *StructGenerator) isSimpleElement(elementName string) bool {
	element, exists := g.elements[elementName]
	if !exists {
		return true // Unknown elements treated as simple
	}

	content := strings.TrimSpace(element.Content)

	// Elements that are explicitly simple
	if content == "( #PCDATA )" || content == "#PCDATA" || content == "EMPTY" {
		return true
	}

	// Elements with no attributes and simple content model
	if len(element.Attributes) == 0 && (content == "( #PCDATA )" || strings.Contains(content, "#PCDATA")) {
		return true
	}

	return false
}

// canContainText determines if an element can contain text content
func (g *StructGenerator) canContainText(content string) bool {
	return strings.Contains(content, "#PCDATA")
}

// toGoStructName converts DTD element name to Go struct name
func (g *StructGenerator) toGoStructName(name string) string {
	// Convert to PascalCase
	words := strings.FieldsFunc(name, func(c rune) bool {
		return c == '-' || c == '_'
	})

	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.Title(word))
		}
	}

	structName := result.String()
	if structName == "" {
		structName = "Element"
	}

	return structName
}

// toGoFieldName converts DTD element/attribute name to Go field name
func (g *StructGenerator) toGoFieldName(name string) string {
	// Convert to PascalCase for field names
	words := strings.FieldsFunc(name, func(c rune) bool {
		return c == '-' || c == '_'
	})

	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			// Capitalize first letter, keep rest as is
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			result.WriteString(string(runes))
		}
	}

	fieldName := result.String()
	if fieldName == "" {
		fieldName = "Field"
	}

	return fieldName
}

// toPascalCase converts kebab-case or snake_case to PascalCase
func (g *StructGenerator) toPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(c rune) bool {
		return c == '-' || c == '_' || c == ' '
	})

	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(string(word[0])))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}

	return result.String()
}

// getGoType maps DTD attribute types to Go types
func (g *StructGenerator) getGoType(dtdType string) string {
	switch strings.ToUpper(dtdType) {
	case "CDATA", "ID", "IDREF", "NMTOKEN":
		return "string"
	case "IDREFS", "NMTOKENS":
		return "[]string"
	default:
		// For enumerated types or unknown types, default to string
		return "string"
	}
}

// getXMLTag generates the XML tag for struct fields
func (g *StructGenerator) getXMLTag(name string, required bool, isAttribute bool) string {
	tag := name
	if isAttribute {
		tag = name + ",attr"
	}
	if !required {
		tag += ",omitempty"
	}
	return tag
}
