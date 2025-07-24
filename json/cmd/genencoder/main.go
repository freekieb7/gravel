package main

import (
	"flag"
	"fmt"
	"go/types"
	"log"
	"os"
	"reflect"

	graveljson "github.com/freekieb7/gravel/json"
	"golang.org/x/tools/go/packages"
)

func main() {
	var (
		packagePath = flag.String("package", "", "Package path to analyze")
		outputFile  = flag.String("output", "encoders_generated.go", "Output file name")
		typeName    = flag.String("type", "", "Specific type name to generate encoder for")
	)
	flag.Parse()

	if *packagePath == "" {
		log.Fatal("Package path is required")
	}

	// Load the package
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedName,
	}

	pkgs, err := packages.Load(cfg, *packagePath)
	if err != nil {
		log.Fatalf("Failed to load package: %v", err)
	}

	if len(pkgs) == 0 {
		log.Fatal("No packages found")
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		for _, err := range pkg.Errors {
			log.Printf("Package error: %v", err)
		}
	}

	// Find struct types to generate encoders for
	var targetTypes []reflect.Type

	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj == nil {
			continue
		}

		if namedType, ok := obj.Type().(*types.Named); ok {
			underlying := namedType.Underlying()
			if structType, ok := underlying.(*types.Struct); ok {
				// Skip if specific type requested and this isn't it
				if *typeName != "" && name != *typeName {
					continue
				}

				// Convert to reflect.Type (simplified approach)
				// In a real implementation, you'd need proper type conversion
				fmt.Printf("Found struct type: %s with %d fields\n", name, structType.NumFields())

				// For demonstration, we'll create a simple registry entry
				// In practice, you'd need to properly convert types.Type to reflect.Type
			}
		}
	}

	// Generate the encoder code
	codegen := &graveljson.CodeGen{
		Package: pkg.Name,
		Types:   targetTypes,
	}

	generatedCode, err := codegen.GenerateEncoders()
	if err != nil {
		log.Fatalf("Failed to generate encoders: %v", err)
	}

	// Write to output file
	file, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(generatedCode)
	if err != nil {
		log.Fatalf("Failed to write generated code: %v", err)
	}

	fmt.Printf("Generated optimized encoders in %s\n", *outputFile)
}
