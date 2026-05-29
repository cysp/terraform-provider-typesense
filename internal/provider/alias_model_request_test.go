package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	typesense_api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAliasModelToAPICollectionAliasSchema(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		model    provider.AliasModel
		expected typesense_api.CollectionAliasSchema
	}{
		"collection name: unknown": {
			model:    provider.AliasModel{CollectionName: types.StringUnknown()},
			expected: typesense_api.CollectionAliasSchema{},
		},
		"collection name: null": {
			model:    provider.AliasModel{CollectionName: types.StringNull()},
			expected: typesense_api.CollectionAliasSchema{},
		},
		"collection name: known": {
			model: provider.AliasModel{
				CollectionName: types.StringValue("posts_v1"),
			},
			expected: typesense_api.CollectionAliasSchema{
				CollectionName: "posts_v1",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expected, test.model.ToAPICollectionAliasSchema())
		})
	}
}
