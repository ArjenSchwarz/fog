package ui

import (
	"io"
	"os"
)

// ConsoleUI is a minimal implementation of OutputHandler that writes to stdout.
type ConsoleUI struct {
	verbose bool
}

// NewConsoleUI creates a new ConsoleUI.
func NewConsoleUI(verbose bool) *ConsoleUI {
	return &ConsoleUI{verbose: verbose}
}

func (c *ConsoleUI) Success(string) {}
func (c *ConsoleUI) Info(string)    {}
func (c *ConsoleUI) Warning(string) {}
func (c *ConsoleUI) Error(string)   {}
func (c *ConsoleUI) Debug(msg string) {
	if c.verbose {
		_ = msg
	}
}
func (c *ConsoleUI) Table(interface{}, TableOptions) error  { return nil }
func (c *ConsoleUI) JSON(interface{}) error                 { return nil }
func (c *ConsoleUI) StartProgress(string) ProgressIndicator { return nil }
func (c *ConsoleUI) SetStatus(string)                       {}
func (c *ConsoleUI) Confirm(string) bool                    { return false }
func (c *ConsoleUI) ConfirmWithDefault(string, bool) bool   { return false }
func (c *ConsoleUI) SetVerbose(v bool)                      { c.verbose = v }
func (c *ConsoleUI) SetQuiet(bool)                          {}
func (c *ConsoleUI) SetOutputFormat(OutputFormat)           {}
func (c *ConsoleUI) GetWriter() io.Writer                   { return os.Stdout }
func (c *ConsoleUI) GetErrorWriter() io.Writer              { return os.Stderr }
func (c *ConsoleUI) GetVerbose() bool                       { return c.verbose }
