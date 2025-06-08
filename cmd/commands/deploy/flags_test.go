package deploy

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestFlagsValidate(t *testing.T) {
	viper.Set("templates.extensions", []string{".yaml"})
	viper.Set("deployments.extensions", []string{".yaml"})
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
			err := tc.flags.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
