package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	typesense_api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionModelToAPICollectionSchema(t *testing.T) {
	t.Parallel()

	var (
		enableNestedFields = true
		facet              = true
		numDim             = 3
		sort               = true
		symbolsToIndex     = []string{"+"}
		tokenSeparators    = []string{"-"}
	)

	tests := map[string]struct {
		model    provider.CollectionModel
		expected typesense_api.CollectionSchema
	}{
		"minimal": {
			model: provider.CollectionModel{
				Name:   types.StringValue("posts"),
				Fields: types.ListValueMust(provider.CollectionFieldObjectType(), []attr.Value{}),
			},
			expected: typesense_api.CollectionSchema{
				Name: "posts",
			},
		},
		"full": {
			model: provider.CollectionModel{
				Name: types.StringValue("posts"),
				Fields: types.ListValueMust(provider.CollectionFieldObjectType(), []attr.Value{
					types.ObjectValueMust(provider.CollectionFieldObjectType().AttrTypes, map[string]attr.Value{
						"name":      types.StringValue("embedding"),
						"type":      types.StringValue("float[]"),
						"facet":     types.BoolValue(true),
						"index":     types.BoolNull(),
						"infix":     types.BoolNull(),
						"locale":    types.StringNull(),
						"num_dim":   types.Int64Value(3),
						"optional":  types.BoolNull(),
						"reference": types.StringNull(),
						"sort":      types.BoolValue(true),
					}),
				}),
				DefaultSortingField: types.StringValue("embedding"),
				EnableNestedFields:  types.BoolValue(true),
				SymbolsToIndex:      types.ListValueMust(types.StringType, []attr.Value{types.StringValue("+")}),
				TokenSeparators:     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("-")}),
			},
			expected: typesense_api.CollectionSchema{
				Name: "posts",
				Fields: []typesense_api.Field{
					{
						Name:   "embedding",
						Type:   "float[]",
						Facet:  &facet,
						NumDim: &numDim,
						Sort:   &sort,
					},
				},
				DefaultSortingField: types.StringValue("embedding").ValueStringPointer(),
				EnableNestedFields:  &enableNestedFields,
				SymbolsToIndex:      &symbolsToIndex,
				TokenSeparators:     &tokenSeparators,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			schema, diags := test.model.ToAPICollectionSchema(t.Context())

			assert.Empty(t, diags)
			assert.Equal(t, test.expected, schema)
		})
	}
}
