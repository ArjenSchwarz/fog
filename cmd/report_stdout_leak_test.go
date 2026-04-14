package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// TestBuildStackReports_NoRawStackNamesToStdout verifies that building a
// report for multiple stacks does not leak raw stack names to stdout. Before
// the fix, the stack iteration loop contained a bare fmt.Println(stackkey)
// which injected plain text into stdout for every stack. This polluted
// machine-readable output (JSON, CSV) and mixed unexpected lines into CLI
// and Lambda report output.
func TestBuildStackReports_NoRawStackNamesToStdout(t *testing.T) {
	// Not parallel: captureBothStreams redirects global os.Stdout,
	// which would interfere with other parallel tests writing to stdout.
	viper.Reset()
	viper.SetDefault("timezone", "UTC")
	t.Cleanup(func() { viper.Reset() })

	oldSettings := settings
	settings = &config.Config{}
	t.Cleanup(func() { settings = oldSettings })

	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	awsConfig := config.AWSConfig{
		AccountID: "111111111111",
		Region:    "us-east-1",
	}

	// Pre-populate events so GetEvents doesn't call AWS
	stackAlpha := lib.CfnStack{
		Name: "alpha-stack",
		Id:   "arn:aws:cloudformation:us-east-1:111111111111:stack/alpha-stack/aaa",
		Events: []lib.StackEvent{
			{
				Type:      "Create",
				Success:   true,
				StartDate: now,
				EndDate:   now.Add(30 * time.Second),
			},
		},
	}
	stackBeta := lib.CfnStack{
		Name: "beta-stack",
		Id:   "arn:aws:cloudformation:us-east-1:111111111111:stack/beta-stack/bbb",
		Events: []lib.StackEvent{
			{
				Type:      "Update",
				Success:   true,
				StartDate: now.Add(1 * time.Hour),
				EndDate:   now.Add(1*time.Hour + 45*time.Second),
			},
		},
	}

	stacks := map[string]lib.CfnStack{
		stackAlpha.Id: stackAlpha,
		stackBeta.Id:  stackBeta,
	}

	stackKeys := []string{stackAlpha.Id, stackBeta.Id}

	var buildErr, renderErr error

	stdout, _ := captureBothStreams(func() {
		ctx := context.Background()
		doc := output.New()
		doc.Header("Test Report")

		// Call the extracted function that contains the stack iteration loop
		if err := buildStackReports(ctx, stacks, stackKeys, doc, awsConfig); err != nil {
			buildErr = err
			return
		}

		builtDoc := doc.Build()
		out := output.NewOutput(
			output.WithFormat(output.JSON()),
			output.WithWriter(output.NewStdoutWriter()),
		)
		if err := out.Render(context.Background(), builtDoc); err != nil {
			renderErr = err
		}
	})

	if buildErr != nil {
		t.Fatalf("buildStackReports returned error: %v", buildErr)
	}
	if renderErr != nil {
		t.Fatalf("failed to render output: %v", renderErr)
	}

	// Verify that raw stack ARNs/keys do not appear as standalone lines in stdout.
	// They may legitimately appear inside rendered JSON values, but should never
	// be on their own line outside of the rendered output.
	for _, key := range stackKeys {
		for _, line := range strings.Split(stdout, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == key {
				t.Errorf("raw stack key %q leaked to stdout as a standalone line", key)
			}
		}
	}
}
