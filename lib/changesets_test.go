package lib

import (
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
)

// func TestChangesetInfo_DeleteChangeset(t *testing.T) {
// 	type fields struct {
// 		Changes      []ChangesetChanges
// 		CreationTime time.Time
// 		HasModule    bool
// 		ID           string
// 		Name         string
// 		Status       string
// 		StatusReason string
// 		StackID      string
// 		StackName    string
// 	}
// 	type args struct {
// 		svc *cloudformation.Client
// 	}
// 	tests := []struct {
// 		name   string
// 		fields fields
// 		args   args
// 		want   bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			changeset := &ChangesetInfo{
// 				Changes:      tt.fields.Changes,
// 				CreationTime: tt.fields.CreationTime,
// 				HasModule:    tt.fields.HasModule,
// 				ID:           tt.fields.ID,
// 				Name:         tt.fields.Name,
// 				Status:       tt.fields.Status,
// 				StatusReason: tt.fields.StatusReason,
// 				StackID:      tt.fields.StackID,
// 				StackName:    tt.fields.StackName,
// 			}
// 			if got := changeset.DeleteChangeset(tt.args.svc); got != tt.want {
// 				t.Errorf("ChangesetInfo.DeleteChangeset() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestChangesetInfo_DeployChangeset(t *testing.T) {
// 	type fields struct {
// 		Changes      []ChangesetChanges
// 		CreationTime time.Time
// 		HasModule    bool
// 		ID           string
// 		Name         string
// 		Status       string
// 		StatusReason string
// 		StackID      string
// 		StackName    string
// 	}
// 	type args struct {
// 		svc *cloudformation.Client
// 	}
// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		args    args
// 		wantErr bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			changeset := &ChangesetInfo{
// 				Changes:      tt.fields.Changes,
// 				CreationTime: tt.fields.CreationTime,
// 				HasModule:    tt.fields.HasModule,
// 				ID:           tt.fields.ID,
// 				Name:         tt.fields.Name,
// 				Status:       tt.fields.Status,
// 				StatusReason: tt.fields.StatusReason,
// 				StackID:      tt.fields.StackID,
// 				StackName:    tt.fields.StackName,
// 			}
// 			if err := changeset.DeployChangeset(tt.args.svc); (err != nil) != tt.wantErr {
// 				t.Errorf("ChangesetInfo.DeployChangeset() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }

