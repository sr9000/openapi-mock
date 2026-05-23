package main

import (
	"errors"
	"flag"
	"log"
	"os"
)

func main() {
	flag.StringVar(&specsDir, "specs-dir", specsDir, "Path to directory with OpenAPI specs")
	flag.StringVar(&openapiGenDir, "generated-dir", openapiGenDir, "Path to generated oapi-codegen packages")
	flag.StringVar(&openapiStubsDir, "stubs-dir", openapiStubsDir, "Path to output stubs directory")
	flag.Parse()

	if err := run(); err != nil {
		log.Printf("upd-stubs finished with errors: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Process OpenAPI specs
	openapiSpecs, err := discoverOpenAPISpecs()
	if err != nil {
		return err
	}
	if len(openapiSpecs) == 0 {
		log.Println("No OpenAPI specs found in specs/ directory")
		return nil
	}

	var joined error
	processed := make([]*openapiSpec, 0, len(openapiSpecs))
	for _, spec := range openapiSpecs {
		log.Printf("Processing OpenAPI spec: %s", spec.SpecPath)
		if err := generateOpenAPIStubs(spec); err != nil {
			joined = errors.Join(joined, errors.New(spec.SpecPath+": "+err.Error()))
			continue
		}
		processed = append(processed, spec)
	}
	if len(processed) == 0 {
		return errors.Join(joined, errors.New("failed to process all discovered specs"))
	}

	// Generate OpenAPI wire file
	if err := generateOpenAPIWireFile(processed); err != nil {
		joined = errors.Join(joined, err)
	}

	if joined == nil {
		log.Printf("OpenAPI stubs generated successfully for %d spec(s)", len(processed))
	}
	return joined
}
