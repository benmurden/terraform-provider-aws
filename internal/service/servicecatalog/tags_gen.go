// Code generated by internal/generate/tags/main.go; DO NOT EDIT.

package servicecatalog

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicecatalog"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

// []*SERVICE.Tag handling

// Tags returns servicecatalog service tags.
func Tags(tags tftags.KeyValueTags) []*servicecatalog.Tag {
	result := make([]*servicecatalog.Tag, 0, len(tags))

	for k, v := range tags.Map() {
		tag := &servicecatalog.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		}

		result = append(result, tag)
	}

	return result
}

// KeyValueTags creates tftags.KeyValueTags from servicecatalog service tags.
func KeyValueTags(tags []*servicecatalog.Tag) tftags.KeyValueTags {
	m := make(map[string]*string, len(tags))

	for _, tag := range tags {
		m[aws.StringValue(tag.Key)] = tag.Value
	}

	return tftags.New(m)
}