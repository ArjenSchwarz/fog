package deployment

import (
	"context"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	cfnTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// TagService implements services.TagService with placeholder logic.
type TagService struct{}

// NewTagService creates a new TagService.
func NewTagService() *TagService { return &TagService{} }

// LoadTags loads tags from files. Placeholder implementation.
func (t *TagService) LoadTags(ctx context.Context, tagFiles []string, defaults map[string]string) ([]cfnTypes.Tag, ferr.FogError) {
	_ = ctx
	_ = tagFiles
	// Real implementation would merge defaults and file contents
	tags := make([]cfnTypes.Tag, 0, len(defaults))
	for k, v := range defaults {
		// copy loop vars so pointers remain stable
		key := k
		value := v
		tags = append(tags, cfnTypes.Tag{Key: &key, Value: &value})
	}
	return tags, nil
}

// ValidateTags validates the provided tags. Placeholder implementation.
func (t *TagService) ValidateTags(ctx context.Context, tags []cfnTypes.Tag) ferr.FogError {
	_ = ctx
	_ = tags
	return nil
}
