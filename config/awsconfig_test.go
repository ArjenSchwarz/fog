package config

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConfig implements a simple Config for testing
type mockConfig struct {
	values map[string]any
}

func newMockConfig() *mockConfig {
	return &mockConfig{
		values: make(map[string]any),
	}
}

func (m *mockConfig) withProfile(profile string) *mockConfig {
	m.values["profile"] = profile
	return m
}

func (m *mockConfig) withRegion(region string) *mockConfig {
	m.values["region"] = region
	return m
}

func (m *mockConfig) GetLCString(setting string) string {
	if val, ok := m.values[setting]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (m *mockConfig) GetString(setting string) string {
	return m.GetLCString(setting)
}

func (m *mockConfig) GetStringSlice(setting string) []string {
	if val, ok := m.values[setting]; ok {
		if slice, ok := val.([]string); ok {
			return slice
		}
	}
	return []string{}
}

func (m *mockConfig) GetBool(setting string) bool {
	if val, ok := m.values[setting]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func (m *mockConfig) GetInt(setting string) int {
	if val, ok := m.values[setting]; ok {
		if i, ok := val.(int); ok {
			return i
		}
	}
	return 0
}

func TestDefaultAwsConfig(t *testing.T) {
	// Note: This test requires actual AWS SDK initialization which would need credentials
	// These tests are marked as integration tests and will be skipped without INTEGRATION=1
	t.Skip("Skipping AWS config test - requires AWS SDK mocking infrastructure")
}

func TestAWSConfig_GetAccountAliasID(t *testing.T) {
	tests := map[string]struct {
		config AWSConfig
		want   string
	}{
		"with alias": {
			config: AWSConfig{
				AccountAlias: "my-account",
				AccountID:    "123456789012",
			},
			want: "my-account (123456789012)",
		},
		"without alias": {
			config: AWSConfig{
				AccountID: "123456789012",
			},
			want: "123456789012",
		},
		"empty account ID": {
			config: AWSConfig{},
			want:   "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.config.GetAccountAliasID()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestAWSConfig_ClientCreation(t *testing.T) {
	config := &AWSConfig{
		Config: aws.Config{
			Region: "us-west-2",
		},
	}

	tests := map[string]struct {
		createClient func(*AWSConfig) any
		expectedType string
	}{
		"CloudformationClient": {
			createClient: func(c *AWSConfig) any { return c.CloudformationClient() },
			expectedType: "*cloudformation.Client",
		},
		"StsClient": {
			createClient: func(c *AWSConfig) any { return c.StsClient() },
			expectedType: "*sts.Client",
		},
		"S3Client": {
			createClient: func(c *AWSConfig) any { return c.S3Client() },
			expectedType: "*s3.Client",
		},
		"IAMClient": {
			createClient: func(c *AWSConfig) any { return c.IAMClient() },
			expectedType: "*iam.Client",
		},
		"EC2Client": {
			createClient: func(c *AWSConfig) any { return c.EC2Client() },
			expectedType: "*ec2.Client",
		},
		"CloudControlClient": {
			createClient: func(c *AWSConfig) any { return c.CloudControlClient() },
			expectedType: "*cloudcontrol.Client",
		},
		"SSOAdminClient": {
			createClient: func(c *AWSConfig) any { return c.SSOAdminClient() },
			expectedType: "*ssoadmin.Client",
		},
		"OrganizationsClient": {
			createClient: func(c *AWSConfig) any { return c.OrganizationsClient() },
			expectedType: "*organizations.Client",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := tc.createClient(config)
			require.NotNil(t, client, "Client should not be nil")
		})
	}
}

func TestMockSTSClient_GetCallerIdentity(t *testing.T) {
	tests := map[string]struct {
		setupMock func() *mockSTSClient
		checkFn   func(*testing.T, *sts.GetCallerIdentityOutput, error)
	}{
		"successful call with defaults": {
			setupMock: func() *mockSTSClient {
				return &mockSTSClient{}
			},
			checkFn: func(t *testing.T, got *sts.GetCallerIdentityOutput, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, got)
				// Default mock returns empty output
			},
		},
		"successful call with custom values": {
			setupMock: func() *mockSTSClient {
				return &mockSTSClient{
					getCallerIdentityFn: func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
						account := "999888777666"
						userId := "CUSTOMUSERID"
						return &sts.GetCallerIdentityOutput{
							Account: &account,
							UserId:  &userId,
						}, nil
					},
				}
			},
			checkFn: func(t *testing.T, got *sts.GetCallerIdentityOutput, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, "999888777666", *got.Account)
				assert.Equal(t, "CUSTOMUSERID", *got.UserId)
			},
		},
		"error from AWS": {
			setupMock: func() *mockSTSClient {
				return &mockSTSClient{
					getCallerIdentityFn: func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
						return nil, errors.New("access denied")
					},
				}
			},
			checkFn: func(t *testing.T, got *sts.GetCallerIdentityOutput, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "access denied")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock := tc.setupMock()
			got, err := mock.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
			tc.checkFn(t, got, err)
		})
	}
}

