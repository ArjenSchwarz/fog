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
func (output OutputArray) GetContentsMap() []map[string]string {
	total := make([]map[string]string, 0, len(output.Contents))
	for _, holder := range output.Contents {
		values := make(map[string]string)
		for _, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[key] = val
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
		output.toCSV()
	case "table":
		output.toTable()
	case "json":
		output.toJSON()
	// case "html":
	// 	output.toHTML()
	default: //If an unknown value is provided, use tables
		output.toTable()
	}
}

func (output OutputArray) toCSV() {
	t := output.buildTable()
	t.RenderCSV()
}

func (output OutputArray) toJSON() {
	jsonString, _ := json.Marshal(output.GetContentsMap())

	err := PrintByteSlice(jsonString, "")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (output OutputArray) toTable() {
	t := output.buildTable()
	switch viper.GetString("table.style") {
	// TODO: Create a command to show examples of all these styles
	case "Default":
		t.SetStyle(table.StyleDefault)
	case "Bold":
		t.SetStyle(table.StyleBold)
	case "ColoredBright":
		t.SetStyle(table.StyleColoredBright)
	case "ColoredDark":
		t.SetStyle(table.StyleColoredDark)
	case "ColoredBlackOnBlueWhite":
		t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	case "ColoredBlackOnCyanWhite":
		t.SetStyle(table.StyleColoredBlackOnCyanWhite)
	case "ColoredBlackOnGreenWhite":
		t.SetStyle(table.StyleColoredBlackOnGreenWhite)
	case "ColoredBlackOnMagentaWhite":
		t.SetStyle(table.StyleColoredBlackOnMagentaWhite)
	case "ColoredBlackOnYellowWhite":
		t.SetStyle(table.StyleColoredBlackOnYellowWhite)
	case "ColoredBlackOnRedWhite":
		t.SetStyle(table.StyleColoredBlackOnRedWhite)
	case "ColoredBlueWhiteOnBlack":
		t.SetStyle(table.StyleColoredBlueWhiteOnBlack)
	case "ColoredCyanWhiteOnBlack":
		t.SetStyle(table.StyleColoredCyanWhiteOnBlack)
	case "ColoredGreenWhiteOnBlack":
		t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
	case "ColoredMagentaWhiteOnBlack":
		t.SetStyle(table.StyleColoredMagentaWhiteOnBlack)
	case "ColoredRedWhiteOnBlack":
		t.SetStyle(table.StyleColoredRedWhiteOnBlack)
	case "ColoredYellowWhiteOnBlack":
		t.SetStyle(table.StyleColoredYellowWhiteOnBlack)
	}
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

func (output OutputArray) buildTable() table.Writer {
	t := table.NewWriter()
	if output.Title != "" {
		t.SetTitle(output.Title)
	}
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(output.KeysAsInterface())
	for _, cont := range output.ContentsAsInterfaces() {
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

func (output *OutputArray) ContentsAsInterfaces() [][]interface{} {
	total := make([][]interface{}, 0)

	for _, holder := range output.Contents {
		values := make([]interface{}, len(output.Keys))
		for counter, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[counter] = val
			}
		}
		total = append(total, values)
	}
	return total
}
