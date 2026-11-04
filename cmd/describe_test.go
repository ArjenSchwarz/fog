package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestDescribeCmd_NoSubcommandShowsHelp drives the real CLI in a subprocess to
// confirm that invoking the describe parent command without a subcommand shows
// its help (including the changeset subcommand) instead of the old
// "describe called" placeholder (T-1373). Both the `stack describe` path and the
// root-level `describe` alias are covered, since the alias re-executes rootCmd.
func TestDescribeCmd_NoSubcommandShowsHelp(t *testing.T) {
	if args := os.Getenv("GO_WANT_DESCRIBE_HELPER_ARGS"); args != "" {
		// Subprocess: run the CLI exactly as a user would.
		rootCmd.SetArgs(strings.Fields(args))
		Execute()
		return
	}

	tests := map[string]struct {
		args string
		// wantErr is empty when the command should show help and exit 0, or
		// the error text expected on stderr with a non-zero exit (T-1454).
		wantErr string
	}{
		"stack describe": {args: "stack describe"},
		"describe alias": {args: "describe"},
		"unknown subcmd": {
			args:    "stack describe nonexistent",
			wantErr: `unknown command "nonexistent" for "fog stack describe"`,
		},
		"alias unknown": {
			args:    "describe nonexistent",
			wantErr: `unknown command "nonexistent" for "fog stack describe"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestDescribeCmd_NoSubcommandShowsHelp$")
			cmd.Dir = t.TempDir()
			cmd.Env = append(
				envWithout("GO_WANT_DESCRIBE_HELPER_ARGS", "HOME"),
				"GO_WANT_DESCRIBE_HELPER_ARGS="+tc.args,
				// Isolate from any user-level fog config.
				"HOME="+t.TempDir(),
			)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			out := stdout.String()
			if strings.Contains(out, "describe called") {
				t.Fatalf("placeholder output still printed: %q", out)
			}

			if tc.wantErr != "" {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) || exitErr.ExitCode() == 0 {
					t.Fatalf("expected non-zero exit, got err=%v stdout=%q stderr=%q", err, out, stderr.String())
				}
				if !strings.Contains(stderr.String(), tc.wantErr) {
					t.Fatalf("expected stderr to contain %q, got %q", tc.wantErr, stderr.String())
				}
				return
			}

			if err != nil {
				t.Fatalf("expected exit code 0, got err=%v stdout=%q stderr=%q", err, out, stderr.String())
			}
			for _, want := range []string{"Usage:", "Available Commands:", "changeset"} {
				if !strings.Contains(out, want) {
					t.Errorf("expected help output to contain %q, got %q", want, out)
				}
			}
		})
	}
}

// TestDescribeCmd_ChangesetSubcommandResolves verifies the changeset
// subcommand is still reachable under describe now that the parent only shows
// help.
func TestDescribeCmd_ChangesetSubcommandResolves(t *testing.T) {

	found, _, err := rootCmd.Find([]string{"stack", "describe", "changeset"})
	if err != nil {
		t.Fatalf("failed to find stack describe changeset: %v", err)
	}
	if found != changesetCmd {
		t.Fatalf("expected changeset command, got %q", found.CommandPath())
	}
	if !found.Runnable() {
		t.Fatal("changeset subcommand should remain runnable")
	}
}
