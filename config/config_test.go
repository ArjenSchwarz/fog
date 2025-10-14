package config

import (
	"testing"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_GetLCString(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  string
	}{
		"value exists": {
			setup: func() {
				viper.Reset()
				viper.Set("profile", "MyProfile")
			},
			key:  "profile",
			want: "myprofile",
		},
		"value does not exist": {
			setup: func() {
				viper.Reset()
			},
			key:  "nonexistent",
			want: "",
		},
		"empty string value": {
			setup: func() {
				viper.Reset()
				viper.Set("empty", "")
			},
			key:  "empty",
			want: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}
			got := config.GetLCString(tc.key)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConfig_GetString(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  string
	}{
		"value exists": {
			setup: func() {
				viper.Reset()
				viper.Set("region", "us-west-2")
			},
			key:  "region",
			want: "us-west-2",
		},
		"value does not exist": {
			setup: func() {
				viper.Reset()
			},
			key:  "nonexistent",
			want: "",
		},
		"preserves case": {
			setup: func() {
				viper.Reset()
				viper.Set("name", "MyName")
			},
			key:  "name",
			want: "MyName",
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

func TestConfig_GetStringSlice(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  []string
	}{
		"value exists": {
			setup: func() {
				viper.Reset()
				viper.Set("tags", []string{"tag1", "tag2", "tag3"})
			},
			key:  "tags",
			want: []string{"tag1", "tag2", "tag3"},
		},
		"value does not exist": {
			setup: func() {
				viper.Reset()
			},
			key:  "nonexistent",
			want: []string{},
		},
		"empty slice": {
			setup: func() {
				viper.Reset()
				viper.Set("empty", []string{})
			},
			key:  "empty",
			want: []string{},
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

func TestConfig_GetBool(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  bool
	}{
		"true value": {
			setup: func() {
				viper.Reset()
				viper.Set("enabled", true)
			},
			key:  "enabled",
			want: true,
		},
		"false value": {
			setup: func() {
				viper.Reset()
				viper.Set("disabled", false)
			},
			key:  "disabled",
			want: false,
		},
		"value does not exist": {
			setup: func() {
				viper.Reset()
			},
			key:  "nonexistent",
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

func TestConfig_GetInt(t *testing.T) {
	tests := map[string]struct {
		setup func()
		key   string
		want  int
	}{
		"positive value": {
			setup: func() {
				viper.Reset()
				viper.Set("timeout", 30)
			},
			key:  "timeout",
			want: 30,
		},
		"zero value": {
			setup: func() {
				viper.Reset()
				viper.Set("zero", 0)
			},
			key:  "zero",
			want: 0,
		},
		"value does not exist": {
			setup: func() {
				viper.Reset()
			},
			key:  "nonexistent",
			want: 0,
		},
		"negative value": {
			setup: func() {
				viper.Reset()
				viper.Set("negative", -5)
			},
			key:  "negative",
			want: -5,
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

func TestConfig_GetSeparator(t *testing.T) {
	tests := map[string]struct {
		setup func()
		want  string
	}{
		"table output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
			},
			want: "\r\n",
		},
		"dot output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "dot")
			},
			want: ",",
		},
		"json output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "json")
			},
			want: ", ",
		},
		"csv output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "csv")
			},
			want: ", ",
		},
		"no output set": {
			setup: func() {
				viper.Reset()
			},
			want: ", ",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setup()
			t.Cleanup(func() {
				viper.Reset()
			})

			config := &Config{}
			got := config.GetSeparator()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConfig_GetFieldOrEmptyValue(t *testing.T) {
	tests := map[string]struct {
		setup func()
		value string
		want  string
	}{
		"non-empty value with table output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
			},
			value: "myvalue",
			want:  "myvalue",
		},
		"empty value with table output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
			},
			value: "",
			want:  "-",
		},
		"non-empty value with json output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "json")
			},
			value: "myvalue",
			want:  "myvalue",
		},
		"empty value with json output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "json")
			},
			value: "",
			want:  "",
		},
		"empty value with csv output": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "csv")
			},
			value: "",
			want:  "-",
		},
		"empty value with no output set": {
			setup: func() {
				viper.Reset()
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

func TestConfig_GetTimezoneLocation(t *testing.T) {
	tests := map[string]struct {
		setup       func()
		want        string
		shouldPanic bool
	}{
		"valid timezone UTC": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", "UTC")
			},
			want:        "UTC",
			shouldPanic: false,
		},
		"valid timezone America/New_York": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", "America/New_York")
			},
			want:        "America/New_York",
			shouldPanic: false,
		},
		"valid timezone Europe/London": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", "Europe/London")
			},
			want:        "Europe/London",
			shouldPanic: false,
		},
		"invalid timezone": {
			setup: func() {
				viper.Reset()
				viper.Set("timezone", "Invalid/Timezone")
			},
			shouldPanic: true,
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

func TestConfig_NewOutputSettings(t *testing.T) {
	tests := map[string]struct {
		setup func()
		check func(*testing.T, *format.OutputSettings)
	}{
		"default settings": {
			setup: func() {
				viper.Reset()
			},
			check: func(t *testing.T, settings *format.OutputSettings) {
				t.Helper()
				assert.True(t, settings.UseEmoji)
				assert.True(t, settings.UseColors)
			},
		},
		"json output format": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "json")
			},
			check: func(t *testing.T, settings *format.OutputSettings) {
				t.Helper()
				assert.True(t, settings.UseEmoji)
				assert.True(t, settings.UseColors)
				assert.Equal(t, "json", settings.OutputFormat)
			},
		},
		"table output format": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "table")
			},
			check: func(t *testing.T, settings *format.OutputSettings) {
				t.Helper()
				assert.Equal(t, "table", settings.OutputFormat)
			},
		},
		"with output file": {
			setup: func() {
				viper.Reset()
				viper.Set("output-file", "output.txt")
			},
			check: func(t *testing.T, settings *format.OutputSettings) {
				t.Helper()
				assert.Equal(t, "output.txt", settings.OutputFile)
			},
		},
		"with output file format": {
			setup: func() {
				viper.Reset()
				viper.Set("output-file-format", "csv")
			},
			check: func(t *testing.T, settings *format.OutputSettings) {
				t.Helper()
				assert.Equal(t, "csv", settings.OutputFileFormat)
			},
		},
		"with table max column width": {
			setup: func() {
				viper.Reset()
				viper.Set("table.max-column-width", 50)
			},
			check: func(t *testing.T, settings *format.OutputSettings) {
				t.Helper()
				assert.Equal(t, 50, settings.TableMaxColumnWidth)
			},
		},
		"with table style": {
			setup: func() {
				viper.Reset()
				viper.Set("table.style", "simple")
			},
			check: func(t *testing.T, settings *format.OutputSettings) {
				t.Helper()
				// The table style should be set from TableStyles map
				assert.NotNil(t, settings.TableStyle)
			},
		},
		"with multiple settings": {
			setup: func() {
				viper.Reset()
				viper.Set("output", "csv")
				viper.Set("output-file", "results.csv")
				viper.Set("output-file-format", "csv")
				viper.Set("table.max-column-width", 100)
			},
			check: func(t *testing.T, settings *format.OutputSettings) {
				t.Helper()
				assert.Equal(t, "csv", settings.OutputFormat)
				assert.Equal(t, "results.csv", settings.OutputFile)
				assert.Equal(t, "csv", settings.OutputFileFormat)
				assert.Equal(t, 100, settings.TableMaxColumnWidth)
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
			got := config.NewOutputSettings()

			require.NotNil(t, got)
			tc.check(t, got)
		})
	}
}
