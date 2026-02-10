package main

import (
	"log"
)

func main() {
	// Process OpenAPI specs
	openapiSpecs, err := discoverOpenAPISpecs()
	if err != nil {
		log.Fatalf("failed to discover OpenAPI specs: %v", err)
	}
	if len(openapiSpecs) == 0 {
		log.Println("No OpenAPI specs found in specs/ directory")
		return
	}
	for _, spec := range openapiSpecs {
		log.Printf("Processing OpenAPI spec: %s", spec.SpecPath)
		if err := generateOpenAPIStubs(spec); err != nil {
			log.Fatalf("failed to generate OpenAPI stubs for %s: %v", spec.SpecPath, err)
		}
	}
	// Generate OpenAPI wire file
	if err := generateOpenAPIWireFile(openapiSpecs); err != nil {
		log.Fatalf("failed to generate OpenAPI wire file: %v", err)
	}
	log.Println("OpenAPI stubs generated successfully")
}
