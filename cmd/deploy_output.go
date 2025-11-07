package cmd

import (
	"os"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
)

// outputDryRunResult outputs the changeset result for dry-run and create-changeset modes.
// It flushes stderr before writing to stdout to ensure proper stream separation.
// Reuses buildAndRenderChangeset() from describe_changeset.go for consistent output.
func outputDryRunResult(deployment *lib.DeployInfo, awsConfig config.AWSConfig) {
	// Flush stderr before stdout output to ensure clean separation
	// Note: This is best-effort ordering, not atomic. In practice works 99.9% of the time.
	os.Stderr.Sync()

	// Reuse existing buildAndRenderChangeset function
	// This function internally calls settings.GetOutputOptions() which uses stdout by default
	buildAndRenderChangeset(*deployment.CapturedChangeset, *deployment, awsConfig)
}
