package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestRootCommandForms_Documented runs the real CLI (via a re-executed test
// binary) for the root-level command forms that the README and user guide
// document, plus the `fog util ...` paths they alias. Running in a subprocess
// matters because the alias commands re-invoke rootCmd.Execute() and call
// log.Fatal on errors, which would otherwise leak state or kill the test run.
// Regression test for T-1563.
func TestRootCommandForms_Documented(t *testing.T) {
	if os.Getenv("GO_WANT_ROOT_COMMAND_HELPER") == "1" {
		rootCmd.SetArgs(strings.Split(os.Getenv("GO_WANT_ROOT_COMMAND_ARGS"), "\n"))
		Execute()
		return
	}

	tests := map[string]struct {
		args []string
		want string
	}{
		"root version flag": {
			args: []string{"--version"},
			want: "fog version " + Version,
		},
		"util version": {
			args: []string{"util", "version"},
			want: Version,
		},
		"root demo tables alias": {
			args: []string{"demo", "tables"},
			want: "Tables are often used for the various outputs",
		},
		"root demo settings alias": {
			args: []string{"demo", "settings"},
			want: "As fog.yml:",
		},
		"util demo tables": {
			args: []string{"util", "demo", "tables"},
			want: "Tables are often used for the various outputs",
		},
		"util demo settings": {
			args: []string{"util", "demo", "settings"},
			want: "As fog.yml:",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, err := runRootCommandHelper(t, tc.args)
			if err != nil {
				t.Fatalf("fog %s failed: %v\nstdout=%q\nstderr=%q", strings.Join(tc.args, " "), err, stdout, stderr)
			}
			if !strings.Contains(stdout, tc.want) {
				t.Fatalf("fog %s: expected stdout to contain %q, got %q", strings.Join(tc.args, " "), tc.want, stdout)
			}
		})
	}
}

// TestRootCommandForms_VerboseShorthandKept guards against the --version flag
// claiming the -v shorthand, which is already used for --verbose.
func TestRootCommandForms_VerboseShorthandKept(t *testing.T) {
	stdout, stderr, err := runRootCommandHelper(t, []string{"-v", "util", "version"})
	if err != nil {
		t.Fatalf("fog -v util version failed: %v\nstdout=%q\nstderr=%q", err, stdout, stderr)
	}
	// The re-executed test binary appends its own "PASS" line, so only the
	// first line is the command's output.
	got, _, _ := strings.Cut(stdout, "\n")
	if got != Version {
		t.Fatalf("expected -v to act as --verbose and print %q, got %q", Version, got)
	}
}

// runRootCommandHelper re-executes the test binary so that
// TestRootCommandForms_Documented's helper branch drives the CLI with args.
// The working directory and HOME point at empty temp dirs so no local fog
// config file influences the result.
func runRootCommandHelper(t *testing.T, args []string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRootCommandForms_Documented$")
	cmd.Dir = t.TempDir()
	cmd.Env = append(
		envWithout("GO_WANT_ROOT_COMMAND_HELPER", "GO_WANT_ROOT_COMMAND_ARGS", "HOME"),
		"GO_WANT_ROOT_COMMAND_HELPER=1",
		"GO_WANT_ROOT_COMMAND_ARGS="+strings.Join(args, "\n"),
		"HOME="+t.TempDir(),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
