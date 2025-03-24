//nolint:revive,stylecheck
package resource_key_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider/resource_key"
	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	typesense_api "github.com/typesense/typesense-go/typesense/api"
)

func TestKeyModelToAPIKeySchema(t *testing.T) {
	t.Parallel()

	var (
		zero  int64  = 0
		value string = "value"
	)

	tests := map[string]struct {
		model    resource_key.KeyModel
		expected typesense_api.ApiKeySchema
	}{
		"actions,collections: null": {
			model: resource_key.KeyModel{
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
			},
			expected: typesense_api.ApiKeySchema{},
		},
		"actions,collections: empty": {
			model: resource_key.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"actions,collections: actions": {
			model: resource_key.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{"*"},
				Collections: []string{},
			},
		},
		"actions,collections: collections": {
			model: resource_key.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{"*"},
			},
		},
		"description: unknown": {
			model: resource_key.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				Description: types.StringUnknown(),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"description: null": {
			model: resource_key.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				Description: types.StringNull(),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
			},
		},
		"description: known": {
			model: resource_key.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
				Description: types.StringValue("description"),
			},
			expected: typesense_api.ApiKeySchema{
				Actions:     []string{},
				Collections: []string{},
				Description: "description",
			},
		},
		"expires at: unknown": {
			model: resource_key.KeyModel{
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
			model: resource_key.KeyModel{
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
			model: resource_key.KeyModel{
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
			model: resource_key.KeyModel{
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
			model: resource_key.KeyModel{
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
			model: resource_key.KeyModel{
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
			assert.EqualValues(t, test.expected, apiKeySchema)
		})
	}
}

func TestReadFromResponse(t *testing.T) {
	t.Parallel()

	var (
		zero               int64  = 0
		value              string = "value"
		prefix             string = "prefix"
		farFutureTimestamp int64  = util.FarFutureTimestamp
	)

	tests := map[string]struct {
		apiKey   typesense_api.ApiKey
		expected resource_key.KeyModel
	}{
		"empty": {
			apiKey: typesense_api.ApiKey{},
			expected: resource_key.KeyModel{
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Description: types.StringValue(""),
			},
		},
		"id": {
			apiKey: typesense_api.ApiKey{
				Id: &zero,
			},
			expected: resource_key.KeyModel{
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
			expected: resource_key.KeyModel{
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
			expected: resource_key.KeyModel{
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
			expected: resource_key.KeyModel{
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
				Description: types.StringValue(""),
			},
		},
		"description": {
			apiKey: typesense_api.ApiKey{
				Description: "description",
			},
			expected: resource_key.KeyModel{
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Description: types.StringValue("description"),
			},
		},
		"expires at": {
			apiKey: typesense_api.ApiKey{
				ExpiresAt: &zero,
			},
			expected: resource_key.KeyModel{
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Description: types.StringValue(""),
				ExpiresAt:   types.Int64Value(0),
			},
		},
		"expires at: far future": {
			apiKey: typesense_api.ApiKey{
				ExpiresAt: &farFutureTimestamp,
			},
			expected: resource_key.KeyModel{
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Description: types.StringValue(""),
				ExpiresAt:   types.Int64Value(farFutureTimestamp),
			},
		},
		"value": {
			apiKey: typesense_api.ApiKey{
				Value: &value,
			},
			expected: resource_key.KeyModel{
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
			expected: resource_key.KeyModel{
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

			model := resource_key.KeyModel{}

			apiKey := test.apiKey
			diags := model.ReadFromResponse(t.Context(), &apiKey)

			assert.Empty(t, diags)
			assert.Equal(t, test.expected, model)
		})
	}
}
