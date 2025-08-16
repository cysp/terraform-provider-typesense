package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/cysp/terraform-provider-typesense/internal/provider/util"
	"github.com/cysp/terraform-provider-typesense/internal/typesense-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestKeyModelReadFromResponse(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		apiKey   typesense.ApiKey
		expected provider.KeyModel
	}{
		"empty": {
			apiKey: typesense.ApiKey{},
			expected: provider.KeyModel{
				Description: types.StringValue(""),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
			},
		},
		"id": {
			apiKey: typesense.ApiKey{
				ID: typesense.NewOptInt64(0),
			},
			expected: provider.KeyModel{
				ID:          types.Int64Value(0),
				Description: types.StringValue(""),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
			},
		},
		"actions,collections: empty": {
			apiKey: typesense.ApiKey{
				Actions:     []string{},
				Collections: []string{},
			},
			expected: provider.KeyModel{
				Description: types.StringValue(""),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
		},
		"actions,collections: actions": {
			apiKey: typesense.ApiKey{
				Actions:     []string{"*"},
				Collections: []string{},
			},
			expected: provider.KeyModel{
				Description: types.StringValue(""),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{}),
			},
		},
		"actions,collections: collections": {
			apiKey: typesense.ApiKey{
				Actions:     []string{},
				Collections: []string{"*"},
			},
			expected: provider.KeyModel{
				Description: types.StringValue(""),
				Actions:     types.ListValueMust(types.StringType, []attr.Value{}),
				Collections: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
			},
		},
		"description": {
			apiKey: typesense.ApiKey{
				Description: "description",
			},
			expected: provider.KeyModel{
				Description: types.StringValue("description"),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
			},
		},
		"expires at": {
			apiKey: typesense.ApiKey{
				ExpiresAt: typesense.NewOptInt64(0),
			},
			expected: provider.KeyModel{
				Description: types.StringValue(""),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				ExpiresAt:   types.Int64Value(0),
			},
		},
		"expires at: far future": {
			apiKey: typesense.ApiKey{
				ExpiresAt: typesense.NewOptInt64(util.FarFutureTimestamp),
			},
			expected: provider.KeyModel{
				Description: types.StringValue(""),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				ExpiresAt:   types.Int64Value(util.FarFutureTimestamp),
			},
		},
		"value": {
			apiKey: typesense.ApiKey{
				Value: typesense.NewOptString("value"),
			},
			expected: provider.KeyModel{
				Description: types.StringValue(""),
				Actions:     types.ListNull(types.StringType),
				Collections: types.ListNull(types.StringType),
				Value:       types.StringValue("value"),
				ValuePrefix: types.StringValue("valu"),
			},
		},
		"value,prefix": {
			apiKey: typesense.ApiKey{
				Value:       typesense.NewOptString("value"),
				ValuePrefix: typesense.NewOptString("prefix"),
			},
			expected: provider.KeyModel{
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

			model := provider.KeyModel{}

			apiKey := test.apiKey
			diags := model.ReadFromResponse(t.Context(), &apiKey)

			assert.Empty(t, diags)
			assert.Equal(t, test.expected, model)
		})
	}
}
