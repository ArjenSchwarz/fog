package cmd

import (
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
)

// TestShowDeploymentInfo_ExistingStackUsesAccountAlias verifies that the
// first line of existing-stack deploy info includes the configured account
// alias, not just the bare account ID. This is a regression test for T-676.
func TestShowDeploymentInfo_ExistingStackUsesAccountAlias(t *testing.T) {
	// Don't run in parallel due to global deployFlags state
	oldFlags := deployFlags
	defer func() { deployFlags = oldFlags }()

	deployFlags = DeployFlags{
		StackName: "existing-stack",
		Dryrun:    false,
	}

	deployment := lib.DeployInfo{
		StackName: "existing-stack",
		IsNew:     false,
	}
	awsCfg := config.AWSConfig{
		Region:       "eu-west-1",
		AccountID:    "987654321098",
		AccountAlias: "staging",
	}

	stderr := captureStderr(func() {
		showDeploymentInfo(deployment, awsCfg, false)
	})

	// Check only the first line — the deployment summary line must use the
	// formatted account display with alias, consistent with new-stack output.
	firstLine := strings.SplitN(stderr, "\n", 2)[0]
	if !strings.Contains(firstLine, "staging (987654321098)") {
		t.Errorf("existing-stack deploy info first line should include account alias.\ngot:  %q\nwant substring: staging (987654321098)", firstLine)
	}
}

// TestShowDeploymentInfo_ExistingStackNoAlias verifies that when no alias is
// configured, the existing-stack first line still shows just the account ID.
func TestShowDeploymentInfo_ExistingStackNoAlias(t *testing.T) {
	oldFlags := deployFlags
	defer func() { deployFlags = oldFlags }()

	deployFlags = DeployFlags{
		StackName: "existing-stack",
		Dryrun:    false,
	}

	deployment := lib.DeployInfo{
		StackName: "existing-stack",
		IsNew:     false,
	}
	awsCfg := config.AWSConfig{
		Region:    "us-east-1",
		AccountID: "111111111111",
	}

	stderr := captureStderr(func() {
		showDeploymentInfo(deployment, awsCfg, false)
	})

	firstLine := strings.SplitN(stderr, "\n", 2)[0]
	if !strings.Contains(firstLine, "account 111111111111") {
		t.Errorf("existing-stack deploy info without alias should show account ID.\ngot: %q", firstLine)
	}
}
