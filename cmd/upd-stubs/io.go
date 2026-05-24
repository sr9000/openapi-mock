package main

import (
	"log"
	"os"
	"path/filepath"
)

func writeGeneratedFile(path string, src []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if dryRun {
		if verboseLogs {
			log.Printf("dry-run: skip writing %s", path)
		}
		return nil
	}
	return os.WriteFile(path, src, 0o644)
}
