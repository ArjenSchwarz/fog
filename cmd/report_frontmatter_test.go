package cmd

import (
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/spf13/viper"
)

// TestGenerateFrontMatter_DeterministicWithMultipleStacks verifies that
// generateFrontMatter produces deterministic output when multiple stacks
// exist in the map. Before the fix, Go map iteration order caused the
// frontmatter to reflect a random stack's event on each run.
func TestGenerateFrontMatter_DeterministicWithMultipleStacks(t *testing.T) {
	t.Parallel()
	viper.SetDefault("timezone", "UTC")

	oldSettings := settings
	settings = &config.Config{}
	t.Cleanup(func() { settings = oldSettings })

	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	awsConfig := config.AWSConfig{
		AccountID: "111111111111",
		Region:    "us-east-1",
	}

	// Create two stacks with distinct events so we can detect which was chosen
	stackA := lib.CfnStack{
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
	stackB := lib.CfnStack{
		Name: "beta-stack",
		Id:   "arn:aws:cloudformation:us-east-1:111111111111:stack/beta-stack/bbb",
		Events: []lib.StackEvent{
			{
				Type:      "Update",
				Success:   false,
				StartDate: now.Add(1 * time.Hour),
				EndDate:   now.Add(1*time.Hour + 45*time.Second),
			},
		},
	}

	stacks := map[string]lib.CfnStack{
		stackA.Id: stackA,
		stackB.Id: stackB,
	}

	// Run multiple times — result must be identical every time
	var firstResult map[string]string
	for i := 0; i < 20; i++ {
		result, err := generateFrontMatter(stacks, awsConfig)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if firstResult == nil {
			firstResult = result
			continue
		}
		for key, want := range firstResult {
			if got := result[key]; got != want {
				t.Fatalf("iteration %d: frontmatter key %q changed: got %q, want %q (non-deterministic)", i, key, got, want)
			}
		}
	}

	// The latest event (by StartDate) should be selected — that's beta-stack's Update
	if got := firstResult["stack"]; got != "beta-stack" {
		t.Errorf("expected frontmatter stack to be 'beta-stack' (latest event), got %q", got)
	}
	if got := firstResult["eventtype"]; got != "Update" {
		t.Errorf("expected frontmatter eventtype to be 'Update' (latest event), got %q", got)
	}
	if got := firstResult["success"]; got != "false" {
		t.Errorf("expected frontmatter success to be 'false' (latest event failed), got %q", got)
	}
}

// TestGenerateFrontMatter_SelectsLatestEventWithinStack verifies that when a
// single stack has multiple events, the frontmatter describes the latest event
// (newest StartDate) rather than an arbitrary one.
func TestGenerateFrontMatter_SelectsLatestEventWithinStack(t *testing.T) {
	t.Parallel()
	viper.SetDefault("timezone", "UTC")

	oldSettings := settings
	settings = &config.Config{}
	t.Cleanup(func() { settings = oldSettings })

	now := time.Date(2025, 3, 10, 8, 0, 0, 0, time.UTC)

	awsConfig := config.AWSConfig{
		AccountID: "222222222222",
		Region:    "eu-west-1",
	}

	stack := lib.CfnStack{
		Name: "my-stack",
		Id:   "arn:aws:cloudformation:eu-west-1:222222222222:stack/my-stack/xxx",
		Events: []lib.StackEvent{
			{
				Type:      "Create",
				Success:   true,
				StartDate: now,
				EndDate:   now.Add(20 * time.Second),
			},
			{
				Type:      "Update",
				Success:   true,
				StartDate: now.Add(2 * time.Hour),
				EndDate:   now.Add(2*time.Hour + 60*time.Second),
			},
			{
				Type:      "Delete",
				Success:   false,
				StartDate: now.Add(5 * time.Hour),
				EndDate:   now.Add(5*time.Hour + 10*time.Second),
			},
		},
	}

	stacks := map[string]lib.CfnStack{
		stack.Id: stack,
	}

	result, err := generateFrontMatter(stacks, awsConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The latest event is the Delete at +5h
	if got := result["eventtype"]; got != "Delete" {
		t.Errorf("expected eventtype 'Delete' (latest event), got %q", got)
	}
	if got := result["success"]; got != "false" {
		t.Errorf("expected success 'false' (Delete failed), got %q", got)
	}
	if got := result["duration"]; got != "10s" {
		t.Errorf("expected duration '10s' (Delete event), got %q", got)
	}
}

// TestGenerateFrontMatter_RespectsLatestOnly verifies that when
// reportFlags.LatestOnly is true, frontmatter only considers the latest event
// per stack — matching the filtering done in the report body by
// generateStackReport. This ensures frontmatter and body describe the same event.
func TestGenerateFrontMatter_RespectsLatestOnly(t *testing.T) {
	t.Parallel()
	viper.SetDefault("timezone", "UTC")

	oldSettings := settings
	settings = &config.Config{}
	t.Cleanup(func() { settings = oldSettings })

	// Save and restore reportFlags
	oldFlags := reportFlags
	t.Cleanup(func() { reportFlags = oldFlags })
	reportFlags = ReportFlags{LatestOnly: true}

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	awsConfig := config.AWSConfig{
		AccountID: "333333333333",
		Region:    "ap-southeast-2",
	}

	// Stack with three events — oldest first (matching chronological order from GetEvents)
	stack := lib.CfnStack{
		Name: "latestonly-stack",
		Id:   "arn:aws:cloudformation:ap-southeast-2:333333333333:stack/latestonly-stack/yyy",
		Events: []lib.StackEvent{
			{
				Type:      "Create",
				Success:   true,
				StartDate: now,
				EndDate:   now.Add(30 * time.Second),
			},
			{
				Type:      "Update",
				Success:   true,
				StartDate: now.Add(1 * time.Hour),
				EndDate:   now.Add(1*time.Hour + 45*time.Second),
			},
			{
				Type:      "Delete",
				Success:   true,
				StartDate: now.Add(3 * time.Hour),
				EndDate:   now.Add(3*time.Hour + 15*time.Second),
			},
		},
	}

	stacks := map[string]lib.CfnStack{
		stack.Id: stack,
	}

	result, err := generateFrontMatter(stacks, awsConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// When LatestOnly is true, only the last event (Delete at +3h) should be used,
	// matching the report body which also shows only the last event.
	if got := result["eventtype"]; got != "Delete" {
		t.Errorf("expected eventtype 'Delete' (latest event with LatestOnly), got %q", got)
	}
	if got := result["duration"]; got != "15s" {
		t.Errorf("expected duration '15s', got %q", got)
	}
}

// TestGenerateFrontMatter_MultipleStacksMultipleEvents tests the combination
// of multiple stacks each with multiple events. Verifies deterministic
// selection of the newest event overall.
func TestGenerateFrontMatter_MultipleStacksMultipleEvents(t *testing.T) {
	t.Parallel()
	viper.SetDefault("timezone", "UTC")

	oldSettings := settings
	settings = &config.Config{}
	t.Cleanup(func() { settings = oldSettings })

	now := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)

	awsConfig := config.AWSConfig{
		AccountID:    "444444444444",
		AccountAlias: "prod-account",
		Region:       "us-west-2",
	}

	// Stack A's latest event is at +2h
	stackA := lib.CfnStack{
		Name: "stack-a",
		Id:   "arn:aws:cloudformation:us-west-2:444444444444:stack/stack-a/aaa",
		Events: []lib.StackEvent{
			{
				Type:      "Create",
				Success:   true,
				StartDate: now,
				EndDate:   now.Add(20 * time.Second),
			},
			{
				Type:      "Update",
				Success:   true,
				StartDate: now.Add(2 * time.Hour),
				EndDate:   now.Add(2*time.Hour + 30*time.Second),
			},
		},
	}

	// Stack B's latest event is at +4h — this should be the selected event
	stackB := lib.CfnStack{
		Name: "stack-b",
		Id:   "arn:aws:cloudformation:us-west-2:444444444444:stack/stack-b/bbb",
		Events: []lib.StackEvent{
			{
				Type:      "Create",
				Success:   true,
				StartDate: now.Add(1 * time.Hour),
				EndDate:   now.Add(1*time.Hour + 15*time.Second),
			},
			{
				Type:      "Delete",
				Success:   false,
				StartDate: now.Add(4 * time.Hour),
				EndDate:   now.Add(4*time.Hour + 5*time.Second),
			},
		},
	}

	stacks := map[string]lib.CfnStack{
		stackA.Id: stackA,
		stackB.Id: stackB,
	}

	result, err := generateFrontMatter(stacks, awsConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stack B's Delete at +4h is the newest event overall
	if got := result["stack"]; got != "stack-b" {
		t.Errorf("expected stack 'stack-b', got %q", got)
	}
	if got := result["eventtype"]; got != "Delete" {
		t.Errorf("expected eventtype 'Delete', got %q", got)
	}
	if got := result["success"]; got != "false" {
		t.Errorf("expected success 'false', got %q", got)
	}
	if got := result["account"]; got != "444444444444" {
		t.Errorf("expected account '444444444444', got %q", got)
	}
	if got := result["accountalias"]; got != "prod-account (444444444444)" {
		t.Errorf("expected accountalias 'prod-account (444444444444)', got %q", got)
	}
	if got := result["region"]; got != "us-west-2" {
		t.Errorf("expected region 'us-west-2', got %q", got)
	}
}

// TestGenerateFrontMatter_EmptyStacks verifies graceful handling of no stacks.
func TestGenerateFrontMatter_EmptyStacks(t *testing.T) {
	t.Parallel()
	viper.SetDefault("timezone", "UTC")

	oldSettings := settings
	settings = &config.Config{}
	t.Cleanup(func() { settings = oldSettings })

	awsConfig := config.AWSConfig{
		AccountID: "555555555555",
		Region:    "us-east-1",
	}

	stacks := map[string]lib.CfnStack{}

	result, err := generateFrontMatter(stacks, awsConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty frontmatter for empty stacks, got %v", result)
	}
}
