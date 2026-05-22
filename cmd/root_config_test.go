package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
	// Search in an empty directory so the default lookup finds nothing.
	dir := t.TempDir()
	viper.AddConfigPath(dir)
	viper.SetConfigName("fog")

	if err := loadConfigFile(""); err != nil {
		t.Fatalf("expected missing implicit config to be tolerated, got %v", err)
	}
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
