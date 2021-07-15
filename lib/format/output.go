package format

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"sort"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/viper"

	"github.com/ArjenSchwarz/fog/config"
)

// OutputHolder holds key-value pairs that belong together in the output
type OutputHolder struct {
	Contents map[string]string
}

// OutputArray holds all the different OutputHolders that will be provided as
// output, as well as the keys (headers) that will actually need to be printed
type OutputArray struct {
	Title    string
	SortKey  string
	Contents []OutputHolder
	Keys     []string
}

// GetContentsMap returns a stringmap of the output contents
func (output OutputArray) GetContentsMap(settings config.Config) []map[string]string {
	total := make([]map[string]string, 0, len(output.Contents))
	for _, holder := range output.Contents {
		values := make(map[string]string)
		for _, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[key] = settings.GetFieldOrEmptyValue(val)
			}
		}
		total = append(total, values)
	}
	return total
}

// Write will provide the output as configured in the configuration
func (output OutputArray) Write(settings config.Config) {
	switch settings.GetLCString("output") {
	case "csv":
		output.toCSV(settings)
	case "table":
		output.toTable(settings)
	case "json":
		output.toJSON(settings)
	// case "html":
	// 	output.toHTML()
	default: //If an unknown value is provided, use tables
		output.toTable(settings)
	}
}

func (output OutputArray) toCSV(settings config.Config) {
	t := output.buildTable(settings)
	t.RenderCSV()
}

func (output OutputArray) toJSON(settings config.Config) {
	jsonString, _ := json.Marshal(output.GetContentsMap(settings))

	err := PrintByteSlice(jsonString, "")
	if err != nil {
		log.Fatal(err.Error())
	}
}

var TableStyles = map[string]table.Style{
	"Default":                    table.StyleDefault,
	"Bold":                       table.StyleBold,
	"ColoredBright":              table.StyleColoredBright,
	"ColoredDark":                table.StyleColoredDark,
	"ColoredBlackOnBlueWhite":    table.StyleColoredBlackOnBlueWhite,
	"ColoredBlackOnCyanWhite":    table.StyleColoredBlackOnCyanWhite,
	"ColoredBlackOnGreenWhite":   table.StyleColoredBlackOnGreenWhite,
	"ColoredBlackOnMagentaWhite": table.StyleColoredBlackOnMagentaWhite,
	"ColoredBlackOnYellowWhite":  table.StyleColoredBlackOnYellowWhite,
	"ColoredBlackOnRedWhite":     table.StyleColoredBlackOnRedWhite,
	"ColoredBlueWhiteOnBlack":    table.StyleColoredBlueWhiteOnBlack,
	"ColoredCyanWhiteOnBlack":    table.StyleColoredCyanWhiteOnBlack,
	"ColoredGreenWhiteOnBlack":   table.StyleColoredGreenWhiteOnBlack,
	"ColoredMagentaWhiteOnBlack": table.StyleColoredMagentaWhiteOnBlack,
	"ColoredRedWhiteOnBlack":     table.StyleColoredRedWhiteOnBlack,
	"ColoredYellowWhiteOnBlack":  table.StyleColoredYellowWhiteOnBlack,
}

func (output OutputArray) toTable(settings config.Config) {
	t := output.buildTable(settings)
	t.SetStyle(TableStyles[viper.GetString("table.style")])
	t.Render()
}

// TODO: Proper HTML output similar to awstools but using table library
// func (output OutputArray) toHTML() {
// 	t := output.buildTable()
// 	t.Style().HTML = table.HTMLOptions{
// 		CSSClass:    "mytable",
// 		EmptyColumn: "&nbsp;",
// 		EscapeText:  true,
// 		Newline:     "<br/>",
// 	}

// 	t.RenderHTML()
// }

func (output OutputArray) buildTable(settings config.Config) table.Writer {
	t := table.NewWriter()
	if output.Title != "" {
		t.SetTitle(output.Title)
	}
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(output.KeysAsInterface())
	for _, cont := range output.ContentsAsInterfaces(settings) {
		t.AppendRow(cont)
	}
	columnConfigs := make([]table.ColumnConfig, 0)
	for _, key := range output.Keys {
		columnConfig := table.ColumnConfig{
			Name:     key,
			WidthMin: 6,
			WidthMax: viper.GetInt("table.max-column-width"),
		}
		columnConfigs = append(columnConfigs, columnConfig)
	}
	t.SetColumnConfigs(columnConfigs)
	return t
}

// PrintByteSlice prints the provided contents to stdout or the provided filepath
func PrintByteSlice(contents []byte, outputFile string) error {
	var target io.Writer
	var err error
	if outputFile == "" {
		target = os.Stdout
	} else {
		target, err = os.Create(outputFile)
		if err != nil {
			return err
		}
	}
	w := bufio.NewWriter(target)
	w.Write(contents)
	err = w.Flush()
	return err
}

// AddHolder adds the provided OutputHolder to the OutputArray
func (output *OutputArray) AddHolder(holder OutputHolder) {
	var contents []OutputHolder
	if output.Contents != nil {
		contents = output.Contents
	}
	contents = append(contents, holder)
	if output.SortKey != "" {
		sort.Slice(contents,
			func(i, j int) bool {
				return contents[i].Contents[output.SortKey] < contents[j].Contents[output.SortKey]
			})
	}
	output.Contents = contents
}

func (output *OutputArray) KeysAsInterface() []interface{} {
	b := make([]interface{}, len(output.Keys))
	for i := range output.Keys {
		b[i] = output.Keys[i]
	}

	return b
}

func (output *OutputArray) ContentsAsInterfaces(settings config.Config) [][]interface{} {
	total := make([][]interface{}, 0)

	for _, holder := range output.Contents {
		values := make([]interface{}, len(output.Keys))
		for counter, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[counter] = settings.GetFieldOrEmptyValue(val)
			}
		}
		total = append(total, values)
	}
	return total
}
