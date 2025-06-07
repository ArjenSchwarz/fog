package errors

// GetErrorCategory returns the category for an error code.
func GetErrorCategory(code ErrorCode) ErrorCategory {
	switch code {
	case ErrValidationFailed, ErrRequiredField, ErrInvalidValue, ErrInvalidFormat, ErrConflictingFlags, ErrDependencyMissing:
		return CategoryValidation
	case ErrConfigNotFound, ErrConfigInvalid, ErrConfigPermission, ErrMissingCredentials, ErrInvalidCredentials:
		return CategoryConfiguration
	case ErrFileNotFound, ErrFilePermission, ErrFileInvalid, ErrDirectoryNotFound, ErrDirectoryPermission:
		return CategoryFileSystem
	case ErrTemplateNotFound, ErrTemplateInvalid, ErrTemplateTooLarge, ErrTemplateUploadFailed, ErrParameterInvalid, ErrParameterMissing:
		return CategoryTemplate
	case ErrAWSAuthentication, ErrAWSPermission, ErrAWSRateLimit, ErrAWSServiceError, ErrAWSRegionInvalid, ErrStackNotFound, ErrStackInvalidState, ErrChangesetFailed, ErrDeploymentFailed, ErrDriftDetectionFailed:
		return CategoryAWS
	case ErrNetworkTimeout, ErrNetworkConnection, ErrNetworkUnreachable:
		return CategoryNetwork
	case ErrResourceNotFound, ErrResourceConflict, ErrResourceLimit, ErrResourceLocked:
		return CategoryResource
	case ErrInternal, ErrNotImplemented:
		return CategoryInternal
	default:
		return CategoryUnknown
	}
}

// GetErrorSeverity returns the severity for an error code.
func GetErrorSeverity(code ErrorCode) ErrorSeverity {
	switch code {
	case ErrInternal, ErrDeploymentFailed, ErrChangesetFailed:
		return SeverityCritical
	case ErrAWSAuthentication, ErrAWSPermission, ErrStackInvalidState, ErrConfigInvalid, ErrMissingCredentials:
		return SeverityHigh
	case ErrValidationFailed, ErrTemplateInvalid, ErrParameterInvalid, ErrFileNotFound, ErrNetworkTimeout:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// IsRetryable returns whether an error with the given code is retryable.
func IsRetryable(code ErrorCode) bool {
	switch code {
	case ErrNetworkTimeout, ErrNetworkConnection, ErrAWSRateLimit, ErrAWSServiceError:
		return true
	case ErrValidationFailed, ErrRequiredField, ErrInvalidValue, ErrConfigInvalid, ErrFileNotFound, ErrTemplateInvalid:
		return false
	default:
		return false
	}
}

// ErrorCodeMetadata provides additional information about error codes.
type ErrorCodeMetadata struct {
	Code        ErrorCode
	Category    ErrorCategory
	Severity    ErrorSeverity
	Retryable   bool
	Description string
	Suggestions []string
}

// GetErrorMetadata returns metadata for an error code.
func GetErrorMetadata(code ErrorCode) ErrorCodeMetadata {
	metadata := ErrorCodeMetadata{
		Code:      code,
		Category:  GetErrorCategory(code),
		Severity:  GetErrorSeverity(code),
		Retryable: IsRetryable(code),
	}

	switch code {
	case ErrTemplateNotFound:
		metadata.Description = "CloudFormation template file not found"
		metadata.Suggestions = []string{
			"Check that the template file path is correct",
			"Ensure the file exists and is readable",
		}
	case ErrStackNotFound:
		metadata.Description = "CloudFormation stack does not exist"
		metadata.Suggestions = []string{
			"Verify the stack name is correct",
			"Check that you're in the correct AWS region",
			"Use 'fog list' to see available stacks",
		}
	case ErrAWSAuthentication:
		metadata.Description = "AWS authentication failed"
		metadata.Suggestions = []string{
			"Check your AWS credentials",
			"Verify AWS CLI configuration",
			"Ensure correct AWS region is set",
		}
	case ErrValidationFailed:
		metadata.Description = "Input validation failed"
		metadata.Suggestions = []string{
			"Review the validation errors",
			"Check command flags and arguments",
			"Refer to the command help for usage information",
		}
	}

	return metadata
}
