package lib

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

func TestDeployInfo_GetCleanedStackName(t *testing.T) {
	type fields struct {
		Changeset            *ChangesetInfo
		ChangesetName        string
		IsDryRun             bool
		IsNew                bool
		Parameters           []types.Parameter
		PrechecksFailed      bool
		RawStack             *types.Stack
		StackArn             string
		StackName            string
		Tags                 []types.Tag
		Template             string
		TemplateLocalPath    string
		TemplateName         string
		TemplateRelativePath string
		TemplateUrl          string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"No arn in the stackname", fields{StackName: "test-stack"}, "test-stack"},
		{"Arn in the stackname", fields{StackName: "arn:aws:cloudformation:ap-southeast-2:12345678901:stack/test-stack/5f584530-013c-11ee-9c69-0a254d5985de"}, "test-stack"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := &DeployInfo{
				Changeset:            tt.fields.Changeset,
				ChangesetName:        tt.fields.ChangesetName,
				IsDryRun:             tt.fields.IsDryRun,
				IsNew:                tt.fields.IsNew,
				Parameters:           tt.fields.Parameters,
				PrechecksFailed:      tt.fields.PrechecksFailed,
				RawStack:             tt.fields.RawStack,
				StackArn:             tt.fields.StackArn,
				StackName:            tt.fields.StackName,
				Tags:                 tt.fields.Tags,
				Template:             tt.fields.Template,
				TemplateLocalPath:    tt.fields.TemplateLocalPath,
				TemplateName:         tt.fields.TemplateName,
				TemplateRelativePath: tt.fields.TemplateRelativePath,
				TemplateUrl:          tt.fields.TemplateUrl,
			}
			if got := deployment.GetCleanedStackName(); got != tt.want {
				t.Errorf("DeployInfo.GetCleanedStackName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeployInfo_GetExecutionTimes(t *testing.T) {
	type fields struct {
		Changeset            *ChangesetInfo
		ChangesetName        string
		IsDryRun             bool
		IsNew                bool
		Parameters           []types.Parameter
		PrechecksFailed      bool
		RawStack             *types.Stack
		StackArn             string
		StackName            string
		Tags                 []types.Tag
		Template             string
		TemplateLocalPath    string
		TemplateName         string
		TemplateRelativePath string
		TemplateUrl          string
	}
	type args struct {
		svc *cloudformation.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]map[string]time.Time
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := &DeployInfo{
				Changeset:            tt.fields.Changeset,
				ChangesetName:        tt.fields.ChangesetName,
				IsDryRun:             tt.fields.IsDryRun,
				IsNew:                tt.fields.IsNew,
				Parameters:           tt.fields.Parameters,
				PrechecksFailed:      tt.fields.PrechecksFailed,
				RawStack:             tt.fields.RawStack,
				StackArn:             tt.fields.StackArn,
				StackName:            tt.fields.StackName,
				Tags:                 tt.fields.Tags,
				Template:             tt.fields.Template,
				TemplateLocalPath:    tt.fields.TemplateLocalPath,
				TemplateName:         tt.fields.TemplateName,
				TemplateRelativePath: tt.fields.TemplateRelativePath,
				TemplateUrl:          tt.fields.TemplateUrl,
			}
			got, err := deployment.GetExecutionTimes(tt.args.svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeployInfo.GetExecutionTimes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeployInfo.GetExecutionTimes() = %v, want %v", got, tt.want)
			}
		})
	}
}
