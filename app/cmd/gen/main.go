// Package main 提供 niko-admin 代码生成器的 CLI 入口。
//
// Usage:
//
//	niko-admin gen <model>        Generate CRUD for a specific model (e.g., "user")
//	niko-admin gen --all          Generate CRUD for all models
//	niko-admin gen --list         List all available models
//	niko-admin gen -f <file>      Generate CRUD from a specific model file
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/niko-admin/niko-admin/pkg/gen"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		printUsage()
		return
	}

	modelDir := gen.DefaultModelDir
	outputDir := gen.DefaultOutputDir
	modulePath := "github.com/niko-admin/niko-admin"

	switch {
	case args[0] == "--all":
		err := gen.GenerateAll(modelDir, outputDir, modulePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case args[0] == "--list":
		models, err := gen.ListModels(modelDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		for _, m := range models {
			fmt.Println(m)
		}

	case args[0] == "-f" && len(args) > 1:
		err := gen.GenerateOne(args[1], outputDir, modulePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated CRUD code for %s\n", args[1])

	default:
		// Treat as model name
		modelName := strings.ToLower(args[0])
		modelFile := fmt.Sprintf("%s/%s.go", modelDir, modelName)
		if _, err := os.Stat(modelFile); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Model file not found: %s\n", modelFile)
			os.Exit(1)
		}
		err := gen.GenerateOne(modelFile, outputDir, modulePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated CRUD code for %s\n", args[0])
	}
}

func printUsage() {
	fmt.Println("Usage: niko-admin gen [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  <model>     Generate CRUD for a specific model (e.g., 'user')")
	fmt.Println("  --all       Generate CRUD for all models")
	fmt.Println("  --list      List all available models")
	fmt.Println("  -f <file>   Generate CRUD from a specific model file")
}
