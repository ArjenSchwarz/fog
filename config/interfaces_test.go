package config

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// TestAWSSDKClientsSatisfyInterfaces verifies that AWS SDK clients implement our config interfaces
func TestAWSSDKClientsSatisfyInterfaces(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		verifyFunc func(t *testing.T)
	}{
		"STS client satisfies STSGetCallerIdentityAPI": {
			verifyFunc: func(t *testing.T) {
				var _ STSGetCallerIdentityAPI = (*sts.Client)(nil)
			},
		},
		"IAM client satisfies IAMListAccountAliasesAPI": {
			verifyFunc: func(t *testing.T) {
				var _ IAMListAccountAliasesAPI = (*iam.Client)(nil)
			},
		},
	}

	for name, tc := range tests {
		tc := tc // capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tc.verifyFunc(t)
		})
	}
}

// Mock implementations for testing interface compliance

type mockAWSConfigLoader struct {
	loadDefaultConfigFn func(context.Context, ...func(*aws_config.LoadOptions) error) (aws.Config, error)
}

func (m *mockAWSConfigLoader) LoadDefaultConfig(ctx context.Context, optFns ...func(*aws_config.LoadOptions) error) (aws.Config, error) {
	if m.loadDefaultConfigFn != nil {
		return m.loadDefaultConfigFn(ctx, optFns...)
	}
	return aws.Config{}, nil
}

type mockSTSClient struct {
	getCallerIdentityFn func(context.Context, *sts.GetCallerIdentityInput, ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

func (m *mockSTSClient) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	if m.getCallerIdentityFn != nil {
		return m.getCallerIdentityFn(ctx, params, optFns...)
	}
	return &sts.GetCallerIdentityOutput{}, nil
}

type mockIAMClient struct {
	listAccountAliasesFn func(context.Context, *iam.ListAccountAliasesInput, ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error)
}

func (m *mockIAMClient) ListAccountAliases(ctx context.Context, params *iam.ListAccountAliasesInput, optFns ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
	if m.listAccountAliasesFn != nil {
		return m.listAccountAliasesFn(ctx, params, optFns...)
	}
	return &iam.ListAccountAliasesOutput{}, nil
}

type mockConfigReader struct {
	readFileFn  func(string) ([]byte, error)
	unmarshalFn func([]byte, any) error
}

func (m *mockConfigReader) ReadFile(filename string) ([]byte, error) {
	if m.readFileFn != nil {
		return m.readFileFn(filename)
	}
	return []byte{}, nil
}

func (m *mockConfigReader) Unmarshal(data []byte, v any) error {
	if m.unmarshalFn != nil {
		return m.unmarshalFn(data, v)
	}
	return nil
}

type mockViperConfig struct {
	isSetFn          func(string) bool
	getStringFn      func(string) string
	getStringSliceFn func(string) []string
	getBoolFn        func(string) bool
	getIntFn         func(string) int
	setConfigFileFn  func(string)
	readInConfigFn   func() error
}

func (m *mockViperConfig) IsSet(key string) bool {
	if m.isSetFn != nil {
		return m.isSetFn(key)
	}
	return false
}

func (m *mockViperConfig) GetString(key string) string {
	if m.getStringFn != nil {
		return m.getStringFn(key)
	}
	return ""
}

func (m *mockViperConfig) GetStringSlice(key string) []string {
	if m.getStringSliceFn != nil {
		return m.getStringSliceFn(key)
	}
	return []string{}
}

func (m *mockViperConfig) GetBool(key string) bool {
	if m.getBoolFn != nil {
		return m.getBoolFn(key)
	}
	return false
}

func (m *mockViperConfig) GetInt(key string) int {
	if m.getIntFn != nil {
		return m.getIntFn(key)
	}
	return 0
}

func (m *mockViperConfig) SetConfigFile(in string) {
	if m.setConfigFileFn != nil {
		m.setConfigFileFn(in)
	}
}

func (m *mockViperConfig) ReadInConfig() error {
	if m.readInConfigFn != nil {
		return m.readInConfigFn()
	}
	return nil
}

// TestMockImplementations verifies that mock implementations satisfy interfaces
func TestMockImplementations(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		verifyFunc func(t *testing.T)
	}{
		"mockAWSConfigLoader satisfies AWSConfigLoader": {
			verifyFunc: func(t *testing.T) {
				var _ AWSConfigLoader = (*mockAWSConfigLoader)(nil)
			},
		},
		"mockSTSClient satisfies STSGetCallerIdentityAPI": {
			verifyFunc: func(t *testing.T) {
				var _ STSGetCallerIdentityAPI = (*mockSTSClient)(nil)
			},
		},
		"mockIAMClient satisfies IAMListAccountAliasesAPI": {
			verifyFunc: func(t *testing.T) {
				var _ IAMListAccountAliasesAPI = (*mockIAMClient)(nil)
			},
		},
		"mockConfigReader satisfies ConfigReader": {
			verifyFunc: func(t *testing.T) {
				var _ ConfigReader = (*mockConfigReader)(nil)
			},
		},
		"mockViperConfig satisfies ViperConfigAPI": {
			verifyFunc: func(t *testing.T) {
				var _ ViperConfigAPI = (*mockViperConfig)(nil)
			},
		},
	}

	for name, tc := range tests {
		tc := tc // capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tc.verifyFunc(t)
		})
	}
}

