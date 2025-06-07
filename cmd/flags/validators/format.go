package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ArjenSchwarz/fog/cmd/flags"
)

// FileExistsRule validates that a file exists
type FileExistsRule struct {
	FieldName   string
	GetValue    func(flags.FlagValidator) string
	Required    bool
	Description string
}

// NewFileExistsRule creates a new file exists rule
func NewFileExistsRule(fieldName string, getValue func(flags.FlagValidator) string, required bool) *FileExistsRule {
	return &FileExistsRule{
		FieldName:   fieldName,
		GetValue:    getValue,
		Required:    required,
		Description: fmt.Sprintf("%s must be a valid file path", fieldName),
	}
}

// Validate checks if the file exists
func (f *FileExistsRule) Validate(ctx context.Context, flags flags.FlagValidator, vCtx *flags.ValidationContext) error {
	value := f.GetValue(flags)
	if value == "" {
		if f.Required {
			return fmt.Errorf("file path for '%s' is required", f.FieldName)
		}
		return nil
	}
	if _, err := os.Stat(value); os.IsNotExist(err) {
		return fmt.Errorf("file '%s' specified for '%s' does not exist", value, f.FieldName)
	}
	return nil
}

// GetDescription returns the rule description
func (f *FileExistsRule) GetDescription() string { return f.Description }

// GetSeverity returns the rule severity
func (f *FileExistsRule) GetSeverity() flags.ValidationSeverity { return flags.SeverityError }

// FileExtensionRule validates file extensions
type FileExtensionRule struct {
	FieldName         string
	GetValue          func(flags.FlagValidator) string
	AllowedExtensions []string
	Description       string
}

// NewFileExtensionRule creates a new file extension rule
func NewFileExtensionRule(fieldName string, getValue func(flags.FlagValidator) string, allowedExtensions []string) *FileExtensionRule {
	return &FileExtensionRule{
		FieldName:         fieldName,
		GetValue:          getValue,
		AllowedExtensions: allowedExtensions,
		Description:       fmt.Sprintf("%s must have one of these extensions: %v", fieldName, allowedExtensions),
	}
}

// Validate checks the file extension
func (f *FileExtensionRule) Validate(ctx context.Context, flags flags.FlagValidator, vCtx *flags.ValidationContext) error {
	value := f.GetValue(flags)
	if value == "" {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(value))
	for _, allowedExt := range f.AllowedExtensions {
		if ext == allowedExt {
			return nil
		}
	}
	return fmt.Errorf("file '%s' has invalid extension '%s', allowed: %v", value, ext, f.AllowedExtensions)
}

// GetDescription returns the rule description
func (f *FileExtensionRule) GetDescription() string { return f.Description }

// GetSeverity returns the rule severity
func (f *FileExtensionRule) GetSeverity() flags.ValidationSeverity { return flags.SeverityWarning }

// RegexRule validates values against a regular expression
type RegexRule struct {
	FieldName   string
	GetValue    func(flags.FlagValidator) string
	Pattern     *regexp.Regexp
	ErrorMsg    string
	Description string
}

// NewRegexRule creates a new regex validation rule
func NewRegexRule(fieldName string, getValue func(flags.FlagValidator) string, pattern string, errorMsg string) *RegexRule {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("Invalid regex pattern: %s", pattern))
	}
	return &RegexRule{
		FieldName:   fieldName,
		GetValue:    getValue,
		Pattern:     regex,
		ErrorMsg:    errorMsg,
		Description: fmt.Sprintf("%s must match pattern: %s", fieldName, pattern),
	}
}

// Validate checks the regex pattern
func (r *RegexRule) Validate(ctx context.Context, flags flags.FlagValidator, vCtx *flags.ValidationContext) error {
	value := r.GetValue(flags)
	if value == "" {
		return nil
	}
	if !r.Pattern.MatchString(value) {
		return fmt.Errorf("'%s' for field '%s': %s", value, r.FieldName, r.ErrorMsg)
	}
	return nil
}

// GetDescription returns the rule description
func (r *RegexRule) GetDescription() string { return r.Description }

// GetSeverity returns the rule severity
func (r *RegexRule) GetSeverity() flags.ValidationSeverity { return flags.SeverityError }
