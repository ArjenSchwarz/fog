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

// DefaultAwsConfig loads default AWS Config
func DefaultAwsConfig(config Config) (AWSConfig, error) {
	awsConfig := AWSConfig{}
	if config.GetLCString("profile") != "" {
		awsConfig.ProfileName = config.GetLCString("profile")
		cfg, err := external.LoadDefaultConfig(context.TODO(), external.WithSharedConfigProfile(config.GetLCString("profile")))
		if err != nil {
			return awsConfig, err
		}
		awsConfig.Config = cfg
	} else {
		cfg, err := external.LoadDefaultConfig(context.TODO(), external.WithRetryer(func() aws.Retryer {
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
	err := awsConfig.setCallerInfo()
	if err != nil {
		return awsConfig, err
	}
	awsConfig.setAlias()
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

// IAMClient returns an EC2 Client
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

func (config *AWSConfig) OrganizationsClient() *organizations.Client {
	return organizations.NewFromConfig(config.Config)
}

func (config *AWSConfig) setCallerInfo() error {
	c := config.StsClient()
	result, err := c.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return err
	}
	config.AccountID = *result.Account
	config.UserID = *result.UserId
	return nil
}

func (config *AWSConfig) setAlias() {
	c := config.IAMClient()
	result, err := c.ListAccountAliases(context.TODO(), &iam.ListAccountAliasesInput{})
	if err != nil || len(result.AccountAliases) == 0 {
		// If the user doesn't have permission to see the aliases or the account has no aliases, continue without
		return
	}
	config.AccountAlias = result.AccountAliases[0]
}

func (config *AWSConfig) GetAccountAliasID() string {
	if config.AccountAlias != "" {
		return fmt.Sprintf("%s (%s)", config.AccountAlias, config.AccountID)
	} else {
		return config.AccountID
	}
}
