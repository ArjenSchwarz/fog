package deploy

import (
	"context"
	"os"
	"testing"

	"github.com/ArjenSchwarz/fog/cmd/registry"
	"github.com/spf13/viper"
)

func TestFlagsValidate(t *testing.T) {
	originalTemplatesExtensions := viper.Get("templates.extensions")
	originalDeploymentsExtensions := viper.Get("deployments.extensions")
	viper.Set("templates.extensions", []string{".yaml"})
	viper.Set("deployments.extensions", []string{".yaml"})
	t.Cleanup(func() {
		viper.Set("templates.extensions", originalTemplatesExtensions)
		viper.Set("deployments.extensions", originalDeploymentsExtensions)
	})
	cases := []struct {
		name    string
		flags   *Flags
		wantErr bool
	}{
		{
			name:    "missing stack name",
			flags:   NewFlags(),
			wantErr: true,
		},
		{
			name: "deployment file conflict",
			flags: func() *Flags {
				f := NewFlags()
				f.StackName = "s"
				f.DeploymentFile = "f"
				f.Template = "t"
				return f
			}(),
			wantErr: true,
		},
		{
			name: "valid",
			flags: func() *Flags {
				f := NewFlags()
				f.StackName = "s"
				tmp := t.TempDir() + "/tmpl.yaml"
				_ = os.WriteFile(tmp, []byte("x"), 0o644)
				f.Template = tmp
				return f
			}(),
			wantErr: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			vCtx := &registry.ValidationContext{
				Command:    nil,
				Args:       []string{},
				AWSRegion:  "",
				ConfigPath: "",
				Verbose:    false,
			}
			err := tc.flags.Validate(ctx, vCtx)
			if tc.wantErr && err == nil {
				t.Errorf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
