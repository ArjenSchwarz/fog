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
