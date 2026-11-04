package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// executeRoot runs the real root command with the given arguments and returns
// the error and combined Cobra output. Config lookup is pointed at empty temp
// dirs so a local fog.yaml cannot influence the result.
func executeRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	chdirTemp(t)
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	err := rootCmd.Execute()
	return out.String(), err
}

// TestUnexpectedPositionalArgsAreRejected covers the T-1454 reproductions:
// extra positional arguments on leaf commands and unknown subcommands on
// command groups must return an error instead of succeeding.
func TestUnexpectedPositionalArgsAreRejected(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantErr string
	}{
		"extra arg on util version": {
			args:    []string{"util", "version", "extra-arg"},
			wantErr: `unknown command "extra-arg" for "fog util version"`,
		},
		"unknown stack subcommand": {
			args:    []string{"stack", "does-not-exist"},
			wantErr: `unknown command "does-not-exist" for "fog stack"`,
		},
		"unknown describe subcommand": {
			args:    []string{"stack", "describe", "does-not-exist"},
			wantErr: `unknown command "does-not-exist" for "fog stack describe"`,
		},
		"unknown resource subcommand": {
			args:    []string{"resource", "does-not-exist"},
			wantErr: `unknown command "does-not-exist" for "fog resource"`,
		},
		"unknown util demo subcommand": {
			args:    []string{"util", "demo", "does-not-exist"},
			wantErr: `unknown command "does-not-exist" for "fog util demo"`,
		},
		"unknown help topic": {
			args:    []string{"help", "does-not-exist"},
			wantErr: `unknown command "does-not-exist" for "fog"`,
		},
		"unknown nested help topic": {
			args:    []string{"help", "stack", "does-not-exist"},
			wantErr: `unknown command "does-not-exist" for "fog stack"`,
		},
		"unknown completion shell": {
			args:    []string{"completion", "does-not-exist"},
			wantErr: `unknown command "does-not-exist" for "fog completion"`,
		},
		// Leaf commands that call AWS are covered by
		// TestAllCommandsValidatePositionalArgs instead, so a regression here
		// can never fall through into AWS code paths.
		"extra arg after flags on util demo settings": {
			args:    []string{"util", "demo", "settings", "--output", "json", "extra-arg"},
			wantErr: `unknown command "extra-arg" for "fog util demo settings"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := executeRoot(t, tc.args...)
			if err == nil {
				t.Fatalf("expected an error for %v, got nil", tc.args)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("got error %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestGroupTypoSuggestsSubcommand checks that a mistyped subcommand on a group
// includes Cobra-style suggestions, as the root command already does.
func TestGroupTypoSuggestsSubcommand(t *testing.T) {
	_, err := executeRoot(t, "stack", "deplyo")
	if err == nil {
		t.Fatal("expected an error for a mistyped subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "Did you mean this?\n\tdeploy") {
		t.Fatalf("expected a suggestion for deploy, got %q", err.Error())
	}
}

// TestValidInvocationsStillSucceed guards against the validators rejecting
// supported usage: groups without arguments show their help, help topics
// resolve, and a leaf command without arguments runs.
func TestValidInvocationsStillSucceed(t *testing.T) {
	tests := map[string]struct {
		args     []string
		wantHelp string
	}{
		"stack group shows help": {
			args:     []string{"stack"},
			wantHelp: "Available Commands:",
		},
		"util version runs": {
			args: []string{"util", "version"},
		},
		"stack describe group shows help": {
			args:     []string{"stack", "describe"},
			wantHelp: "changeset",
		},
		"help for nested command": {
			args:     []string{"help", "stack", "describe", "changeset"},
			wantHelp: "fog stack describe changeset",
		},
		"help for root alias": {
			args:     []string{"help", "deploy"},
			wantHelp: "Alias for 'stack deploy'",
		},
		"completion group shows help": {
			args:     []string{"completion"},
			wantHelp: "Available Commands:",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			out, err := executeRoot(t, tc.args...)
			if err != nil {
				t.Fatalf("expected no error for %v, got %v", tc.args, err)
			}
			if tc.wantHelp != "" && !strings.Contains(out, tc.wantHelp) {
				t.Fatalf("expected output to contain %q, got %q", tc.wantHelp, out)
			}
		})
	}
}

// TestAllCommandsValidatePositionalArgs walks the real command tree, including
// Cobra's generated help and completion commands, so a newly added command
// without an Args validator is caught. Commands with DisableFlagParsing (the
// root-level aliases and Cobra's hidden __complete) are skipped because they
// forward their arguments to another command, which performs the validation.
func TestAllCommandsValidatePositionalArgs(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.DisableFlagParsing {
				continue
			}
			t.Run(sub.CommandPath(), func(t *testing.T) {
				if sub.Args == nil {
					t.Fatalf("%s has no Args validator", sub.CommandPath())
				}
				if !sub.Runnable() {
					t.Fatalf("%s is not runnable, so Cobra would skip its Args validator", sub.CommandPath())
				}
				if err := sub.ValidateArgs([]string{"unexpected"}); err == nil {
					t.Fatalf("%s accepted an unexpected positional argument", sub.CommandPath())
				}
				if err := sub.ValidateArgs(nil); err != nil {
					t.Fatalf("%s rejected an empty argument list: %v", sub.CommandPath(), err)
				}
			})
			walk(sub)
		}
	}
	walk(rootCmd)
}

// TestHelpTopicThroughAlias drives the real CLI in a subprocess to check that
// help topics given through a root-level alias are validated against the
// alias target: valid paths show help and exit 0, unknown words exit non-zero.
func TestHelpTopicThroughAlias(t *testing.T) {
	if args := os.Getenv("GO_WANT_HELP_ALIAS_ARGS"); args != "" {
		// Subprocess: run the CLI exactly as a user would.
		rootCmd.SetArgs(strings.Fields(args))
		Execute()
		return
	}

	tests := map[string]struct {
		args    string
		wantErr string
	}{
		"alias topic":                   {args: "help deploy"},
		"alias topic with subcommand":   {args: "help describe changeset"},
		"alias to nested command":       {args: "help resources"},
		"unknown word after alias":      {args: "help deploy nope", wantErr: `unknown command "nope" for "fog stack deploy"`},
		"unknown subcommand of alias":   {args: "help describe nope", wantErr: `unknown command "nope" for "fog stack describe"`},
		"extra word after alias subcmd": {args: "help describe changeset nope", wantErr: `unknown command "nope" for "fog stack describe changeset"`},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestHelpTopicThroughAlias$")
			cmd.Dir = t.TempDir()
			cmd.Env = append(
				envWithout("GO_WANT_HELP_ALIAS_ARGS", "HOME"),
				"GO_WANT_HELP_ALIAS_ARGS="+tc.args,
				// Isolate from any user-level fog config.
				"HOME="+t.TempDir(),
			)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected exit 0, got err=%v stderr=%q", err, stderr.String())
				}
				if !strings.Contains(stdout.String(), "Usage:") {
					t.Fatalf("expected help output, got %q", stdout.String())
				}
				return
			}

			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() == 0 {
				t.Fatalf("expected non-zero exit, got err=%v stdout=%q", err, stdout.String())
			}
			if !strings.Contains(stderr.String(), tc.wantErr) {
				t.Fatalf("expected stderr to contain %q, got %q", tc.wantErr, stderr.String())
			}
		})
	}
}
