package config

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	external "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

//AWSConfig is a holder for AWS Config type information
type AWSConfig struct {
	Config    aws.Config
	Region    string
	AccountID string
	UserID    string
}

// DefaultAwsConfig loads default AWS Config
func DefaultAwsConfig(config Config) AWSConfig {
	awsConfig := AWSConfig{}
	if config.GetLCString("profile") != "" {
		cfg, err := external.LoadDefaultConfig(context.TODO(), external.WithSharedConfigProfile(config.GetLCString("profile")))
		if err != nil {
			panic(err)
		}
		awsConfig.Config = cfg
	} else {
		cfg, err := external.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic(err)
		}
		awsConfig.Config = cfg
	}
	if config.GetLCString("region") != "" {
		awsConfig.Config.Region = config.GetLCString("region")
	}
	awsConfig.Region = awsConfig.Config.Region
	awsConfig.setCallerInfo()
	return awsConfig
}

// StsClient returns an STS Client
func (config *AWSConfig) StsClient() *sts.Client {
	return sts.NewFromConfig(config.Config)
}

//CloudformationClient returns an cloudformation Client
func (config *AWSConfig) CloudformationClient() *cloudformation.Client {
	return cloudformation.NewFromConfig(config.Config)
}

func (config *AWSConfig) setCallerInfo() {
	c := config.StsClient()
	result, err := c.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		panic(err)
	}
	config.AccountID = *result.Account
	config.UserID = *result.UserId
}
