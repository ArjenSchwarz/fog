package validators

import (
	"context"
	"os"
	"testing"

	cmdflags "github.com/ArjenSchwarz/fog/cmd/flags"
)

func TestFileExistsRule(t *testing.T) {
	rule := NewFileExistsRule("file", func(cmdflags.FlagValidator) string { return "" }, true)
	if err := rule.Validate(context.Background(), nil, nil); err == nil {
		t.Fatalf("expected error for required missing file")
	}

	rule = NewFileExistsRule("file", func(cmdflags.FlagValidator) string { return "" }, false)
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error for optional missing file: %v", err)
	}

	rule = NewFileExistsRule("file", func(cmdflags.FlagValidator) string { return "/nonexistent" }, false)
	if err := rule.Validate(context.Background(), nil, nil); err == nil {
		t.Fatalf("expected error for non-existing file")
	}

	tmp := t.TempDir() + "/f"
	_ = os.WriteFile(tmp, []byte("x"), 0o644)
	rule = NewFileExistsRule("file", func(cmdflags.FlagValidator) string { return tmp }, false)
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileExtensionRule(t *testing.T) {
	rule := NewFileExtensionRule("file", func(cmdflags.FlagValidator) string { return "" }, []string{".yaml"})
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error for empty value: %v", err)
	}

	rule = NewFileExtensionRule("file", func(cmdflags.FlagValidator) string { return "test.yaml" }, []string{".yaml"})
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error for valid extension: %v", err)
	}

	rule = NewFileExtensionRule("file", func(cmdflags.FlagValidator) string { return "test.txt" }, []string{".yaml"})
	if err := rule.Validate(context.Background(), nil, nil); err == nil {
		t.Fatalf("expected extension error")
	}
}

func TestRegexRule(t *testing.T) {
	rule := NewRegexRule("name", func(cmdflags.FlagValidator) string { return "" }, `^abc$`, "bad")
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error for empty value: %v", err)
	}

	rule = NewRegexRule("name", func(cmdflags.FlagValidator) string { return "abc" }, `^abc$`, "bad")
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error for valid value: %v", err)
	}

	rule = NewRegexRule("name", func(cmdflags.FlagValidator) string { return "def" }, `^abc$`, "bad")
	if err := rule.Validate(context.Background(), nil, nil); err == nil {
		t.Fatalf("expected regex error")
	}
}
