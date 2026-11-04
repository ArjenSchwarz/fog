package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ArjenSchwarz/fog/lib"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/viper"
)

// These tests cover T-1702: comma-separated --tags and --parameters lists
// must tolerate whitespace around names (e.g. "common, app") and skip empty
// entries from trailing or doubled commas, rather than looking up files with
// names such as " app" or "".

func TestSplitFileList(t *testing.T) {
	tests := map[string]struct {
		input string
		want  []string
	}{
		"single name":           {input: "common", want: []string{"common"}},
		"no spaces":             {input: "common,app", want: []string{"common", "app"}},
		"space after comma":     {input: "common, app", want: []string{"common", "app"}},
		"surrounding spaces":    {input: " common ,  app ", want: []string{"common", "app"}},
		"trailing comma":        {input: "common,app,", want: []string{"common", "app"}},
		"doubled comma":         {input: "common,,app", want: []string{"common", "app"}},
		"only separators":       {input: " , ,", want: []string{}},
		"empty string":          {input: "", want: []string{}},
		"inner spaces retained": {input: "my tags, app", want: []string{"my tags", "app"}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := splitFileList(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitFileList(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// writeListTestFile writes content to dir/name, failing the test on error.
func writeListTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

func TestLoadTagsFromFiles_TrimsCommaSeparatedNames(t *testing.T) {
	t.Cleanup(viper.Reset)
	dir := t.TempDir()
	viper.Set("tags.directory", dir)
	viper.Set("tags.extensions", []string{".json"})
	writeListTestFile(t, dir, "common.json", `[{"Key":"Owner","Value":"platform"}]`)
	writeListTestFile(t, dir, "app.json", `[{"Key":"App","Value":"web"}]`)

	// A failed lookup would call failWithError; fail the test instead of exiting.
	origExit := osExitFunc
	osExitFunc = func(code int) { t.Fatalf("unexpected exit with code %d", code) }
	t.Cleanup(func() { osExitFunc = origExit })

	tags := loadTagsFromFiles("common, app,", &lib.DeployInfo{})

	got := make(map[string]string, len(tags))
	for _, tag := range tags {
		got[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	want := map[string]string{"Owner": "platform", "App": "web"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loadTagsFromFiles returned %v, want %v", got, want)
	}
}

func TestLoadParametersFromFiles_TrimsCommaSeparatedNames(t *testing.T) {
	t.Cleanup(viper.Reset)
	dir := t.TempDir()
	viper.Set("parameters.directory", dir)
	viper.Set("parameters.extensions", []string{".json"})
	writeListTestFile(t, dir, "base.json", `[{"ParameterKey":"VpcCidr","ParameterValue":"10.0.0.0/16"}]`)
	writeListTestFile(t, dir, "prod.json", `[{"ParameterKey":"Environment","ParameterValue":"prod"}]`)

	origExit := osExitFunc
	osExitFunc = func(code int) { t.Fatalf("unexpected exit with code %d", code) }
	t.Cleanup(func() { osExitFunc = origExit })

	params := loadParametersFromFiles("base, prod,")

	got := make(map[string]string, len(params))
	for _, p := range params {
		got[aws.ToString(p.ParameterKey)] = aws.ToString(p.ParameterValue)
	}
	want := map[string]string{"VpcCidr": "10.0.0.0/16", "Environment": "prod"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loadParametersFromFiles returned %v, want %v", got, want)
	}
}
