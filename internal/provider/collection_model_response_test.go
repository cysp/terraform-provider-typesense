package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	typesense_api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionModelReadFromResponse(t *testing.T) {
	t.Parallel()

	var (
		createdAt          int64 = 1
		enableNestedFields       = true
		facet                    = true
		numDim                   = 3
		numDocuments       int64 = 2
		sort                     = true
		symbolsToIndex           = []string{"+"}
		tokenSeparators          = []string{"-"}
	)

	tests := map[string]struct {
		collection typesense_api.CollectionResponse
		expected   provider.CollectionModel
	}{
		"empty": {
			collection: typesense_api.CollectionResponse{},
			expected: provider.CollectionModel{
				Name:                types.StringValue(""),
				DefaultSortingField: types.StringNull(),
				EnableNestedFields:  types.BoolNull(),
				Fields:              types.ListValueMust(provider.CollectionFieldObjectType(), []attr.Value{}),
				SymbolsToIndex:      types.ListNull(types.StringType),
				TokenSeparators:     types.ListNull(types.StringType),
				CreatedAt:           types.Int64Null(),
				NumDocuments:        types.Int64Null(),
			},
		},
		"full": {
			collection: typesense_api.CollectionResponse{
				Name:                "posts",
				DefaultSortingField: types.StringValue("embedding").ValueStringPointer(),
				EnableNestedFields:  &enableNestedFields,
				Fields: []typesense_api.Field{
					{
						Name:   "embedding",
						Type:   "float[]",
						Facet:  &facet,
						NumDim: &numDim,
						Sort:   &sort,
					},
				},
				SymbolsToIndex:  &symbolsToIndex,
				TokenSeparators: &tokenSeparators,
				CreatedAt:       &createdAt,
				NumDocuments:    &numDocuments,
			},
			expected: provider.CollectionModel{
				Name:                types.StringValue("posts"),
				DefaultSortingField: types.StringValue("embedding"),
				EnableNestedFields:  types.BoolValue(true),
				Fields: types.ListValueMust(provider.CollectionFieldObjectType(), []attr.Value{
					types.ObjectValueMust(provider.CollectionFieldObjectType().AttrTypes, map[string]attr.Value{
						"name":             types.StringValue("embedding"),
						"type":             types.StringValue("float[]"),
						"facet":            types.BoolValue(true),
						"index":            types.BoolValue(true),
						"infix":            types.BoolValue(false),
						"locale":           types.StringValue(""),
						"num_dim":          types.Int64Value(3),
						"optional":         types.BoolValue(false),
						"reference":        types.StringNull(),
						"sort":             types.BoolValue(true),
						"store":            types.BoolValue(true),
						"range_index":      types.BoolValue(false),
						"stem":             types.BoolValue(false),
						"stem_dictionary":  types.StringValue(""),
						"vec_dist":         types.StringValue("cosine"),
						"token_separators": types.ListValueMust(types.StringType, []attr.Value{}),
						"symbols_to_index": types.ListValueMust(types.StringType, []attr.Value{}),
					}),
				}),
				SymbolsToIndex:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("+")}),
				TokenSeparators: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("-")}),
				CreatedAt:       types.Int64Value(1),
				NumDocuments:    types.Int64Value(2),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			model := provider.CollectionModel{}

			collection := test.collection
			diags := model.ReadFromResponse(t.Context(), &collection)

			assert.Empty(t, diags)
			assert.Equal(t, test.expected, model)
		})
	}
}
