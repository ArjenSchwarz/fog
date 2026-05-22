package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/spf13/viper"
)

// errRoutingExitPanic signals our stubbed osExitFunc fired so we can stop the
// flow without terminating the test process.
const errRoutingExitPanic = "osExit called"

// runDeployWithInterceptedExit runs f with osExitFunc stubbed and os.Stdout/os.Stderr
// captured. The stubbed exit panics to unwind execution; the panic is recovered
// here so the captured streams are still returned. This proves error paths
// route through failWithError (osExitFunc + stderr) rather than log.Fatalln,
// which would terminate the test process and bypass the stub.
func runDeployWithInterceptedExit(t *testing.T, f func()) (code int, called bool, stdout, stderr string) {
	t.Helper()
	origExit := osExitFunc
	osExitFunc = func(c int) {
		code = c
		called = true
		panic(errRoutingExitPanic)
	}

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}
	defer rOut.Close()
	defer rErr.Close()
	os.Stdout = wOut
	os.Stderr = wErr

	func() {
		defer func() {
			wOut.Close()
			wErr.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr
			osExitFunc = origExit
			if r := recover(); r != nil && r != errRoutingExitPanic {
				panic(r)
			}
		}()
		f()
	}()

	var bufOut, bufErr bytes.Buffer
	if _, err := io.Copy(&bufOut, rOut); err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	if _, err := io.Copy(&bufErr, rErr); err != nil {
		t.Fatalf("failed to read captured stderr: %v", err)
	}
	return code, called, bufOut.String(), bufErr.String()
}

// TestLoadParametersFromFiles_RoutesThroughSharedHandler is a regression test
// for T-1185: a parameter-file read failure must route through failWithError
// (osExitFunc + stderr), not log.Fatalln. stdout must stay clean.
func TestLoadParametersFromFiles_RoutesThroughSharedHandler(t *testing.T) {
	origFlags := deployFlags
	defer func() { deployFlags = origFlags; viper.Reset() }()
	viper.Set("debug", false)

	code, called, stdout, stderr := runDeployWithInterceptedExit(t, func() {
		loadParametersFromFiles("does-not-exist-t1185")
	})

	if !called {
		t.Fatal("expected osExitFunc to be called via failWithError, got log.Fatalln behavior instead")
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if stdout != "" {
		t.Errorf("expected clean stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error:") {
		t.Errorf("expected shared handler error on stderr, got %q", stderr)
	}
}

// TestLoadTagsFromFiles_RoutesThroughSharedHandler is a regression test for
// T-1185: a tag-file read failure must route through failWithError.
func TestLoadTagsFromFiles_RoutesThroughSharedHandler(t *testing.T) {
	origFlags := deployFlags
	defer func() { deployFlags = origFlags; viper.Reset() }()
	viper.Set("debug", false)

	code, called, stdout, stderr := runDeployWithInterceptedExit(t, func() {
		loadTagsFromFiles("does-not-exist-t1185", &lib.DeployInfo{})
	})

	if !called {
		t.Fatal("expected osExitFunc to be called via failWithError, got log.Fatalln behavior instead")
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if stdout != "" {
		t.Errorf("expected clean stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error:") {
		t.Errorf("expected shared handler error on stderr, got %q", stderr)
	}
}

// TestReadTemplateFromSource_RoutesThroughSharedHandler is a regression test
// for T-1185: a template read failure must route through failWithError.
func TestReadTemplateFromSource_RoutesThroughSharedHandler(t *testing.T) {
	origFlags := deployFlags
	defer func() { deployFlags = origFlags; viper.Reset() }()
	viper.Set("debug", false)
	viper.Set("templates.directory", t.TempDir())
	deployFlags = DeployFlags{Template: "does-not-exist-t1185"}

	code, called, stdout, stderr := runDeployWithInterceptedExit(t, func() {
		readTemplateFromSource(&lib.DeployInfo{})
	})

	if !called {
		t.Fatal("expected osExitFunc to be called via failWithError, got log.Fatalln behavior instead")
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if stdout != "" {
		t.Errorf("expected clean stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error:") {
		t.Errorf("expected shared handler error on stderr, got %q", stderr)
	}
}

// TestDeployTemplate_PrepareErrorRoutesThroughSharedHandler is a regression
// test for T-1185: prepareDeployment failures in deployTemplate must route
// through failWithError (testable osExitFunc) instead of a bare os.Exit.
func TestDeployTemplate_PrepareErrorRoutesThroughSharedHandler(t *testing.T) {
	origLoadAWSConfig := loadAWSConfig
	origFlags := deployFlags
	defer func() {
		loadAWSConfig = origLoadAWSConfig
		deployFlags = origFlags
		viper.Reset()
	}()
	viper.Set("debug", false)

	// An invalid flag combination makes deployFlags.Validate() return an error
	// from prepareDeployment before any AWS calls.
	deployFlags = DeployFlags{}
	loadAWSConfig = func(_ context.Context, _ config.Config) (config.AWSConfig, error) {
		t.Fatal("loadAWSConfig should not be reached when validation fails")
		return config.AWSConfig{}, nil
	}

	code, called, stdout, stderr := runDeployWithInterceptedExit(t, func() {
		deployTemplate(nil, nil)
	})

	if !called {
		t.Fatal("expected osExitFunc to be called via failWithError")
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if stdout != "" {
		t.Errorf("expected clean stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error:") {
		t.Errorf("expected shared handler error on stderr, got %q", stderr)
	}
}
