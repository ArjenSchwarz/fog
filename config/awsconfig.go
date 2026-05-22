package config

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	external "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudcontrol"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// AWSConfig is a holder for AWS Config type information
type AWSConfig struct {
	AccountAlias string
	AccountID    string
	Config       aws.Config
	ProfileName  string
	Region       string
	UserID       string
}

// profileReader is the minimal Config surface sharedConfigProfile needs.
// It exists so the helper can be exercised in tests without depending on the
// global viper-backed Config.
type profileReader interface {
	GetString(setting string) string
}

// sharedConfigProfile returns the AWS shared config profile name from config,
// preserving the original case.
//
// AWS profile names (in ~/.aws/config and ~/.aws/credentials) are
// case-sensitive, so the raw GetString value is used instead of the
// lowercased GetLCString value. See T-880.
func sharedConfigProfile(config profileReader) string {
	return config.GetString("profile")
}

// DefaultAwsConfig loads default AWS Config
func DefaultAwsConfig(ctx context.Context, config Config) (AWSConfig, error) {
	awsConfig := AWSConfig{}
	profile := sharedConfigProfile(&config)
	if profile != "" {
		awsConfig.ProfileName = profile
		cfg, err := external.LoadDefaultConfig(ctx, external.WithSharedConfigProfile(profile))
		if err != nil {
			return awsConfig, err
		}
		awsConfig.Config = cfg
	} else {
		cfg, err := external.LoadDefaultConfig(ctx, external.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 0)
		}))

		if err != nil {
			return awsConfig, err
		}
		awsConfig.Config = cfg
	}
	if config.GetLCString("region") != "" {
		awsConfig.Config.Region = config.GetLCString("region")
	}
	awsConfig.Region = awsConfig.Config.Region
	err := awsConfig.setCallerInfo(ctx)
	if err != nil {
		return awsConfig, err
	}
	awsConfig.setAlias(ctx)
	return awsConfig, nil
}

// StsClient returns an STS Client
func (config *AWSConfig) StsClient() *sts.Client {
	return sts.NewFromConfig(config.Config)
}

// CloudformationClient returns a Cloudformation Client
func (config *AWSConfig) CloudformationClient() *cloudformation.Client {
	return cloudformation.NewFromConfig(config.Config)
}

// S3Client returns an S3 Client
func (config *AWSConfig) S3Client() *s3.Client {
	return s3.NewFromConfig(config.Config)
}

// IAMClient returns an IAM Client
func (config *AWSConfig) IAMClient() *iam.Client {
	return iam.NewFromConfig(config.Config)
}

// EC2Client returns an EC2 Client
func (config *AWSConfig) EC2Client() *ec2.Client {
	return ec2.NewFromConfig(config.Config)
}

// CloudControlClient returns a CloudControl Client
func (config *AWSConfig) CloudControlClient() *cloudcontrol.Client {
	return cloudcontrol.NewFromConfig(config.Config)
}

// SSOAdminClient returns an SSO Admin Client
func (config *AWSConfig) SSOAdminClient() *ssoadmin.Client {
	return ssoadmin.NewFromConfig(config.Config)
}

// OrganizationsClient returns an Organizations Client
func (config *AWSConfig) OrganizationsClient() *organizations.Client {
	return organizations.NewFromConfig(config.Config)
}

func (config *AWSConfig) setCallerInfo(ctx context.Context) error {
	accountID, userID, err := resolveCallerInfo(ctx, config.StsClient())
	if err != nil {
		return err
	}
	config.AccountID = accountID
	config.UserID = userID
	return nil
}

// resolveCallerInfo calls STS GetCallerIdentity and extracts the account and
// user IDs. STS returns these as optional *string fields, so a successful
// (nil-error) response can still omit them — for example via an SDK mock,
// middleware, or partial response. Missing values are reported as an error
// rather than dereferenced, which previously panicked at startup (T-1110).
func resolveCallerInfo(ctx context.Context, client STSGetCallerIdentityAPI) (accountID string, userID string, err error) {
	result, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", "", err
	}
	if result == nil || result.Account == nil || result.UserId == nil {
		return "", "", fmt.Errorf("STS GetCallerIdentity returned an incomplete identity: missing account or user ID")
	}
	return *result.Account, *result.UserId, nil
}

func (config *AWSConfig) setAlias(ctx context.Context) {
	config.AccountAlias = resolveAccountAlias(ctx, config.IAMClient())
}

// resolveAccountAlias calls IAM ListAccountAliases and returns the first alias,
// or an empty string when none is available. A nil/empty output, an empty
// alias list, or an error (e.g. the caller lacks iam:ListAccountAliases
// permission) all yield an empty alias without failing. Previously len() was
// evaluated on a result that could be nil on a successful call, which panicked
// (T-1110).
func resolveAccountAlias(ctx context.Context, client IAMListAccountAliasesAPI) string {
	result, err := client.ListAccountAliases(ctx, &iam.ListAccountAliasesInput{})
	if err != nil || result == nil || len(result.AccountAliases) == 0 {
		// If the user doesn't have permission to see the aliases or the account has no aliases, continue without
		return ""
	}
	return result.AccountAliases[0]
}

// GetAccountAliasID returns the account alias with ID in parentheses if alias exists, otherwise just the ID
func (config *AWSConfig) GetAccountAliasID() string {
	if config.AccountAlias != "" {
		return fmt.Sprintf("%s (%s)", config.AccountAlias, config.AccountID)
	} else {
		return config.AccountID
	}
}
