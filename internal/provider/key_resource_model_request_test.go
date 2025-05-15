package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	typesense_api "github.com/typesense/typesense-go/typesense/api"
)

func TestKeyResourceModelToAPIKeySchema(t *testing.T) {
	t.Parallel()

	var (
		zero  int64
		value = "value"
	)

	tests := map[string]struct {
		model    provider.KeyResourceModel
		expected typesense_api.ApiKeySchema
	}{
		"actions,collections: null": {
			model: provider.KeyResourceModel{
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
			},
			expected: typesense_api.ApiKeySchema{},
		},
		"actions,collections: empty": {
			model: provider.KeyResourceModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"actions,collections: actions": {
			model: provider.KeyResourceModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{"*"},
				Collections: []string{},
			},
		},
		"actions,collections: collections": {
			model: provider.KeyResourceModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{"*"},
			},
		},
		"description: unknown": {
			model: provider.KeyResourceModel{
				Description: types.StringUnknown(),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"description: null": {
			model: provider.KeyResourceModel{
				Description: types.StringNull(),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"description: known": {
			model: provider.KeyResourceModel{
				Description: types.StringValue("description"),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense_api.ApiKeySchema{
				Description: "description",
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"expires at: unknown": {
			model: provider.KeyResourceModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				ExpiresAt:   types.Int64Unknown(),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"expires at: null": {
			model: provider.KeyResourceModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				ExpiresAt:   types.Int64Null(),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"expires at: known": {
			model: provider.KeyResourceModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				ExpiresAt:   types.Int64Value(0),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
				ExpiresAt:   &zero,
			},
		},
		"value: unknown": {
			model: provider.KeyResourceModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				Value:       types.StringUnknown(),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"value: null": {
			model: provider.KeyResourceModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				Value:       types.StringNull(),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"value: known": {
			model: provider.KeyResourceModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				Value:       types.StringValue("value"),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
				Value:       &value,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			apiKeySchema, diags := test.model.ToAPIKeySchema(t.Context())

			assert.Empty(t, diags)
			assert.Equal(t, test.expected, apiKeySchema)
		})
	}
}
