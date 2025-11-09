package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_GetString_EdgeCases tests edge cases for GetString
func TestConfig_GetString_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  string
	}{
		"very long string value": {
			setup: func() {
				viper.Reset()
				longString := ""
				for i := 0; i < 10000; i++ {
					longString += "a"
				}
				viper.Set("long", longString)
			},
			key: "long",
			want: func() string {
				result := ""
				for i := 0; i < 10000; i++ {
					result += "a"
				}
				return result
			}(),
		},
		"string with special characters": {
			setup: func() {
				viper.Reset()
				viper.Set("special", "!@#$%^&*()_+-=[]{}|;:',.<>?/~`")
			},
			key:  "special",
			want: "!@#$%^&*()_+-=[]{}|;:',.<>?/~`",
		},
		"string with unicode": {
			setup: func() {
				viper.Reset()
				viper.Set("unicode", "Hello 世界 🌍")
			},
			key:  "unicode",
			want: "Hello 世界 🌍",
		},
		"string with newlines": {
			setup: func() {
				viper.Reset()
				viper.Set("multiline", "line1\nline2\nline3")
			},
			key:  "multiline",
			want: "line1\nline2\nline3",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
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

// TestConfig_GetInt_EdgeCases tests edge cases for GetInt
func TestConfig_GetInt_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  int
	}{
		"maximum int value": {
			setup: func() {
				viper.Reset()
				viper.Set("maxint", int(^uint(0)>>1))
			},
			key:  "maxint",
			want: int(^uint(0) >> 1),
		},
		"minimum int value": {
			setup: func() {
				viper.Reset()
				viper.Set("minint", -int(^uint(0)>>1)-1)
			},
			key:  "minint",
			want: -int(^uint(0)>>1) - 1,
		},
		"string representation of number": {
			setup: func() {
				viper.Reset()
				viper.Set("stringnum", "123")
			},
			key:  "stringnum",
			want: 0, // Viper returns 0 for non-int types
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
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

// TestConfig_GetStringSlice_EdgeCases tests edge cases for GetStringSlice
func TestConfig_GetStringSlice_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  []string
	}{
		"slice with empty strings": {
			setup: func() {
				viper.Reset()
				viper.Set("withempty", []string{"", "value", "", "another"})
			},
			key:  "withempty",
			want: []string{"", "value", "", "another"},
		},
		"slice with one element": {
			setup: func() {
				viper.Reset()
				viper.Set("single", []string{"only"})
			},
			key:  "single",
			want: []string{"only"},
		},
		"very large slice": {
			setup: func() {
				viper.Reset()
				large := make([]string, 1000)
				for i := 0; i < 1000; i++ {
					large[i] = "item"
				}
				viper.Set("large", large)
			},
			key: "large",
			want: func() []string {
				result := make([]string, 1000)
				for i := 0; i < 1000; i++ {
					result[i] = "item"
				}
				return result
			}(),
		},
		"slice with special characters": {
			setup: func() {
				viper.Reset()
				viper.Set("special", []string{"a,b,c", "x|y|z", "m;n;o"})
			},
			key:  "special",
			want: []string{"a,b,c", "x|y|z", "m;n;o"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
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

// TestConfig_GetBool_EdgeCases tests edge cases for GetBool
func TestConfig_GetBool_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  bool
	}{
		"string true": {
			setup: func() {
				viper.Reset()
				viper.Set("strtrue", "true")
			},
			key:  "strtrue",
			want: false, // Viper GetBool expects actual boolean, not string
		},
		"numeric 1": {
			setup: func() {
				viper.Reset()
				viper.Set("num1", 1)
			},
			key:  "num1",
			want: false, // Viper GetBool expects actual boolean
		},
		"numeric 0": {
			setup: func() {
				viper.Reset()
				viper.Set("num0", 0)
			},
			key:  "num0",
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}
			got := config.GetBool(tc.key)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestConfig_GetTimezoneLocation_EdgeCases tests edge cases for GetTimezoneLocation
func TestConfig_GetTimezoneLocation_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup       func()
		want        string
		shouldPanic bool
	}{
		"empty timezone string": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", "")
			},
			shouldPanic: true, // Empty string is invalid timezone
		},
		"Local timezone": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", "Local")
			},
			want:        "Local",
			shouldPanic: false,
		},
		"timezone with spaces": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", " UTC ")
			},
			shouldPanic: true, // Spaces in timezone name are invalid
		},
		"case sensitive timezone": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", "utc") // Should be "UTC"
			},
			shouldPanic: true, // Timezone names are case-sensitive
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}

			if tc.shouldPanic {
				assert.Panics(t, func() {
					config.GetTimezoneLocation()
				})
				return
			}

			got := config.GetTimezoneLocation()
			require.NotNil(t, got)
			assert.Equal(t, tc.want, got.String())
		})
	}
}

