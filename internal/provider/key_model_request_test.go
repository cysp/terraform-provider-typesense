package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/cysp/terraform-provider-typesense/internal/typesense-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestKeyModelToAPIKeySchema(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		model    provider.KeyModel
		expected typesense.ApiKeySchema
	}{
		"actions,collections: null": {
			model: provider.KeyModel{
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
			},
			expected: typesense.ApiKeySchema{},
		},
		"actions,collections: empty": {
			model: provider.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"actions,collections: actions": {
			model: provider.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense.ApiKeySchema{
				Actions:     []string{"*"},
				Collections: []string{},
			},
		},
		"actions,collections: collections": {
			model: provider.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
			},
			expected: typesense.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{"*"},
			},
		},
		"description: unknown": {
			model: provider.KeyModel{
				Description: types.StringUnknown(),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"description: null": {
			model: provider.KeyModel{
				Description: types.StringNull(),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"description: known": {
			model: provider.KeyModel{
				Description: types.StringValue("description"),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense.ApiKeySchema{
				Description: "description",
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"expires at: unknown": {
			model: provider.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				ExpiresAt:   types.Int64Unknown(),
			},
			expected: typesense.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"expires at: null": {
			model: provider.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				ExpiresAt:   types.Int64Null(),
			},
			expected: typesense.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"expires at: known": {
			model: provider.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				ExpiresAt:   types.Int64Value(0),
			},
			expected: typesense.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
				ExpiresAt:   typesense.NewOptInt64(0),
			},
		},
		"value: unknown": {
			model: provider.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				Value:       types.StringUnknown(),
			},
			expected: typesense.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"value: null": {
			model: provider.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				Value:       types.StringNull(),
			},
			expected: typesense.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"value: known": {
			model: provider.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				Value:       types.StringValue("value"),
			},
			expected: typesense.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
				Value:       typesense.NewOptString("value"),
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
