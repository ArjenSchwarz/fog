package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/texts"
	"github.com/spf13/viper"
)

// withResetViper saves and restores global viper and cfgFile state so config
// tests do not leak settings into one another.
func withResetViper(t *testing.T) {
	t.Helper()
	originalCfgFile := cfgFile
	t.Cleanup(func() {
		cfgFile = originalCfgFile
		viper.Reset()
	})
	viper.Reset()
}

// TestLoadConfigFile_ExplicitMissingReturnsError verifies that an explicitly
// requested --config file that does not exist produces an error rather than
// being silently ignored.
func TestLoadConfigFile_ExplicitMissingReturnsError(t *testing.T) {
	withResetViper(t)
	missing := filepath.Join(t.TempDir(), "does-not-exist.yaml")

	if err := loadConfigFile(missing); err == nil {
		t.Fatalf("expected error for missing explicit config %q, got nil", missing)
	}
}

// TestLoadConfigFile_ValidConfigSucceeds verifies a valid --config file is read
// without error and its values are applied.
func TestLoadConfigFile_ValidConfigSucceeds(t *testing.T) {
	withResetViper(t)
	path := filepath.Join(t.TempDir(), "fog.yaml")
	if err := os.WriteFile(path, []byte("rootdir: from-config\n"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := loadConfigFile(path); err != nil {
		t.Fatalf("expected no error for valid config, got %v", err)
	}
	if got := viper.GetString("rootdir"); got != "from-config" {
		t.Fatalf("expected config value to be loaded, got %q", got)
	}
}

// TestLoadConfigFile_ImplicitMissingTolerated verifies that when no explicit
// --config is provided, a missing default config file is tolerated.
func TestLoadConfigFile_ImplicitMissingTolerated(t *testing.T) {
	withResetViper(t)
	// loadConfigFile("") searches "." and $HOME, so point both at empty temp
	// dirs to guarantee the implicit lookup genuinely finds no config.
	chdirTemp(t)
	t.Setenv("HOME", t.TempDir())

	if err := loadConfigFile(""); err != nil {
		t.Fatalf("expected missing implicit config to be tolerated, got %v", err)
	}
	if used := viper.ConfigFileUsed(); used != "" {
		t.Fatalf("expected no config file to be loaded, got %q", used)
	}
}

// chdirTemp switches the working directory to an empty temp dir for the test
// and restores the original directory on cleanup.
func chdirTemp(t *testing.T) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}

// TestInitConfig_ExplicitMissingExitsNonZero exercises the full CLI path to
// confirm that a missing explicit --config fails fast with a config error,
// matching the bug's reproduction (previously exited 0).
func TestInitConfig_ExplicitMissingExitsNonZero(t *testing.T) {
	if os.Getenv("GO_WANT_INIT_CONFIG_HELPER") == "1" {
		// Subprocess: drive the real CLI so initConfig's error path executes.
		rootCmd.SetArgs([]string{"--config", os.Getenv("GO_WANT_INIT_CONFIG_PATH"), "util", "version"})
		Execute()
		return
	}

	missing := filepath.Join(t.TempDir(), "fog-missing-config.yaml")
	cmd := exec.Command(os.Args[0], "-test.run=^TestInitConfig_ExplicitMissingExitsNonZero$")
	cmd.Env = append(
		envWithout("GO_WANT_INIT_CONFIG_HELPER", "GO_WANT_INIT_CONFIG_PATH"),
		"GO_WANT_INIT_CONFIG_HELPER=1",
		"GO_WANT_INIT_CONFIG_PATH="+missing,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected subprocess to exit non-zero, got err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code, got 0; stdout=%q", stdout.String())
	}
	if !strings.Contains(stderr.String(), missing) {
		t.Fatalf("expected stderr to mention the missing config path, got %q", stderr.String())
	}
}

// TestLoadConfigFile_EnvOverridesDashedAndNestedKeys verifies that
// shell-friendly environment variables (dots and dashes replaced by
// underscores) override values from the config file for dashed top-level keys
// and nested keys, as read through the production settings getters.
func TestLoadConfigFile_EnvOverridesDashedAndNestedKeys(t *testing.T) {
	withResetViper(t)
	path := filepath.Join(t.TempDir(), "fog.yaml")
	configContent := `output-file: from-config.txt
output-file-format: csv
table:
  max-column-width: 20
drift:
  ignore-tags:
    - ConfigTag
`
	if err := os.WriteFile(path, []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	t.Setenv("OUTPUT_FILE", "from-env.json")
	t.Setenv("OUTPUT_FILE_FORMAT", "JSON")
	t.Setenv("TABLE_MAX_COLUMN_WIDTH", "120")
	t.Setenv("DRIFT_IGNORE_TAGS", "EnvTagA EnvTagB")

	if err := loadConfigFile(path); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got, want := settings.GetString("output-file"), "from-env.json"; got != want {
		t.Errorf("output-file: got %q, want %q", got, want)
	}
	if got, want := settings.GetLCString("output-file-format"), "json"; got != want {
		t.Errorf("output-file-format: got %q, want %q", got, want)
	}
	if got, want := settings.GetInt("table.max-column-width"), 120; got != want {
		t.Errorf("table.max-column-width: got %d, want %d", got, want)
	}
	got := settings.GetStringSlice("drift.ignore-tags")
	want := []string{"EnvTagA", "EnvTagB"}
	if !slices.Equal(got, want) {
		t.Errorf("drift.ignore-tags: got %v, want %v", got, want)
	}
}

// TestLoadConfigFile_ConfigUsedWithoutEnv verifies that config file values are
// still used when no matching environment variable is set.
func TestLoadConfigFile_ConfigUsedWithoutEnv(t *testing.T) {
	withResetViper(t)
	path := filepath.Join(t.TempDir(), "fog.yaml")
	if err := os.WriteFile(path, []byte("output-file: from-config.txt\ntable:\n  max-column-width: 20\n"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	// t.Setenv registers restoration; Unsetenv then guarantees the variables
	// are absent even if the developer's shell defines them.
	for _, name := range []string{"OUTPUT_FILE", "TABLE_MAX_COLUMN_WIDTH"} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("failed to unset %s: %v", name, err)
		}
	}

	if err := loadConfigFile(path); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got, want := settings.GetString("output-file"), "from-config.txt"; got != want {
		t.Errorf("output-file: got %q, want %q", got, want)
	}
	if got, want := settings.GetInt("table.max-column-width"), 20; got != want {
		t.Errorf("table.max-column-width: got %d, want %d", got, want)
	}
}

// setRootFlag sets a root persistent flag as if passed on the command line and
// restores its value and Changed state after the test.
func setRootFlag(t *testing.T, name, value string) {
	t.Helper()
	flag := rootCmd.PersistentFlags().Lookup(name)
	if flag == nil {
		t.Fatalf("unknown root flag %q", name)
	}
	original := flag.Value.String()
	t.Cleanup(func() {
		if err := flag.Value.Set(original); err != nil {
			t.Errorf("failed to restore flag %q: %v", name, err)
		}
		flag.Changed = false
	})
	if err := rootCmd.PersistentFlags().Set(name, value); err != nil {
		t.Fatalf("failed to set flag %q: %v", name, err)
	}
}

// writeConfig writes a fog.yaml with the given content to a temp dir and
// returns its path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fog.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	return path
}

// TestLoadConfigFile_FlagOverridesEnv verifies that an explicitly passed flag
// still wins over the corresponding environment variable and config value,
// using the real root flag bindings.
func TestLoadConfigFile_FlagOverridesEnv(t *testing.T) {
	withResetViper(t)
	bindPersistentFlags()
	path := writeConfig(t, "output: table\noutput-file: from-config.txt\n")

	t.Setenv("OUTPUT", "csv")
	t.Setenv("OUTPUT_FILE", "from-env.txt")
	setRootFlag(t, "output", "json")
	setRootFlag(t, "file", "from-flag.txt")

	if err := loadConfigFile(path); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got, want := settings.GetLCString("output"), "json"; got != want {
		t.Errorf("output: got %q, want %q", got, want)
	}
	if got, want := settings.GetString("output-file"), "from-flag.txt"; got != want {
		t.Errorf("output-file: got %q, want %q", got, want)
	}
}

// TestLoadConfigFile_SimpleEnvKeysWithFlagBindings verifies that simple keys
// keep their existing environment names (no prefix) and that, with the real
// flag bindings in place but no flag passed, the environment beats both the
// config file and the flag defaults.
func TestLoadConfigFile_SimpleEnvKeysWithFlagBindings(t *testing.T) {
	withResetViper(t)
	bindPersistentFlags()
	path := writeConfig(t, "output: table\nprofile: config-profile\nregion: us-east-1\n")

	t.Setenv("OUTPUT", "JSON")
	t.Setenv("PROFILE", "env-profile")
	t.Setenv("REGION", "ap-southeast-2")

	if err := loadConfigFile(path); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	tests := map[string]string{
		"output":  "json",
		"profile": "env-profile",
		"region":  "ap-southeast-2",
	}
	for key, want := range tests {
		t.Run(key, func(t *testing.T) {
			if got := settings.GetLCString(key); got != want {
				t.Errorf("%s: got %q, want %q", key, got, want)
			}
		})
	}
}

// TestLoadConfigFile_PrechecksFromEnv verifies that TEMPLATES_PRECHECKS keeps
// command boundaries when read by runPrechecks: a single command with
// arguments stays one command, multiple commands are newline-separated, and an
// unset variable leaves the config file list in place. The config file
// precheck always fails, so a passing result proves the env value replaced it.
func TestLoadConfigFile_PrechecksFromEnv(t *testing.T) {
	tests := map[string]struct {
		env           *string
		wantCount     int
		wantLogStatus lib.DeploymentLogPreChecks
	}{
		"single command with arguments": {
			env:           new("echo hello world"),
			wantCount:     1,
			wantLogStatus: lib.DeploymentLogPreChecksPassed,
		},
		"newline separated commands": {
			env:           new("echo first $TEMPLATEPATH\n  echo second\n\n"),
			wantCount:     2,
			wantLogStatus: lib.DeploymentLogPreChecksPassed,
		},
		"unset env uses config list": {
			wantCount:     1,
			wantLogStatus: lib.DeploymentLogPreChecksFailed,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			withResetViper(t)
			path := writeConfig(t, "templates:\n  prechecks:\n    - sh -c 'exit 1'\n")
			t.Setenv("TEMPLATES_PRECHECKS", "")
			if tc.env != nil {
				t.Setenv("TEMPLATES_PRECHECKS", *tc.env)
			} else if err := os.Unsetenv("TEMPLATES_PRECHECKS"); err != nil {
				t.Fatalf("failed to unset TEMPLATES_PRECHECKS: %v", err)
			}

			if err := loadConfigFile(path); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			info := lib.DeployInfo{TemplateRelativePath: "template.yaml"}
			logObj := lib.DeploymentLog{}
			out, _ := runPrechecks(&info, &logObj)

			if want := fmt.Sprintf(string(texts.FilePrecheckStarted), tc.wantCount); !strings.Contains(out, want) {
				t.Errorf("expected output to contain %q, got %q", want, out)
			}
			if logObj.PreChecks != tc.wantLogStatus {
				t.Errorf("log status: got %q, want %q; output %q", logObj.PreChecks, tc.wantLogStatus, out)
			}
		})
	}
}