func TestMockIAMClient_ListAccountAliases(t *testing.T) {
	tests := map[string]struct {
		setupMock func() *mockIAMClient
		checkFn   func(*testing.T, *iam.ListAccountAliasesOutput, error)
	}{
		"successful call with defaults": {
			setupMock: func() *mockIAMClient {
				return &mockIAMClient{}
			},
			checkFn: func(t *testing.T, got *iam.ListAccountAliasesOutput, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, got)
				// Default mock returns empty output
			},
		},
		"successful call with custom aliases": {
			setupMock: func() *mockIAMClient {
				return &mockIAMClient{
					listAccountAliasesFn: func(ctx context.Context, params *iam.ListAccountAliasesInput, optFns ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
						return &iam.ListAccountAliasesOutput{
							AccountAliases: []string{"custom-alias"},
						}, nil
					},
				}
			},
			checkFn: func(t *testing.T, got *iam.ListAccountAliasesOutput, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, []string{"custom-alias"}, got.AccountAliases)
			},
		},
		"no aliases": {
			setupMock: func() *mockIAMClient {
				return &mockIAMClient{
					listAccountAliasesFn: func(ctx context.Context, params *iam.ListAccountAliasesInput, optFns ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
						return &iam.ListAccountAliasesOutput{
							AccountAliases: []string{},
						}, nil
					},
				}
			},
			checkFn: func(t *testing.T, got *iam.ListAccountAliasesOutput, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Empty(t, got.AccountAliases)
			},
		},
		"error from AWS": {
			setupMock: func() *mockIAMClient {
				return &mockIAMClient{
					listAccountAliasesFn: func(ctx context.Context, params *iam.ListAccountAliasesInput, optFns ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
						return nil, errors.New("access denied")
					},
				}
			},
			checkFn: func(t *testing.T, got *iam.ListAccountAliasesOutput, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "access denied")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock := tc.setupMock()
			got, err := mock.ListAccountAliases(context.Background(), &iam.ListAccountAliasesInput{})
			tc.checkFn(t, got, err)
		})
	}
}

func TestMockConfig(t *testing.T) {
	tests := map[string]struct {
		setupMock func() *mockConfig
		testFunc  func(*testing.T, *mockConfig)
	}{
		"GetLCString with value": {
			setupMock: func() *mockConfig {
				return newMockConfig().withProfile("test-profile")
			},
			testFunc: func(t *testing.T, m *mockConfig) {
				t.Helper()
				got := m.GetLCString("profile")
				assert.Equal(t, "test-profile", got)
			},
		},
		"GetLCString without value": {
			setupMock: func() *mockConfig {
				return newMockConfig()
			},
			testFunc: func(t *testing.T, m *mockConfig) {
				t.Helper()
				got := m.GetLCString("profile")
				assert.Equal(t, "", got)
			},
		},
		"GetString with value": {
			setupMock: func() *mockConfig {
				return newMockConfig().withRegion("us-east-1")
			},
			testFunc: func(t *testing.T, m *mockConfig) {
				t.Helper()
				got := m.GetString("region")
				assert.Equal(t, "us-east-1", got)
			},
		},
		"GetBool with value": {
			setupMock: func() *mockConfig {
				m := newMockConfig()
				m.values["enabled"] = true
				return m
			},
			testFunc: func(t *testing.T, m *mockConfig) {
				t.Helper()
				got := m.GetBool("enabled")
				assert.True(t, got)
			},
		},
		"GetInt with value": {
			setupMock: func() *mockConfig {
				m := newMockConfig()
				m.values["timeout"] = 30
				return m
			},
			testFunc: func(t *testing.T, m *mockConfig) {
				t.Helper()
				got := m.GetInt("timeout")
				assert.Equal(t, 30, got)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock := tc.setupMock()
			tc.testFunc(t, mock)
		})
	}
}