// TestConfig_GetFieldOrEmptyValue_EdgeCases tests edge cases for GetFieldOrEmptyValue
func TestConfig_GetFieldOrEmptyValue_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
		value string
		want  string
	}{
		"whitespace only with table output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
			},
			value: "   ",
			want:  "   ", // Whitespace is not empty
		},
		"very long value with table output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
			},
			value: func() string {
				result := ""
				for i := 0; i < 1000; i++ {
					result += "x"
				}
				return result
			}(),
			want: func() string {
				result := ""
				for i := 0; i < 1000; i++ {
					result += "x"
				}
				return result
			}(),
		},
		"unicode with json output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "json")
			},
			value: "世界",
			want:  "世界",
		},
		"empty with markdown output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "markdown")
			},
			value: "",
			want:  "-",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
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

// TestConfig_GetTableFormat_EdgeCases tests edge cases for GetTableFormat
func TestConfig_GetTableFormat_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
		check func(*testing.T, any)
	}{
		"negative column width": {
			setup: func() {
				viper.Reset()
				viper.Set("table.max-column-width", -10)
			},
			check: func(t *testing.T, fmt any) {
				t.Helper()
				assert.NotNil(t, fmt)
			},
		},
		"extremely large column width": {
			setup: func() {
				viper.Reset()
				viper.Set("table.max-column-width", 999999)
			},
			check: func(t *testing.T, fmt any) {
				t.Helper()
				assert.NotNil(t, fmt)
			},
		},
		"empty style string": {
			setup: func() {
				viper.Reset()
				viper.Set("table.style", "")
				viper.SetDefault("table.max-column-width", 50)
			},
			check: func(t *testing.T, fmt any) {
				t.Helper()
				assert.NotNil(t, fmt)
			},
		},
		"style with special characters": {
			setup: func() {
				viper.Reset()
				viper.Set("table.style", "Default@#$")
				viper.SetDefault("table.max-column-width", 50)
			},
			check: func(t *testing.T, fmt any) {
				t.Helper()
				assert.NotNil(t, fmt)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}
			got := config.GetTableFormat()

			require.NotNil(t, got)
			tc.check(t, got)
		})
	}
}

// TestConfig_Concurrency tests concurrent access to Config methods
func TestConfig_Concurrency(t *testing.T) {
	t.Parallel()

	const numGoroutines = 50
	done := make(chan bool, numGoroutines)

	// Note: This test documents that Config (via Viper) is not thread-safe
	// In production, each goroutine should have its own config or use proper synchronization

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			config := &Config{}

			// Perform various config operations
			_ = config.GetString("test")
			_ = config.GetInt("number")
			_ = config.GetBool("flag")
			_ = config.GetStringSlice("list")
			_ = config.GetLCString("profile")
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestConfig_GetOutputOptions_EdgeCases tests edge cases for GetOutputOptions
func TestConfig_GetOutputOptions_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup func()
		check func(*testing.T, any)
	}{
		"invalid output format": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "invalid")
			},
			check: func(t *testing.T, opts any) {
				t.Helper()
				assert.NotNil(t, opts)
			},
		},
		"empty output format": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "")
			},
			check: func(t *testing.T, opts any) {
				t.Helper()
				assert.NotNil(t, opts)
			},
		},
		"output file with empty path": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "json")
				viper.Set("output-file", "")
			},
			check: func(t *testing.T, opts any) {
				t.Helper()
				assert.NotNil(t, opts)
			},
		},
		"output file with invalid path characters": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "json")
				viper.Set("output-file", "/invalid\x00path")
			},
			check: func(t *testing.T, opts any) {
				t.Helper()
				assert.NotNil(t, opts)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}
			got := config.GetOutputOptions()

			require.NotNil(t, got)
			tc.check(t, got)
		})
	}
}
