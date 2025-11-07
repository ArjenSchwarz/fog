package cmd

import (
	"context"
	"fmt"
	"os"

	output "github.com/ArjenSchwarz/go-output/v2"
)

// formatInfo formats an informational message with emoji and styling.
// Replicates the behavior of v1's outputsettings.StringInfo().
// Output format: "ℹ️  <text>\n\n"
func formatInfo(text string) string {
	return fmt.Sprintf("ℹ️  %s\n\n", text)
}

// formatSuccess formats a success message with emoji, color, and styling.
// Replicates the behavior of v1's outputsettings.StringSuccess().
// Output format: "✅ <green-bold-text>\n\n"
func formatSuccess(text string) string {
	return fmt.Sprintf("✅ %s\n\n", output.StylePositive(text))
}

// formatError formats an error message with emoji, color, and styling.
// Replicates the behavior of v1's outputsettings.StringFailure().
// Output format: "\n🚨 <red-bold-text> 🚨\n\n"
func formatError(text string) string {
	return fmt.Sprintf("\n🚨 %s 🚨\n\n", output.StyleWarning(text))
}

// formatPositive formats a positive message with color and styling.
// Replicates the behavior of v1's outputsettings.StringPositive().
// Output format: "<green-bold-text>\n"
func formatPositive(text string) string {
	return fmt.Sprintf("%s\n", output.StylePositive(text))
}

// formatBold formats text in bold.
// Replicates the behavior of v1's outputsettings.StringBold().
// Output format: "<bold-text>"
func formatBold(text string) string {
	return output.StyleBold(text)
}

// printMessage renders a formatted message using the go-output document builder.
// This ensures proper table separation when messages are followed by tables.
func printMessage(message string) {
	doc := output.New().Text(message).Build()
	out := createStderrOutput()
	if err := out.Render(context.Background(), doc); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to render message: %v\n", err)
	}
}