// TestMockFunctionality verifies that mocks work as expected
func TestMockFunctionality(t *testing.T) {
	t.Helper()

	t.Run("mockSTSClient returns configured values", func(t *testing.T) {
		t.Parallel()

		accountID := "123456789012"
		userID := "AIDAI23HXS2O3EXAMPLE"

		mock := &mockSTSClient{
			getCallerIdentityFn: func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
				return &sts.GetCallerIdentityOutput{
					Account: &accountID,
					UserId:  &userID,
				}, nil
			},
		}

		ctx := context.Background()
		got, err := mock.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			t.Errorf("GetCallerIdentity() error = %v", err)
		}

		if *got.Account != accountID {
			t.Errorf("GetCallerIdentity() Account = %v, want %v", *got.Account, accountID)
		}

		if *got.UserId != userID {
			t.Errorf("GetCallerIdentity() UserId = %v, want %v", *got.UserId, userID)
		}
	})

	t.Run("mockIAMClient returns configured values", func(t *testing.T) {
		t.Parallel()

		aliases := []string{"my-account-alias"}

		mock := &mockIAMClient{
			listAccountAliasesFn: func(ctx context.Context, params *iam.ListAccountAliasesInput, optFns ...func(*iam.Options)) (*iam.ListAccountAliasesOutput, error) {
				return &iam.ListAccountAliasesOutput{
					AccountAliases: aliases,
				}, nil
			},
		}

		ctx := context.Background()
		got, err := mock.ListAccountAliases(ctx, &iam.ListAccountAliasesInput{})
		if err != nil {
			t.Errorf("ListAccountAliases() error = %v", err)
		}

		if len(got.AccountAliases) != len(aliases) {
			t.Errorf("ListAccountAliases() returned %d aliases, want %d", len(got.AccountAliases), len(aliases))
		}

		if got.AccountAliases[0] != aliases[0] {
			t.Errorf("ListAccountAliases() alias = %v, want %v", got.AccountAliases[0], aliases[0])
		}
	})

	t.Run("mockConfigReader returns configured values", func(t *testing.T) {
		t.Parallel()

		fileContent := []byte("test content")

		mock := &mockConfigReader{
			readFileFn: func(filename string) ([]byte, error) {
				return fileContent, nil
			},
		}

		got, err := mock.ReadFile("test.yaml")
		if err != nil {
			t.Errorf("ReadFile() error = %v", err)
		}

		if string(got) != string(fileContent) {
			t.Errorf("ReadFile() = %v, want %v", string(got), string(fileContent))
		}
	})

	t.Run("mockViperConfig returns configured values", func(t *testing.T) {
		t.Parallel()

		mock := &mockViperConfig{
			isSetFn: func(key string) bool {
				return key == "profile"
			},
			getStringFn: func(key string) string {
				if key == "profile" {
					return "default"
				}
				return ""
			},
		}

		if !mock.IsSet("profile") {
			t.Error("IsSet(\"profile\") = false, want true")
		}

		if mock.IsSet("nonexistent") {
			t.Error("IsSet(\"nonexistent\") = true, want false")
		}

		got := mock.GetString("profile")
		want := "default"
		if got != want {
			t.Errorf("GetString(\"profile\") = %v, want %v", got, want)
		}
	})
}
