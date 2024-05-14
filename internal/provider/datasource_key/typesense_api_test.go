//nolint:revive,stylecheck
package datasource_key_test

import (
	"context"
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider/datasource_key"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	typesense_api "github.com/typesense/typesense-go/typesense/api"
)

func TestReadFromResponse(t *testing.T) {
	t.Parallel()

	var (
		zero   int64  = 0
		value  string = "value"
		prefix string = "prefix"
	)

	tests := map[string]struct {
		apiKey   typesense_api.ApiKey
		expected datasource_key.KeyModel
	}{
		"empty": {
			apiKey: typesense_api.ApiKey{},
			expected: datasource_key.KeyModel{
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Description: types.StringValue(""),
			},
		},
		"id": {
			apiKey: typesense_api.ApiKey{
				Id: &zero,
			},
			expected: datasource_key.KeyModel{
				Id:          types.Int64Value(0),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Description: types.StringValue(""),
			},
		},
		"actions,collections: empty": {
			apiKey: typesense_api.ApiKey{
				Actions:     []string{},
				Collections: []string{},
			},
			expected: datasource_key.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				Description: types.StringValue(""),
			},
		},
		"actions,collections: actions": {
			apiKey: typesense_api.ApiKey{
				Actions:     []string{"*"},
				Collections: []string{},
			},
			expected: datasource_key.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				Description: types.StringValue(""),
			},
		},
		"actions,collections: collections": {
			apiKey: typesense_api.ApiKey{
				Actions:     []string{},
				Collections: []string{"*"},
			},
			expected: datasource_key.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
				Description: types.StringValue(""),
			},
		},
		"description": {
			apiKey: typesense_api.ApiKey{
				Description: "description",
			},
			expected: datasource_key.KeyModel{
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Description: types.StringValue("description"),
			},
		},
		"value": {
			apiKey: typesense_api.ApiKey{
				Value: &value,
			},
			expected: datasource_key.KeyModel{
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Description: types.StringValue(""),
				Value:       types.StringValue("value"),
				ValuePrefix: types.StringValue("valu"),
			},
		},
		"value,prefix": {
			apiKey: typesense_api.ApiKey{
				Value:       &value,
				ValuePrefix: &prefix,
			},
			expected: datasource_key.KeyModel{
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Description: types.StringValue(""),
				Value:       types.StringValue("value"),
				ValuePrefix: types.StringValue("prefix"),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			model := datasource_key.KeyModel{}

			apiKey := test.apiKey
			diags := model.ReadFromResponse(context.Background(), &apiKey)

			assert.Empty(t, diags)
			assert.Equal(t, test.expected, model)
		})
	}
}
