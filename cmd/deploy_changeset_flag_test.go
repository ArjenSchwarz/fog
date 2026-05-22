package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/testutil"
	output "github.com/ArjenSchwarz/go-output/v2"
)

// TestDeployFlags_Validate_DeployChangeset is a regression test for T-865.
// The --deploy-changeset flag is supposed to deploy a specific existing
// changeset. It therefore requires --changeset to identify it and is mutually
// exclusive with flags that describe creation of a new changeset
// (--dry-run, --create-changeset, --template, --parameters, --tags,
// --deployment-file).
func TestDeployFlags_Validate_DeployChangeset(t *testing.T) {
	tests := map[string]struct {
		flags   DeployFlags
		wantErr bool
		errMsg  string
	}{
		"deploy-changeset with changeset name is valid": {
			flags: DeployFlags{
				StackName:       "my-stack",
				ChangesetName:   "my-changeset",
				DeployChangeset: true,
			},
			wantErr: false,
		},
		"deploy-changeset without changeset name fails": {
			flags: DeployFlags{
				StackName:       "my-stack",
				DeployChangeset: true,
			},
			wantErr: true,
			errMsg:  "--deploy-changeset requires --changeset",
		},
		"deploy-changeset with dry-run fails": {
			flags: DeployFlags{
				StackName:       "my-stack",
				ChangesetName:   "my-changeset",
				DeployChangeset: true,
				Dryrun:          true,
			},
			wantErr: true,
			errMsg:  "cannot be combined",
		},
		"deploy-changeset with create-changeset fails": {
			flags: DeployFlags{
				StackName:       "my-stack",
				ChangesetName:   "my-changeset",
				DeployChangeset: true,
				CreateChangeset: true,
			},
			wantErr: true,
			errMsg:  "cannot be combined",
		},
		"deploy-changeset with template fails": {
			flags: DeployFlags{
				StackName:       "my-stack",
				ChangesetName:   "my-changeset",
				DeployChangeset: true,
				Template:        "my-template",
			},
			wantErr: true,
			errMsg:  "cannot be combined",
		},
		"deploy-changeset with parameters fails": {
			flags: DeployFlags{
				StackName:       "my-stack",
				ChangesetName:   "my-changeset",
				DeployChangeset: true,
				Parameters:      "params.json",
			},
			wantErr: true,
			errMsg:  "cannot be combined",
		},
		"deploy-changeset with tags fails": {
			flags: DeployFlags{
				StackName:       "my-stack",
				ChangesetName:   "my-changeset",
				DeployChangeset: true,
				Tags:            "tags.json",
			},
			wantErr: true,
			errMsg:  "cannot be combined",
		},
		"deploy-changeset with deployment-file fails": {
			flags: DeployFlags{
				StackName:       "my-stack",
				ChangesetName:   "my-changeset",
				DeployChangeset: true,
				DeploymentFile:  "deploy.yaml",
			},
			wantErr: true,
			errMsg:  "cannot be combined",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.flags.Validate()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("expected error containing %q, got %q", tc.errMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestRunDeployChangesetFlow_DeployChangeset_SkipsCreation is a regression
// test for T-865. When --deploy-changeset is set, the deploy-changeset
// execution flow must NOT call createChangesetFunc (which creates a new
// changeset) and must instead retrieve and execute the existing changeset
// named by --changeset.
func TestRunDeployChangesetFlow_DeployChangeset_SkipsCreation(t *testing.T) {
	createCalled := false
	fetchCalled := false
	deployCalled := false
	showCalled := false

	origCreate := createChangesetFunc
	origFetch := fetchChangesetFunc
	origShow := showChangesetFunc
	origDeploy := deployChangesetFunc
	origAsk := askForConfirmationFunc
	origFlags := deployFlags
	defer func() {
		createChangesetFunc = origCreate
		fetchChangesetFunc = origFetch
		showChangesetFunc = origShow
		deployChangesetFunc = origDeploy
		askForConfirmationFunc = origAsk
		deployFlags = origFlags
	}()

	createChangesetFunc = func(info *lib.DeployInfo, cfg config.AWSConfig) *lib.ChangesetInfo {
		createCalled = true
		return &lib.ChangesetInfo{Name: "new-changeset"}
	}
	fetchChangesetFunc = func(info *lib.DeployInfo, cfg config.AWSConfig) *lib.ChangesetInfo {
		fetchCalled = true
		return &lib.ChangesetInfo{Name: "existing-changeset"}
	}
	showChangesetFunc = func(cs lib.ChangesetInfo, info lib.DeployInfo, cfg config.AWSConfig, optionalBuilder ...*output.Builder) {
		showCalled = true
	}
	deployChangesetFunc = func(info lib.DeployInfo, cfg config.AWSConfig) error {
		deployCalled = true
		return nil
	}
	askForConfirmationFunc = func(string) bool { return true }

	deployFlags = DeployFlags{
		StackName:       "my-stack",
		ChangesetName:   "existing-changeset",
		DeployChangeset: true,
	}

	info := lib.DeployInfo{
		StackName:     "my-stack",
		ChangesetName: "existing-changeset",
	}
	logObj := lib.DeploymentLog{}

	changeset := runDeployChangesetFlow(&info, config.AWSConfig{}, &logObj, false)

	if createCalled {
		t.Error("createChangesetFunc must not be called when --deploy-changeset is set")
	}
	if !fetchCalled {
		t.Error("fetchChangesetFunc must be called when --deploy-changeset is set")
	}
	if !showCalled {
		t.Error("existing changeset must be displayed before deployment")
	}
	if changeset == nil || changeset.Name != "existing-changeset" {
		t.Errorf("expected existing-changeset, got %+v", changeset)
	}
	if info.Changeset == nil || info.Changeset.Name != "existing-changeset" {
		t.Errorf("expected deployment.Changeset to be populated with existing changeset, got %+v", info.Changeset)
	}
	if info.CapturedChangeset == nil || info.CapturedChangeset.Name != "existing-changeset" {
		t.Errorf("expected deployment.CapturedChangeset to be populated with existing changeset, got %+v", info.CapturedChangeset)
	}
	// Ensure the caller can still confirm+deploy against the fetched changeset
	deployed, err := confirmAndDeployChangeset(changeset, &info, config.AWSConfig{})
	if err != nil {
		t.Fatalf("confirmAndDeployChangeset returned error: %v", err)
	}
	if !deployed {
		t.Error("expected confirmAndDeployChangeset to return deployed=true")
	}
	if !deployCalled {
		t.Error("expected deployChangesetFunc to be called")
	}
}

// TestRunDeployChangesetFlow_NilFetchReturn verifies that when
// fetchChangesetFunc returns nil (e.g. an error path that returned after
// calling osExitFunc, or a test stub), runDeployChangesetFlow returns nil
// gracefully instead of panicking on a nil dereference.
func TestRunDeployChangesetFlow_NilFetchReturn(t *testing.T) {
	origFetch := fetchChangesetFunc
	origShow := showChangesetFunc
	defer func() {
		fetchChangesetFunc = origFetch
		showChangesetFunc = origShow
	}()

	fetchChangesetFunc = func(info *lib.DeployInfo, cfg config.AWSConfig) *lib.ChangesetInfo {
		return nil
	}
	showCalled := false
	showChangesetFunc = func(cs lib.ChangesetInfo, info lib.DeployInfo, cfg config.AWSConfig, optionalBuilder ...*output.Builder) {
		showCalled = true
	}

	info := lib.DeployInfo{StackName: "my-stack", ChangesetName: "missing"}
	logObj := lib.DeploymentLog{}

	changeset := runDeployChangesetFlow(&info, config.AWSConfig{}, &logObj, false)
	if changeset != nil {
		t.Errorf("expected nil return when fetchChangesetFunc returns nil, got %+v", changeset)
	}
	if showCalled {
		t.Error("showChangesetFunc must not be called when fetchChangesetFunc returns nil")
	}
}

// TestDeployTemplate_NilDeployChangesetFetchAborts is a regression test for
// T-1252. When --deploy-changeset is set and fetchChangesetFunc returns nil
// (e.g. the changeset is missing and fetchChangeset returned after osExitFunc),
// deployTemplate must abort instead of continuing into deployment. Previously
// the nil changeset fell through to confirmAndDeployChangeset, which would
// invoke deployChangesetFunc and dereference the nil deployment.Changeset.
func TestDeployTemplate_NilDeployChangesetFetchAborts(t *testing.T) {
	origLoadAWSConfig := loadAWSConfig
	origGetCfnClient := getCfnClient
	origFetch := fetchChangesetFunc
	origDeploy := deployChangesetFunc
	origAsk := askForConfirmationFunc
	origOsExit := osExitFunc
	origFlags := deployFlags
	defer func() {
		loadAWSConfig = origLoadAWSConfig
		getCfnClient = origGetCfnClient
		fetchChangesetFunc = origFetch
		deployChangesetFunc = origDeploy
		askForConfirmationFunc = origAsk
		osExitFunc = origOsExit
		deployFlags = origFlags
	}()

	mockClient := testutil.NewMockCFNClient()
	mockClient.WithError(errors.New("stack not found"))

	loadAWSConfig = func(_ context.Context, _ config.Config) (config.AWSConfig, error) {
		return config.AWSConfig{AccountID: "123456789012", Region: "us-east-1"}, nil
	}
	getCfnClient = func(_ config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
		return mockClient
	}

	// Simulate the missing-changeset path: fetchChangeset reports the error and
	// exits via osExitFunc, but the stub returns nil so the flow can continue.
	exitCalled := false
	osExitFunc = func(int) { exitCalled = true }
	fetchChangesetFunc = func(_ *lib.DeployInfo, _ config.AWSConfig) *lib.ChangesetInfo {
		osExitFunc(1)
		return nil
	}

	deployCalled := false
	deployChangesetFunc = func(_ lib.DeployInfo, _ config.AWSConfig) error {
		deployCalled = true
		return nil
	}
	askForConfirmationFunc = func(string) bool { return true }

	deployFlags = DeployFlags{
		StackName:       "my-stack",
		ChangesetName:   "missing-changeset",
		DeployChangeset: true,
		NonInteractive:  true,
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("deployTemplate panicked on nil changeset: %v", r)
			}
		}()
		deployTemplate(nil, nil)
	}()

	if !exitCalled {
		t.Error("expected fetchChangeset to signal exit via osExitFunc")
	}
	if deployCalled {
		t.Error("deployChangesetFunc must NOT be called when fetchChangesetFunc returns nil")
	}
}
