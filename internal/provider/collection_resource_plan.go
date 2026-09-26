package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	api "github.com/typesense/typesense-go/v3/typesense/api"
)

var _ resource.ResourceWithModifyPlan = (*collectionResource)(nil)

func (r *collectionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var previous, planned CollectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planned)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !previous.Name.Equal(planned.Name) {
		return
	}

	if req.Config.Schema != nil {
		var configured CollectionModel
		resp.Diagnostics.Append(req.Config.Get(ctx, &configured)...)

		if resp.Diagnostics.HasError() || collectionConfiguredReplacement(configured, previous, planned) {
			return
		}
	}

	oldFallback, oldFound := collectionExactFallback(previous.Fields)
	newFallback, newFound := collectionExactFallback(planned.Fields)

	if !oldFound || !newFound || !collectionFieldValueKnown(oldFallback) || !collectionFieldValueKnown(newFallback) {
		return
	}

	oldField, oldDiags := collectionFallbackAPIField(ctx, oldFallback)

	newField, newDiags := collectionFallbackAPIField(ctx, newFallback)
	if oldDiags.HasError() || newDiags.HasError() {
		// Apply checks the live schema. A plan whose field values cannot be
		// converted should not be rejected here as a fallback replacement.
		return
	}

	if !sameCollectionField(oldField, newField) {
		resp.Diagnostics.AddAttributeError(path.Root("fields"), "Cannot replace Typesense fallback in one update", "The exact .* fallback cannot be dropped and readded in one Typesense schema alteration. Remove it in one apply and add the desired declaration in a second apply, or use a new collection and alias cutover.")
	}
}

// Schema plan modifiers run before ModifyPlan, but their replacement paths are
// not passed to it. Mirror the collection-level replacement conditions here.
func collectionConfiguredReplacement(configured, previous, planned CollectionModel) bool {
	return !configured.DefaultSortingField.IsNull() && !previous.DefaultSortingField.Equal(planned.DefaultSortingField) ||
		!configured.EnableNestedFields.IsNull() && !previous.EnableNestedFields.Equal(planned.EnableNestedFields) ||
		!configured.SymbolsToIndex.IsNull() && !previous.SymbolsToIndex.Equal(planned.SymbolsToIndex) ||
		!configured.TokenSeparators.IsNull() && !previous.TokenSeparators.Equal(planned.TokenSeparators)
}

func collectionFallbackAPIField(ctx context.Context, value types.Object) (api.Field, diag.Diagnostics) {
	var model CollectionFieldModel

	diags := value.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return api.Field{}, diags
	}

	field, fieldDiags := model.ToAPIField(ctx)
	diags.Append(fieldDiags...)

	return field, diags
}

func collectionExactFallback(fields types.List) (types.Object, bool) {
	if fields.IsNull() || fields.IsUnknown() {
		return types.Object{}, false
	}

	for _, value := range fields.Elements() {
		if value.IsNull() || value.IsUnknown() {
			continue
		}

		field := value.(types.Object)                     //nolint:forcetypeassert // The schema guarantees object elements.
		name := field.Attributes()["name"].(types.String) //nolint:forcetypeassert // The schema guarantees a string name.

		if !name.IsNull() && !name.IsUnknown() && name.ValueString() == ".*" {
			return field, true
		}
	}

	return types.Object{}, false
}

func collectionFieldValueKnown(field types.Object) bool {
	for _, value := range field.Attributes() {
		if value.IsUnknown() {
			return false
		}
	}

	return true
}
