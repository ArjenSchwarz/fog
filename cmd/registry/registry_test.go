package registry

import (
	"testing"

	"github.com/spf13/cobra"
)

type stubBuilder struct {
	cmd *cobra.Command
}

func (s stubBuilder) BuildCommand() *cobra.Command { return s.cmd }
func (s stubBuilder) GetHandler() CommandHandler   { return nil }

type nilBuilder struct{}

func (nilBuilder) BuildCommand() *cobra.Command { return nil }
func (nilBuilder) GetHandler() CommandHandler   { return nil }

func TestRegistryRegisterAndBuild(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	registry := NewCommandRegistry(root)

	if err := registry.Register("test", stubBuilder{cmd: &cobra.Command{Use: "test"}}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	if err := registry.Register("test", stubBuilder{cmd: &cobra.Command{Use: "test"}}); err == nil {
		t.Errorf("expected error on duplicate registration")
	}

	if err := registry.BuildAll(); err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	if len(root.Commands()) != 1 {
		t.Fatalf("expected command added to root")
	}
	if root.Commands()[0].Use != "test" {
		t.Errorf("command use mismatch")
	}
}

func TestRegistryBuildAllNilCommand(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	registry := NewCommandRegistry(root)
	if err := registry.Register("bad", nilBuilder{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := registry.BuildAll(); err == nil {
		t.Errorf("expected error when builder returns nil")
	}
}

// TestRegistryBuildAllMultiple ensures that multiple registered commands are
// added to the root command when BuildAll is called.
func TestRegistryBuildAllMultiple(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	registry := NewCommandRegistry(root)

	err := registry.Register("one", stubBuilder{cmd: &cobra.Command{Use: "one"}})
	if err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	err = registry.Register("two", stubBuilder{cmd: &cobra.Command{Use: "two"}})
	if err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	if err := registry.BuildAll(); err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	if len(root.Commands()) != 2 {
		t.Fatalf("expected two commands to be registered")
	}
	uses := []string{root.Commands()[0].Use, root.Commands()[1].Use}
	condA := uses[0] == "one" && uses[1] == "two"
	condB := uses[0] == "two" && uses[1] == "one"
	if !condA && !condB {
		t.Errorf("commands not registered correctly: %v", uses)
	}
}
