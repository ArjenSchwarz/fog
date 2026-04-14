package cmd

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// TestAwsManagedPrefixListIDs_NilFields verifies that the filtering helper
// does not panic when OwnerId or PrefixListId is nil, as can happen with
// partial EC2 SDK responses.
func TestAwsManagedPrefixListIDs_NilFields(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input []ec2types.ManagedPrefixList
		want  []string
	}{
		"nil_OwnerId_skipped": {
			input: []ec2types.ManagedPrefixList{
				{OwnerId: nil, PrefixListId: aws.String("pl-111")},
			},
			want: []string{},
		},
		"nil_PrefixListId_skipped": {
			input: []ec2types.ManagedPrefixList{
				{OwnerId: aws.String("AWS"), PrefixListId: nil},
			},
			want: []string{},
		},
		"both_nil_skipped": {
			input: []ec2types.ManagedPrefixList{
				{OwnerId: nil, PrefixListId: nil},
			},
			want: []string{},
		},
		"empty_PrefixListId_skipped": {
			input: []ec2types.ManagedPrefixList{
				{OwnerId: aws.String("AWS"), PrefixListId: aws.String("")},
			},
			want: []string{},
		},
		"non_aws_owner_excluded": {
			input: []ec2types.ManagedPrefixList{
				{OwnerId: aws.String("123456789012"), PrefixListId: aws.String("pl-222")},
			},
			want: []string{},
		},
		"aws_owned_included": {
			input: []ec2types.ManagedPrefixList{
				{OwnerId: aws.String("AWS"), PrefixListId: aws.String("pl-333")},
			},
			want: []string{"pl-333"},
		},
		"mixed_entries": {
			input: []ec2types.ManagedPrefixList{
				{OwnerId: nil, PrefixListId: aws.String("pl-nil-owner")},
				{OwnerId: aws.String("AWS"), PrefixListId: nil},
				{OwnerId: aws.String("AWS"), PrefixListId: aws.String("pl-good")},
				{OwnerId: aws.String("123456789012"), PrefixListId: aws.String("pl-customer")},
				{OwnerId: aws.String("AWS"), PrefixListId: aws.String("")},
			},
			want: []string{"pl-good"},
		},
		"empty_list": {
			input: []ec2types.ManagedPrefixList{},
			want:  []string{},
		},
		"nil_list": {
			input: nil,
			want:  []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := awsManagedPrefixListIDs(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d IDs %v, want %d IDs %v", len(got), got, len(tc.want), tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
