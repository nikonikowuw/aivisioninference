// Package gen provides CRUD code generation for niko-admin models.
package gen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Field represents a parsed struct field with its metadata.
type Field struct {
	Name          string // Go field name (e.g., "Username")
	Type          string // Go type (e.g., "string")
	JSONName      string // JSON tag name (e.g., "username")
	GormTag       string // Full gorm tag
	IsPK          bool   // Is primary key
	IsUnique      bool   // Has uniqueIndex
	IsNotNull     bool   // Has not null
	IsSearchable  bool   // string type without json:"-"
	IsTime        bool   // Is time.Time
	IsSlice       bool   // Is a slice type (e.g., []Role)
	IsJSONExcluded bool  // Has json:"-" (excluded from JSON output)
	IsComplexType bool   // Is a complex type (slice, struct, pointer to struct)
	SwaggerType   string // Swagger type (string, integer, boolean)
}

// ShouldIncludeInDTO returns true if this field should be included in Create/Update DTOs.
func (f *Field) ShouldIncludeInDTO() bool {
	// Exclude fields with json:"-"
	if f.IsJSONExcluded {
		return false
	}

	// Exclude primary key (ID is auto-generated)
	if f.IsPK {
		return false
	}

	// Exclude time fields (CreatedAt, UpdatedAt, DeletedAt)
	if f.Name == "CreatedAt" || f.Name == "UpdatedAt" || f.Name == "DeletedAt" {
		return false
	}

	// Exclude complex types (slices, maps, structs) - these are relationships
	if f.IsComplexType {
		return false
	}

	return true
}

// ModelInfo contains all metadata extracted from a model struct.
type ModelInfo struct {
	Name         string   // Model name (e.g., "User")
	Plural       string   // Plural name (e.g., "users")
	PackageName  string   // Lowercase name (e.g., "user")
	Fields       []Field  // All non-relationship fields
	SearchFields []Field  // Fields eligible for search
	HasSoftDelete bool    // Has DeletedAt field
	FilePath     string   // Source file path
}

// ParseModel parses a Go source file and extracts model struct information.
func ParseModel(filePath string) ([]*ModelInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var models []*ModelInfo

	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			return true
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Skip join tables and small structs (they don't need CRUD)
			if typeSpec.Name.Name == "" {
				continue
			}

			modelName := typeSpec.Name.Name

			// Skip non-model structs (join tables, etc.)
			if !isMainModel(structType) {
				continue
			}

			model := &ModelInfo{
				Name:        modelName,
				Plural:      toPlural(strings.ToLower(modelName)),
				PackageName: strings.ToLower(modelName),
				FilePath:    filePath,
			}

			// Parse fields
			for _, field := range structType.Fields.List {
				for _, name := range field.Names {
					if !name.IsExported() {
						continue
					}

					fieldInfo := parseField(name.Name, field)
					model.Fields = append(model.Fields, *fieldInfo)

					if fieldInfo.Name == "DeletedAt" {
						model.HasSoftDelete = true
					}

					if fieldInfo.IsSearchable {
						model.SearchFields = append(model.SearchFields, *fieldInfo)
					}
				}
			}

			models = append(models, model)
		}
		return true
	})

	return models, nil
}

// isMainModel checks if a struct is a main model (has more than 2 fields, likely a real model).
func isMainModel(structType *ast.StructType) bool {
	if structType.Fields == nil {
		return false
	}

	fieldCount := 0
	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 {
			fieldCount++
		}
	}

	// Models should have at least 3 fields (ID, CreatedAt, UpdatedAt + at least one more)
	return fieldCount >= 3
}

// parseField extracts field metadata from an AST field.
func parseField(name string, field *ast.Field) *Field {
	f := &Field{
		Name: name,
	}

	// Get type string
	if field.Type != nil {
		f.Type = getTypeString(field.Type)
	}

	// Parse tags
	if field.Tag != nil {
		tag := field.Tag.Value
		f.JSONName = getJSONName(tag)
		f.GormTag = getGormTag(tag)
		f.IsPK = strings.Contains(f.GormTag, "primaryKey")
		f.IsUnique = strings.Contains(f.GormTag, "uniqueIndex") || strings.Contains(f.GormTag, "unique")
		f.IsNotNull = strings.Contains(f.GormTag, "not null")
	}

	// Determine JSON excluded
	f.IsJSONExcluded = f.JSONName == "-"

	// Determine time type
	f.IsTime = f.Type == "time.Time" || f.Type == "*time.Time"

	// Determine if slice
	f.IsSlice = strings.HasPrefix(f.Type, "[]")

	// Determine if complex type (slice, struct, pointer to struct)
	f.IsComplexType = f.IsSlice || isStructType(f.Type)

	// Determine searchable
	f.IsSearchable = isSearchable(f)

	// Determine swagger type
	f.SwaggerType = getSwaggerType(f.Type)

	return f
}

