package provider //nolint:testpackage // Exercise resource plan validation with prior state.

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

func TestCollectionFallbackPlanComparesEffectiveFields(t *testing.T) {
	t.Parallel()

	previous := collectionUpdateModel(t, []api.Field{{Name: ".*", Type: "auto", Optional: new(true)}}, "5s")

	previousField := CollectionFieldModelsFromAPI([]api.Field{{Name: ".*", Type: "auto", Optional: new(true)}})[0]
	// These options did not exist in older provider state.
	previousField.Store = types.BoolNull()
	previousField.RangeIndex = types.BoolNull()
	previousField.Stem = types.BoolNull()
	previousField.StemDictionary = types.StringNull()
	previousField.TokenSeparators = types.ListNull(types.StringType)
	previousField.SymbolsToIndex = types.ListNull(types.StringType)
	previousFields, fieldDiags := types.ListValueFrom(t.Context(), CollectionFieldObjectType(), []CollectionFieldModel{previousField})
	require.Empty(t, fieldDiags)

	previous.Fields = previousFields

	for _, test := range []struct {
		name, fieldType string
		wantError       bool
	}{
		{"new defaults match legacy state", "auto", false},
		{"actual fallback replacement", "string", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			planned := collectionUpdateModel(t, []api.Field{{Name: ".*", Type: test.fieldType, Optional: new(true)}}, "5s")
			schema := previous.ResourceSchema(t.Context())
			req := resource.ModifyPlanRequest{State: tfsdk.State{Schema: schema}, Plan: tfsdk.Plan{Schema: schema}}
			require.Empty(t, req.State.Set(t.Context(), &previous))
			require.Empty(t, req.Plan.Set(t.Context(), &planned))

			var resp resource.ModifyPlanResponse
			(&collectionResource{}).ModifyPlan(t.Context(), req, &resp)
			require.Equal(t, test.wantError, resp.Diagnostics.HasError())
		})
	}
}

func TestCollectionFallbackPlanAllowsCollectionReplacement(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		change func(*CollectionModel)
	}{
		{"name change", func(model *CollectionModel) { model.Name = types.StringValue("renamed") }},
		{"configured nested setting", func(model *CollectionModel) { model.EnableNestedFields = types.BoolValue(true) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			previous := collectionUpdateModel(t, []api.Field{{Name: ".*", Type: "auto", Optional: new(true)}}, "5s")
			planned := collectionUpdateModel(t, []api.Field{{Name: ".*", Type: "string", Optional: new(true)}}, "5s")
			test.change(&planned)

			schema := previous.ResourceSchema(t.Context())
			req := resource.ModifyPlanRequest{State: tfsdk.State{Schema: schema}, Plan: tfsdk.Plan{Schema: schema}}
			require.Empty(t, req.State.Set(t.Context(), &previous))
			require.Empty(t, req.Plan.Set(t.Context(), &planned))
			req.Config = tfsdk.Config{Schema: schema, Raw: req.Plan.Raw}

			var resp resource.ModifyPlanResponse
			(&collectionResource{}).ModifyPlan(t.Context(), req, &resp)
			require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
		})
	}
}

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
