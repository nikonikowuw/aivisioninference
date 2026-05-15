package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultModelDir is the default directory containing model files.
	DefaultModelDir = "internal/model"

	// DefaultOutputDir is the default output directory for generated code.
	DefaultOutputDir = "internal"
)

// GenerateAll generates CRUD code for all models in the model directory.
func GenerateAll(modelDir, outputDir, modulePath string) error {
	models, err := ListModels(modelDir)
	if err != nil {
		return fmt.Errorf("list models: %w", err)
	}

	if len(models) == 0 {
		fmt.Println("No models found")
		return nil
	}

	renderer := NewRenderer(outputDir, modulePath)
	var generated int
	var skipped int

	for _, modelName := range models {
		modelFile := filepath.Join(modelDir, modelName+".go")

		// Skip files that are not model definitions (join tables, etc.)
		if !isModelFile(modelFile) {
			skipped++
			continue
		}

		fmt.Printf("Generating CRUD for %s...\n", modelName)
		if err := generateFromModelFile(modelFile, renderer); err != nil {
			fmt.Printf("  Warning: %v\n", err)
			skipped++
			continue
		}
		generated++
	}

	fmt.Printf("\nGenerated CRUD code for %d models (%d skipped)\n", generated, skipped)
	return nil
}

// GenerateOne generates CRUD code for a single model file.
func GenerateOne(modelFile, outputDir, modulePath string) error {
	// Normalize path
	modelFile = filepath.Clean(modelFile)

	// Check if file exists
	if _, err := os.Stat(modelFile); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", modelFile)
	}

	renderer := NewRenderer(outputDir, modulePath)
	return generateFromModelFile(modelFile, renderer)
}

// generateFromModelFile parses a model file and generates CRUD code.
func generateFromModelFile(modelFile string, renderer *Renderer) error {
	// Parse model file
	models, err := ParseModel(modelFile)
	if err != nil {
		return fmt.Errorf("parse model file %s: %w", modelFile, err)
	}

	if len(models) == 0 {
		return fmt.Errorf("no valid models found in %s", modelFile)
	}

	// Generate CRUD for each model in the file
	for _, model := range models {
		if err := renderer.Render(model); err != nil {
			return fmt.Errorf("render model %s: %w", model.Name, err)
		}
		fmt.Printf("  Generated files for %s:\n", model.Name)
		fmt.Printf("    - internal/handler/%s.go\n", model.PackageName)
		fmt.Printf("    - internal/service/%s.go\n", model.PackageName)
		fmt.Printf("    - internal/repository/%s.go\n", model.PackageName)
		fmt.Printf("    - internal/router/%s.go\n", model.PackageName)
		fmt.Printf("    - internal/dto/%s.go\n", model.PackageName)
	}

	return nil
}

// isModelFile checks if a Go file contains model definitions.
func isModelFile(filePath string) bool {
	// Read first few bytes to check for package declaration
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// Check if it's a Go file with model package
	contentStr := string(content)
	return strings.Contains(contentStr, "package model")
}
