package cmd

import (
	"bytes"
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

	tests := map[string]string{
		"stack describe": "stack describe",
		"describe alias": "describe",
		"unknown subcmd": "stack describe nonexistent",
		"alias unknown":  "describe nonexistent",
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestDescribeCmd_NoSubcommandShowsHelp$")
			cmd.Dir = t.TempDir()
			cmd.Env = append(
				envWithout("GO_WANT_DESCRIBE_HELPER_ARGS", "HOME"),
				"GO_WANT_DESCRIBE_HELPER_ARGS="+tc,
				// Isolate from any user-level fog config.
				"HOME="+t.TempDir(),
			)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				t.Fatalf("expected exit code 0, got err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
			}

			out := stdout.String()
			if strings.Contains(out, "describe called") {
				t.Fatalf("placeholder output still printed: %q", out)
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
// subcommand is still reachable under describe after the parent stopped being
// runnable.
func TestDescribeCmd_ChangesetSubcommandResolves(t *testing.T) {
	if describeCmd.Runnable() {
		t.Fatal("describe parent command should not be runnable on its own")
	}

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
