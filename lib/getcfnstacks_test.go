package lib

import (
	"context"
	"testing"

	"github.com/ArjenSchwarz/fog/lib/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetCfnStacks_MapKeysAreStackNames verifies that GetCfnStacks returns a
// map keyed by stack name (not stack ID/ARN). Callers rely on map keys being
// human-readable stack names for filtering, sorting, and display.
func TestGetCfnStacks_MapKeysAreStackNames(t *testing.T) {
	stack1 := testutil.NewStackBuilder("my-app-stack").
		WithDescription("Application stack").
		Build()
	stack2 := testutil.NewStackBuilder("my-db-stack").
		WithDescription("Database stack").
		Build()

	client := testutil.NewMockCFNClient().
		WithStack(stack1).
		WithStack(stack2)

	emptyFilter := ""
	result, err := GetCfnStacks(context.Background(), &emptyFilter, client)
	require.NoError(t, err)

	// The map must be keyed by stack name, NOT by stack ID (ARN)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "my-app-stack", "map key should be the stack name, not the stack ID")
	assert.Contains(t, result, "my-db-stack", "map key should be the stack name, not the stack ID")

	// Verify stack names are NOT ARN-style IDs
	for key := range result {
		assert.NotContains(t, key, "arn:aws:", "map key should not be an ARN")
	}
}

// TestGetCfnStacks_StackNameFieldMatchesKey verifies that each CfnStack's Name
// field matches the map key it is stored under.
func TestGetCfnStacks_StackNameFieldMatchesKey(t *testing.T) {
	stack := testutil.NewStackBuilder("test-stack").Build()

	client := testutil.NewMockCFNClient().WithStack(stack)

	emptyFilter := ""
	result, err := GetCfnStacks(context.Background(), &emptyFilter, client)
	require.NoError(t, err)
	require.Len(t, result, 1)

	for key, cfnStack := range result {
		assert.Equal(t, key, cfnStack.Name, "map key must equal the CfnStack.Name field")
	}
}

// TestGetCfnStacks_GlobFilterUsesStackName verifies that glob-style stack name
// filtering works correctly when the map is keyed by stack name.
func TestGetCfnStacks_GlobFilterUsesStackName(t *testing.T) {
	stack1 := testutil.NewStackBuilder("dev-app-stack").Build()
	stack2 := testutil.NewStackBuilder("prod-app-stack").Build()
	stack3 := testutil.NewStackBuilder("dev-db-stack").Build()

	client := testutil.NewMockCFNClient().
		WithStack(stack1).
		WithStack(stack2).
		WithStack(stack3)

	filter := "dev-*"
	result, err := GetCfnStacks(context.Background(), &filter, client)
	require.NoError(t, err)

	assert.Len(t, result, 2, "glob filter 'dev-*' should match exactly 2 stacks")
	assert.Contains(t, result, "dev-app-stack")
	assert.Contains(t, result, "dev-db-stack")
	assert.NotContains(t, result, "prod-app-stack")
}

// TestGetCfnStacks_SpecificStackFilter verifies that filtering by exact stack
// name returns only that stack.
func TestGetCfnStacks_SpecificStackFilter(t *testing.T) {
	stack := testutil.NewStackBuilder("my-specific-stack").Build()

	client := testutil.NewMockCFNClient().WithStack(stack)

	filter := "my-specific-stack"
	result, err := GetCfnStacks(context.Background(), &filter, client)
	require.NoError(t, err)

	assert.Len(t, result, 1)
	assert.Contains(t, result, "my-specific-stack")
}
