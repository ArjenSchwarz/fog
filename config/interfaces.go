package config

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// AWSConfigLoader defines the interface for loading AWS SDK configuration.
// This interface abstracts the AWS SDK's config.LoadDefaultConfig function
// to enable testing without requiring actual AWS credentials.
type AWSConfigLoader interface {
	LoadDefaultConfig(ctx context.Context, optFns ...func(*aws_config.LoadOptions) error) (aws.Config, error)
}

// STSGetCallerIdentityAPI defines the STS operation for retrieving caller identity information.
// This is used to get the AWS account ID and user ID during AWS configuration setup.
type STSGetCallerIdentityAPI interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// IAMListAccountAliasesAPI defines the IAM operation for listing account aliases.
// This is used to retrieve the account alias during AWS configuration setup.
type IAMListAccountAliasesAPI interface {
	ListAccountAliases(ctx context.Context, params *iam.ListAccountAliasesInput, optFns ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error)
}

// ConfigReader defines the interface for reading configuration files.
// This interface abstracts file system operations and configuration parsing
// to enable testing with in-memory configurations.
type ConfigReader interface {
	// ReadFile reads the contents of a configuration file
	ReadFile(filename string) ([]byte, error)

	// Unmarshal parses configuration data into a structure
	// The data parameter contains the raw configuration bytes
	// The v parameter is a pointer to the target structure
	Unmarshal(data []byte, v any) error
}

// ViperConfigAPI defines the interface for Viper configuration operations.
// This interface wraps the commonly used Viper methods to enable testing
// without requiring actual configuration files.
type ViperConfigAPI interface {
	// IsSet checks if a configuration key exists
	IsSet(key string) bool

	// GetString retrieves a string value for the given key
	GetString(key string) string

	// GetStringSlice retrieves a string slice value for the given key
	GetStringSlice(key string) []string

	// GetBool retrieves a boolean value for the given key
	GetBool(key string) bool

	// GetInt retrieves an integer value for the given key
	GetInt(key string) int

	// SetConfigFile sets the path to the configuration file
	SetConfigFile(in string)

	// ReadInConfig reads the configuration file
	ReadInConfig() error
}