// func TestChangesetInfo_AddChange(t *testing.T) {
// 	type fields struct {
// 		Changes      []ChangesetChanges
// 		CreationTime time.Time
// 		HasModule    bool
// 		ID           string
// 		Name         string
// 		Status       string
// 		StatusReason string
// 		StackID      string
// 		StackName    string
// 	}
// 	type args struct {
// 		changes ChangesetChanges
// 	}
// 	tests := []struct {
// 		name   string
// 		fields fields
// 		args   args
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			changeset := &ChangesetInfo{
// 				Changes:      tt.fields.Changes,
// 				CreationTime: tt.fields.CreationTime,
// 				HasModule:    tt.fields.HasModule,
// 				ID:           tt.fields.ID,
// 				Name:         tt.fields.Name,
// 				Status:       tt.fields.Status,
// 				StatusReason: tt.fields.StatusReason,
// 				StackID:      tt.fields.StackID,
// 				StackName:    tt.fields.StackName,
// 			}
// 			changeset.AddChange(tt.args.changes)
// 		})
// 	}
// }

// func TestChangesetInfo_GetStack(t *testing.T) {
// 	type fields struct {
// 		Changes      []ChangesetChanges
// 		CreationTime time.Time
// 		HasModule    bool
// 		ID           string
// 		Name         string
// 		Status       string
// 		StatusReason string
// 		StackID      string
// 		StackName    string
// 	}
// 	type args struct {
// 		svc *cloudformation.Client
// 	}
// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		args    args
// 		want    types.Stack
// 		wantErr bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			changeset := &ChangesetInfo{
// 				Changes:      tt.fields.Changes,
// 				CreationTime: tt.fields.CreationTime,
// 				HasModule:    tt.fields.HasModule,
// 				ID:           tt.fields.ID,
// 				Name:         tt.fields.Name,
// 				Status:       tt.fields.Status,
// 				StatusReason: tt.fields.StatusReason,
// 				StackID:      tt.fields.StackID,
// 				StackName:    tt.fields.StackName,
// 			}
// 			got, err := changeset.GetStack(tt.args.svc)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("ChangesetInfo.GetStack() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("ChangesetInfo.GetStack() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

func TestChangesetInfo_GenerateChangesetUrl(t *testing.T) {
	type fields struct {
		Changes      []ChangesetChanges
		CreationTime time.Time
		HasModule    bool
		ID           string
		Name         string
		Status       string
		StatusReason string
		StackID      string
		StackName    string
	}
	type args struct {
		settings config.AWSConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changeset := &ChangesetInfo{
				Changes:      tt.fields.Changes,
				CreationTime: tt.fields.CreationTime,
				HasModule:    tt.fields.HasModule,
				ID:           tt.fields.ID,
				Name:         tt.fields.Name,
				Status:       tt.fields.Status,
				StatusReason: tt.fields.StatusReason,
				StackID:      tt.fields.StackID,
				StackName:    tt.fields.StackName,
			}
			if got := changeset.GenerateChangesetUrl(tt.args.settings); got != tt.want {
				t.Errorf("ChangesetInfo.GenerateChangesetUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetStackAndChangesetFromURL(t *testing.T) {
	type args struct {
		changeseturl string
		region       string
	}
	url1 := "https://ap-southeast-2.console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/changesets/changes?stackId=arn%3Aaws%3Acloudformation%3Aap-southeast-2%3A12345678901%3Astack%2Fvpc-go-demo%2F5f584530-013c-11ee-9c69-0a254d5985de&changeSetId=arn%3Aaws%3Acloudformation%3Aap-southeast-2%3A12345678901%3AchangeSet%2Ffog-2023-06-03T22-26-08%2F6fa5185b-1442-4c1d-b192-6e0d92891bd5"
	url2 := "https://us-east-1.console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/changesets/changes?stackId=arn%3Aaws%3Acloudformation%3Aus-east-1%3A12345678901%3Astack%2Fvpc-cb-demo%2F7c838330-d92e-11ed-b124-0ab9b6430001&changeSetId=arn%3Aaws%3Acloudformation%3Aus-east-1%3A12345678901%3AchangeSet%2Ffog-2023-04-13T19-19-09%2F85a00a3c-5aa7-4a26-95ca-a06cece6956e"
	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
	}{
		{"ap-southeast-2", args{changeseturl: url1, region: "ap-southeast-2"}, "arn:aws:cloudformation:ap-southeast-2:12345678901:stack/vpc-go-demo/5f584530-013c-11ee-9c69-0a254d5985de", "arn:aws:cloudformation:ap-southeast-2:12345678901:changeSet/fog-2023-06-03T22-26-08/6fa5185b-1442-4c1d-b192-6e0d92891bd5"},
		{"us-east-1", args{changeseturl: url2, region: "us-east-1"}, "arn:aws:cloudformation:us-east-1:12345678901:stack/vpc-cb-demo/7c838330-d92e-11ed-b124-0ab9b6430001", "arn:aws:cloudformation:us-east-1:12345678901:changeSet/fog-2023-04-13T19-19-09/85a00a3c-5aa7-4a26-95ca-a06cece6956e"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetStackAndChangesetFromURL(tt.args.changeseturl, tt.args.region)
			if got != tt.want {
				t.Errorf("GetStackAndChangesetFromURL() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetStackAndChangesetFromURL() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

// func TestChangesetChanges_GetDangerDetails(t *testing.T) {
// 	type fields struct {
// 		Action      string
// 		LogicalID   string
// 		Replacement string
// 		ResourceID  string
// 		Type        string
// 		Module      string
// 		Details     []types.ResourceChangeDetail
// 	}
// 	tests := []struct {
// 		name   string
// 		fields fields
// 		want   []string
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			changes := &ChangesetChanges{
// 				Action:      tt.fields.Action,
// 				LogicalID:   tt.fields.LogicalID,
// 				Replacement: tt.fields.Replacement,
// 				ResourceID:  tt.fields.ResourceID,
// 				Type:        tt.fields.Type,
// 				Module:      tt.fields.Module,
// 				Details:     tt.fields.Details,
// 			}
// 			if got := changes.GetDangerDetails(); !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("ChangesetChanges.GetDangerDetails() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
