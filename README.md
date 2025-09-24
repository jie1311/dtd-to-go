# DTD to Go Struct Generator

A Go program that reads Document Type Definition (DTD) files and generates corresponding Go structs with XML tags for easy marshaling/unmarshaling.

## Features

- Parse DTD files and extract element and attribute definitions
- Generate Go structs with appropriate XML tags
- Handle various DTD content models (EMPTY, ANY, PCDATA, element sequences, choices)
- Support for optional and repeating elements
- Proper Go naming conventions (CamelCase)
- Configurable package names
- Output to file or stdout

## Usage

### Build the program

```bash
go build -o dtd-to-go
```

### Run with command line options

```bash
# Generate structs to stdout
./dtd-to-go -input sample.dtd

# Generate structs to a file with custom package name
./dtd-to-go -input sample.dtd -output models/book.go -package models
```

### Command line options

- `-input`: Path to the DTD file to parse (required)
- `-output`: Path to output Go file (default: stdout)
- `-package`: Go package name for generated structs (default: main)

## Example

Given this DTD file:

```dtd
<!ELEMENT catalog (book+)>
<!ATTLIST catalog version CDATA #REQUIRED>

<!ELEMENT book (title, author, publisher, price?)>
<!ATTLIST book id ID #REQUIRED
               isbn CDATA #IMPLIED
               category (fiction|non-fiction|technical) "fiction">

<!ELEMENT title (#PCDATA)>
<!ELEMENT author (first-name, last-name)>
<!ELEMENT first-name (#PCDATA)>
<!ELEMENT last-name (#PCDATA)>
<!ELEMENT publisher (#PCDATA)>
<!ELEMENT price (#PCDATA)>
<!ATTLIST price currency CDATA "USD">
```

The program generates Go structs like:

```go
package main

import "encoding/xml"

// Catalog represents the <catalog> element
type Catalog struct {
    XMLName xml.Name `xml:"catalog"`
    Version string   `xml:"version,attr"`
    Book    []Book   `xml:"book,omitempty"`
}

// Book represents the <book> element
type Book struct {
    XMLName   xml.Name `xml:"book"`
    Id        string   `xml:"id,attr"`
    Isbn      string   `xml:"isbn,attr,omitempty"`
    Category  string   `xml:"category,attr,omitempty"`
    Title     *string  `xml:"title,omitempty"`
    Author    *Author  `xml:"author,omitempty"`
    Publisher *string  `xml:"publisher,omitempty"`
    Price     *Price   `xml:"price,omitempty"`
}

// Author represents the <author> element
type Author struct {
    XMLName   xml.Name `xml:"author"`
    FirstName *string  `xml:"first-name,omitempty"`
    LastName  *string  `xml:"last-name,omitempty"`
}

// Price represents the <price> element
type Price struct {
    XMLName  xml.Name `xml:"price"`
    Currency string   `xml:"currency,attr,omitempty"`
    Text     string   `xml:",chardata"`
}
```

## DTD Support

The parser supports:

- Element declarations (`<!ELEMENT>`)
- Attribute lists (`<!ATTLIST>`)
- Content models:
  - `EMPTY` - elements with no content
  - `ANY` - elements with any content
  - `(#PCDATA)` - text-only content
  - Element sequences: `(a, b, c)`
  - Occurrence indicators: `?` (optional), `+` (one or more), `*` (zero or more)
- Attribute types: `CDATA`, `ID`, `IDREF`, etc.
- Attribute defaults: `#REQUIRED`, `#IMPLIED`, or literal values

## Limitations

- **Choice Elements**: Choice content models like `(a | b | c)` are converted to structs with all possible options as array fields, rather than implementing a proper union type
- **EMPTY Elements**: Elements declared as `EMPTY` are represented as string pointers, which may not be the most appropriate representation
- **Entity Declarations**: Parameter entities are parsed but not fully expanded in content models
- **Mixed Content**: Complex mixed content models may need manual adjustment
- **Namespaces**: Not fully supported
- **Complex Occurrence Patterns**: Nested occurrence indicators may not be handled optimally
- **Attribute Enumerations**: Enumerated attribute types are converted to simple string types

## Requirements

- Go 1.24.6 or later

