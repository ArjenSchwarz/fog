package cmd

import "testing"

// TestGetDefaultExtension verifies that getDefaultExtension returns the
// correct file extension for each supported output format. Regression test
// for T-850: previously "html" incorrectly returned ".md" because markdown
// and html shared a switch case, so S3 uploads of HTML reports used the
// wrong extension.
func TestGetDefaultExtension(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		format string
		want   string
	}{
		"markdown returns .md":        {format: "markdown", want: ".md"},
		"html returns .html":          {format: "html", want: ".html"},
		"json returns .json":          {format: "json", want: ".json"},
		"csv returns .csv":            {format: "csv", want: ".csv"},
		"yaml returns .yaml":          {format: "yaml", want: ".yaml"},
		"dot returns .dot":            {format: "dot", want: ".dot"},
		"table returns .txt":          {format: "table", want: ".txt"},
		"unknown format returns .txt": {format: "unknown", want: ".txt"},
		"empty format returns .txt":   {format: "", want: ".txt"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := getDefaultExtension(tc.format)
			if got != tc.want {
				t.Errorf("getDefaultExtension(%q) = %q, want %q", tc.format, got, tc.want)
			}
		})
	}
}
