package deploy

import (
	"context"
	"strings"

	"github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/flags/groups"
	"github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/ArjenSchwarz/fog/cmd/validation"
	"github.com/ArjenSchwarz/fog/config"
)

// Handler implements the deploy command logic.
type Handler struct {
	flags             *groups.DeploymentFlags
	deploymentService services.DeploymentService
	config            *config.Config
}

// NewHandler creates a new deploy command handler.
func NewHandler(flags *groups.DeploymentFlags, deploymentService services.DeploymentService, config *config.Config) *Handler {
	return &Handler{
		flags:             flags,
		deploymentService: deploymentService,
		config:            config,
	}
}

// Execute runs the deploy command using the deployment service.
func (h *Handler) Execute(ctx context.Context) error {
	errorCtx := errors.NewErrorContext("deploy", "command").WithStackName(h.flags.StackName)
	ctx = errors.WithErrorContext(ctx, errorCtx)

	if err := h.ValidateFlags(); err != nil {
		return err
	}

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
		return errors.WrapError(errorCtx, err, errors.ErrInternal, "failed to prepare deployment")
	}

	if err := h.deploymentService.ValidateDeployment(ctx, plan); err != nil {
		return errors.WrapError(errorCtx, err, errors.ErrInternal, "deployment validation failed")
	}

	changeset, err := h.deploymentService.CreateChangeset(ctx, plan)
	if err != nil {
		return errors.WrapError(errorCtx, err, errors.ErrChangesetFailed, "failed to create changeset")
	}

	if opts.DryRun {
		return nil
	}

	if opts.CreateOnly {
		return nil
	}

	result, err := h.deploymentService.ExecuteDeployment(ctx, plan, changeset)
	if err != nil {
		return errors.WrapError(errorCtx, err, errors.ErrDeploymentFailed, "deployment failed")
	}

	if result.Success {
		return nil
	}
	return errors.ContextualError(errorCtx, errors.ErrDeploymentFailed, "deployment completed with errors")
}

// ValidateFlags validates the command flags using the Flags struct.
func (h *Handler) ValidateFlags() error {
	if h.flags == nil {
		return errors.ContextualError(
			errors.NewErrorContext("deploy", "validation"),
			errors.ErrInternal,
			"no flags provided",
		)
	}

	vb := validation.NewValidationErrorBuilder("deploy-flags")

	if h.flags.StackName == "" {
		vb.RequiredField("stackname")
	}

	if h.flags.DeploymentFile != "" && (h.flags.Template != "" || h.flags.Parameters != "" || h.flags.Tags != "") {
		flags := []string{"deployment-file"}
		if h.flags.Template != "" {
			flags = append(flags, "template")
		}
		if h.flags.Parameters != "" {
			flags = append(flags, "parameters")
		}
		if h.flags.Tags != "" {
			flags = append(flags, "tags")
		}
		vb.ConflictingFlags(flags)
	}

	return vb.Build()
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
