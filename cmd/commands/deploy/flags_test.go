package deploy

import "testing"

func TestFlagsValidate(t *testing.T) {
	cases := []struct {
		name    string
		flags   Flags
		wantErr bool
	}{
		{"missing stack name", Flags{}, true},
		{"deployment file conflict", Flags{StackName: "s", DeploymentFile: "f", Template: "t"}, true},
		{"valid", Flags{StackName: "s"}, false},
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
