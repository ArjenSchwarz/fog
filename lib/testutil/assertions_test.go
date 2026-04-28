package testutil

import (
	"os"
	"os/exec"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertStackParameter_NilValue(t *testing.T) {
	t.Parallel()

	output := runAssertionFailureHelper(t, "parameter-nil-value")

	assertAssertionFailure(t, output,
		"Parameter Environment value mismatch",
		"Got: <nil>",
		"Expected: production",
	)
}

func TestAssertStackOutput_NilValue(t *testing.T) {
	t.Parallel()

	output := runAssertionFailureHelper(t, "output-nil-value")

	assertAssertionFailure(t, output,
		"Output ServiceURL value mismatch",
		"Got: <nil>",
		"Expected: https://example.com",
	)
}

func TestAssertStackTag_NilValue(t *testing.T) {
	t.Parallel()

	output := runAssertionFailureHelper(t, "tag-nil-value")

	assertAssertionFailure(t, output,
		"Tag Environment value mismatch",
		"Got: <nil>",
		"Expected: production",
	)
}

func TestAssertionFailureHelper(t *testing.T) {
	t.Helper()

	if os.Getenv("FOG_ASSERTION_HELPER") != "1" {
		t.Skip("helper process only")
	}

	switch os.Getenv("FOG_ASSERTION_SCENARIO") {
	case "parameter-nil-value":
		AssertStackParameter(t, &types.Stack{
			Parameters: []types.Parameter{
				{
					ParameterKey: aws.String("Environment"),
				},
			},
		}, "Environment", "production")
	case "output-nil-value":
		AssertStackOutput(t, &types.Stack{
			Outputs: []types.Output{
				{
					OutputKey: aws.String("ServiceURL"),
				},
			},
		}, "ServiceURL", "https://example.com")
	case "tag-nil-value":
		AssertStackTag(t, &types.Stack{
			Tags: []types.Tag{
				{
					Key: aws.String("Environment"),
				},
			},
		}, "Environment", "production")
	default:
		t.Fatalf("unknown helper scenario %q", os.Getenv("FOG_ASSERTION_SCENARIO"))
	}
}

func runAssertionFailureHelper(t *testing.T, scenario string) string {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestAssertionFailureHelper$")
	cmd.Env = append(os.Environ(),
		"FOG_ASSERTION_HELPER=1",
		"FOG_ASSERTION_SCENARIO="+scenario,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err, "expected helper assertion to fail")

	return string(output)
}

func assertAssertionFailure(t *testing.T, output string, expectedParts ...string) {
	t.Helper()

	assert.NotContains(t, output, "panic:", "assertion helper should fail cleanly without panicking")

	for _, expectedPart := range expectedParts {
		assert.Contains(t, output, expectedPart)
	}
}