// TestLoadConfigFile_PrechecksFromEnvWithSpacedTemplatePath combines the
// T-1668 newline-separated TEMPLATES_PRECHECKS form with T-1403's
// split-before-substitution: each line must stay one precheck, and a
// template path containing spaces must reach the command as a single argument
// whether $TEMPLATEPATH is quoted or not. "test X = Y" only succeeds with
// exactly three arguments where X equals Y, so either a split command or a
// split path makes the precheck fail. The config file precheck always fails,
// so a pass also proves the env value replaced it.
func TestLoadConfigFile_PrechecksFromEnvWithSpacedTemplatePath(t *testing.T) {
	withResetViper(t)
	path := writeConfig(t, "templates:\n  prechecks:\n    - sh -c 'exit 1'\n")
	t.Setenv("TEMPLATES_PRECHECKS", `test $TEMPLATEPATH = "path with spaces/template.yaml"
test "$TEMPLATEPATH" = "path with spaces/template.yaml"`)

	if err := loadConfigFile(path); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	info := lib.DeployInfo{TemplateRelativePath: "path with spaces/template.yaml"}
	logObj := lib.DeploymentLog{}
	out, abort := runPrechecks(&info, &logObj)

	if want := fmt.Sprintf(string(texts.FilePrecheckStarted), 2); !strings.Contains(out, want) {
		t.Errorf("expected output to contain %q, got %q", want, out)
	}
	if info.PrechecksFailed || logObj.PreChecks != lib.DeploymentLogPreChecksPassed {
		t.Errorf("expected prechecks to pass, got PrechecksFailed=%v status=%q; output %q", info.PrechecksFailed, logObj.PreChecks, out)
	}
	if abort {
		t.Errorf("expected abort=false")
	}
}
