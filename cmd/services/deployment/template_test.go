package deployment

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/spf13/viper"
)

// TestTemplateServiceLoadAndValidate covers template loading from the configured
// directory and basic validation error scenarios.
func TestTemplateServiceLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "tmpl.yaml")
	content := "Resources: {}"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("setup file: %v", err)
	}
	viper.Set("templates.directory", dir)
	viper.Set("templates.extensions", []string{".yaml"})

	ts := NewTemplateService(nil)
	ctx := context.Background()

	tmpl, err := ts.LoadTemplate(ctx, "tmpl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl.Content != content {
		t.Errorf("template content mismatch: got %q, want %q", tmpl.Content, content)
	}
	if tmpl.LocalPath != filePath {
		t.Errorf("template local path mismatch: got %q, want %q", tmpl.LocalPath, filePath)
	}
	if err := ts.ValidateTemplate(ctx, tmpl); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	_, err = ts.LoadTemplate(ctx, "missing")
	if err == nil {
		t.Fatalf("expected error for missing template")
	}

	if err := ts.ValidateTemplate(ctx, &services.Template{}); err == nil {
		t.Fatalf("expected validation error for empty content")
	}
}
