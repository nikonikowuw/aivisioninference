// Package validator provides a configured validator instance with
// custom validation rules for request binding.
package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// New creates a new validator.Validate instance with custom validators registered.
func New() *validator.Validate {
	v := validator.New()

	// Use JSON field names for validation error messages.
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return v
}
