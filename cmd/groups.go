/*
Copyright © 2021 Arjen Schwarz <developer@arjen.eu>

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
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

// Create command group containers
var (
	// Stack management commands
	stackCmd = &cobra.Command{
		Use:   "stack",
		Short: "Stack management commands",
		Long:  `Commands for managing CloudFormation stacks, including deployment, updates, and drift detection.`,
	}

	// Resource management commands
	resourceGroupCmd = &cobra.Command{
		Use:   "resource",
		Short: "Resource management commands",
		Long:  `Commands for working with CloudFormation resources and exports.`,
	}

	// Utility commands
	utilGroupCmd = &cobra.Command{
		Use:   "util",
		Short: "Utility commands",
		Long:  `Utility commands for various helper functions.`,
	}

	// Names of commands already migrated to the new registry-based structure
	migratedStackCommands = []string{"deploy"}
)

// InitGroups initializes all command groups and adds them to the root command
func InitGroups() {
	// Add command groups to root command
	rootCmd.AddCommand(stackCmd)
	rootCmd.AddCommand(resourceGroupCmd)
	rootCmd.AddCommand(utilGroupCmd)

	// For migrated commands in the stack group, create aliases so old paths
	// continue to work while the new command registry attaches them at the
	// root level.
	for _, name := range migratedStackCommands {
		short := fmt.Sprintf("Alias for '%s'", name)
		stackCmd.AddCommand(NewCommandAlias(name, name, short))
	}
}

// NewCommandAlias creates a command that redirects to another command path
func NewCommandAlias(name, target, short string) *cobra.Command {
	// Create the alias command
	aliasCmd := &cobra.Command{
		Use:                name,
		Short:              short,
		Long:               short,
		Hidden:             true, // Hide from help to avoid confusion
		DisableFlagParsing: true, // Pass all flags to the target command
		Run: func(cmd *cobra.Command, args []string) {
			// Split the target into command parts
			targetParts := strings.Split(target, " ")

			// Reconstruct the command line with the target command and original args
			newArgs := append(targetParts[1:], args...)
			rootCmd.SetArgs(append([]string{targetParts[0]}, newArgs...))

			// Execute the target command
			err := rootCmd.Execute()
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	return aliasCmd
}
