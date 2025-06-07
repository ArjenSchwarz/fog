package errors

// ErrorCode represents a specific error condition.
type ErrorCode string

// ErrorCategory defines the broad category an error belongs to.
type ErrorCategory int

const (
	CategoryUnknown ErrorCategory = iota
	CategoryValidation
	CategoryConfiguration
	CategoryNetwork
	CategoryAWS
	CategoryFileSystem
	CategoryTemplate
	CategoryPermission
	CategoryResource
	CategoryInternal
)

// ErrorSeverity indicates how severe an error is considered.
type ErrorSeverity int

const (
	SeverityLow ErrorSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// General error codes.
const (
	ErrUnknown        ErrorCode = "UNKNOWN"
	ErrInternal       ErrorCode = "INTERNAL"
	ErrNotImplemented ErrorCode = "NOT_IMPLEMENTED"
	ErrMultipleErrors ErrorCode = "MULTIPLE_ERRORS"
)

// Validation error codes.
const (
	ErrValidationFailed  ErrorCode = "VALIDATION_FAILED"
	ErrRequiredField     ErrorCode = "REQUIRED_FIELD"
	ErrInvalidValue      ErrorCode = "INVALID_VALUE"
	ErrInvalidFormat     ErrorCode = "INVALID_FORMAT"
	ErrConflictingFlags  ErrorCode = "CONFLICTING_FLAGS"
	ErrDependencyMissing ErrorCode = "DEPENDENCY_MISSING"
)

// Configuration error codes.
const (
	ErrConfigNotFound     ErrorCode = "CONFIG_NOT_FOUND"
	ErrConfigInvalid      ErrorCode = "CONFIG_INVALID"
	ErrConfigPermission   ErrorCode = "CONFIG_PERMISSION"
	ErrMissingCredentials ErrorCode = "MISSING_CREDENTIALS"
	ErrInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
)

// File system error codes.
const (
	ErrFileNotFound        ErrorCode = "FILE_NOT_FOUND"
	ErrFilePermission      ErrorCode = "FILE_PERMISSION"
	ErrFileInvalid         ErrorCode = "FILE_INVALID"
	ErrDirectoryNotFound   ErrorCode = "DIRECTORY_NOT_FOUND"
	ErrDirectoryPermission ErrorCode = "DIRECTORY_PERMISSION"
)

// Template error codes.
const (
	ErrTemplateNotFound     ErrorCode = "TEMPLATE_NOT_FOUND"
	ErrTemplateInvalid      ErrorCode = "TEMPLATE_INVALID"
	ErrTemplateTooLarge     ErrorCode = "TEMPLATE_TOO_LARGE"
	ErrTemplateUploadFailed ErrorCode = "TEMPLATE_UPLOAD_FAILED"
	ErrParameterInvalid     ErrorCode = "PARAMETER_INVALID"
	ErrParameterMissing     ErrorCode = "PARAMETER_MISSING"
)

// AWS error codes.
const (
	ErrAWSAuthentication    ErrorCode = "AWS_AUTHENTICATION"
	ErrAWSPermission        ErrorCode = "AWS_PERMISSION"
	ErrAWSRateLimit         ErrorCode = "AWS_RATE_LIMIT"
	ErrAWSServiceError      ErrorCode = "AWS_SERVICE_ERROR"
	ErrAWSRegionInvalid     ErrorCode = "AWS_REGION_INVALID"
	ErrStackNotFound        ErrorCode = "STACK_NOT_FOUND"
	ErrStackInvalidState    ErrorCode = "STACK_INVALID_STATE"
	ErrChangesetFailed      ErrorCode = "CHANGESET_FAILED"
	ErrDeploymentFailed     ErrorCode = "DEPLOYMENT_FAILED"
	ErrDriftDetectionFailed ErrorCode = "DRIFT_DETECTION_FAILED"
)

// Network error codes.
const (
	ErrNetworkTimeout     ErrorCode = "NETWORK_TIMEOUT"
	ErrNetworkConnection  ErrorCode = "NETWORK_CONNECTION"
	ErrNetworkUnreachable ErrorCode = "NETWORK_UNREACHABLE"
)

// Resource error codes.
const (
	ErrResourceNotFound ErrorCode = "RESOURCE_NOT_FOUND"
	ErrResourceConflict ErrorCode = "RESOURCE_CONFLICT"
	ErrResourceLimit    ErrorCode = "RESOURCE_LIMIT"
	ErrResourceLocked   ErrorCode = "RESOURCE_LOCKED"
)

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
