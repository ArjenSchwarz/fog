package deployment

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfnTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/services"
)

// validateStackState checks if the stack is in a valid state for deployment
func (s *Service) validateStackState(ctx context.Context, plan *services.DeploymentPlan) ferr.FogError {
	errorCtx := ferr.GetErrorContext(ctx)

	// Check if stack exists
	input := &cloudformation.DescribeStacksInput{
		StackName: aws.String(plan.StackName),
	}

	output, err := s.cfnClient.DescribeStacks(ctx, input)
	if err != nil {
		// Stack doesn't exist - this is valid for new stacks
		plan.IsNewStack = true
		return nil
	}

	if len(output.Stacks) == 0 {
		// Stack doesn't exist - this is valid for new stacks
		plan.IsNewStack = true
		return nil
	}

	// Stack exists - check its state
	stack := output.Stacks[0]
	status := stack.StackStatus
	plan.IsNewStack = false

	// Check if stack is in a state that allows updates
	validUpdateStates := []cfnTypes.StackStatus{
		cfnTypes.StackStatusCreateComplete,
		cfnTypes.StackStatusUpdateComplete,
		cfnTypes.StackStatusUpdateRollbackComplete,
		cfnTypes.StackStatusImportComplete,
		cfnTypes.StackStatusImportRollbackComplete,
	}

	for _, validState := range validUpdateStates {
		if status == validState {
			return nil // Stack is in valid state for update
		}
	}

	// Stack is in invalid state for deployment
	return ferr.ContextualError(errorCtx, ferr.ErrStackInvalidState,
		fmt.Sprintf("stack %s is in state %s which does not allow updates", plan.StackName, status))
}
