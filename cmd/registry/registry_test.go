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
	r := NewCommandRegistry(root)

	if err := r.Register("test", stubBuilder{cmd: &cobra.Command{Use: "test"}}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	if err := r.Register("test", stubBuilder{cmd: &cobra.Command{Use: "test"}}); err == nil {
		t.Errorf("expected error on duplicate registration")
	}

	if err := r.BuildAll(); err != nil {
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
	r := NewCommandRegistry(root)
	if err := r.Register("bad", nilBuilder{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := r.BuildAll(); err == nil {
		t.Errorf("expected error when builder returns nil")
	}
}
