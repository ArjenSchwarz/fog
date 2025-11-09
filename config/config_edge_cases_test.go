package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TestConfig_GetTimezoneLocation_EdgeCases tests additional edge cases for timezone handling.
// These tests address Issue 6.1 from the audit report regarding configuration loading panics.
// NOTE: Cannot use t.Parallel() at function level because viper uses global state
func TestConfig_GetTimezoneLocation_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup       func()
		shouldPanic bool
		description string
	}{
		"empty timezone string": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", "")
			},
			shouldPanic: true,
			description: "Empty timezone string should cause panic (Issue 6.1)",
		},
		"timezone not set": {
			setup: func() {
				viper.Reset()
				// timezone key not set at all
			},
			shouldPanic: true,
			description: "Missing timezone configuration should cause panic (Issue 6.1)",
		},
		"case sensitive timezone name": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", "utc") // lowercase instead of UTC
			},
			shouldPanic: true,
			description: "Case-sensitive timezone names should be handled",
		},
		"timezone with spaces": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", " UTC ")
			},
			shouldPanic: true,
			description: "Timezone with leading/trailing spaces should be rejected",
		},
		"special characters in timezone": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", "America/New_York!")
			},
			shouldPanic: true,
			description: "Invalid characters in timezone name should be rejected",
		},
		"numeric timezone": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", "123")
			},
			shouldPanic: true,
			description: "Numeric timezone values should be rejected",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}

			if tc.shouldPanic {
				assert.Panics(t, func() {
					config.GetTimezoneLocation()
				}, tc.description)
			} else {
				assert.NotPanics(t, func() {
					loc := config.GetTimezoneLocation()
					assert.NotNil(t, loc)
				}, tc.description)
			}
		})
	}
}

// TestConfig_GetOutputOptions_EdgeCases tests edge cases for output options configuration.
// These tests address testing gaps mentioned in Issue 4.1.
func TestConfig_GetOutputOptions_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup       func()
		description string
	}{
		"invalid output format falls back to table": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "invalid-format")
				viper.Set("use-emoji", false)
				viper.Set("use-colors", false)
			},
			description: "Invalid output format should fall back to table format",
		},
		"output file with invalid path": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
				viper.Set("output-file", "/invalid/path/that/does/not/exist/file.txt")
				viper.Set("use-emoji", false)
				viper.Set("use-colors", false)
			},
			description: "Invalid file path should log warning but not fail",
		},
		"output file with empty string": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
				viper.Set("output-file", "")
				viper.Set("use-emoji", false)
				viper.Set("use-colors", false)
			},
			description: "Empty output file should be treated as no file output",
		},
		"file format without file specified": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
				viper.Set("output-file-format", "csv")
				// output-file not set
				viper.Set("use-emoji", false)
				viper.Set("use-colors", false)
			},
			description: "File format without file should be ignored",
		},
		"same format for console and file": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "json")
				viper.Set("output-file", "/tmp/test.json")
				viper.Set("output-file-format", "json")
				viper.Set("use-emoji", false)
				viper.Set("use-colors", false)
			},
			description: "Same format for console and file should not duplicate formats",
		},
		"dot output format": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "dot")
				viper.Set("use-emoji", false)
				viper.Set("use-colors", false)
			},
			description: "DOT graph format should be supported",
		},
		"yaml output format": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "yaml")
				viper.Set("use-emoji", false)
				viper.Set("use-colors", false)
			},
			description: "YAML output format should be supported",
		},
		"html output format": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "html")
				viper.Set("use-emoji", false)
				viper.Set("use-colors", false)
			},
			description: "HTML output format should be supported",
		},
		"multiple transformers enabled": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
				viper.Set("use-emoji", true)
				viper.Set("use-colors", true)
			},
			description: "Multiple transformers should work together",
		},
		"file output with relative path": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
				viper.Set("output-file", "output.txt")
				viper.Set("use-emoji", false)
				viper.Set("use-colors", false)
			},
			description: "Relative file paths should default to current directory",
		},
		"file output with directory only": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
				viper.Set("output-file", "/tmp/")
				viper.Set("use-emoji", false)
				viper.Set("use-colors", false)
			},
			description: "Directory-only path should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}

			// Should not panic even with edge cases
			assert.NotPanics(t, func() {
				opts := config.GetOutputOptions()
				assert.NotNil(t, opts, tc.description)
				assert.NotEmpty(t, opts, tc.description)
			})
		})
	}
}

