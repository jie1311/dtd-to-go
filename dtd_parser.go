package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// DTDElement represents an element definition in a DTD
type DTDElement struct {
	Name       string
	Content    string
	Attributes []DTDAttribute
}

// DTDAttribute represents an attribute definition in a DTD
type DTDAttribute struct {
	Name         string
	Type         string
	DefaultValue string
	Required     bool
}

// ParseResult contains the result of DTD parsing
type ParseResult struct {
	Elements map[string]*DTDElement
	Order    []string
}

// DTDParser handles parsing of DTD files
type DTDParser struct {
	elements     map[string]*DTDElement
	attributes   map[string][]DTDAttribute
	elementOrder []string          // Track the order of element declarations
	entities     map[string]string // Store parameter entity definitions
}

// NewDTDParser creates a new DTD parser
func NewDTDParser() *DTDParser {
	return &DTDParser{
		elements:     make(map[string]*DTDElement),
		attributes:   make(map[string][]DTDAttribute),
		elementOrder: make([]string, 0),
		entities:     make(map[string]string),
	}
}

// ParseFile parses a DTD file and returns the elements with their order
func (p *DTDParser) ParseFile(filename string) (*ParseResult, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentLine strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "<!--") {
			continue
		}

		currentLine.WriteString(line)
		currentLine.WriteString(" ")

		// Check if we have a complete declaration
		if strings.HasSuffix(line, ">") && (strings.Contains(currentLine.String(), "<!ELEMENT") ||
			strings.Contains(currentLine.String(), "<!ATTLIST") ||
			strings.Contains(currentLine.String(), "<!ENTITY")) {

			completeLine := strings.TrimSpace(currentLine.String())
			p.parseLine(completeLine)
			currentLine.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// Associate attributes with their elements
	for elementName, attrs := range p.attributes {
		if element, exists := p.elements[elementName]; exists {
			element.Attributes = attrs
		}
	}

	return &ParseResult{
		Elements: p.elements,
		Order:    p.elementOrder,
	}, nil
}

// parseLine parses a single complete DTD line
func (p *DTDParser) parseLine(line string) {
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "<!ENTITY") {
		p.parseEntity(line)
	} else if strings.HasPrefix(line, "<!ELEMENT") {
		p.parseElement(line)
	} else if strings.HasPrefix(line, "<!ATTLIST") {
		p.parseAttributeList(line)
	}
}

// parseEntity parses an ENTITY declaration
func (p *DTDParser) parseEntity(line string) {
	// Handle parameter entities like <!ENTITY % status_sellable "...">
	re := regexp.MustCompile(`<!ENTITY\s+%\s+(\w+)\s+"(.+?)">`)
	matches := re.FindStringSubmatch(line)

	if len(matches) >= 3 {
		entityName := matches[1]
		entityValue := matches[2]
		p.entities[entityName] = entityValue
	}
}

// parseElement parses an ELEMENT declaration
func (p *DTDParser) parseElement(line string) {
	// Regular expression to match <!ELEMENT name content>
	// Updated to handle hyphenated element names
	re := regexp.MustCompile(`<!ELEMENT\s+([\w-]+)\s+(.+?)>`)
	matches := re.FindStringSubmatch(line)

	if len(matches) >= 3 {
		name := matches[1]
		content := strings.TrimSpace(matches[2])

		// Only add to order if this is the first time we see this element
		if _, exists := p.elements[name]; !exists {
			p.elementOrder = append(p.elementOrder, name)
		}

		p.elements[name] = &DTDElement{
			Name:    name,
			Content: content,
		}
	}
}

// parseEntityValue parses an entity value and adds attributes
func (p *DTDParser) parseEntityValue(elementName, entityValue string, attributes *[]DTDAttribute) {
	// Split the entity value into parts
	parts := strings.Fields(entityValue)
	if len(parts) < 3 {
		return
	}

	// Extract attribute name, type, and requirement
	// Format: "status ( current | withdrawn | offmarket | sold | deleted ) #REQUIRED"
	attrName := parts[0]

	// Find the closing parenthesis to get the complete type definition
	typeEnd := -1
	for i, part := range parts {
		if strings.Contains(part, ")") {
			typeEnd = i
			break
		}
	}

	var defaultInfo string
	if typeEnd+1 < len(parts) {
		defaultInfo = parts[typeEnd+1]
	}

	attr := DTDAttribute{
		Name: attrName,
		Type: "string", // Simplify enumerated types to string
	}

	// Check if required or has default value
	if defaultInfo == "#REQUIRED" {
		attr.Required = true
	} else if defaultInfo != "#IMPLIED" {
		attr.DefaultValue = strings.Trim(defaultInfo, `"`)
	}

	*attributes = append(*attributes, attr)
}

// parseAttributeList parses an ATTLIST declaration
func (p *DTDParser) parseAttributeList(line string) {
	// Remove <!ATTLIST and >
	content := strings.TrimPrefix(line, "<!ATTLIST")
	content = strings.TrimSuffix(content, ">")
	content = strings.TrimSpace(content)

	parts := strings.Fields(content)
	if len(parts) < 1 {
		return
	}

	elementName := parts[0]
	parts = parts[1:]

	var attributes []DTDAttribute

	// Parse attributes (simplified parsing for complex DTD constructs)
	for i := 0; i < len(parts); {
		if i >= len(parts) {
			break
		}

		// Handle entity references like %status_sellable;
		if strings.HasPrefix(parts[i], "%") && strings.HasSuffix(parts[i], ";") {
			entityName := strings.TrimPrefix(parts[i], "%")
			entityName = strings.TrimSuffix(entityName, ";")

			if entityValue, exists := p.entities[entityName]; exists {
				// Recursively parse the entity value
				p.parseEntityValue(elementName, entityValue, &attributes)
			}
			i++
			continue
		}

		// Basic attribute parsing
		if i+2 < len(parts) {
			attrName := parts[i]
			attrType := parts[i+1]
			defaultInfo := parts[i+2]

			// Skip attributes with complex type definitions (parentheses)
			if strings.Contains(attrType, "(") {
				// Find the end of the parenthetical expression
				j := i + 1
				parenCount := 0
				for j < len(parts) {
					for _, char := range parts[j] {
						if char == '(' {
							parenCount++
						} else if char == ')' {
							parenCount--
						}
					}
					if parenCount == 0 && strings.Contains(parts[j], ")") {
						break
					}
					j++
				}

				if j+1 < len(parts) {
					defaultInfo = parts[j+1]

					attr := DTDAttribute{
						Name: attrName,
						Type: "string", // Simplify enumerated types to string
					}

					// Check if required or has default value
					if defaultInfo == "#REQUIRED" {
						attr.Required = true
					} else if defaultInfo != "#IMPLIED" {
						attr.DefaultValue = strings.Trim(defaultInfo, `"`)
					}

					attributes = append(attributes, attr)
				}

				i = j + 2
			} else {
				attr := DTDAttribute{
					Name: attrName,
					Type: attrType,
				}

				// Check if required or has default value
				if defaultInfo == "#REQUIRED" {
					attr.Required = true
				} else if defaultInfo != "#IMPLIED" {
					attr.DefaultValue = strings.Trim(defaultInfo, `"`)
				}

				attributes = append(attributes, attr)
				i += 3
			}
		} else {
			i++
		}
	}

	// Append to existing attributes instead of overwriting
	if existingAttrs, exists := p.attributes[elementName]; exists {
		p.attributes[elementName] = append(existingAttrs, attributes...)
	} else {
		p.attributes[elementName] = attributes
	}
}
