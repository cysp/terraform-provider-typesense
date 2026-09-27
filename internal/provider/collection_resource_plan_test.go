package provider //nolint:testpackage // Exercise resource plan validation with prior state.

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionFallbackUnknownConfigurationIsRejected(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		fields types.List
	}{
		{"unknown list", types.ListUnknown(CollectionFieldObjectType())},
		{"unknown element", types.ListValueMust(CollectionFieldObjectType(), []attr.Value{
			types.ObjectUnknown(CollectionFieldObjectType().AttrTypes),
		})},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			planned := collectionUpdateModel(t, []api.Field{{Name: ".*", Type: "auto", Optional: new(true)}}, "5s")
			configured := planned
			configured.Fields = test.fields

			schema := planned.ResourceSchema(t.Context())
			configPlan := tfsdk.Plan{Schema: schema}
			require.Empty(t, configPlan.Set(t.Context(), &configured))
			config := tfsdk.Config{Schema: schema, Raw: configPlan.Raw}

			diags := validatePlannedFallbackFieldConfig(t.Context(), config, planned.Fields)
			require.True(t, diags.HasError(), "%v", diags)
		})
	}
}
