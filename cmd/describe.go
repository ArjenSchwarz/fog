/*
Copyright © 2023 Arjen Schwarz <developer@arjen.eu>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"github.com/spf13/cobra"
)

var describeFlags DescribeFlags

// describeCmd represents the describe command. It is a group command like
// `stack` and `resource`: without a subcommand it shows its help (listing the
// subcommands), and an unknown subcommand returns an error.
var describeCmd = &cobra.Command{
	Use:   "describe",
	Args:  rejectUnknownSubcommand,
	RunE:  showGroupHelp,
	Short: "Describe lets you describe/see various items",
	Long: `Describe lets you describe/see various items, such as stacks, changesets, templates, etc.

For details see the subcommands.`,
}

func init() {
	stackCmd.AddCommand(describeCmd)
	describeFlags.RegisterFlags(describeCmd)
}
