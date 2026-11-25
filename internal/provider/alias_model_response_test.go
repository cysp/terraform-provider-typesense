package provider_test

import (
	"testing"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

func TestAliasModelReadFromResponse(t *testing.T) {
	t.Parallel()

	name := "posts"

	tests := map[string]struct {
		alias    api.CollectionAlias
		expected provider.AliasModel
	}{
		"empty": {
			alias: api.CollectionAlias{},
			expected: provider.AliasModel{
				Name:           types.StringNull(),
				CollectionName: types.StringValue(""),
			},
		},
		"name,collection": {
			alias: api.CollectionAlias{
				Name:           &name,
				CollectionName: "posts_v1",
			},
			expected: provider.AliasModel{
				Name:           types.StringValue("posts"),
				CollectionName: types.StringValue("posts_v1"),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			model := provider.AliasModel{}

			alias := test.alias
			model.ReadFromResponse(&alias)

			assert.Equal(t, test.expected, model)
		})
	}
}
