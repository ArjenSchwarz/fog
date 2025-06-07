package deploy

import (
	"context"
	"fmt"
	"strings"

	"github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/ArjenSchwarz/fog/config"
)

// Handler implements the deploy command logic.
type Handler struct {
	flags             *Flags
	deploymentService services.DeploymentService
	config            *config.Config
}

// NewHandler creates a new deploy command handler.
func NewHandler(flags *Flags, deploymentService services.DeploymentService, config *config.Config) *Handler {
	return &Handler{
		flags:             flags,
		deploymentService: deploymentService,
		config:            config,
	}
}

// Execute runs the deploy command using the deployment service.
func (h *Handler) Execute(ctx context.Context) error {
	opts := services.DeploymentOptions{
		StackName:      h.flags.StackName,
		TemplateSource: h.flags.Template,
		ParameterFiles: parseCommaSeparated(h.flags.Parameters),
		TagFiles:       parseCommaSeparated(h.flags.Tags),
		DefaultTags:    h.flags.DefaultTags,
		Bucket:         h.flags.Bucket,
		ChangesetName:  h.flags.ChangesetName,
		DeploymentFile: h.flags.DeploymentFile,
		DryRun:         h.flags.Dryrun,
		NonInteractive: h.flags.NonInteractive,
		CreateOnly:     h.flags.CreateChangeset,
		DeployOnly:     h.flags.DeployChangeset,
	}

	plan, err := h.deploymentService.PrepareDeployment(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to prepare deployment: %w", err)
	}

	if err := h.deploymentService.ValidateDeployment(ctx, plan); err != nil {
		return fmt.Errorf("deployment validation failed: %w", err)
	}

	changeset, err := h.deploymentService.CreateChangeset(ctx, plan)
	if err != nil {
		return fmt.Errorf("failed to create changeset: %w", err)
	}

	if opts.DryRun {
		return nil
	}

	if opts.CreateOnly {
		return nil
	}

	result, err := h.deploymentService.ExecuteDeployment(ctx, plan, changeset)
	if err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	if result.Success {
		return nil
	}
	return fmt.Errorf("deployment completed with errors: %s", result.ErrorMessage)
}

// ValidateFlags validates the command flags using the Flags struct.
func (h *Handler) ValidateFlags() error {
	if h.flags == nil {
		return fmt.Errorf("no flags provided")
	}
	return h.flags.Validate()
}

// parseCommaSeparated splits a comma-separated string and trims whitespace.
func parseCommaSeparated(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			result = append(result, v)
		}
	}
	return result
}