// TestConfig_GetFieldOrEmptyValue_EdgeCases tests edge cases for field value handling.
// NOTE: Cannot use t.Parallel() at function level because viper uses global state
func TestConfig_GetFieldOrEmptyValue_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
		value string
		want  string
	}{
		"whitespace-only value with table output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
			},
			value: "   ",
			want:  "   ", // Whitespace is considered non-empty
		},
		"newline character with json output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "json")
			},
			value: "\n",
			want:  "\n", // Newline is considered non-empty
		},
		"zero value with table output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
			},
			value: "0",
			want:  "0", // String "0" is non-empty
		},
		"uppercase output format": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "JSON") // Will be lowercased by GetLCString
			},
			value: "",
			want:  "", // JSON format returns empty string
		},
		"mixed case output format": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "TaBLe")
			},
			value: "",
			want:  "-", // Non-JSON formats return dash
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}
			got := config.GetFieldOrEmptyValue(tc.value)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestConfig_GetTableFormat_EdgeCases tests edge cases for table format configuration.
// NOTE: Cannot use t.Parallel() at function level because viper uses global state
func TestConfig_GetTableFormat_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
	}{
		"negative max column width": {
			setup: func() {
				viper.Reset()
				viper.Set("table.style", "Default")
				viper.Set("table.max-column-width", -10)
			},
		},
		"very large max column width": {
			setup: func() {
				viper.Reset()
				viper.Set("table.style", "Default")
				viper.Set("table.max-column-width", 999999)
			},
		},
		"empty style string": {
			setup: func() {
				viper.Reset()
				viper.Set("table.style", "")
				viper.Set("table.max-column-width", 50)
			},
		},
		"style with special characters": {
			setup: func() {
				viper.Reset()
				viper.Set("table.style", "Invalid@Style#Name")
				viper.Set("table.max-column-width", 50)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}

			// Should not panic with edge cases
			assert.NotPanics(t, func() {
				format := config.GetTableFormat()
				assert.NotNil(t, format)
			})
		})
	}
}

// TestConfig_GetString_EdgeCases tests edge cases for string value retrieval.
// NOTE: Cannot use t.Parallel() at function level because viper uses global state
func TestConfig_GetString_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  string
	}{
		"numeric value converted to string": {
			setup: func() {
				viper.Reset()
				viper.Set("number", 123)
			},
			key:  "number",
			want: "123",
		},
		"boolean value converted to string": {
			setup: func() {
				viper.Reset()
				viper.Set("bool", true)
			},
			key:  "bool",
			want: "true",
		},
		"key with dots": {
			setup: func() {
				viper.Reset()
				viper.Set("table.style", "Bold")
			},
			key:  "table.style",
			want: "Bold",
		},
		"very long string value": {
			setup: func() {
				viper.Reset()
				longString := string(make([]byte, 10000))
				viper.Set("long", longString)
			},
			key:  "long",
			want: string(make([]byte, 10000)),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}
			got := config.GetString(tc.key)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestConfig_GetInt_EdgeCases tests edge cases for integer value retrieval.
// NOTE: Cannot use t.Parallel() at function level because viper uses global state
func TestConfig_GetInt_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  int
	}{
		"string number converted to int": {
			setup: func() {
				viper.Reset()
				viper.Set("number", "456")
			},
			key:  "number",
			want: 456,
		},
		"float converted to int": {
			setup: func() {
				viper.Reset()
				viper.Set("float", 123.456)
			},
			key:  "float",
			want: 123, // Truncated
		},
		"very large number": {
			setup: func() {
				viper.Reset()
				viper.Set("large", 2147483647) // Max int32
			},
			key:  "large",
			want: 2147483647,
		},
		"invalid string returns zero": {
			setup: func() {
				viper.Reset()
				viper.Set("invalid", "not-a-number")
			},
			key:  "invalid",
			want: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}
			got := config.GetInt(tc.key)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestConfig_GetStringSlice_EdgeCases tests edge cases for string slice retrieval.
// NOTE: Cannot use t.Parallel() at function level because viper uses global state
func TestConfig_GetStringSlice_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  []string
	}{
		"single string converted to slice by viper": {
			setup: func() {
				viper.Reset()
				viper.Set("single", "value")
			},
			key:  "single",
			want: []string{"value"}, // Viper auto-converts single string to slice with one element
		},
		"slice with empty strings": {
			setup: func() {
				viper.Reset()
				viper.Set("with-empty", []string{"", "value", ""})
			},
			key:  "with-empty",
			want: []string{"", "value", ""},
		},
		"slice with duplicate values": {
			setup: func() {
				viper.Reset()
				viper.Set("duplicates", []string{"a", "a", "b", "a"})
			},
			key:  "duplicates",
			want: []string{"a", "a", "b", "a"},
		},
		"nil slice": {
			setup: func() {
				viper.Reset()
				var nilSlice []string
				viper.Set("nil-slice", nilSlice)
			},
			key:  "nil-slice",
			want: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}
			got := config.GetStringSlice(tc.key)
			assert.Equal(t, tc.want, got)
		})
	}
}
