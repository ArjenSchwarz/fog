package cmd

import (
	"strings"
	"testing"
)

func TestFormatInfo(t *testing.T) {
	result := formatInfo("Test message")

	if !strings.Contains(result, "ℹ️") {
		t.Error("formatInfo should include info emoji")
	}
	if !strings.Contains(result, "Test message") {
		t.Error("formatInfo should include the message text")
	}
	if !strings.HasSuffix(result, "\n\n") {
		t.Error("formatInfo should end with double newline")
	}
}

func TestFormatSuccess(t *testing.T) {
	result := formatSuccess("Operation completed")

	if !strings.Contains(result, "✅") {
		t.Error("formatSuccess should include success emoji")
	}
	if !strings.Contains(result, "Operation completed") {
		t.Error("formatSuccess should include the message text")
	}
	if !strings.HasSuffix(result, "\n\n") {
		t.Error("formatSuccess should end with double newline")
	}
	// Check for ANSI color codes (green bold)
	if !strings.Contains(result, "\x1b[32") {
		t.Error("formatSuccess should include green color code")
	}
}

func TestFormatError(t *testing.T) {
	result := formatError("Something failed")

	if !strings.Contains(result, "🚨") {
		t.Error("formatError should include warning emoji")
	}
	// Should have emoji on both sides
	if strings.Count(result, "🚨") != 2 {
		t.Error("formatError should have warning emoji on both sides")
	}
	if !strings.Contains(result, "Something failed") {
		t.Error("formatError should include the message text")
	}
	if !strings.HasSuffix(result, "\n\n") {
		t.Error("formatError should end with double newline")
	}
	if !strings.HasPrefix(result, "\n") {
		t.Error("formatError should start with newline")
	}
	// Check for ANSI color codes (red bold)
	if !strings.Contains(result, "\x1b[31") {
		t.Error("formatError should include red color code")
	}
}

func TestFormatPositive(t *testing.T) {
	result := formatPositive("All checks passed")

	if !strings.Contains(result, "All checks passed") {
		t.Error("formatPositive should include the message text")
	}
	if !strings.HasSuffix(result, "\n") {
		t.Error("formatPositive should end with single newline")
	}
	// Check for ANSI color codes (green bold)
	if !strings.Contains(result, "\x1b[32") {
		t.Error("formatPositive should include green color code")
	}
}

func TestFormatBold(t *testing.T) {
	result := formatBold("Important text")

	if !strings.Contains(result, "Important text") {
		t.Error("formatBold should include the message text")
	}
	// Check for ANSI bold code
	if !strings.Contains(result, "\x1b[") {
		t.Error("formatBold should include ANSI formatting")
	}
}
