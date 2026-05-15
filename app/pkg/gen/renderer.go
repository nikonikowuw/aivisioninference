package gen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Renderer generates code files from templates.
type Renderer struct {
	OutputDir  string
	ModulePath string
}

// TemplateData is the data passed to templates.
type TemplateData struct {
	Name          string
	Plural        string
	PackageName   string
	Fields        []Field
	SearchFields  []Field
	HasSoftDelete bool
	HasTimeField  bool
	FilePath      string
	Timestamp     string
}

// NewRenderer creates a new Renderer with the given output directory and module path.
func NewRenderer(outputDir, modulePath string) *Renderer {
	return &Renderer{
		OutputDir:  outputDir,
		ModulePath: modulePath,
	}
}

// Render generates all 5 CRUD files for a model.
func (r *Renderer) Render(model *ModelInfo) error {
	// Check if any field uses time.Time
	hasTimeField := false
	for _, f := range model.Fields {
		if f.ShouldIncludeInDTO() && f.IsTime {
			hasTimeField = true
			break
		}
	}

	// Prepare template data
	data := TemplateData{
		Name:          model.Name,
		Plural:        model.Plural,
		PackageName:   model.PackageName,
		Fields:        model.Fields,
		SearchFields:  model.SearchFields,
		HasSoftDelete: model.HasSoftDelete,
		HasTimeField:  hasTimeField,
		FilePath:      model.FilePath,
		Timestamp:     time.Now().Format("2006-01-02 15:04:05"),
	}

	// Create output directories
	dirs := []string{
		filepath.Join(r.OutputDir, "handler"),
		filepath.Join(r.OutputDir, "service"),
		filepath.Join(r.OutputDir, "repository"),
		filepath.Join(r.OutputDir, "router"),
		filepath.Join(r.OutputDir, "dto"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Render each template
	templates := []struct {
		name     string
		filePath string
	}{
		{"handler.go.tmpl", filepath.Join(r.OutputDir, "handler", model.PackageName+".go")},
		{"service.go.tmpl", filepath.Join(r.OutputDir, "service", model.PackageName+".go")},
		{"repository.go.tmpl", filepath.Join(r.OutputDir, "repository", model.PackageName+".go")},
		{"router.go.tmpl", filepath.Join(r.OutputDir, "router", model.PackageName+".go")},
		{"dto.go.tmpl", filepath.Join(r.OutputDir, "dto", model.PackageName+".go")},
	}

	for _, tmpl := range templates {
		if err := r.renderTemplate(tmpl.name, tmpl.filePath, data); err != nil {
			return fmt.Errorf("render %s: %w", tmpl.name, err)
		}
	}

	return nil
}

// renderTemplate renders a single template file.
func (r *Renderer) renderTemplate(templateName, outputPath string, data TemplateData) error {
	// Find template file
	templatePath, err := findTemplateFile(templateName)
	if err != nil {
		return err
	}

	// Read template
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template %s: %w", templatePath, err)
	}

	// Create template with custom functions
	funcMap := template.FuncMap{
		"toUpper": strings.ToUpper,
		"toLower": strings.ToLower,
		"plural":  toPlural,
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
	}

	tmpl, err := template.New(templateName).Funcs(funcMap).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("parse template %s: %w", templateName, err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template %s: %w", templateName, err)
	}

	// Write output
	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write file %s: %w", outputPath, err)
	}

	return nil
}

// findTemplateFile finds the template file by searching in the package directory.
func findTemplateFile(name string) (string, error) {
	// Try relative path first
	relPath := filepath.Join("pkg", "gen", "templates", name)
	if _, err := os.Stat(relPath); err == nil {
		return relPath, nil
	}

	// Try from current directory
	if wd, err := os.Getwd(); err == nil {
		path := filepath.Join(wd, "pkg", "gen", "templates", name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Try from module root (look for go.mod)
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod, look for template
			tmplPath := filepath.Join(dir, "pkg", "gen", "templates", name)
			if _, err := os.Stat(tmplPath); err == nil {
				return tmplPath, nil
			}
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("template file %s not found", name)
}