// getTypeString converts an AST expression to a type string.
func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	case *ast.ArrayType:
		return "[]" + getTypeString(t.Elt)
	case *ast.SelectorExpr:
		return getTypeString(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + getTypeString(t.Key) + "]" + getTypeString(t.Value)
	default:
		return "interface{}"
	}
}

// getJSONName extracts the JSON tag name from a struct tag.
func getJSONName(tag string) string {
	// Remove backticks
	tag = strings.Trim(tag, "`")

	// Find json:"..." tag
	for _, part := range strings.Fields(tag) {
		if strings.HasPrefix(part, "json:") {
			value := strings.TrimPrefix(part, "json:")
			value = strings.Trim(value, "\"")
			value = strings.Trim(value, ",")
			if value == "-" {
				return "-"
			}
			return value
		}
	}

	return ""
}

// getGormTag extracts the gorm tag from a struct tag.
func getGormTag(tag string) string {
	// Remove backticks
	tag = strings.Trim(tag, "`")

	// Find gorm:"..." tag
	for _, part := range strings.Fields(tag) {
		if strings.HasPrefix(part, "gorm:") {
			value := strings.TrimPrefix(part, "gorm:")
			value = strings.Trim(value, "\"")
			return value
		}
	}

	return ""
}

// isSearchable determines if a field is eligible for search filtering.
func isSearchable(f *Field) bool {
	// Must be a string type (not pointer, not slice, not time)
	if f.Type != "string" && f.Type != "*string" {
		return false
	}

	// JSON name must not be "-" (excluded from JSON)
	if f.JSONName == "-" || f.JSONName == "" {
		return false
	}

	// Skip soft delete field
	if f.Name == "DeletedAt" {
		return false
	}

	// Skip ID field (shouldn't be searchable via LIKE)
	if f.Name == "ID" {
		return false
	}

	// Skip password and other sensitive fields
	if f.Name == "Password" || f.Name == "LoginAttempts" || f.Name == "LockedUntil" {
		return false
	}

	return true
}

// isStructType checks if a type is a struct type (not a pointer).
func isStructType(t string) bool {
	// Remove pointer prefix
	t = strings.TrimPrefix(t, "*")

	// Common struct types in models
	structTypes := []string{
		"Role", "Permission", "User", "File", "Task",
		"UserRole", "RolePermission",
	}

	for _, st := range structTypes {
		if t == st {
			return true
		}
	}

	return false
}

// getSwaggerType maps Go types to Swagger/OpenAPI types.
func getSwaggerType(goType string) string {
	// Remove pointer
	goType = strings.TrimPrefix(goType, "*")

	switch {
	case goType == "string":
		return "string"
	case goType == "bool":
		return "boolean"
	case goType == "time.Time" || goType == "*time.Time":
		return "string" // date-time format
	case isNumericType(goType):
		return "integer"
	case strings.HasPrefix(goType, "[]"):
		return "array"
	case strings.HasPrefix(goType, "map["):
		return "object"
	default:
		return "object"
	}
}

// isNumericType checks if a type is numeric.
func isNumericType(t string) bool {
	numericTypes := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
	}
	for _, n := range numericTypes {
		if t == n {
			return true
		}
	}
	return false
}

// toPlural converts a singular word to plural (simple English rules).
func toPlural(word string) string {
	// Special cases
	specials := map[string]string{
		"user":    "users",
		"role":    "roles",
		"file":    "files",
		"task":    "tasks",
		"audit":   "audits",
		"menu":    "menus",
		"config":  "configs",
		"setting": "settings",
	}
	if p, ok := specials[word]; ok {
		return p
	}

	// General rules
	if strings.HasSuffix(word, "y") && len(word) > 1 {
		runeBeforeY := rune(word[len(word)-2])
		if !isVowel(runeBeforeY) {
			return word[:len(word)-1] + "ies"
		}
	}

	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") ||
		strings.HasSuffix(word, "z") || strings.HasSuffix(word, "ch") ||
		strings.HasSuffix(word, "sh") {
		return word + "es"
	}

	return word + "s"
}

// isVowel checks if a rune is a vowel.
func isVowel(r rune) bool {
	vowels := map[rune]bool{
		'a': true, 'e': true, 'i': true, 'o': true, 'u': true,
		'A': true, 'E': true, 'I': true, 'O': true, 'U': true,
	}
	return vowels[r]
}

// ListModels lists all available model files in the given directory.
func ListModels(modelDir string) ([]string, error) {
	// Check if directory exists
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		return nil, err
	}

	entries, err := os.ReadDir(modelDir)
	if err != nil {
		return nil, err
	}

	var models []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			// Remove .go extension
			modelName := strings.TrimSuffix(name, ".go")
			// Skip files that start with underscore
			if !strings.HasPrefix(modelName, "_") {
				models = append(models, modelName)
			}
		}
	}

	return models, nil
}

// GetModelFilePath returns the full path for a model file.
func GetModelFilePath(modelDir, modelName string) string {
	return filepath.Join(modelDir, modelName+".go")
}
