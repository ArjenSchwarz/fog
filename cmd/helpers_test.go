package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestFailWithError_WritesToStderr verifies failWithError keeps diagnostics on
// stderr so structured stdout pipelines only receive command results.
func TestFailWithError_WritesToStderr(t *testing.T) {
	if os.Getenv("GO_WANT_FAIL_WITH_ERROR_HELPER") == "1" {
		failWithError(errors.New("boom"))
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestFailWithError_WritesToStderr$")
	cmd.Env = append(os.Environ(), "GO_WANT_FAIL_WITH_ERROR_HELPER=1")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected subprocess to exit with an error, got %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	if got := stdout.String(); got != "" {
		t.Fatalf("expected no stdout output, got %q", got)
	}
	got := stderr.String()
	if got == "" {
		t.Fatal("expected error output on stderr, got nothing")
	}
	if !strings.Contains(got, "Error: boom") {
		t.Fatalf("expected stderr to contain error message, got %q", got)
	}
}
