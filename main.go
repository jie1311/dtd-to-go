package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var (
		inputFile   = flag.String("input", "", "Path to the DTD file to parse")
		outputFile  = flag.String("output", "", "Path to output Go file (default: stdout)")
		packageName = flag.String("package", "main", "Go package name for generated structs")
	)
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -input <dtd-file> [-output <go-file>] [-package <package-name>]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  -input    Path to the DTD file to parse (required)\n")
		fmt.Fprintf(os.Stderr, "  -output   Path to output Go file (default: stdout)\n")
		fmt.Fprintf(os.Stderr, "  -package  Go package name for generated structs (default: main)\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -input example.dtd -output structs.go -package models\n", os.Args[0])
		os.Exit(1)
	}

	// Parse the DTD file
	fmt.Printf("Parsing DTD file: %s\n", *inputFile)
	parser := NewDTDParser()
	result, err := parser.ParseFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing DTD file: %v\n", err)
		os.Exit(1)
	}

	if len(result.Elements) == 0 {
		fmt.Printf("No elements found in DTD file\n")
		return
	}

	fmt.Printf("Found %d elements in DTD file\n", len(result.Elements))
	for _, name := range result.Order {
		fmt.Printf("  - %s\n", name)
	}

	// Generate Go structs
	generator := NewStructGenerator(*packageName, result.Elements, result.Order)
	structCode := generator.GenerateStructs()

	// Output the generated code
	if *outputFile == "" {
		// Output to stdout
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("Generated Go Structs:")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Print(structCode)
	} else {
		// Output to file
		err := writeToFile(*outputFile, structCode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated Go structs written to: %s\n", *outputFile)
	}
}

// writeToFile writes content to the specified file
func writeToFile(filename, content string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write content to file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	return nil
}
