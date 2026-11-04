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
	"errors"
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
		Args:  rejectUnknownSubcommand,
		RunE:  showGroupHelp,
	}

	// Resource management commands
	resourceGroupCmd = &cobra.Command{
		Use:   "resource",
		Short: "Resource management commands",
		Long:  `Commands for working with CloudFormation resources and exports.`,
		Args:  rejectUnknownSubcommand,
		RunE:  showGroupHelp,
	}

	// Utility commands
	utilGroupCmd = &cobra.Command{
		Use:   "util",
		Short: "Utility commands",
		Long:  `Utility commands for various helper functions.`,
		Args:  rejectUnknownSubcommand,
		RunE:  showGroupHelp,
	}
)

// rejectUnknownSubcommand is the Args validator for command groups. Cobra only
// checks for unknown subcommands on the root command, so without this a typo
// such as "fog stack deplyo" prints the group help and exits successfully.
func rejectUnknownSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	msg := fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath())
	// Match the default distance Cobra uses for root-level suggestions.
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
		msg += "\n\nDid you mean this?\n\t" + strings.Join(suggestions, "\n\t")
	}
	return errors.New(msg)
}

// showGroupHelp makes a command group runnable so Cobra applies its Args
// validator (Cobra skips argument validation for non-runnable commands).
// Running the group without arguments still just shows its help.
func showGroupHelp(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}

// validateHelpTopic is the Args validator for the help command. Cobra's own
// help command prints "Unknown help topic" but still exits successfully, so a
// typo such as "fog help deplyo" would go unnoticed in scripts.
func validateHelpTopic(cmd *cobra.Command, args []string) error {
	target, rest, err := cmd.Root().Find(args)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return nil
	}
	// Aliases have no subcommands of their own, so check the remaining words
	// against the alias target ("fog help describe changeset" is valid
	// because "fog stack describe changeset" exists).
	if aliasTarget, ok := target.Annotations[aliasTargetAnnotation]; ok {
		return validateHelpTopic(cmd, append(strings.Fields(aliasTarget), rest...))
	}
	return rejectUnknownSubcommand(target, rest)
}

// initGeneratedCommands creates Cobra's help and completion commands up front
// (Cobra would otherwise only add them during Execute) and adds argument
// validation to them, so "fog help nope" and "fog completion nope" fail like
// any other unknown command. Execute keeps these instances.
func initGeneratedCommands(root *cobra.Command) {
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	for _, c := range root.Commands() {
		switch c.Name() {
		case "help":
			c.Args = validateHelpTopic
		case "completion":
			// The completion group already uses cobra.NoArgs, but Cobra skips
			// Args validation for commands that aren't runnable.
			c.Args = rejectUnknownSubcommand
			c.RunE = showGroupHelp
		}
	}
}

// InitGroups initializes all command groups and adds them to the root command
func InitGroups() {
	// Add command groups to root command
	rootCmd.AddCommand(stackCmd)
	rootCmd.AddCommand(resourceGroupCmd)
	rootCmd.AddCommand(utilGroupCmd)
}

// aliasTargetAnnotation records an alias's target command path, so help
// topics given through an alias can be validated against the real command.
const aliasTargetAnnotation = "fog-alias-target"

// NewCommandAlias creates a command that redirects to another command path
func NewCommandAlias(name, target, short string) *cobra.Command {
	// Create the alias command
	aliasCmd := &cobra.Command{
		Use:                name,
		Short:              short,
		Long:               short,
		Hidden:             true, // Hide from help to avoid confusion
		DisableFlagParsing: true, // Pass all flags to the target command
		Annotations:        map[string]string{aliasTargetAnnotation: target},
		Run: func(cmd *cobra.Command, args []string) {
			// Split the target into command parts
			targetParts := strings.Split(target, " ")

			// Reconstruct the command line with the target command and original args
			newArgs := make([]string, 0, len(targetParts[1:])+len(args))
			newArgs = append(newArgs, targetParts[1:]...)
			newArgs = append(newArgs, args...)
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
