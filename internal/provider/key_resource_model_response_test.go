package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	typesense_api "github.com/typesense/typesense-go/typesense/api"
)

func TestKeyResourceModelReadFromResponse(t *testing.T) {
	t.Parallel()

	var (
		zero               int64
		value              = "value"
		prefix             = "prefix"
		farFutureTimestamp = util.FarFutureTimestamp
	)

	//nolint:dupl
	tests := map[string]struct {
		apiKey   typesense_api.ApiKey
		expected provider.KeyResourceModel
	}{
		"empty": {
			apiKey: typesense_api.ApiKey{},
			expected: provider.KeyResourceModel{
				Description: types.StringValue(""),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
			},
		},
		"id": {
			apiKey: typesense_api.ApiKey{
				Id: &zero,
			},
			expected: provider.KeyResourceModel{
				ID:          types.Int64Value(0),
				Description: types.StringValue(""),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
			},
		},
		"actions,collections: empty": {
			apiKey: typesense_api.ApiKey{
				Actions:     []string{},
				Collections: []string{},
			},
			expected: provider.KeyResourceModel{
				Description: types.StringValue(""),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
		},
		"actions,collections: actions": {
			apiKey: typesense_api.ApiKey{
				Actions:     []string{"*"},
				Collections: []string{},
			},
			expected: provider.KeyResourceModel{
				Description: types.StringValue(""),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
		},
		"actions,collections: collections": {
			apiKey: typesense_api.ApiKey{
				Actions:     []string{},
				Collections: []string{"*"},
			},
			expected: provider.KeyResourceModel{
				Description: types.StringValue(""),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
			},
		},
		"description": {
			apiKey: typesense_api.ApiKey{
				Description: "description",
			},
			expected: provider.KeyResourceModel{
				Description: types.StringValue("description"),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
			},
		},
		"expires at": {
			apiKey: typesense_api.ApiKey{
				ExpiresAt: &zero,
			},
			expected: provider.KeyResourceModel{
				Description: types.StringValue(""),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				ExpiresAt:   types.Int64Value(0),
			},
		},
		"expires at: far future": {
			apiKey: typesense_api.ApiKey{
				ExpiresAt: &farFutureTimestamp,
			},
			expected: provider.KeyResourceModel{
				Description: types.StringValue(""),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				ExpiresAt:   types.Int64Value(farFutureTimestamp),
			},
		},
		"value": {
			apiKey: typesense_api.ApiKey{
				Value: &value,
			},
			expected: provider.KeyResourceModel{
				Description: types.StringValue(""),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Value:       types.StringValue("value"),
				ValuePrefix: types.StringValue("valu"),
			},
		},
		"value,prefix": {
			apiKey: typesense_api.ApiKey{
				Value:       &value,
				ValuePrefix: &prefix,
			},
			expected: provider.KeyResourceModel{
				Description: types.StringValue(""),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Value:       types.StringValue("value"),
				ValuePrefix: types.StringValue("prefix"),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			model := provider.KeyResourceModel{}

			apiKey := test.apiKey
			diags := model.ReadFromResponse(t.Context(), &apiKey)

			assert.Empty(t, diags)
			assert.Equal(t, test.expected, model)
		})
	}
}
